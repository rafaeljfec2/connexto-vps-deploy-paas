package webhook

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/paasdeploy/backend/internal/ghclient"
)

const (
	providerGitHub  = "github"
	errParseRepoURL = "parse repository URL: %w"
)

var _ Manager = (*GitHubManager)(nil)

type GitHubManager struct {
	provider      ghclient.Provider
	webhookURL    string
	webhookSecret string
}

func NewGitHubManager(provider ghclient.Provider, webhookURL, webhookSecret string) *GitHubManager {
	return &GitHubManager{
		provider:      provider,
		webhookURL:    webhookURL,
		webhookSecret: webhookSecret,
	}
}

func (m *GitHubManager) Setup(ctx context.Context, input SetupInput) (*SetupResult, error) {
	owner, repo, err := parseGitHubURL(input.RepositoryURL)
	if err != nil {
		return nil, fmt.Errorf(errParseRepoURL, err)
	}

	targetURL := input.TargetURL
	if targetURL == "" {
		targetURL = m.webhookURL
	}

	secret := input.Secret
	if secret == "" {
		secret = m.webhookSecret
	}

	config := ghclient.WebhookConfig{
		URL:         targetURL,
		ContentType: "json",
		Secret:      secret,
		InsecureSSL: "0",
	}

	webhook, err := m.provider.CreateWebhook(ctx, owner, repo, config)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "hook already exists") ||
			strings.Contains(strings.ToLower(err.Error()), "already exists") {
			existing, listErr := m.findExistingWebhookByURL(ctx, owner, repo, targetURL)
			if listErr == nil && existing != nil {
				return &SetupResult{
					WebhookID: existing.ID,
					Provider:  providerGitHub,
					Active:    existing.Active,
				}, nil
			}
		}
		return nil, fmt.Errorf("create webhook: %w", err)
	}

	return &SetupResult{
		WebhookID: webhook.ID,
		Provider:  providerGitHub,
		Active:    webhook.Active,
	}, nil
}

func normalizeURL(u string) string {
	return strings.TrimSuffix(strings.TrimSpace(u), "/")
}

func (m *GitHubManager) findExistingWebhookByURL(ctx context.Context, owner, repo, targetURL string) (*ghclient.Webhook, error) {
	hooks, err := m.provider.ListWebhooks(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	normalized := normalizeURL(targetURL)
	for i := range hooks {
		h := &hooks[i]
		if normalizeURL(h.Config.URL) == normalized {
			return h, nil
		}
	}
	return nil, nil
}

func (m *GitHubManager) Remove(ctx context.Context, input RemoveInput) error {
	owner, repo, err := parseGitHubURL(input.RepositoryURL)
	if err != nil {
		return fmt.Errorf(errParseRepoURL, err)
	}

	if err := m.provider.DeleteWebhook(ctx, owner, repo, input.WebhookID); err != nil {
		return fmt.Errorf("delete webhook: %w", err)
	}

	return nil
}

func (m *GitHubManager) Status(ctx context.Context, repoURL string, webhookID int64) (*Status, error) {
	owner, repo, err := parseGitHubURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf(errParseRepoURL, err)
	}

	webhook, err := m.provider.GetWebhook(ctx, owner, repo, webhookID)
	if err != nil {
		return &Status{
			Exists: false,
			Error:  err.Error(),
		}, nil
	}

	if webhook == nil {
		return &Status{
			Exists: false,
		}, nil
	}

	return &Status{
		Exists: true,
		Active: webhook.Active,
	}, nil
}

func (m *GitHubManager) WebhookURL() string {
	return m.webhookURL
}

func (m *GitHubManager) ListCommits(ctx context.Context, repoURL, branch string, perPage int) ([]ghclient.CommitInfo, error) {
	owner, repo, err := parseGitHubURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf(errParseRepoURL, err)
	}

	commits, err := m.provider.ListCommits(ctx, owner, repo, branch, perPage)
	if err != nil {
		return nil, fmt.Errorf("list commits: %w", err)
	}

	return commits, nil
}

var (
	httpsPattern = regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+?)(?:\.git)?$`)
	sshPattern   = regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+?)(?:\.git)?$`)
)

func parseGitHubURL(url string) (owner, repo string, err error) {
	url = strings.TrimSpace(url)

	if matches := httpsPattern.FindStringSubmatch(url); len(matches) == 3 {
		return matches[1], matches[2], nil
	}

	if matches := sshPattern.FindStringSubmatch(url); len(matches) == 3 {
		return matches[1], matches[2], nil
	}

	return "", "", fmt.Errorf("invalid GitHub repository URL: %s", url)
}
