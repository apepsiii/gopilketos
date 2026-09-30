package services

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type GDriveConfig struct {
	Enabled             bool
	FolderID            string
	ServiceAccountJSON  string
	DeleteLocal         bool
	AutoApprove         bool
	TestimonialsEnabled bool
}

type GDriveClient struct {
	config     GDriveConfig
	httpClient *http.Client
}

type serviceAccountKey struct {
	Type        string `json:"type"`
	ProjectID   string `json:"project_id"`
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
	TokenURI    string `json:"token_uri"`
}

type oauthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error,omitempty"`
	ErrorDesc   string `json:"error_description,omitempty"`
}

type gdriveUploadResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// NewGDriveClient loads configuration from settings table
func NewGDriveClient(db *sql.DB) (*GDriveClient, error) {
	var (
		enabledNull     sql.NullInt64
		folderIDNull    sql.NullString
		saJSONNull      sql.NullString
		deleteLocalNull sql.NullInt64
		autoApproveNull sql.NullInt64
		testimNull      sql.NullInt64
	)

	_ = db.QueryRow(`SELECT 
		COALESCE(gdrive_enabled, 0),
		COALESCE(gdrive_folder_id, ''),
		COALESCE(gdrive_service_account_json, ''),
		COALESCE(gdrive_delete_local, 0),
		COALESCE(testimonials_auto_approve, 1),
		COALESCE(testimonials_enabled, 1)
		FROM settings LIMIT 1`).Scan(
		&enabledNull,
		&folderIDNull,
		&saJSONNull,
		&deleteLocalNull,
		&autoApproveNull,
		&testimNull,
	)

	cfg := GDriveConfig{
		Enabled:             enabledNull.Int64 == 1,
		FolderID:            strings.TrimSpace(folderIDNull.String),
		ServiceAccountJSON:  strings.TrimSpace(saJSONNull.String),
		DeleteLocal:         deleteLocalNull.Int64 == 1,
		AutoApprove:         autoApproveNull.Int64 == 1,
		TestimonialsEnabled: testimNull.Int64 == 1,
	}

	return &GDriveClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 90 * time.Second, // Allow sufficient time for video uploads
		},
	}, nil
}

// GetConfig returns the active configuration
func (c *GDriveClient) GetConfig() GDriveConfig {
	return c.config
}

// getAccessToken authenticates using Google Service Account JWT bearer flow
func (c *GDriveClient) getAccessToken() (string, error) {
	if c.config.ServiceAccountJSON == "" {
		return "", fmt.Errorf("Service Account JSON belum dikonfigurasi")
	}

	var sa serviceAccountKey
	if err := json.Unmarshal([]byte(c.config.ServiceAccountJSON), &sa); err != nil {
		return "", fmt.Errorf("format Service Account JSON tidak valid: %v", err)
	}

	if sa.ClientEmail == "" || sa.PrivateKey == "" {
		return "", fmt.Errorf("Service Account JSON wajib memiliki client_email dan private_key")
	}

	tokenURI := sa.TokenURI
	if tokenURI == "" {
		tokenURI = "https://oauth2.googleapis.com/token"
	}

	// Parse RSA private key
	block, _ := pem.Decode([]byte(sa.PrivateKey))
	if block == nil {
		return "", fmt.Errorf("gagal mendecode PEM private key")
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS1
		var err2 error
		parsedKey, err2 = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err2 != nil {
			return "", fmt.Errorf("gagal mem-parsing private key RSA: %v", err)
		}
	}

	rsaKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("private key bukan RSA")
	}

	// Build JWT header & claims
	now := time.Now().Unix()
	headerBytes, _ := json.Marshal(map[string]string{
		"alg": "RS256",
		"typ": "JWT",
	})
	claimsBytes, _ := json.Marshal(map[string]interface{}{
		"iss":   sa.ClientEmail,
		"scope": "https://www.googleapis.com/auth/drive.file",
		"aud":   tokenURI,
		"exp":   now + 3600,
		"iat":   now,
	})

	unsignedToken := fmt.Sprintf("%s.%s",
		base64.RawURLEncoding.EncodeToString(headerBytes),
		base64.RawURLEncoding.EncodeToString(claimsBytes),
	)

	// Sign with RS256
	hasher := sha256.New()
	hasher.Write([]byte(unsignedToken))
	digest := hasher.Sum(nil)

	signature, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, digest)
	if err != nil {
		return "", fmt.Errorf("gagal menandatangani JWT: %v", err)
	}

	signedAssertion := fmt.Sprintf("%s.%s",
		unsignedToken,
		base64.RawURLEncoding.EncodeToString(signature),
	)

	// Exchange JWT for access token
	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", signedAssertion)

	resp, err := c.httpClient.PostForm(tokenURI, form)
	if err != nil {
		return "", fmt.Errorf("gagal menghubungi oauth2.googleapis.com: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("google oauth error (%d): %s", resp.StatusCode, string(body))
	}

	var tokenResp oauthTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("gagal mem-parsing response token: %v", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("access token kosong: %s", tokenResp.ErrorDesc)
	}

	return tokenResp.AccessToken, nil
}

// UploadVideo uploads a local video file to Google Drive and sets public read permission
func (c *GDriveClient) UploadVideo(localFilePath, filename, mimeType string) (fileID, previewURL string, err error) {
	if !c.config.Enabled {
		return "", "", fmt.Errorf("sinkronisasi Google Drive dinonaktifkan")
	}

	file, err := os.Open(localFilePath)
	if err != nil {
		return "", "", fmt.Errorf("gagal membuka file lokal: %v", err)
	}
	defer file.Close()

	accessToken, err := c.getAccessToken()
	if err != nil {
		return "", "", fmt.Errorf("gagal autentikasi Google Drive: %v", err)
	}

	// Prepare multipart body for upload
	bodyBuf := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuf)

	// Part 1: Metadata (JSON)
	metaPartHeader := make(textproto.MIMEHeader)
	metaPartHeader.Set("Content-Type", "application/json; charset=UTF-8")
	metaPart, err := writer.CreatePart(metaPartHeader)
	if err != nil {
		return "", "", err
	}

	metaObj := map[string]interface{}{
		"name": filename,
	}
	if c.config.FolderID != "" {
		metaObj["parents"] = []string{c.config.FolderID}
	}
	metaBytes, _ := json.Marshal(metaObj)
	metaPart.Write(metaBytes)

	// Part 2: Media binary
	if mimeType == "" {
		mimeType = "video/webm"
	}
	mediaPartHeader := make(textproto.MIMEHeader)
	mediaPartHeader.Set("Content-Type", mimeType)
	mediaPart, err := writer.CreatePart(mediaPartHeader)
	if err != nil {
		return "", "", err
	}

	if _, err := io.Copy(mediaPart, file); err != nil {
		file.Close()
		return "", "", fmt.Errorf("gagal menyalin isi video ke request: %v", err)
	}
	file.Close()

	if err := writer.Close(); err != nil {
		return "", "", err
	}

	// Upload request
	uploadURL := "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart&fields=id,name,webViewLink"
	req, err := http.NewRequest("POST", uploadURL, bodyBuf)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	uploadResp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("gagal mengunggah video ke Drive: %v", err)
	}
	defer uploadResp.Body.Close()

	respBody, _ := io.ReadAll(uploadResp.Body)
	if uploadResp.StatusCode != http.StatusOK && uploadResp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("drive upload error (%d): %s", uploadResp.StatusCode, string(respBody))
	}

	var upResult gdriveUploadResponse
	if err := json.Unmarshal(respBody, &upResult); err != nil {
		return "", "", fmt.Errorf("gagal mem-parsing response upload Drive: %v", err)
	}

	fileID = upResult.ID
	previewURL = fmt.Sprintf("https://drive.google.com/file/d/%s/preview", fileID)

	// Set public reader permission so anyone can stream/view video
	c.setFilePublicRead(accessToken, fileID)

	// Delete local file if configured to save server disk space
	if c.config.DeleteLocal {
		_ = file.Close()
		_ = os.Remove(localFilePath)
	}

	return fileID, previewURL, nil
}

// setFilePublicRead makes the uploaded Google Drive file publicly viewable
func (c *GDriveClient) setFilePublicRead(accessToken, fileID string) {
	permURL := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s/permissions", fileID)
	permPayload := []byte(`{"role":"reader","type":"anyone"}`)

	req, err := http.NewRequest("POST", permURL, bytes.NewBuffer(permPayload))
	if err == nil {
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.httpClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}
}

// TestConnection tests service account authentication and folder access
func (c *GDriveClient) TestConnection() error {
	token, err := c.getAccessToken()
	if err != nil {
		return err
	}

	// Check folder if specified
	if c.config.FolderID != "" {
		folderURL := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s?fields=id,name", c.config.FolderID)
		req, _ := http.NewRequest("GET", folderURL, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("koneksi berhasil tapi gagal mengecek folder: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("folder ID tidak ditemukan atau akun service belum memiliki akses (status %d): %s", resp.StatusCode, string(body))
		}
	}

	return nil
}

// FindVideosUploadDir returns the absolute path for local video storage
func FindVideosUploadDir() string {
	exeDir, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(exeDir), "public", "uploads", "videos")
		if err := os.MkdirAll(dir, 0755); err == nil {
			return dir
		}
	}
	dir := filepath.Join("public", "uploads", "videos")
	_ = os.MkdirAll(dir, 0755)
	return dir
}
