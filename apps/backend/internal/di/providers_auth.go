package di

import (
	"context"
	"log/slog"
	"os"

	"github.com/google/wire"

	"github.com/paasdeploy/backend/internal/config"
	"github.com/paasdeploy/backend/internal/crypto"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/ghclient"
	"github.com/paasdeploy/backend/internal/handler"
	"github.com/paasdeploy/backend/internal/middleware"
	"github.com/paasdeploy/backend/internal/repository"
	"github.com/paasdeploy/backend/internal/service"
)

var AuthSet = wire.NewSet(
	ProvideTokenEncryptor,
	ProvideOAuthClient,
	ProvideGitHubAppClient,
	ProvideAuthMiddleware,
	ProvideAuthHandler,
	ProvideGitHubHandler,
)

var GitHubSet = wire.NewSet(
	ProvideGitHubWebhookHandler,
)

type webhookDeployAuditAdapter struct {
	audit *service.AuditService
}

func (a *webhookDeployAuditAdapter) LogDeployStarted(ctx context.Context, deployID, appID, appName, commitSHA string) {
	if a.audit != nil {
		auditCtx := service.AuditContext{}
		a.audit.LogDeployStarted(ctx, auditCtx, deployID, appID, appName, commitSHA)
	}
}

func ProvideTokenEncryptor(cfg *config.Config, logger *slog.Logger) *crypto.TokenEncryptor {
	if cfg.Auth.TokenEncryptionKey == "" {
		if cfg.Server.Env == "production" {
			logger.Error("TOKEN_ENCRYPTION_KEY is required in production")
			os.Exit(1)
		}
		logger.Warn("TOKEN_ENCRYPTION_KEY not set, credentials will not be encrypted")
		return nil
	}

	encryptor, err := crypto.NewTokenEncryptor(cfg.Auth.TokenEncryptionKey)
	if err != nil {
		logger.Error("failed to create token encryptor", "error", err)
		if cfg.Server.Env == "production" {
			os.Exit(1)
		}
		return nil
	}

	return encryptor
}

func ProvideOAuthClient(cfg *config.Config, logger *slog.Logger) *ghclient.OAuthClient {
	if cfg.GitHub.ClientID == "" || cfg.GitHub.ClientSecret == "" {
		logger.Info("GitHub OAuth not configured: GIT_HUB_CLIENT_ID or GIT_HUB_CLIENT_SECRET not set")
		return nil
	}

	return ghclient.NewOAuthClient(ghclient.OAuthConfig{
		ClientID:     cfg.GitHub.ClientID,
		ClientSecret: cfg.GitHub.ClientSecret,
		CallbackURL:  cfg.GitHub.CallbackURL,
	})
}

func ProvideGitHubAppClient(cfg *config.Config, logger *slog.Logger) *ghclient.AppClient {
	if cfg.GitHub.AppID == 0 || len(cfg.GitHub.AppPrivateKey) == 0 {
		logger.Info("GitHub App not configured: GIT_HUB_APP_ID or private key not set")
		return nil
	}

	client, err := ghclient.NewAppClient(ghclient.AppConfig{
		AppID:      cfg.GitHub.AppID,
		PrivateKey: cfg.GitHub.AppPrivateKey,
	})
	if err != nil {
		logger.Error("failed to create GitHub App client", "error", err)
		return nil
	}

	return client
}

func ProvideAuthMiddleware(
	cfg *config.Config,
	sessionRepo domain.SessionRepository,
	userRepo domain.UserRepository,
	logger *slog.Logger,
) *middleware.AuthMiddleware {
	return middleware.NewAuthMiddleware(middleware.AuthMiddlewareConfig{
		SessionRepo:       sessionRepo,
		UserRepo:          userRepo,
		Logger:            logger,
		SessionCookieName: cfg.Auth.SessionCookieName,
	})
}

func ProvideAuthHandler(
	cfg *config.Config,
	oauthClient *ghclient.OAuthClient,
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	tokenEncryptor *crypto.TokenEncryptor,
	auditService *service.AuditService,
	logger *slog.Logger,
) *handler.AuthHandler {
	if oauthClient == nil || tokenEncryptor == nil {
		logger.Info("AuthHandler not created: OAuth client or token encryptor not available")
		return nil
	}

	return handler.NewAuthHandler(handler.AuthHandlerConfig{
		OAuthClient:       oauthClient,
		UserRepo:          userRepo,
		SessionRepo:       sessionRepo,
		TokenEncryptor:    tokenEncryptor,
		AuditService:      auditService,
		Logger:            logger,
		SessionCookieName: cfg.Auth.SessionCookieName,
		SessionMaxAge:     cfg.Auth.SessionMaxAge,
		SecureCookie:      cfg.Auth.SecureCookie,
		CookieDomain:      cfg.Auth.CookieDomain,
		FrontendURL:       cfg.Auth.FrontendURL,
	})
}

func ProvideGitHubHandler(
	cfg *config.Config,
	appClient *ghclient.AppClient,
	installationRepo domain.InstallationRepository,
	userRepo domain.UserRepository,
	logger *slog.Logger,
) *handler.GitHubHandler {
	return handler.NewGitHubHandler(handler.GitHubHandlerConfig{
		AppClient:        appClient,
		InstallationRepo: installationRepo,
		UserRepo:         userRepo,
		Logger:           logger,
		WebhookSecret:    cfg.GitHub.WebhookSecret,
		AppInstallURL:    cfg.GitHub.AppInstallURL,
		SetupURL:         cfg.GitHub.AppSetupURL,
	})
}

func ProvideGitHubWebhookHandler(
	cfg *config.Config,
	appRepo *repository.PostgresAppRepository,
	deploymentRepo *repository.PostgresDeploymentRepository,
	payloadStore ghclient.WebhookPayloadStore,
	auditService *service.AuditService,
	logger *slog.Logger,
) *ghclient.WebhookHandler {
	adapter := &webhookDeployAuditAdapter{audit: auditService}
	return ghclient.NewWebhookHandler(
		appRepo,
		deploymentRepo,
		adapter,
		payloadStore,
		cfg.GitHub.WebhookSecret,
		logger,
	)
}

func ProvideCloudflareAuthHandler(
	cfg *config.Config,
	connectionRepo domain.CloudflareConnectionRepository,
	tokenEncryptor *crypto.TokenEncryptor,
	logger *slog.Logger,
) *handler.CloudflareAuthHandler {
	if tokenEncryptor == nil {
		logger.Info("CloudflareAuthHandler not created: token encryptor not available")
		return nil
	}

	return handler.NewCloudflareAuthHandler(handler.CloudflareAuthHandlerConfig{
		ClientID:       cfg.Cloudflare.ClientID,
		ClientSecret:   cfg.Cloudflare.ClientSecret,
		CallbackURL:    cfg.Cloudflare.CallbackURL,
		ConnectionRepo: connectionRepo,
		TokenEncryptor: tokenEncryptor,
		Logger:         logger,
		FrontendURL:    cfg.Auth.FrontendURL,
		SecureCookie:   cfg.Auth.SecureCookie,
		CookieDomain:   cfg.Auth.CookieDomain,
	})
}
