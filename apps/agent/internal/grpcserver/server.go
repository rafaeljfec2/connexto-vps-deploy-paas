package grpcserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

type Config struct {
	Port     int
	CertPath string
	KeyPath  string
}

type Server struct {
	cfg        Config
	grpcServer *grpc.Server
	logger     *slog.Logger
}

func New(cfg Config, logger *slog.Logger) (*Server, error) {
	if cfg.Port == 0 || cfg.CertPath == "" || cfg.KeyPath == "" {
		return nil, fmt.Errorf("missing grpc server config")
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.NoClientCert,
		MinVersion:   tls.VersionTLS13,
	}

	grpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	pb.RegisterAgentServiceServer(grpcServer, &AgentService{})

	return &Server{
		cfg:        cfg,
		grpcServer: grpcServer,
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

type AgentService struct {
	pb.UnimplementedAgentServiceServer
}

func (s *AgentService) ExecuteDeploy(ctx context.Context, req *pb.DeployRequest) (*pb.DeployResponse, error) {
	return &pb.DeployResponse{
		Success: false,
		Message: "deploy not implemented",
	}, nil
}
