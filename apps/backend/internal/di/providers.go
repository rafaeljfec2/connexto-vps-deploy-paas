package di

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/wire"
	_ "github.com/lib/pq"
	"github.com/lmittmann/tint"

	"github.com/paasdeploy/backend/internal/config"
	"github.com/paasdeploy/backend/internal/crypto"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/engine"
	"github.com/paasdeploy/backend/internal/github"
	"github.com/paasdeploy/backend/internal/handler"
	"github.com/paasdeploy/backend/internal/middleware"
	"github.com/paasdeploy/backend/internal/repository"
	"github.com/paasdeploy/backend/internal/server"
	"github.com/paasdeploy/backend/internal/service"
	"github.com/paasdeploy/backend/internal/webhook"
)

var ConfigSet = wire.NewSet(
	ProvideConfig,
)

var LoggerSet = wire.NewSet(
	ProvideLogger,
)

var DatabaseSet = wire.NewSet(
	ProvideDatabase,
)

var RepositorySet = wire.NewSet(
	repository.NewPostgresAppRepository,
	wire.Bind(new(domain.AppRepository), new(*repository.PostgresAppRepository)),
	repository.NewPostgresDeploymentRepository,
	wire.Bind(new(domain.DeploymentRepository), new(*repository.PostgresDeploymentRepository)),
	repository.NewPostgresEnvVarRepository,
	wire.Bind(new(domain.EnvVarRepository), new(*repository.PostgresEnvVarRepository)),
	repository.NewPostgresUserRepository,
	wire.Bind(new(domain.UserRepository), new(*repository.PostgresUserRepository)),
	repository.NewPostgresSessionRepository,
	wire.Bind(new(domain.SessionRepository), new(*repository.PostgresSessionRepository)),
	repository.NewPostgresInstallationRepository,
	wire.Bind(new(domain.InstallationRepository), new(*repository.PostgresInstallationRepository)),
)

var AuthSet = wire.NewSet(
	ProvideTokenEncryptor,
	ProvideOAuthClient,
	ProvideGitHubAppClient,
	ProvideAuthMiddleware,
	ProvideAuthHandler,
	ProvideGitHubHandler,
)

var WebhookSet = wire.NewSet(
	ProvideWebhookManager,
)

var ServiceSet = wire.NewSet(
	ProvideAppCleaner,
	ProvideAppService,
)

var EngineSet = wire.NewSet(
	engine.New,
)

var GitHubSet = wire.NewSet(
	ProvideGitHubWebhookHandler,
)

var HandlerSet = wire.NewSet(
	ProvideHealthHandler,
	handler.NewAppHandler,
	handler.NewSSEHandler,
	handler.NewSwaggerHandler,
	handler.NewEnvVarHandler,
	handler.NewContainerHealthHandler,
	ProvideAppAdminHandler,
)

var ServerSet = wire.NewSet(
	ProvideServerConfig,
	server.New,
)

var AppSet = wire.NewSet(
	ConfigSet,
	LoggerSet,
	DatabaseSet,
	RepositorySet,
	WebhookSet,
	ServiceSet,
	EngineSet,
	GitHubSet,
	AuthSet,
	HandlerSet,
	ServerSet,
	wire.Struct(new(Application), "*"),
)

const Version = "0.1.0"

func ProvideConfig() (*config.Config, error) {
	cfg := config.Load()
	if err := cfg.EnsureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to ensure directories: %w", err)
	}
	return cfg, nil
}

func ProvideHealthHandler() *handler.HealthHandler {
	return handler.NewHealthHandler(Version)
}

func ProvideWebhookManager(cfg *config.Config, logger *slog.Logger) webhook.Manager {
	if cfg.GitHub.PAT == "" || cfg.GitHub.WebhookURL == "" {
		logger.Info("webhook management disabled: GITHUB_PAT or GITHUB_WEBHOOK_URL not configured")
		return webhook.NewNoOpManager()
	}

	provider := github.NewPATProvider(cfg.GitHub.PAT)
	return webhook.NewGitHubManager(provider, cfg.GitHub.WebhookURL, cfg.GitHub.WebhookSecret)
}

func ProvideAppCleaner(cfg *config.Config, logger *slog.Logger) *engine.AppCleaner {
	return engine.NewAppCleaner(cfg.Deploy.DataDir, logger)
}

func ProvideAppAdminHandler(appRepo domain.AppRepository, eng *engine.Engine, cfg *config.Config) *handler.AppAdminHandler {
	return handler.NewAppAdminHandler(appRepo, eng, cfg.Deploy.DataDir)
}

func ProvideAppService(
	appRepo domain.AppRepository,
	deploymentRepo domain.DeploymentRepository,
	envVarRepo domain.EnvVarRepository,
	webhookManager webhook.Manager,
	appCleaner *engine.AppCleaner,
	logger *slog.Logger,
) *service.AppService {
	return service.NewAppService(appRepo, deploymentRepo, envVarRepo, webhookManager, appCleaner, logger)
}

func ProvideGitHubWebhookHandler(
	cfg *config.Config,
	appRepo *repository.PostgresAppRepository,
	deploymentRepo *repository.PostgresDeploymentRepository,
	logger *slog.Logger,
) *github.WebhookHandler {
	return github.NewWebhookHandler(
		appRepo,
		deploymentRepo,
		cfg.GitHub.WebhookSecret,
		logger,
	)
}

func ProvideLogger(cfg *config.Config) *slog.Logger {
	var logLevel slog.Level
	switch cfg.Server.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	var handler slog.Handler

	if cfg.Server.Env == "development" {
		handler = tint.NewHandler(os.Stdout, &tint.Options{
			Level:      logLevel,
			TimeFormat: "15:04:05",
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		})
	}

	return slog.New(handler)
}

func ProvideDatabase(cfg *config.Config) (*sql.DB, func(), error) {
	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("failed to ping database: %w", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup, nil
}

func ProvideServerConfig(cfg *config.Config) server.Config {
	return server.Config{
		Host:         cfg.Server.Host,
		Port:         cfg.Server.Port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}
}

type Application struct {
	Config                 *config.Config
	Logger                 *slog.Logger
	DB                     *sql.DB
	Engine                 *engine.Engine
	Server                 *server.Server
	HealthHandler          *handler.HealthHandler
	AppHandler             *handler.AppHandler
	SSEHandler             *handler.SSEHandler
	SwaggerHandler         *handler.SwaggerHandler
	EnvVarHandler          *handler.EnvVarHandler
	ContainerHealthHandler *handler.ContainerHealthHandler
	AppAdminHandler        *handler.AppAdminHandler
	WebhookHandler         *github.WebhookHandler
	AuthHandler            *handler.AuthHandler
	GitHubHandler          *handler.GitHubHandler
	AuthMiddleware         *middleware.AuthMiddleware
}

func ProvideTokenEncryptor(cfg *config.Config, logger *slog.Logger) *crypto.TokenEncryptor {
	if cfg.Auth.TokenEncryptionKey == "" {
		logger.Warn("TOKEN_ENCRYPTION_KEY not set, authentication will not work properly")
		return nil
	}

	encryptor, err := crypto.NewTokenEncryptor(cfg.Auth.TokenEncryptionKey)
	if err != nil {
		logger.Error("failed to create token encryptor", "error", err)
		return nil
	}

	return encryptor
}

func ProvideOAuthClient(cfg *config.Config, logger *slog.Logger) *github.OAuthClient {
	if cfg.GitHub.ClientID == "" || cfg.GitHub.ClientSecret == "" {
		logger.Info("GitHub OAuth not configured: GITHUB_CLIENT_ID or GITHUB_CLIENT_SECRET not set")
		return nil
	}

	return github.NewOAuthClient(github.OAuthConfig{
		ClientID:     cfg.GitHub.ClientID,
		ClientSecret: cfg.GitHub.ClientSecret,
		CallbackURL:  cfg.GitHub.CallbackURL,
	})
}

func ProvideGitHubAppClient(cfg *config.Config, logger *slog.Logger) *github.AppClient {
	if cfg.GitHub.AppID == 0 || len(cfg.GitHub.AppPrivateKey) == 0 {
		logger.Info("GitHub App not configured: GITHUB_APP_ID or private key not set")
		return nil
	}

	client, err := github.NewAppClient(github.AppConfig{
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
	oauthClient *github.OAuthClient,
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	tokenEncryptor *crypto.TokenEncryptor,
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
		Logger:            logger,
		SessionCookieName: cfg.Auth.SessionCookieName,
		SessionMaxAge:     cfg.Auth.SessionMaxAge,
		SecureCookie:      cfg.Auth.SecureCookie,
		FrontendURL:       cfg.Auth.FrontendURL,
	})
}

func ProvideGitHubHandler(
	cfg *config.Config,
	appClient *github.AppClient,
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
