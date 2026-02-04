package webhook

import (
	"context"
	"errors"
	"testing"

	"github.com/paasdeploy/backend/internal/ghclient"
)

const (
	testWebhookURL    = "https://example.com/webhooks/github"
	testWebhookSecret = "test-secret"
	testRepoURL       = "https://github.com/owner/repo"
	errFmtUnexpected  = "unexpected error: %v"
)

type mockProvider struct {
	createWebhookFunc func(ctx context.Context, owner, repo string, config ghclient.WebhookConfig) (*ghclient.Webhook, error)
	deleteWebhookFunc func(ctx context.Context, owner, repo string, webhookID int64) error
	getWebhookFunc    func(ctx context.Context, owner, repo string, webhookID int64) (*ghclient.Webhook, error)
	listWebhooksFunc  func(ctx context.Context, owner, repo string) ([]ghclient.Webhook, error)
	listCommitsFunc   func(ctx context.Context, owner, repo, branch string, perPage int) ([]ghclient.CommitInfo, error)
}

func (m *mockProvider) CreateWebhook(ctx context.Context, owner, repo string, config ghclient.WebhookConfig) (*ghclient.Webhook, error) {
	if m.createWebhookFunc != nil {
		return m.createWebhookFunc(ctx, owner, repo, config)
	}
	return &ghclient.Webhook{ID: 123, Active: true}, nil
}

func (m *mockProvider) DeleteWebhook(ctx context.Context, owner, repo string, webhookID int64) error {
	if m.deleteWebhookFunc != nil {
		return m.deleteWebhookFunc(ctx, owner, repo, webhookID)
	}
	return nil
}

func (m *mockProvider) GetWebhook(ctx context.Context, owner, repo string, webhookID int64) (*ghclient.Webhook, error) {
	if m.getWebhookFunc != nil {
		return m.getWebhookFunc(ctx, owner, repo, webhookID)
	}
	return &ghclient.Webhook{ID: webhookID, Active: true}, nil
}

func (m *mockProvider) ListWebhooks(ctx context.Context, owner, repo string) ([]ghclient.Webhook, error) {
	if m.listWebhooksFunc != nil {
		return m.listWebhooksFunc(ctx, owner, repo)
	}
	return []ghclient.Webhook{}, nil
}

func (m *mockProvider) ListCommits(ctx context.Context, owner, repo, branch string, perPage int) ([]ghclient.CommitInfo, error) {
	if m.listCommitsFunc != nil {
		return m.listCommitsFunc(ctx, owner, repo, branch, perPage)
	}
	return []ghclient.CommitInfo{}, nil
}

func TestGitHubManagerSetup(t *testing.T) {
	provider := &mockProvider{}
	manager := NewGitHubManager(provider, testWebhookURL, testWebhookSecret)

	result, err := manager.Setup(context.Background(), SetupInput{
		RepositoryURL: testRepoURL,
	})

	if err != nil {
		t.Fatalf(errFmtUnexpected, err)
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
	manager := NewGitHubManager(provider, testWebhookURL, testWebhookSecret)

	result, err := manager.Setup(context.Background(), SetupInput{
		RepositoryURL: "git@github.com:owner/repo.git",
	})

	if err != nil {
		t.Fatalf(errFmtUnexpected, err)
	}

	if result.WebhookID != 123 {
		t.Errorf("expected webhook ID 123, got %d", result.WebhookID)
	}
}

func TestGitHubManagerSetupInvalidURL(t *testing.T) {
	provider := &mockProvider{}
	manager := NewGitHubManager(provider, testWebhookURL, testWebhookSecret)

	_, err := manager.Setup(context.Background(), SetupInput{
		RepositoryURL: "https://gitlab.com/owner/repo",
	})

	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestGitHubManagerRemove(t *testing.T) {
	provider := &mockProvider{}
	manager := NewGitHubManager(provider, testWebhookURL, testWebhookSecret)

	err := manager.Remove(context.Background(), RemoveInput{
		RepositoryURL: testRepoURL,
		WebhookID:     123,
	})

	if err != nil {
		t.Fatalf(errFmtUnexpected, err)
	}
}

func TestGitHubManagerStatus(t *testing.T) {
	provider := &mockProvider{}
	manager := NewGitHubManager(provider, testWebhookURL, testWebhookSecret)

	status, err := manager.Status(context.Background(), testRepoURL, 123)

	if err != nil {
		t.Fatalf(errFmtUnexpected, err)
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
		getWebhookFunc: func(ctx context.Context, owner, repo string, webhookID int64) (*ghclient.Webhook, error) {
			return nil, nil
		},
	}
	manager := NewGitHubManager(provider, testWebhookURL, testWebhookSecret)

	status, err := manager.Status(context.Background(), testRepoURL, 123)

	if err != nil {
		t.Fatalf(errFmtUnexpected, err)
	}

	if status.Exists {
		t.Error("expected webhook to not exist")
	}
}

func TestGitHubManagerStatusError(t *testing.T) {
	provider := &mockProvider{
		getWebhookFunc: func(ctx context.Context, owner, repo string, webhookID int64) (*ghclient.Webhook, error) {
			return nil, errors.New("API error")
		},
	}
	manager := NewGitHubManager(provider, testWebhookURL, testWebhookSecret)

	status, err := manager.Status(context.Background(), testRepoURL, 123)

	if err != nil {
		t.Fatalf(errFmtUnexpected, err)
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
			assertParseResult(t, owner, repo, err, tt.wantOwner, tt.wantRepo, tt.wantErr)
		})
	}
}

func assertParseResult(t *testing.T, owner, repo string, err error, wantOwner, wantRepo string, wantErr bool) {
	t.Helper()
	if wantErr {
		if err == nil {
			t.Error("expected error")
		}
		return
	}
	if err != nil {
		t.Fatalf(errFmtUnexpected, err)
	}
	if owner != wantOwner {
		t.Errorf("expected owner %q, got %q", wantOwner, owner)
	}
	if repo != wantRepo {
		t.Errorf("expected repo %q, got %q", wantRepo, repo)
	}
}
