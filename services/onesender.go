package services

import (
	"database/sql"
)

// Deprecated: OneSenderConfig is maintained for backwards compatibility. Use WhatsAppConfig instead.
type OneSenderConfig = WhatsAppConfig

// Deprecated: OneSenderClient is maintained for backwards compatibility. Use WhatsAppClient instead.
type OneSenderClient struct {
	waClient *WhatsAppClient
}

// Deprecated: NewOneSenderClient is maintained for backwards compatibility. Use NewWhatsAppClient instead.
func NewOneSenderClient(db *sql.DB) (*OneSenderClient, error) {
	client, err := NewWhatsAppClient(db)
	if err != nil {
		return nil, err
	}
	return &OneSenderClient{waClient: client}, nil
}

// SendMessage sends message via the underlying WhatsAppClient
func (c *OneSenderClient) SendMessage(phoneNumber string, vars ...string) error {
	msg := ""
	if len(vars) > 0 {
		msg = vars[0]
	}
	return c.waClient.SendMessage(phoneNumber, msg)
}
