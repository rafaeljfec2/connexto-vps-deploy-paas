package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"

	"github.com/paasdeploy/backend/internal/database"
	"github.com/paasdeploy/backend/internal/di"
	"github.com/paasdeploy/backend/internal/engine"
	"github.com/paasdeploy/backend/internal/handler"
	"github.com/paasdeploy/backend/internal/server"
)

func main() {
	_ = godotenv.Load()

	if err := runMigrationsFirst(); err != nil {
		panic(err)
	}

	app, cleanup, err := di.InitializeApplication()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	app.Logger.Info("Starting FlowDeploy API", "version", di.Version)
	go handleEngineEvents(app)
	startEngine(app)
	startGrpcServer(app)
	registerHandlers(app)
	registerProtectedRoutes(app)
	startServer(app)
	waitForShutdown(app)
}

func runMigrationsFirst() error {
	cfg, err := di.ProvideConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	logger := di.ProvideLogger(cfg)
	db, cleanup, err := di.ProvideDatabase(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer cleanup()

	migrationsPath := getMigrationsPath()
	if err := database.RunMigrations(db, migrationsPath, logger); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

func handleEngineEvents(app *di.Application) {
	for event := range app.Engine.Events() {
		processEvent(app, event)
	}
}

func processEvent(app *di.Application, event engine.DeployEvent) {
	switch event.Type {
	case engine.EventTypeRunning:
		app.SSEHandler.EmitDeployRunning(event.DeployID, event.AppID)
		app.NotificationService.NotifyDeployRunning(event.DeployID, event.AppID)
	case engine.EventTypeSuccess:
		app.SSEHandler.EmitDeploySuccess(event.DeployID, event.AppID)
		app.NotificationService.NotifyDeploySuccess(event.DeployID, event.AppID)
	case engine.EventTypeFailed:
		app.SSEHandler.EmitDeployFailed(event.DeployID, event.AppID, event.Message)
		app.NotificationService.NotifyDeployFailed(event.DeployID, event.AppID, event.Message)
	case engine.EventTypeLog:
		app.SSEHandler.EmitLog(event.DeployID, event.AppID, event.Message)
	case engine.EventTypeHealth:
		emitHealthEvent(app, event)
		emitNotificationHealth(app, event)
	case engine.EventTypeStats:
		emitStatsEvent(app, event)
	}
}

func emitHealthEvent(app *di.Application, event engine.DeployEvent) {
	if event.Health == nil {
		return
	}
	app.SSEHandler.EmitHealth(event.AppID, handler.SSEHealthStatus{
		Status:    event.Health.Status,
		Health:    event.Health.Health,
		StartedAt: event.Health.StartedAt,
		Uptime:    event.Health.Uptime,
	})
}

func emitNotificationHealth(app *di.Application, event engine.DeployEvent) {
	if event.Health == nil {
		return
	}
	app.NotificationService.NotifyHealthChange(event.AppID, event.Health.Status, event.Health.Health)
}

func emitStatsEvent(app *di.Application, event engine.DeployEvent) {
	if event.Stats == nil {
		return
	}
	app.SSEHandler.EmitStats(event.AppID, handler.SSEContainerStats{
		CPUPercent:    event.Stats.CPUPercent,
		MemoryUsage:   event.Stats.MemoryUsage,
		MemoryLimit:   event.Stats.MemoryLimit,
		MemoryPercent: event.Stats.MemoryPercent,
		NetworkRx:     event.Stats.NetworkRx,
		NetworkTx:     event.Stats.NetworkTx,
		PIDs:          event.Stats.PIDs,
	})
}

func startEngine(app *di.Application) {
	if err := app.Engine.Start(); err != nil {
		app.Logger.Error("Failed to start deploy engine", "error", err)
		os.Exit(1)
	}
}

func startGrpcServer(app *di.Application) {
	if !app.Config.GRPC.Enabled || app.GrpcServer == nil {
		return
	}
	address := fmt.Sprintf("%s:%d", app.Config.Server.Host, app.Config.GRPC.Port)
	go func() {
		if err := app.GrpcServer.Start(address); err != nil {
			app.Logger.Error("gRPC server error", "error", err)
		}
	}()
}

func registerHandlers(app *di.Application) {
	app.HealthHandler.Register(app.Server.App())
	app.SwaggerHandler.Register(app.Server.App())

	if app.AuthHandler != nil {
		app.AuthHandler.Register(app.Server.App(), server.AuthRateLimiter())
	}

	if app.GitHubHandler != nil {
		app.GitHubHandler.Register(app.Server.App())
	}

	if app.AuthMiddleware != nil {
		app.Server.App().Use(app.AuthMiddleware.Optional())
	}

	app.WebhookHandler.Register(app.Server.App())

	if app.AgentDownloadHandler != nil {
		app.Server.App().Get("/paas-deploy/v1/agent/binary", app.AgentDownloadHandler.ServeBinary)
	}
}

func registerProtectedRoutes(app *di.Application) {
	if app.AuthHandler == nil || app.AuthMiddleware == nil {
		return
	}

	authRequired := app.Server.App().Group("")
	authRequired.Use(app.AuthMiddleware.Require())
	app.AuthHandler.RegisterProtected(authRequired)

	registerOptionalProtectedHandler(app.GitHubHandler, authRequired)
	registerOptionalProtectedHandler(app.CloudflareAuthHandler, authRequired)
	registerOptionalProtectedHandler(app.DomainHandler, authRequired)
	registerOptionalProtectedHandler(app.MigrationHandler, authRequired)
	registerOptionalProtectedHandler(app.NotificationHandler, authRequired)
	registerOptionalProtectedHandler(app.ServerHandler, authRequired)

	app.AppHandler.Register(authRequired)
	app.EnvVarHandler.Register(authRequired)
	app.SSEHandler.Register(authRequired)
	app.ContainerHealthHandler.Register(authRequired)
	app.AppAdminHandler.Register(authRequired)
	app.ContainerHandler.Register(authRequired)
	app.ContainerExecHandler.Register(authRequired)
	app.TemplateHandler.Register(authRequired)

	registerOptionalProtectedHandler(app.ImageHandler, authRequired)
	registerOptionalProtectedHandler(app.ResourceHandler, authRequired)

	app.SystemHandler.Register(authRequired)

	if app.CertificateHandler != nil {
		app.CertificateHandler.RegisterRoutes(authRequired.Group("/api"))
	}

	if app.AuditHandler != nil {
		app.AuditHandler.Register(authRequired.Group(handler.APIPrefix))
	}
}

type protectedRegistrar interface {
	Register(router fiber.Router)
}

type gitHubProtectedRegistrar interface {
	RegisterProtected(router fiber.Router)
}

func registerOptionalProtectedHandler(h any, router fiber.Router) {
	if h == nil {
		return
	}
	if registrar, ok := h.(protectedRegistrar); ok {
		registrar.Register(router)
	}
	if registrar, ok := h.(gitHubProtectedRegistrar); ok {
		registrar.RegisterProtected(router)
	}
}

func startServer(app *di.Application) {
	go func() {
		if err := app.Server.Start(); err != nil {
			app.Logger.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()
}

func waitForShutdown(app *di.Application) {
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
