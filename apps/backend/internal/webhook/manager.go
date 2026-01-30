package webhook

import (
	"context"
	"time"
)

type SetupInput struct {
	RepositoryURL string
	TargetURL     string
	Secret        string
	Events        []string
}

type SetupResult struct {
	WebhookID int64
	Provider  string
	Active    bool
}

type RemoveInput struct {
	RepositoryURL string
	WebhookID     int64
}

type Status struct {
	Exists     bool
	Active     bool
	LastPingAt *time.Time
	Error      string
}

type Manager interface {
	Setup(ctx context.Context, input SetupInput) (*SetupResult, error)
	Remove(ctx context.Context, input RemoveInput) error
	Status(ctx context.Context, repoURL string, webhookID int64) (*Status, error)
}
