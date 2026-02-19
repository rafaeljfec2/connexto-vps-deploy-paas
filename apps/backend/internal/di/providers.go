package di

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/lmittmann/tint"

	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/agentdownload"
	"github.com/paasdeploy/backend/internal/config"
	"github.com/paasdeploy/backend/internal/crypto"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/engine"
	"github.com/paasdeploy/backend/internal/ghclient"
	"github.com/paasdeploy/backend/internal/grpcserver"
	"github.com/paasdeploy/backend/internal/handler"
	"github.com/paasdeploy/backend/internal/middleware"
	"github.com/paasdeploy/backend/internal/pki"
	"github.com/paasdeploy/backend/internal/provisioner"
	"github.com/paasdeploy/backend/internal/repository"
	"github.com/paasdeploy/backend/internal/server"
	"github.com/paasdeploy/backend/internal/service"
	"github.com/paasdeploy/backend/internal/webhook"
	"github.com/paasdeploy/shared/pkg/cleaner"
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
	ProvideNotificationService,
)

var EngineSet = wire.NewSet(
	ProvideGitTokenProvider,
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
	ProvideAgentHealthChecker,
	ProvideAgentClient,
	ProvideServerHandlerAgentDeps,
	ProvideServerHandler,
)

var ServerSet = wire.NewSet(
	ProvideServerConfig,
	server.New,
)

var AgentDownloadSet = wire.NewSet(
	agentdownload.NewTokenStore,
	ProvideAgentDownloadHandler,
)

var ProvisionerSet = wire.NewSet(
	ProvidePKI,
	ProvideSSHProvisioner,
	ProvideGrpcServer,
	AgentDownloadSet,
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
	ProvisionerSet,
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
		logger.Info("webhook management disabled: GIT_HUB_PAT or GIT_HUB_WEBHOOK_URL not configured")
		return webhook.NewNoOpManager()
	}

	provider := ghclient.NewPATProvider(cfg.GitHub.PAT)
	return webhook.NewGitHubManager(provider, cfg.GitHub.WebhookURL, cfg.GitHub.WebhookSecret)
}

func ProvideAppCleaner(cfg *config.Config, logger *slog.Logger) *cleaner.Cleaner {
	return cleaner.New(cfg.Deploy.DataDir, logger)
}

func ProvideGitTokenProvider(
	appClient *ghclient.AppClient,
	installationRepo domain.InstallationRepository,
	logger *slog.Logger,
) engine.GitTokenProvider {
	if appClient == nil {
		logger.Info("git token provider disabled: GitHub App not configured")
		return nil
	}
	return engine.NewAppGitTokenProvider(appClient, installationRepo, logger)
}

func ProvideAppAdminHandler(appRepo domain.AppRepository, eng *engine.Engine, cfg *config.Config, logger *slog.Logger) *handler.AppAdminHandler {
	return handler.NewAppAdminHandler(appRepo, eng, cfg.Deploy.DataDir, logger)
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

type webhookDeployAuditAdapter struct {
	audit *service.AuditService
}

func (a *webhookDeployAuditAdapter) LogDeployStarted(ctx context.Context, deployID, appID, appName, commitSHA string) {
	if a.audit != nil {
		auditCtx := service.AuditContext{}
		a.audit.LogDeployStarted(ctx, auditCtx, deployID, appID, appName, commitSHA)
	}
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
	connConfig, err := pgx.ParseConfig(cfg.Database.URL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse database URL: %w", err)
	}
	connConfig.ConnectTimeout = 10 * time.Second

	db := stdlib.OpenDB(*connConfig)
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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
		WriteTimeout: 10 * time.Minute,
		IdleTimeout:  60 * time.Second,
		CorsOrigins:  cfg.Server.CorsOrigins,
	}
}

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
	AgentDownloadHandler   *agentdownload.Handler
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

func ProvideDomainHandler(
	cfg *config.Config,
	appRepo domain.AppRepository,
	domainRepo domain.CustomDomainRepository,
	connectionRepo domain.CloudflareConnectionRepository,
	serverRepo domain.ServerRepository,
	tokenEncryptor *crypto.TokenEncryptor,
	eng *engine.Engine,
	logger *slog.Logger,
) *handler.DomainHandler {
	if tokenEncryptor == nil {
		logger.Info("DomainHandler not created: token encryptor not available")
		return nil
	}

	return handler.NewDomainHandler(handler.DomainHandlerConfig{
		AppRepo:        appRepo,
		DomainRepo:     domainRepo,
		ConnectionRepo: connectionRepo,
		ServerRepo:     serverRepo,
		TokenEncryptor: tokenEncryptor,
		ServerIP:       cfg.Cloudflare.ServerIP,
		Logger:         logger,
		DomainUpdater:  eng,
	})
}

func ProvideMigrationHandler(logger *slog.Logger) *handler.MigrationHandler {
	return handler.NewMigrationHandler(logger)
}

func ProvideContainerHandler(eng *engine.Engine, logger *slog.Logger) *handler.ContainerHandler {
	return handler.NewContainerHandler(eng.Docker(), logger)
}

func ProvideContainerExecHandler(logger *slog.Logger) *handler.ContainerExecHandler {
	return handler.NewContainerExecHandler(logger)
}

func ProvideTemplateHandler(eng *engine.Engine, logger *slog.Logger) *handler.TemplateHandler {
	return handler.NewTemplateHandler(eng.Docker(), logger)
}

func ProvideImageHandler(eng *engine.Engine, logger *slog.Logger) *handler.ImageHandler {
	return handler.NewImageHandler(eng.Docker(), logger)
}

func ProvideAuditService(db *sql.DB, logger *slog.Logger) *service.AuditService {
	repo := repository.NewPostgresAuditLogRepository(db)
	return service.NewAuditService(repo, logger)
}

func ProvideAuditHandler(auditService *service.AuditService, webhookPayloadRepo *repository.PostgresWebhookPayloadRepository) *handler.AuditHandler {
	return handler.NewAuditHandler(auditService, webhookPayloadRepo)
}

func ProvideResourceHandler(eng *engine.Engine, logger *slog.Logger) *handler.ResourceHandler {
	return handler.NewResourceHandler(eng.Docker(), logger)
}

func ProvideCertificateHandler(cfg *config.Config, logger *slog.Logger) *handler.CertificateHandler {
	return handler.NewCertificateHandler(handler.CertificateHandlerConfig{
		TraefikURL: cfg.Traefik.URL,
		Logger:     logger,
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

func ProvidePKI(
	logger *slog.Logger,
	caRepo domain.CertificateAuthorityRepository,
) (*pki.CertificateAuthority, error) {
	record, err := caRepo.GetRoot()
	if err == nil {
		ca, loadErr := pki.LoadCA(record.CertPEM, record.KeyPEM)
		if loadErr != nil {
			return nil, fmt.Errorf("load CA: %w", loadErr)
		}
		logger.Info("PKI CA loaded")
		return ca, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("read CA: %w", err)
	}

	ca, err := pki.NewCA()
	if err != nil {
		return nil, fmt.Errorf("create CA: %w", err)
	}
	if err := caRepo.UpsertRoot(domain.CertificateAuthorityRecord{
		CertPEM: ca.GetCACertPEM(),
		KeyPEM:  ca.GetCAKeyPEM(),
	}); err != nil {
		return nil, fmt.Errorf("persist CA: %w", err)
	}
	logger.Info("PKI CA initialized")
	return ca, nil
}

func ProvideAgentHealthChecker(ca *pki.CertificateAuthority, cfg *config.Config) *agentclient.HealthChecker {
	timeout := 10 * time.Second
	if cfg.Deploy.HealthCheckTimeout > 0 {
		timeout = cfg.Deploy.HealthCheckTimeout
	}
	return agentclient.NewHealthChecker(ca, timeout, cfg.GRPC.AgentTLSInsecureSkipVerify)
}

func ProvideAgentClient(ca *pki.CertificateAuthority, cfg *config.Config) *agentclient.AgentClient {
	timeout := 10 * time.Second
	if cfg.Deploy.HealthCheckTimeout > 0 {
		timeout = cfg.Deploy.HealthCheckTimeout
	}
	return agentclient.NewAgentClient(ca, timeout, cfg.GRPC.AgentTLSInsecureSkipVerify)
}

func ProvideSSHProvisioner(
	ca *pki.CertificateAuthority,
	cfg *config.Config,
	logger *slog.Logger,
	serverRepo domain.ServerRepository,
) *provisioner.SSHProvisioner {
	serverAddr := cfg.GRPC.ServerAddr
	if serverAddr == "" && cfg.GRPC.Port > 0 {
		serverAddr = fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.GRPC.Port)
		if cfg.Server.Host == "0.0.0.0" {
			serverAddr = fmt.Sprintf("localhost:%d", cfg.GRPC.Port)
		}
	}
	return provisioner.NewSSHProvisioner(provisioner.SSHProvisionerConfig{
		CA:              ca,
		ServerAddr:      serverAddr,
		AgentBinaryPath: cfg.GRPC.AgentBinaryPath,
		AgentPort:       cfg.GRPC.AgentPort,
		Logger:          logger,
		HostKeyStore:    serverRepo,
	})
}

func ProvideAgentDownloadHandler(
	store *agentdownload.TokenStore,
	cfg *config.Config,
	logger *slog.Logger,
) *agentdownload.Handler {
	return agentdownload.NewHandler(store, cfg.GRPC.AgentBinaryPath, logger)
}

func ProvideGrpcServer(
	cfg *config.Config,
	ca *pki.CertificateAuthority,
	serverRepo domain.ServerRepository,
	agentTokenStore *agentdownload.TokenStore,
	sseHandler *handler.SSEHandler,
	logger *slog.Logger,
) *grpcserver.Server {
	server, err := grpcserver.NewServer(cfg, ca, serverRepo, agentTokenStore, sseHandler, logger)
	if err != nil {
		logger.Error("failed to create gRPC server", "error", err)
		return nil
	}
	return server
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
