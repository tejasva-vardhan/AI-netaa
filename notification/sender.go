package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"finalneta/models"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Sender is the interface for notification senders
type Sender interface {
	Send(ctx context.Context, notification *models.Notification) error
	Channel() models.NotificationChannel
	Validate(notification *models.Notification) error
}

// getShadowAddress returns the only allowed recipient when EMAIL_MODE=shadow. All emails go here.
func getShadowAddress() string {
	if os.Getenv("EMAIL_MODE") != "shadow" {
		return ""
	}
	addr := os.Getenv("EMAIL_SHADOW_ADDRESS")
	if addr == "" {
		addr = "aineta502@gmail.com"
	}
	return addr
}

// EmailSender sends email. If EMAIL_MODE=shadow, forces recipient to EMAIL_SHADOW_ADDRESS.
// If SENDGRID_API_KEY is set, uses SendGrid; otherwise no-op (pilot without real send).
type EmailSender struct {
	apiKey     string
	shadowAddr string
}

// NewEmailSender creates an email sender (reads EMAIL_MODE, EMAIL_SHADOW_ADDRESS, SENDGRID_API_KEY from env).
func NewEmailSender() *EmailSender {
	apiKey := os.Getenv("SENDGRID_API_KEY")
	shadowAddr := getShadowAddress()
	return &EmailSender{apiKey: apiKey, shadowAddr: shadowAddr}
}

// Channel returns the email channel type
func (s *EmailSender) Channel() models.NotificationChannel {
	return models.ChannelEmail
}

// Validate validates email notification
func (s *EmailSender) Validate(notification *models.Notification) error {
	if notification.Recipient == "" {
		return ErrInvalidRecipient
	}
	return nil
}

// Send sends an email. In shadow mode, recipient is forced to shadow address. Async/retries are handled by caller.
func (s *EmailSender) Send(ctx context.Context, notification *models.Notification) error {
	if s.shadowAddr != "" {
		notification.Recipient = s.shadowAddr
	}
	if err := s.Validate(notification); err != nil {
		return err
	}
	if s.apiKey == "" {
		return nil
	}
	return s.sendViaSendGrid(ctx, notification)
}

const sendGridURL = "https://api.sendgrid.com/v3/mail/send"
const maxSendGridRetries = 3

func (s *EmailSender) sendViaSendGrid(ctx context.Context, n *models.Notification) error {
	subject := ""
	if n.Subject.Valid {
		subject = n.Subject.String
	}
	fromEmail := os.Getenv("SENDGRID_FROM_EMAIL")
	if fromEmail == "" {
		fromEmail = "noreply@aineta.in"
	}
	fromName := os.Getenv("SENDGRID_FROM_NAME")
	if fromName == "" {
		fromName = "AI Neta"
	}
	body := map[string]interface{}{
		"personalizations": []map[string]interface{}{
			{"to": []map[string]interface{}{{"email": n.Recipient}}},
		},
		"from":    map[string]string{"email": fromEmail, "name": fromName},
		"subject": subject,
		"content": []map[string]string{{"type": "text/plain", "value": n.Body}},
	}
	payload, _ := json.Marshal(body)
	var lastErr error
	for attempt := 0; attempt < maxSendGridRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendGridURL, bytes.NewReader(payload))
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("sendgrid status %d", resp.StatusCode)
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}
	return lastErr
}

// SMSSender handles SMS notifications
type SMSSender struct {
	// In production, this would contain SMS gateway config, API keys, etc.
}

// NewSMSSender creates a new SMS sender
func NewSMSSender() *SMSSender {
	return &SMSSender{}
}

// Channel returns the SMS channel type
func (s *SMSSender) Channel() models.NotificationChannel {
	return models.ChannelSMS
}

// Validate validates SMS notification
func (s *SMSSender) Validate(notification *models.Notification) error {
	if notification.Recipient == "" {
		return ErrInvalidRecipient
	}
	// Add phone number format validation if needed
	return nil
}

// Send sends an SMS notification (mock implementation)
func (s *SMSSender) Send(ctx context.Context, notification *models.Notification) error {
	// Mock implementation - in production, this would:
	// 1. Send via SMS gateway (Twilio, AWS SNS, etc.)
	// 2. Return error if sending fails
	
	// For now, simulate success
	return nil
}

// WhatsAppSender handles WhatsApp notifications
type WhatsAppSender struct {
	// In production, this would contain WhatsApp Business API config
}

// NewWhatsAppSender creates a new WhatsApp sender
func NewWhatsAppSender() *WhatsAppSender {
	return &WhatsAppSender{}
}

// Channel returns the WhatsApp channel type
func (s *WhatsAppSender) Channel() models.NotificationChannel {
	return models.ChannelWhatsApp
}

// Validate validates WhatsApp notification
func (s *WhatsAppSender) Validate(notification *models.Notification) error {
	if notification.Recipient == "" {
		return ErrInvalidRecipient
	}
	// Add WhatsApp number format validation if needed
	return nil
}

// Send sends a WhatsApp notification (mock implementation)
func (s *WhatsAppSender) Send(ctx context.Context, notification *models.Notification) error {
	// Mock implementation - in production, this would:
	// 1. Send via WhatsApp Business API
	// 2. Return error if sending fails
	
	// For now, simulate success
	return nil
}

// Errors
var (
	ErrInvalidRecipient = &NotificationError{Message: "invalid recipient"}
	ErrUnsupportedChannel = &NotificationError{Message: "unsupported channel"}
	ErrMaxRetriesExceeded = &NotificationError{Message: "max retries exceeded"}
)

// NotificationError represents a notification error
type NotificationError struct {
	Message string
	Err     error
}

func (e *NotificationError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *NotificationError) Unwrap() error {
	return e.Err
}
