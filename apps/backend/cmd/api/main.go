package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/paasdeploy/backend/internal/database"
	"github.com/paasdeploy/backend/internal/di"
	"github.com/paasdeploy/backend/internal/engine"
	"github.com/paasdeploy/backend/internal/handler"
)

func main() {
	_ = godotenv.Load()

	app, cleanup, err := di.InitializeApplication()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	app.Logger.Info("Starting PaaSDeploy API", "version", di.Version)

	migrationsPath := getMigrationsPath()
	if err := database.RunMigrations(app.DB, migrationsPath, app.Logger); err != nil {
		app.Logger.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	go func() {
		for event := range app.Engine.Events() {
			switch event.Type {
			case engine.EventTypeRunning:
				app.SSEHandler.EmitDeployRunning(event.DeployID, event.AppID)
			case engine.EventTypeSuccess:
				app.SSEHandler.EmitDeploySuccess(event.DeployID, event.AppID)
			case engine.EventTypeFailed:
				app.SSEHandler.EmitDeployFailed(event.DeployID, event.AppID, event.Message)
			case engine.EventTypeLog:
				app.SSEHandler.EmitLog(event.DeployID, event.AppID, event.Message)
			case engine.EventTypeHealth:
				if event.Health != nil {
					app.SSEHandler.EmitHealth(event.AppID, handler.SSEHealthStatus{
						Status:    event.Health.Status,
						Health:    event.Health.Health,
						StartedAt: event.Health.StartedAt,
						Uptime:    event.Health.Uptime,
					})
				}
			}
		}
	}()

	if err := app.Engine.Start(); err != nil {
		app.Logger.Error("Failed to start deploy engine", "error", err)
		os.Exit(1)
	}

	app.HealthHandler.Register(app.Server.App())
	app.AppHandler.Register(app.Server.App())
	app.EnvVarHandler.Register(app.Server.App())
	app.SSEHandler.Register(app.Server.App())
	app.ContainerHealthHandler.Register(app.Server.App())
	app.WebhookHandler.Register(app.Server.App())
	app.SwaggerHandler.Register(app.Server.App())

	go func() {
		if err := app.Server.Start(); err != nil {
			app.Logger.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	app.Engine.Stop()

	if err := app.Server.Shutdown(); err != nil {
		app.Logger.Error("Server forced to shutdown", "error", err)
	}

	app.Logger.Info("Server stopped")
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
