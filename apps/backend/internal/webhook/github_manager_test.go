package webhook

import (
	"context"
	"errors"
	"testing"

	"github.com/paasdeploy/backend/internal/github"
)

type mockProvider struct {
	createWebhookFunc func(ctx context.Context, owner, repo string, config github.WebhookConfig) (*github.Webhook, error)
	deleteWebhookFunc func(ctx context.Context, owner, repo string, webhookID int64) error
	getWebhookFunc    func(ctx context.Context, owner, repo string, webhookID int64) (*github.Webhook, error)
	listWebhooksFunc  func(ctx context.Context, owner, repo string) ([]github.Webhook, error)
}

func (m *mockProvider) CreateWebhook(ctx context.Context, owner, repo string, config github.WebhookConfig) (*github.Webhook, error) {
	if m.createWebhookFunc != nil {
		return m.createWebhookFunc(ctx, owner, repo, config)
	}
	return &github.Webhook{ID: 123, Active: true}, nil
}

func (m *mockProvider) DeleteWebhook(ctx context.Context, owner, repo string, webhookID int64) error {
	if m.deleteWebhookFunc != nil {
		return m.deleteWebhookFunc(ctx, owner, repo, webhookID)
	}
	return nil
}

func (m *mockProvider) GetWebhook(ctx context.Context, owner, repo string, webhookID int64) (*github.Webhook, error) {
	if m.getWebhookFunc != nil {
		return m.getWebhookFunc(ctx, owner, repo, webhookID)
	}
	return &github.Webhook{ID: webhookID, Active: true}, nil
}

func (m *mockProvider) ListWebhooks(ctx context.Context, owner, repo string) ([]github.Webhook, error) {
	if m.listWebhooksFunc != nil {
		return m.listWebhooksFunc(ctx, owner, repo)
	}
	return []github.Webhook{}, nil
}

func TestGitHubManagerSetup(t *testing.T) {
	provider := &mockProvider{}
	manager := NewGitHubManager(provider, "https://example.com/webhooks/github", "test-secret")

	result, err := manager.Setup(context.Background(), SetupInput{
		RepositoryURL: "https://github.com/owner/repo",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.WebhookID != 123 {
		t.Errorf("expected webhook ID 123, got %d", result.WebhookID)
	}

	if result.Provider != "github" {
		t.Errorf("expected provider 'github', got %s", result.Provider)
	}
}

func TestGitHubManagerSetupWithSSHURL(t *testing.T) {
	provider := &mockProvider{}
	manager := NewGitHubManager(provider, "https://example.com/webhooks/github", "test-secret")

	result, err := manager.Setup(context.Background(), SetupInput{
		RepositoryURL: "git@github.com:owner/repo.git",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.WebhookID != 123 {
		t.Errorf("expected webhook ID 123, got %d", result.WebhookID)
	}
}

func TestGitHubManagerSetupInvalidURL(t *testing.T) {
	provider := &mockProvider{}
	manager := NewGitHubManager(provider, "https://example.com/webhooks/github", "test-secret")

	_, err := manager.Setup(context.Background(), SetupInput{
		RepositoryURL: "https://gitlab.com/owner/repo",
	})

	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestGitHubManagerRemove(t *testing.T) {
	provider := &mockProvider{}
	manager := NewGitHubManager(provider, "https://example.com/webhooks/github", "test-secret")

	err := manager.Remove(context.Background(), RemoveInput{
		RepositoryURL: "https://github.com/owner/repo",
		WebhookID:     123,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubManagerStatus(t *testing.T) {
	provider := &mockProvider{}
	manager := NewGitHubManager(provider, "https://example.com/webhooks/github", "test-secret")

	status, err := manager.Status(context.Background(), "https://github.com/owner/repo", 123)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !status.Exists {
		t.Error("expected webhook to exist")
	}

	if !status.Active {
		t.Error("expected webhook to be active")
	}
}

func TestGitHubManagerStatusNotFound(t *testing.T) {
	provider := &mockProvider{
		getWebhookFunc: func(ctx context.Context, owner, repo string, webhookID int64) (*github.Webhook, error) {
			return nil, nil
		},
	}
	manager := NewGitHubManager(provider, "https://example.com/webhooks/github", "test-secret")

	status, err := manager.Status(context.Background(), "https://github.com/owner/repo", 123)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.Exists {
		t.Error("expected webhook to not exist")
	}
}

func TestGitHubManagerStatusError(t *testing.T) {
	provider := &mockProvider{
		getWebhookFunc: func(ctx context.Context, owner, repo string, webhookID int64) (*github.Webhook, error) {
			return nil, errors.New("API error")
		},
	}
	manager := NewGitHubManager(provider, "https://example.com/webhooks/github", "test-secret")

	status, err := manager.Status(context.Background(), "https://github.com/owner/repo", 123)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.Exists {
		t.Error("expected webhook to not exist on error")
	}

	if status.Error == "" {
		t.Error("expected error message")
	}
}

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "https URL",
			url:       "https://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "https URL with .git",
			url:       "https://github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "SSH URL",
			url:       "git@github.com:owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "SSH URL without .git",
			url:       "git@github.com:owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:    "invalid URL",
			url:     "https://gitlab.com/owner/repo",
			wantErr: true,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseGitHubURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if owner != tt.wantOwner {
				t.Errorf("expected owner %q, got %q", tt.wantOwner, owner)
			}

			if repo != tt.wantRepo {
				t.Errorf("expected repo %q, got %q", tt.wantRepo, repo)
			}
		})
	}
}
