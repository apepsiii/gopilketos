package services

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type OneSenderConfig struct {
	Enabled  bool
	URL      string
	APIKey   string
	Template string
}

type OneSenderClient struct {
	config     OneSenderConfig
	httpClient *http.Client
}

func NewOneSenderClient(db *sql.DB) (*OneSenderClient, error) {
	var enabled int
	var url, apiKey, template string

	err := db.QueryRow(`SELECT onesender_enabled, onesender_api_url, onesender_api_key, onesender_template FROM settings ORDER BY id DESC LIMIT 1`).Scan(&enabled, &url, &apiKey, &template)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return &OneSenderClient{
		config: OneSenderConfig{
			Enabled:  enabled == 1,
			URL:      url,
			APIKey:   apiKey,
			Template: template,
		},
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}, nil
}

func FormatPhoneNumber(phone string) string {
	phone = strings.TrimSpace(phone)

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

func (c *OneSenderClient) SendMessage(phoneNumber string, vars ...string) error {
	if !c.config.Enabled {
		return fmt.Errorf("onesender belum diaktifkan")
	}
	if c.config.URL == "" || c.config.APIKey == "" {
		return fmt.Errorf("konfigurasi onesender belum lengkap")
	}

	phoneNumber = FormatPhoneNumber(phoneNumber)

	message := c.config.Template
	if len(vars) > 0 && vars[0] != "" {
		message = vars[0]
	}

	if message == "" {
		message = "Terima kasih telah berpartisipasi dalam pemilihan OSIS SMK NIBA. Suara Anda telah tercatat."
	}

	payload := map[string]interface{}{
		"recipient_type": "individual",
		"to":             phoneNumber,
		"type":           "text",
		"text": map[string]string{
			"body": message,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.config.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("onesender request failed: %s", resp.Status)
	}

	return nil
}
