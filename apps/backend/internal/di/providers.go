package di

import (
	"database/sql"
	"log/slog"

	"github.com/google/wire"

	"github.com/paasdeploy/backend/internal/agentdownload"
	"github.com/paasdeploy/backend/internal/config"
	"github.com/paasdeploy/backend/internal/engine"
	"github.com/paasdeploy/backend/internal/ghclient"
	"github.com/paasdeploy/backend/internal/grpcserver"
	"github.com/paasdeploy/backend/internal/handler"
	"github.com/paasdeploy/backend/internal/middleware"
	"github.com/paasdeploy/backend/internal/server"
	"github.com/paasdeploy/backend/internal/service"
)

const Version = "0.2.0"

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
	ProvisionerSet,
	HandlerSet,
	ServerSet,
	wire.Struct(new(Application), "*"),
)

type Application struct {
	Config                 *config.Config
	Logger                 *slog.Logger
	DB                     *sql.DB
	Engine                 *engine.Engine
	Server                 *server.Server
	GrpcServer             *grpcserver.Server
	HealthHandler          *handler.HealthHandler
	AppHandler             *handler.AppHandler
	SSEHandler             *handler.SSEHandler
	SwaggerHandler         *handler.SwaggerHandler
	EnvVarHandler          *handler.EnvVarHandler
	ContainerHealthHandler *handler.ContainerHealthHandler
	AppAdminHandler        *handler.AppAdminHandler
	WebhookHandler         *ghclient.WebhookHandler
	AuthHandler            *handler.AuthHandler
	GitHubHandler          *handler.GitHubHandler
	AuthMiddleware         *middleware.AuthMiddleware
	CloudflareAuthHandler  *handler.CloudflareAuthHandler
	DomainHandler          *handler.DomainHandler
	MigrationHandler       *handler.MigrationHandler
	ContainerHandler       *handler.ContainerHandler
	ContainerExecHandler   *handler.ContainerExecHandler
	TemplateHandler        *handler.TemplateHandler
	ImageHandler           *handler.ImageHandler
	CertificateHandler     *handler.CertificateHandler
	AuditService           *service.AuditService
	AuditHandler           *handler.AuditHandler
	ResourceHandler        *handler.ResourceHandler
	NotificationService    *service.NotificationService
	NotificationHandler    *handler.NotificationHandler
	ServerHandler          *handler.ServerHandler
	SystemHandler          *handler.SystemHandler
	AgentDownloadHandler   *agentdownload.Handler
	CleanupHandler         *handler.CleanupHandler
	ContainerSSLHandler    *handler.ContainerSSLHandler
}
