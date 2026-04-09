package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type OneSenderClient struct {
	url        string
	apiKey     string
	httpClient *http.Client
}

func NewOneSenderClient() *OneSenderClient {
	return &OneSenderClient{
		url:    os.Getenv("ONESENDER_API_URL"),
		apiKey: os.Getenv("ONESENDER_API_KEY"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *OneSenderClient) SendMessage(phoneNumber, message string) error {
	if c.url == "" || c.apiKey == "" {
		return nil
	}
	payload := map[string]string{
		"phone":   phoneNumber,
		"message": message,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

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
