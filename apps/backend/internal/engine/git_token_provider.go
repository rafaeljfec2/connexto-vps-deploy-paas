package engine

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/ghclient"
)

type AppGitTokenProvider struct {
	appClient        *ghclient.AppClient
	installationRepo domain.InstallationRepository
	logger           *slog.Logger
}

func NewAppGitTokenProvider(
	appClient *ghclient.AppClient,
	installationRepo domain.InstallationRepository,
	logger *slog.Logger,
) *AppGitTokenProvider {
	return &AppGitTokenProvider{
		appClient:        appClient,
		installationRepo: installationRepo,
		logger:           logger,
	}
}

func (p *AppGitTokenProvider) GetToken(ctx context.Context, repoURL string) (string, error) {
	if p.appClient == nil || p.installationRepo == nil {
		return "", nil
	}

	owner := extractOwnerFromURL(repoURL)
	if owner == "" {
		return "", fmt.Errorf("could not extract owner from repository URL: %s", repoURL)
	}

	installation, err := p.installationRepo.FindByAccountLogin(ctx, owner)
	if err != nil {
		return "", fmt.Errorf("failed to find installation for owner %s: %w", owner, err)
	}
	if installation == nil {
		return "", fmt.Errorf("no GitHub App installation found for owner %s", owner)
	}

	p.logger.Debug("Found installation for repository",
		"owner", owner,
		"installation_id", installation.InstallationID,
	)

	token, err := p.appClient.GetInstallationToken(ctx, installation.InstallationID)
	if err != nil {
		return "", fmt.Errorf("failed to get installation token: %w", err)
	}

	return token.Token, nil
}

func extractOwnerFromURL(repoURL string) string {
	repoURL = strings.TrimSuffix(repoURL, ".git")

	if strings.HasPrefix(repoURL, "https://github.com/") {
		parts := strings.Split(strings.TrimPrefix(repoURL, "https://github.com/"), "/")
		if len(parts) >= 1 {
			return parts[0]
		}
	}

	if strings.HasPrefix(repoURL, "git@github.com:") {
		parts := strings.Split(strings.TrimPrefix(repoURL, "git@github.com:"), "/")
		if len(parts) >= 1 {
			return parts[0]
		}
	}

	return ""
}
