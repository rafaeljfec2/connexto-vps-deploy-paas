package di

import (
	"database/sql"
	"log/slog"

	"github.com/google/wire"

	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/config"
	"github.com/paasdeploy/backend/internal/crypto"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/engine"
	"github.com/paasdeploy/backend/internal/ghclient"
	"github.com/paasdeploy/backend/internal/grpcserver"
	"github.com/paasdeploy/backend/internal/handler"
	"github.com/paasdeploy/backend/internal/provisioner"
	"github.com/paasdeploy/backend/internal/repository"
	"github.com/paasdeploy/backend/internal/service"
	"github.com/paasdeploy/backend/internal/webhook"
	"github.com/paasdeploy/shared/pkg/cleaner"
)

var WebhookSet = wire.NewSet(
	ProvideWebhookManager,
)

var ServiceSet = wire.NewSet(
	ProvideAppCleaner,
	ProvideAppService,
	ProvideNotificationService,
)

var HandlerSet = wire.NewSet(
	ProvideHealthHandler,
	handler.NewAppHandler,
	handler.NewSSEHandler,
	handler.NewSwaggerHandler,
	handler.NewEnvVarHandler,
	handler.NewContainerHealthHandler,
	ProvideAppAdminHandler,
	ProvideCloudflareAuthHandler,
	ProvideDomainHandler,
	ProvideMigrationHandler,
	ProvideContainerHandler,
	ProvideContainerExecHandler,
	ProvideTemplateHandler,
	ProvideImageHandler,
	ProvideCertificateHandler,
	ProvideAuditService,
	ProvideAuditHandler,
	ProvideNotificationHandler,
	ProvideResourceHandler,
	handler.NewSystemHandler,
	ProvideAgentClient,
	ProvideAgentHealthChecker,
	ProvideServerHandlerAgentDeps,
	ProvideServerHandler,
	ProvideCleanupHandler,
	ProvideContainerSSLHandler,
)

func ProvideHealthHandler() *handler.HealthHandler {
	return handler.NewHealthHandler(Version)
}

func ProvideWebhookManager(cfg *config.Config, logger *slog.Logger) webhook.Manager {
	if cfg.GitHub.PAT == "" || cfg.GitHub.WebhookURL == "" {
		logger.Info("webhook management disabled: GIT_HUB_PAT or GIT_HUB_WEBHOOK_URL not configured")
		return webhook.NewNoOpManager()
	}

	provider := ghclient.NewPATProvider(cfg.GitHub.PAT)
	return webhook.NewGitHubManager(provider, cfg.GitHub.WebhookURL, cfg.GitHub.WebhookSecret)
}

func ProvideAppCleaner(cfg *config.Config, logger *slog.Logger) *cleaner.Cleaner {
	return cleaner.New(cfg.Deploy.DataDir, logger)
}

func ProvideAppService(
	appRepo domain.AppRepository,
	deploymentRepo domain.DeploymentRepository,
	envVarRepo domain.EnvVarRepository,
	webhookManager webhook.Manager,
	appCleaner *cleaner.Cleaner,
	logger *slog.Logger,
) *service.AppService {
	return service.NewAppService(appRepo, deploymentRepo, envVarRepo, webhookManager, appCleaner, logger)
}

type AppAdminHandlerDeps struct {
	AppRepo          domain.AppRepository
	ServerRepo       domain.ServerRepository
	CustomDomainRepo domain.CustomDomainRepository
	EnvVarRepo       domain.EnvVarRepository
	Engine           *engine.Engine
	AgentClient      *agentclient.AgentClient
	Config           *config.Config
	Logger           *slog.Logger
}

func ProvideAppAdminHandler(deps AppAdminHandlerDeps) *handler.AppAdminHandler {
	return handler.NewAppAdminHandler(handler.AppAdminHandlerConfig{
		AppRepo:          deps.AppRepo,
		ServerRepo:       deps.ServerRepo,
		CustomDomainRepo: deps.CustomDomainRepo,
		EnvVarRepo:       deps.EnvVarRepo,
		Engine:           deps.Engine,
		AgentClient:      deps.AgentClient,
		AgentPort:        deps.Config.GRPC.AgentPort,
		DataDir:          deps.Config.Deploy.DataDir,
		Logger:           deps.Logger,
	})
}

type DomainHandlerDeps struct {
	Config         *config.Config
	AppRepo        domain.AppRepository
	DomainRepo     domain.CustomDomainRepository
	ConnectionRepo domain.CloudflareConnectionRepository
	ServerRepo     domain.ServerRepository
	TokenEncryptor *crypto.TokenEncryptor
	Engine         *engine.Engine
	Logger         *slog.Logger
}

func ProvideDomainHandler(deps DomainHandlerDeps) *handler.DomainHandler {
	if deps.TokenEncryptor == nil {
		deps.Logger.Info("DomainHandler not created: token encryptor not available")
		return nil
	}

	return handler.NewDomainHandler(handler.DomainHandlerConfig{
		AppRepo:        deps.AppRepo,
		DomainRepo:     deps.DomainRepo,
		ConnectionRepo: deps.ConnectionRepo,
		ServerRepo:     deps.ServerRepo,
		TokenEncryptor: deps.TokenEncryptor,
		ServerIP:       deps.Config.Cloudflare.ServerIP,
		Logger:         deps.Logger,
		DomainUpdater:  deps.Engine,
	})
}

func ProvideMigrationHandler(logger *slog.Logger) *handler.MigrationHandler {
	return handler.NewMigrationHandler(logger)
}

func ProvideContainerHandler(
	eng *engine.Engine,
	serverRepo domain.ServerRepository,
	agentClient *agentclient.AgentClient,
	cfg *config.Config,
	logger *slog.Logger,
) *handler.ContainerHandler {
	return handler.NewContainerHandler(handler.ContainerHandlerConfig{
		Docker:      eng.Docker(),
		AgentClient: agentClient,
		ServerRepo:  serverRepo,
		AgentPort:   cfg.GRPC.AgentPort,
		Logger:      logger,
	})
}

func ProvideContainerExecHandler(
	serverRepo domain.ServerRepository,
	agentClient *agentclient.AgentClient,
	cfg *config.Config,
	logger *slog.Logger,
) *handler.ContainerExecHandler {
	return handler.NewContainerExecHandler(handler.ContainerExecHandlerConfig{
		AgentClient: agentClient,
		ServerRepo:  serverRepo,
		AgentPort:   cfg.GRPC.AgentPort,
		Logger:      logger,
	})
}

func ProvideTemplateHandler(
	eng *engine.Engine,
	serverRepo domain.ServerRepository,
	agentClient *agentclient.AgentClient,
	cfg *config.Config,
	logger *slog.Logger,
) *handler.TemplateHandler {
	return handler.NewTemplateHandler(handler.TemplateHandlerConfig{
		Docker:      eng.Docker(),
		AgentClient: agentClient,
		ServerRepo:  serverRepo,
		AgentPort:   cfg.GRPC.AgentPort,
		Logger:      logger,
	})
}

func ProvideImageHandler(
	eng *engine.Engine,
	serverRepo domain.ServerRepository,
	agentClient *agentclient.AgentClient,
	cfg *config.Config,
	logger *slog.Logger,
) *handler.ImageHandler {
	return handler.NewImageHandler(handler.ImageHandlerConfig{
		Docker:      eng.Docker(),
		AgentClient: agentClient,
		ServerRepo:  serverRepo,
		AgentPort:   cfg.GRPC.AgentPort,
		Logger:      logger,
	})
}

func ProvideAuditService(db *sql.DB, logger *slog.Logger) *service.AuditService {
	repo := repository.NewPostgresAuditLogRepository(db)
	return service.NewAuditService(repo, logger)
}

func ProvideAuditHandler(auditService *service.AuditService, webhookPayloadRepo *repository.PostgresWebhookPayloadRepository) *handler.AuditHandler {
	return handler.NewAuditHandler(auditService, webhookPayloadRepo)
}

func ProvideResourceHandler(
	eng *engine.Engine,
	serverRepo domain.ServerRepository,
	agentClient *agentclient.AgentClient,
	cfg *config.Config,
	logger *slog.Logger,
) *handler.ResourceHandler {
	return handler.NewResourceHandler(handler.ResourceHandlerConfig{
		Docker:      eng.Docker(),
		AgentClient: agentClient,
		ServerRepo:  serverRepo,
		AgentPort:   cfg.GRPC.AgentPort,
		Logger:      logger,
	})
}

func ProvideCertificateHandler(
	cfg *config.Config,
	serverRepo domain.ServerRepository,
	agentClient *agentclient.AgentClient,
	logger *slog.Logger,
) *handler.CertificateHandler {
	return handler.NewCertificateHandler(handler.CertificateHandlerConfig{
		TraefikURL:  cfg.Traefik.URL,
		AgentClient: agentClient,
		ServerRepo:  serverRepo,
		AgentPort:   cfg.GRPC.AgentPort,
		Logger:      logger,
	})
}

func ProvideNotificationService(
	channelRepo domain.NotificationChannelRepository,
	ruleRepo domain.NotificationRuleRepository,
	appRepo domain.AppRepository,
	logger *slog.Logger,
) *service.NotificationService {
	return service.NewNotificationService(channelRepo, ruleRepo, appRepo, logger)
}

func ProvideNotificationHandler(
	channelRepo domain.NotificationChannelRepository,
	ruleRepo domain.NotificationRuleRepository,
	appRepo domain.AppRepository,
	logger *slog.Logger,
) *handler.NotificationHandler {
	return handler.NewNotificationHandler(channelRepo, ruleRepo, appRepo, logger)
}

func ProvideServerHandlerAgentDeps(
	healthChecker *agentclient.HealthChecker,
	agentClient *agentclient.AgentClient,
	cfg *config.Config,
	grpcServer *grpcserver.Server,
) handler.ServerHandlerAgentDeps {
	return handler.ServerHandlerAgentDeps{
		HealthChecker:       healthChecker,
		AgentClient:         agentClient,
		AgentPort:           cfg.GRPC.AgentPort,
		AgentBinaryPath:     cfg.GRPC.AgentBinaryPath,
		UpdateAgentEnqueuer: grpcServer,
	}
}

func ProvideServerHandler(
	serverRepo domain.ServerRepository,
	tokenEncryptor *crypto.TokenEncryptor,
	prov *provisioner.SSHProvisioner,
	sseHandler *handler.SSEHandler,
	agentDeps handler.ServerHandlerAgentDeps,
	appService *service.AppService,
	logger *slog.Logger,
) *handler.ServerHandler {
	return handler.NewServerHandler(
		serverRepo,
		tokenEncryptor,
		prov,
		sseHandler,
		agentDeps,
		appService,
		logger,
	)
}

func ProvideContainerSSLHandler(
	serverRepo domain.ServerRepository,
	agentClient *agentclient.AgentClient,
	cfg *config.Config,
	logger *slog.Logger,
) *handler.ContainerSSLHandler {
	return handler.NewContainerSSLHandler(handler.ContainerSSLHandlerConfig{
		AgentClient: agentClient,
		ServerRepo:  serverRepo,
		AgentPort:   cfg.GRPC.AgentPort,
		Logger:      logger,
	})
}

func ProvideCleanupHandler(
	serverRepo domain.ServerRepository,
	cleanupLogRepo domain.CleanupLogRepository,
	agentClient *agentclient.AgentClient,
	cfg *config.Config,
	logger *slog.Logger,
) *handler.CleanupHandler {
	return handler.NewCleanupHandler(handler.CleanupHandlerConfig{
		AgentClient:    agentClient,
		ServerRepo:     serverRepo,
		CleanupLogRepo: cleanupLogRepo,
		AgentPort:      cfg.GRPC.AgentPort,
		Logger:         logger,
	})
}
