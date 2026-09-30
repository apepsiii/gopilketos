package services

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// WhatsAppConfig holds connection settings for WhatsApp Gateway (GOWA or OneSender)
type WhatsAppConfig struct {
	Enabled   bool
	Provider  string // "gowa" (default) or "onesender"
	URL       string // default: http://127.0.0.1:8054/api/whatsapp/send
	DeviceID  string // default: Pionir
	Username  string // default: admin
	Password  string // default: PutihAbu123!
	APIKey    string // optional Bearer token
	Template  string
}

// WhatsAppClient manages requests to the WhatsApp Gateway
type WhatsAppClient struct {
	config     WhatsAppConfig
	httpClient *http.Client
}

// GowaRequest payload for NIBA-Gowa Shim & Gowa
type GowaRequest struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

// GowaResponse returned by NIBA-Gowa Shim & Gowa
type GowaResponse struct {
	Success   bool   `json:"success"`
	MessageID string `json:"message_id,omitempty"`
	Error     string `json:"error,omitempty"`
	Message   string `json:"message,omitempty"`
}

// NewWhatsAppClient loads settings from the database and initializes the client
func NewWhatsAppClient(db *sql.DB) (*WhatsAppClient, error) {
	var (
		providerNull sql.NullString
		enabledNull  sql.NullInt64
		urlNull      sql.NullString
		deviceNull   sql.NullString
		userNull     sql.NullString
		passNull     sql.NullString
		apiKeyNull   sql.NullString
		templateNull sql.NullString
	)

	err := db.QueryRow(`
		SELECT 
			COALESCE(wa_provider, 'gowa'),
			COALESCE(wa_enabled, onesender_enabled, 0),
			COALESCE(NULLIF(wa_api_url, ''), NULLIF(onesender_api_url, ''), 'http://127.0.0.1:8054/api/whatsapp/send'),
			COALESCE(NULLIF(wa_device_id, ''), 'Pionir'),
			COALESCE(NULLIF(wa_username, ''), 'admin'),
			COALESCE(NULLIF(wa_password, ''), 'PutihAbu123!'),
			COALESCE(onesender_api_key, ''),
			COALESCE(NULLIF(wa_template, ''), NULLIF(onesender_template, ''), '')
		FROM settings ORDER BY id DESC LIMIT 1
	`).Scan(&providerNull, &enabledNull, &urlNull, &deviceNull, &userNull, &passNull, &apiKeyNull, &templateNull)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	cfg := WhatsAppConfig{
		Enabled:  enabledNull.Int64 == 1,
		Provider: strings.ToLower(strings.TrimSpace(providerNull.String)),
		URL:      strings.TrimSpace(urlNull.String),
		DeviceID: strings.TrimSpace(deviceNull.String),
		Username: strings.TrimSpace(userNull.String),
		Password: strings.TrimSpace(passNull.String),
		APIKey:   strings.TrimSpace(apiKeyNull.String),
		Template: strings.TrimSpace(templateNull.String),
	}

	if cfg.Provider == "" {
		cfg.Provider = "gowa"
	}
	if cfg.URL == "" {
		cfg.URL = "http://127.0.0.1:8054/api/whatsapp/send"
	}
	if cfg.DeviceID == "" {
		cfg.DeviceID = "Pionir"
	}

	return &WhatsAppClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// IsEnabled returns whether automatic WhatsApp notifications are turned on
func (c *WhatsAppClient) IsEnabled() bool {
	return c.config.Enabled
}

// GetConfig returns a copy of the current configuration
func (c *WhatsAppClient) GetConfig() WhatsAppConfig {
	return c.config
}

// FormatPhoneNumber sanitizes and converts local Indonesian phone numbers to standard 62xxx format
func FormatPhoneNumber(phone string) string {
	// Remove all non-digit characters except leading +
	reg := regexp.MustCompile(`[^0-9+]`)
	phone = reg.ReplaceAllString(strings.TrimSpace(phone), "")

	if strings.HasPrefix(phone, "+") {
		phone = strings.TrimPrefix(phone, "+")
	}

	if strings.HasPrefix(phone, "0") {
		phone = "62" + strings.TrimPrefix(phone, "0")
	}

	if !strings.HasPrefix(phone, "62") {
		phone = "62" + phone
	}

	return phone
}

// BuildVoteMessage formats the confirmation message with available variables
func (c *WhatsAppClient) BuildVoteMessage(voterName, chairmanName, viceChairmanName string, voteTime time.Time) string {
	timeStr := voteTime.Format("02 Jan 2006 15:04 WIB")
	template := c.config.Template

	if strings.TrimSpace(template) == "" {
		return fmt.Sprintf("Halo %s! 🙏\n\n"+
			"Terima kasih telah berpartisipasi dalam Pemilihan Ketua & Wakil Ketua OSIS SMK NIBA Business School.\n\n"+
			"✅ Suara Anda telah tercatat dengan aman pada %s.\n\n"+
			"Partisipasi Anda sangat berarti bagi kemajuan sekolah kita.\n\n"+
			"Salam hangat,\nPanitia Pilketos", voterName, timeStr)
	}

	msg := template
	msg = strings.ReplaceAll(msg, "{nama}", voterName)
	msg = strings.ReplaceAll(msg, "{kandidat_ketua}", chairmanName)
	msg = strings.ReplaceAll(msg, "{kandidat_wakil}", viceChairmanName)
	msg = strings.ReplaceAll(msg, "{waktu}", timeStr)

	return msg
}

// SendMessage sends a WhatsApp message via GOWA (or OneSender fallback)
func (c *WhatsAppClient) SendMessage(phoneNumber, message string) error {
	if !c.config.Enabled {
		return fmt.Errorf("layanan notifikasi WhatsApp belum diaktifkan")
	}
	if c.config.URL == "" {
		return fmt.Errorf("URL gateway WhatsApp belum dikonfigurasi")
	}

	phoneNumber = FormatPhoneNumber(phoneNumber)

	if c.config.Provider == "onesender" {
		return c.sendOneSender(phoneNumber, message)
	}

	return c.sendGowa(phoneNumber, message)
}

// sendGowa sends via NIBA-Gowa Shim or Gowa API directly
func (c *WhatsAppClient) sendGowa(phoneNumber, message string) error {
	payload := GowaRequest{
		Phone:   phoneNumber,
		Message: message,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("gagal encode payload JSON: %w", err)
	}

	req, err := http.NewRequest("POST", c.config.URL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("gagal membuat request HTTP: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if c.config.DeviceID != "" {
		req.Header.Set("X-Device-Id", c.config.DeviceID)
	}

	if c.config.Username != "" && c.config.Password != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(c.config.Username + ":" + c.config.Password))
		req.Header.Set("Authorization", "Basic "+auth)
	} else if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("koneksi ke Gowa/Shim gagal (%s): %w", c.config.URL, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	var gowaResp GowaResponse
	_ = json.Unmarshal(bodyBytes, &gowaResp)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := string(bodyBytes)
		if gowaResp.Error != "" {
			errMsg = gowaResp.Error
		} else if gowaResp.Message != "" {
			errMsg = gowaResp.Message
		}
		return fmt.Errorf("Gowa gateway error (status %d): %s", resp.StatusCode, errMsg)
	}

	if !gowaResp.Success && gowaResp.Error != "" {
		return fmt.Errorf("Gowa status error: %s", gowaResp.Error)
	}

	return nil
}

// sendOneSender sends via legacy OneSender API
func (c *WhatsAppClient) sendOneSender(phoneNumber, message string) error {
	payload := map[string]interface{}{
		"recipient_type": "individual",
		"to":             phoneNumber,
		"type":           "text",
		"text": map[string]string{
			"body": message,
		},
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.config.URL, bytes.NewReader(jsonBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("onesender request failed (status %s): %s", resp.Status, string(body))
	}

	return nil
}
