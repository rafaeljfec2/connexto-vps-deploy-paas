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
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/engine"
	"github.com/paasdeploy/backend/internal/github"
	"github.com/paasdeploy/backend/internal/handler"
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
	Config         *config.Config
	Logger         *slog.Logger
	DB             *sql.DB
	Engine         *engine.Engine
	Server         *server.Server
	HealthHandler  *handler.HealthHandler
	AppHandler     *handler.AppHandler
	SSEHandler     *handler.SSEHandler
	SwaggerHandler *handler.SwaggerHandler
	EnvVarHandler  *handler.EnvVarHandler
	WebhookHandler *github.WebhookHandler
}
