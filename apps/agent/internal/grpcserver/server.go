package grpcserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/paasdeploy/agent/internal/deploy"
	"github.com/paasdeploy/agent/internal/sysinfo"
	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

const logStreamBuffer = 512

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
	agentService := &AgentService{
		deployExecutor: deploy.NewExecutor(logger),
		logger:         logger.With("component", "agent-service"),
	}
	pb.RegisterAgentServiceServer(grpcServer, agentService)

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
	deployExecutor *deploy.Executor
	logStreams     sync.Map
	logger         *slog.Logger
}

func (s *AgentService) ExecuteDeploy(ctx context.Context, req *pb.DeployRequest) (*pb.DeployResponse, error) {
	logFn := s.buildLogFunc(req.DeploymentId)
	resp := s.deployExecutor.Execute(ctx, req, logFn)
	s.closeLogStream(req.DeploymentId)
	return resp, nil
}

func (s *AgentService) StreamDeployLogs(sub *pb.DeployLogSubscription, stream pb.AgentService_StreamDeployLogsServer) error {
	ch := make(chan *pb.DeployLogEntry, logStreamBuffer)
	s.logStreams.Store(sub.DeploymentId, ch)
	defer s.logStreams.Delete(sub.DeploymentId)

	s.logger.Info("Deploy log stream opened", "deploymentId", sub.DeploymentId)

	for entry := range ch {
		if err := stream.Send(entry); err != nil {
			s.logger.Warn("Failed to send deploy log entry", "deploymentId", sub.DeploymentId, "error", err)
			return err
		}
	}

	s.logger.Info("Deploy log stream closed", "deploymentId", sub.DeploymentId)
	return nil
}

func (s *AgentService) buildLogFunc(deploymentID string) deploy.LogFunc {
	return func(stage pb.DeployStage, level pb.DeployLogLevel, message string) {
		val, ok := s.logStreams.Load(deploymentID)
		if !ok {
			return
		}
		ch := val.(chan *pb.DeployLogEntry)
		entry := &pb.DeployLogEntry{
			DeploymentId: deploymentID,
			Timestamp:    timestamppb.Now(),
			Level:        level,
			Stage:        stage,
			Message:      message,
		}
		select {
		case ch <- entry:
		default:
			s.logger.Debug("Deploy log channel full, dropping entry", "deploymentId", deploymentID)
		}
	}
}

func (s *AgentService) closeLogStream(deploymentID string) {
	val, ok := s.logStreams.LoadAndDelete(deploymentID)
	if !ok {
		return
	}
	close(val.(chan *pb.DeployLogEntry))
}

func (s *AgentService) GetSystemInfo(ctx context.Context, _ *emptypb.Empty) (*pb.SystemInfo, error) {
	return sysinfo.GetSystemInfo()
}

func (s *AgentService) GetSystemMetrics(ctx context.Context, _ *emptypb.Empty) (*pb.SystemMetrics, error) {
	return sysinfo.GetSystemMetrics()
}
