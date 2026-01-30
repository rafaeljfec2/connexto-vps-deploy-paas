package github

import "context"

type WebhookConfig struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Secret      string `json:"secret,omitempty"`
	InsecureSSL string `json:"insecure_ssl"`
}

type Webhook struct {
	ID        int64         `json:"id"`
	Type      string        `json:"type"`
	Name      string        `json:"name"`
	Active    bool          `json:"active"`
	Events    []string      `json:"events"`
	Config    WebhookConfig `json:"config"`
}

type Provider interface {
	CreateWebhook(ctx context.Context, owner, repo string, config WebhookConfig) (*Webhook, error)
	DeleteWebhook(ctx context.Context, owner, repo string, webhookID int64) error
	GetWebhook(ctx context.Context, owner, repo string, webhookID int64) (*Webhook, error)
	ListWebhooks(ctx context.Context, owner, repo string) ([]Webhook, error)
}
