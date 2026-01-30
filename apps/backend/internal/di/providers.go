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

	"github.com/paasdeploy/backend/internal/config"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/engine"
	"github.com/paasdeploy/backend/internal/handler"
	"github.com/paasdeploy/backend/internal/repository"
	"github.com/paasdeploy/backend/internal/server"
	"github.com/paasdeploy/backend/internal/service"
)

var ConfigSet = wire.NewSet(
	config.Load,
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
)

var ServiceSet = wire.NewSet(
	service.NewAppService,
)

var EngineSet = wire.NewSet(
	engine.New,
)

var HandlerSet = wire.NewSet(
	ProvideHealthHandler,
	handler.NewAppHandler,
	handler.NewSSEHandler,
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
	ServiceSet,
	EngineSet,
	HandlerSet,
	ServerSet,
	wire.Struct(new(Application), "*"),
)

const Version = "0.1.0"

func ProvideHealthHandler() *handler.HealthHandler {
	return handler.NewHealthHandler(Version)
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

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	h := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(h)
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
	Config        *config.Config
	Logger        *slog.Logger
	DB            *sql.DB
	Engine        *engine.Engine
	Server        *server.Server
	HealthHandler *handler.HealthHandler
	AppHandler    *handler.AppHandler
	SSEHandler    *handler.SSEHandler
}
