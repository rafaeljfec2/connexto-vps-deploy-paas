package di

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/wire"

	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/agentdownload"
	"github.com/paasdeploy/backend/internal/config"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/engine"
	"github.com/paasdeploy/backend/internal/ghclient"
	"github.com/paasdeploy/backend/internal/grpcserver"
	"github.com/paasdeploy/backend/internal/handler"
	"github.com/paasdeploy/backend/internal/pki"
	"github.com/paasdeploy/backend/internal/provisioner"
	"github.com/paasdeploy/backend/internal/server"
)

var EngineSet = wire.NewSet(
	ProvideGitTokenProvider,
	engine.New,
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

const (
	httpReadTimeout  = 15 * time.Second
	httpWriteTimeout = 10 * time.Minute
	httpIdleTimeout  = 60 * time.Second
	defaultAgentTimeout = 10 * time.Second
)

func ProvideServerConfig(cfg *config.Config) server.Config {
	return server.Config{
		Host:         cfg.Server.Host,
		Port:         cfg.Server.Port,
		ReadTimeout:  httpReadTimeout,
		WriteTimeout: httpWriteTimeout,
		IdleTimeout:  httpIdleTimeout,
		CorsOrigins:  cfg.Server.CorsOrigins,
	}
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

func ProvideAgentClient(ca *pki.CertificateAuthority, cfg *config.Config) (*agentclient.AgentClient, error) {
	timeout := defaultAgentTimeout
	if cfg.Deploy.HealthCheckTimeout > 0 {
		timeout = cfg.Deploy.HealthCheckTimeout
	}
	return agentclient.NewAgentClient(ca, timeout, cfg.GRPC.AgentTLSInsecureSkipVerify)
}

func ProvideAgentHealthChecker(ac *agentclient.AgentClient, cfg *config.Config) *agentclient.HealthChecker {
	timeout := defaultAgentTimeout
	if cfg.Deploy.HealthCheckTimeout > 0 {
		timeout = cfg.Deploy.HealthCheckTimeout
	}
	return agentclient.NewHealthChecker(ac, timeout)
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

func ProvideAgentDownloadHandler(
	store *agentdownload.TokenStore,
	cfg *config.Config,
	logger *slog.Logger,
) *agentdownload.Handler {
	return agentdownload.NewHandler(store, cfg.GRPC.AgentBinaryPath, logger)
}
