package di

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/lmittmann/tint"

	"github.com/paasdeploy/backend/internal/config"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/ghclient"
	"github.com/paasdeploy/backend/internal/repository"
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
	repository.NewPostgresCertificateAuthorityRepository,
	wire.Bind(new(domain.CertificateAuthorityRepository), new(*repository.PostgresCertificateAuthorityRepository)),
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
	repository.NewPostgresCloudflareConnectionRepository,
	wire.Bind(new(domain.CloudflareConnectionRepository), new(*repository.PostgresCloudflareConnectionRepository)),
	repository.NewPostgresCustomDomainRepository,
	wire.Bind(new(domain.CustomDomainRepository), new(*repository.PostgresCustomDomainRepository)),
	repository.NewPostgresNotificationChannelRepository,
	wire.Bind(new(domain.NotificationChannelRepository), new(*repository.PostgresNotificationChannelRepository)),
	repository.NewPostgresNotificationRuleRepository,
	wire.Bind(new(domain.NotificationRuleRepository), new(*repository.PostgresNotificationRuleRepository)),
	repository.NewPostgresServerRepository,
	wire.Bind(new(domain.ServerRepository), new(*repository.PostgresServerRepository)),
	repository.NewPostgresWebhookPayloadRepository,
	wire.Bind(new(ghclient.WebhookPayloadStore), new(*repository.PostgresWebhookPayloadRepository)),
	repository.NewPostgresCleanupLogRepository,
	wire.Bind(new(domain.CleanupLogRepository), new(*repository.PostgresCleanupLogRepository)),
)

func ProvideConfig() (*config.Config, error) {
	cfg := config.Load()
	if err := cfg.EnsureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to ensure directories: %w", err)
	}
	return cfg, nil
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

const (
	dbConnectTimeout  = 10 * time.Second
	dbMaxOpenConns    = 25
	dbMaxIdleConns    = 5
	dbConnMaxLifetime = 5 * time.Minute
	dbPingTimeout     = 15 * time.Second
)

func ProvideDatabase(cfg *config.Config) (*sql.DB, func(), error) {
	connConfig, err := pgx.ParseConfig(cfg.Database.URL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse database URL: %w", err)
	}
	connConfig.ConnectTimeout = dbConnectTimeout

	db := stdlib.OpenDB(*connConfig)
	db.SetMaxOpenConns(dbMaxOpenConns)
	db.SetMaxIdleConns(dbMaxIdleConns)
	db.SetConnMaxLifetime(dbConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), dbPingTimeout)
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
