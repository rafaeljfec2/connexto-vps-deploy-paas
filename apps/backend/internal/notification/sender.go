package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/paasdeploy/backend/internal/domain"
)

const (
	formatApp     = "App: %s\n"
	formatMessage = "Message: %s\n"
	formatStatus  = "Status: %s\n"
	formatHealth  = "Health: %s\n"
)

type NotificationPayload struct {
	EventType string
	AppID     string
	AppName   string
	Message   string
	DeployID  string
	Status    string
	Health    string
	Timestamp time.Time
}

type Sender interface {
	Send(channel *domain.NotificationChannel, payload NotificationPayload) error
}

type SlackSender struct {
	client *http.Client
}

func NewSlackSender() *SlackSender {
	return &SlackSender{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

type slackPayload struct {
	Text string `json:"text"`
}

func (s *SlackSender) Send(channel *domain.NotificationChannel, payload NotificationPayload) error {
	var cfg struct {
		WebhookURL string `json:"webhookUrl"`
	}
	if err := json.Unmarshal(channel.Config, &cfg); err != nil || cfg.WebhookURL == "" {
		return fmt.Errorf("invalid slack config: webhookUrl required")
	}

	text := formatSlackMessage(payload)
	body, err := json.Marshal(slackPayload{Text: text})
	if err != nil {
		return err
	}

	resp, err := s.client.Post(cfg.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack webhook returned %d", resp.StatusCode)
	}
	return nil
}

func formatSlackMessage(payload NotificationPayload) string {
	msg := fmt.Sprintf("*[FlowDeploy] %s*\n", payload.EventType)
	msg += buildPayloadBody(payload)
	msg += fmt.Sprintf("Time: %s", payload.Timestamp.Format(time.RFC3339))
	return msg
}

func buildPayloadBody(payload NotificationPayload) string {
	var b strings.Builder
	if payload.AppName != "" {
		b.WriteString(fmt.Sprintf(formatApp, payload.AppName))
	}
	if payload.Message != "" {
		b.WriteString(fmt.Sprintf(formatMessage, payload.Message))
	}
	if payload.Status != "" {
		b.WriteString(fmt.Sprintf(formatStatus, payload.Status))
	}
	if payload.Health != "" {
		b.WriteString(fmt.Sprintf(formatHealth, payload.Health))
	}
	return b.String()
}

type DiscordSender struct {
	client *http.Client
}

func NewDiscordSender() *DiscordSender {
	return &DiscordSender{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

type discordPayload struct {
	Content string `json:"content"`
}

func (d *DiscordSender) Send(channel *domain.NotificationChannel, payload NotificationPayload) error {
	var cfg struct {
		WebhookURL string `json:"webhookUrl"`
	}
	if err := json.Unmarshal(channel.Config, &cfg); err != nil || cfg.WebhookURL == "" {
		return fmt.Errorf("invalid discord config: webhookUrl required")
	}

	content := formatDiscordMessage(payload)
	body, err := json.Marshal(discordPayload{Content: content})
	if err != nil {
		return err
	}

	resp, err := d.client.Post(cfg.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord webhook returned %d", resp.StatusCode)
	}
	return nil
}

func formatDiscordMessage(payload NotificationPayload) string {
	msg := fmt.Sprintf("**[FlowDeploy] %s**\n", payload.EventType)
	if payload.AppName != "" {
		msg += fmt.Sprintf("App: %s\n", payload.AppName)
	}
	if payload.Message != "" {
		msg += fmt.Sprintf("Message: %s\n", payload.Message)
	}
	if payload.Status != "" {
		msg += fmt.Sprintf("Status: %s\n", payload.Status)
	}
	if payload.Health != "" {
		msg += fmt.Sprintf("Health: %s\n", payload.Health)
	}
	msg += fmt.Sprintf("Time: %s", payload.Timestamp.Format(time.RFC3339))
	return msg
}

type EmailSender struct{}

func NewEmailSender() *EmailSender {
	return &EmailSender{}
}

func (e *EmailSender) Send(channel *domain.NotificationChannel, payload NotificationPayload) error {
	var cfg struct {
		SMTPHost  string `json:"smtpHost"`
		SMTPPort  int    `json:"smtpPort"`
		From      string `json:"from"`
		To        string `json:"to"`
		Username  string `json:"username"`
		Password  string `json:"password"`
	}
	if err := json.Unmarshal(channel.Config, &cfg); err != nil {
		return fmt.Errorf("invalid email config: %w", err)
	}
	if cfg.SMTPHost == "" || cfg.From == "" || cfg.To == "" {
		return fmt.Errorf("invalid email config: smtpHost, from, to required")
	}
	if cfg.SMTPPort == 0 {
		cfg.SMTPPort = 587
	}

	subject := fmt.Sprintf("[FlowDeploy] %s", payload.EventType)
	body := formatEmailBody(payload)

	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)
	msg := []byte(
		"To: " + cfg.To + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" +
			body + "\r\n",
	)

	var auth smtp.Auth
	if cfg.Username != "" && cfg.Password != "" {
		host := cfg.SMTPHost
		if idx := strings.Index(host, ":"); idx > 0 {
			host = host[:idx]
		}
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, host)
	}

	if err := smtp.SendMail(addr, auth, cfg.From, []string{cfg.To}, msg); err != nil {
		return fmt.Errorf("smtp send failed: %w", err)
	}
	return nil
}

func formatEmailBody(payload NotificationPayload) string {
	msg := fmt.Sprintf("FlowDeploy Notification: %s\n\n", payload.EventType)
	msg += buildPayloadBody(payload)
	msg += fmt.Sprintf("\nTime: %s", payload.Timestamp.Format(time.RFC3339))
	return msg
}
