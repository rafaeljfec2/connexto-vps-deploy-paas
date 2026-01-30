package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/paasdeploy/backend/internal/config"
	"github.com/paasdeploy/backend/internal/database"
	"github.com/paasdeploy/backend/internal/engine"
	"github.com/paasdeploy/backend/internal/handler"
	"github.com/paasdeploy/backend/internal/repository"
	"github.com/paasdeploy/backend/internal/server"
	"github.com/paasdeploy/backend/internal/service"
)

const version = "0.1.0"

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	logger := setupLogger(cfg.Server.LogLevel)
	logger.Info("Starting PaaSDeploy API", "version", version)

	db, err := setupDatabase(cfg.Database.URL)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	migrationsPath := getMigrationsPath()
	if err := database.RunMigrations(db, migrationsPath, logger); err != nil {
		logger.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	appRepo := repository.NewPostgresAppRepository(db)
	deploymentRepo := repository.NewPostgresDeploymentRepository(db)

	appService := service.NewAppService(appRepo, deploymentRepo)

	deployEngine := engine.New(cfg, db, logger)

	healthHandler := handler.NewHealthHandler(version)
	appHandler := handler.NewAppHandler(appService)
	sseHandler := handler.NewSSEHandler()

	go func() {
		for event := range deployEngine.Events() {
			switch event.Type {
			case engine.EventTypeRunning:
				sseHandler.EmitDeployRunning(event.DeployID, event.AppID)
			case engine.EventTypeSuccess:
				sseHandler.EmitDeploySuccess(event.DeployID, event.AppID)
			case engine.EventTypeFailed:
				sseHandler.EmitDeployFailed(event.DeployID, event.AppID, event.Message)
			case engine.EventTypeLog:
				sseHandler.EmitLog(event.DeployID, event.AppID, event.Message)
			}
		}
	}()

	if err := deployEngine.Start(); err != nil {
		logger.Error("Failed to start deploy engine", "error", err)
		os.Exit(1)
	}

	srv := server.New(server.Config{
		Host:         cfg.Server.Host,
		Port:         cfg.Server.Port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}, logger)

	healthHandler.Register(srv.App())
	appHandler.Register(srv.App())
	sseHandler.Register(srv.App())

	go func() {
		if err := srv.Start(); err != nil {
			logger.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	deployEngine.Stop()

	if err := srv.Shutdown(); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Server stopped")
}

func setupLogger(level string) *slog.Logger {
	var logLevel slog.Level
	switch level {
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

	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler)
}

func setupDatabase(url string) (*sql.DB, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func getMigrationsPath() string {
	execPath, err := os.Executable()
	if err != nil {
		return "migrations"
	}

	execDir := filepath.Dir(execPath)

	possiblePaths := []string{
		filepath.Join(execDir, "migrations"),
		filepath.Join(execDir, "..", "..", "migrations"),
		"migrations",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return "migrations"
}
