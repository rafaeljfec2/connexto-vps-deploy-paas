package grpcserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"

	"github.com/paasdeploy/agent/internal/deploy"
	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"github.com/paasdeploy/shared/pkg/docker"
	"github.com/paasdeploy/shared/pkg/executor"
	"github.com/paasdeploy/shared/pkg/paths"
	"github.com/paasdeploy/shared/pkg/traefik"
)

const (
	logStreamBuffer     = 512
	defaultTraefikURL   = "http://127.0.0.1:8081"
	maxConnectionIdle   = 5 * time.Minute
	keepaliveTime       = 10 * time.Second
	keepaliveTimeout    = 5 * time.Second
	keepaliveMinTime    = 5 * time.Second
	executorTimeout     = 2 * time.Minute
)

type Config struct {
	CAPath   string
	Port     int
	CertPath string
	KeyPath  string
}

type Server struct {
	cfg        Config
	grpcServer *grpc.Server
	docker     *docker.Client
	logger     *slog.Logger
}

type AgentService struct {
	pb.UnimplementedAgentServiceServer
	deployExecutor *deploy.Executor
	docker         *docker.Client
	executor       *executor.Executor
	traefikClient  *traefik.Client
	logStreams     sync.Map
	logger         *slog.Logger
}

func buildTLSConfig(cfg Config, cert tls.Certificate, logger *slog.Logger) (*tls.Config, error) {
	if cfg.CAPath == "" {
		logger.Warn("No CA cert provided — running without mTLS (not recommended for production)")
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.NoClientCert,
			MinVersion:   tls.VersionTLS13,
		}, nil
	}

	caPEM, err := os.ReadFile(cfg.CAPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert: %w", err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("failed to parse CA cert")
	}
	logger.Info("mTLS enabled for gRPC server")
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

func New(cfg Config, logger *slog.Logger) (*Server, error) {
	if cfg.Port == 0 || cfg.CertPath == "" || cfg.KeyPath == "" {
		return nil, fmt.Errorf("missing grpc server config")
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
	if err != nil {
		return nil, err
	}

	tlsConfig, tlsErr := buildTLSConfig(cfg, cert, logger)
	if tlsErr != nil {
		return nil, tlsErr
	}

	grpcServer := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsConfig)),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: maxConnectionIdle,
			Time:              keepaliveTime,
			Timeout:           keepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             keepaliveMinTime,
			PermitWithoutStream: true,
		}),
	)
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	registry := os.Getenv("DOCKER_REGISTRY")
	dockerClient := docker.NewClient(paths.ResolveDataDir(), registry, logger)

	traefikURL := os.Getenv("TRAEFIK_API_URL")
	if traefikURL == "" {
		traefikURL = defaultTraefikURL
	}

	agentService := &AgentService{
		deployExecutor: deploy.NewExecutor(logger),
		docker:         dockerClient,
		executor:       executor.New("", executorTimeout, logger),
		traefikClient:  traefik.NewClient(traefikURL),
		logger:         logger.With("component", "agent-service"),
	}
	pb.RegisterAgentServiceServer(grpcServer, agentService)

	return &Server{
		cfg:        cfg,
		grpcServer: grpcServer,
		docker:     dockerClient,
		logger:     logger.With("component", "agent-grpc"),
	}, nil
}

func (s *Server) Start() error {
	addr := net.JoinHostPort("0.0.0.0", fmt.Sprintf("%d", s.cfg.Port))
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.logger.Info("Agent gRPC listening", "address", addr)
	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}

func (s *Server) Docker() *docker.Client {
	return s.docker
}
