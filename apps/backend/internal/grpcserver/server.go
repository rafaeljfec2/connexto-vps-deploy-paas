package grpcserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"github.com/paasdeploy/backend/internal/config"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/pki"
)

type Server struct {
	pb.UnimplementedAgentServiceServer

	grpcServer           *grpc.Server
	ca                   *pki.CertificateAuthority
	serverRepo           domain.ServerRepository
	hub                  *AgentHub
	cmdQueue             *AgentCommandQueue
	agentTokenStore      AgentTokenStore
	agentDownloadURL     string
	agentUpdateNotifier  AgentUpdateNotifier
	logger               *slog.Logger
}

type AgentTokenStore interface {
	Create() (string, error)
}

type AgentUpdateNotifier interface {
	NotifyUpdateDelivered(serverID string)
	NotifyUpdateCompleted(serverID, newVersion string)
	EmitAgentUpdateError(serverID, message string)
}

func grpcServerCertHostname(cfg *config.Config) string {
	if cfg.GRPC.ServerAddr != "" {
		if host, _, err := net.SplitHostPort(cfg.GRPC.ServerAddr); err == nil && host != "" {
			return host
		}
	}
	return cfg.Server.Host
}

func NewServer(
	cfg *config.Config,
	ca *pki.CertificateAuthority,
	serverRepo domain.ServerRepository,
	agentTokenStore AgentTokenStore,
	agentUpdateNotifier AgentUpdateNotifier,
	logger *slog.Logger,
) (*Server, error) {
	certHostname := grpcServerCertHostname(cfg)
	serverCert, err := ca.GenerateServerCert(certHostname)
	if err != nil {
		return nil, fmt.Errorf("generate server cert: %w", err)
	}

	cert, err := tls.X509KeyPair(serverCert.CertPEM, serverCert.KeyPEM)
	if err != nil {
		return nil, fmt.Errorf("load server cert: %w", err)
	}

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(ca.GetCACertPEM())

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
		MinVersion:   tls.VersionTLS13,
	}

	grpcServer := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsConfig)),
		grpc.UnaryInterceptor(authInterceptor),
		grpc.StreamInterceptor(streamAuthInterceptor),
	)

	s := &Server{
		grpcServer:          grpcServer,
		ca:                  ca,
		serverRepo:          serverRepo,
		hub:                 NewAgentHub(),
		cmdQueue:            NewAgentCommandQueue(),
		agentTokenStore:     agentTokenStore,
		agentDownloadURL:    buildAgentDownloadURL(cfg),
		agentUpdateNotifier: agentUpdateNotifier,
		logger:              logger.With("component", "grpc"),
	}

	pb.RegisterAgentServiceServer(grpcServer, s)

	return s, nil
}

func (s *Server) Start(address string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	s.logger.Info("gRPC server listening", "address", address)
	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}

func buildAgentDownloadURL(cfg *config.Config) string {
	if cfg.Server.ApiBaseURL == "" || cfg.GRPC.AgentBinaryPath == "" {
		return ""
	}
	return strings.TrimSuffix(cfg.Server.ApiBaseURL, "/") + "/paas-deploy/v1/agent/binary"
}

func (s *Server) EnqueueUpdateAgent(serverID string) {
	payload := ""
	if s.agentDownloadURL == "" {
		s.logger.Warn("agent download URL is empty, update will have no payload", "serverId", serverID)
	} else if s.agentTokenStore == nil {
		s.logger.Warn("agent token store is nil, update will have no payload", "serverId", serverID)
	} else if token, err := s.agentTokenStore.Create(); err != nil {
		s.logger.Error("failed to create agent download token", "serverId", serverID, "error", err)
	} else {
		payload = s.agentDownloadURL + "?token=" + token
	}

	s.logger.Info("enqueuing agent update", "serverId", serverID, "hasPayload", payload != "")
	s.cmdQueue.Enqueue(serverID, &pb.AgentCommand{
		Type:    pb.AgentCommandType_AGENT_COMMAND_UPDATE_AGENT,
		Payload: payload,
	})
}

func authInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	_, err := extractServerIDFromCert(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid certificate")
	}
	return handler(ctx, req)
}

func streamAuthInterceptor(
	srv any,
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	_, err := extractServerIDFromCert(ss.Context())
	if err != nil {
		return status.Error(codes.Unauthenticated, "invalid certificate")
	}
	return handler(srv, ss)
}

func extractServerIDFromCert(ctx context.Context) (string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "", fmt.Errorf("no peer info")
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return "", fmt.Errorf("no TLS info")
	}

	if len(tlsInfo.State.VerifiedChains) == 0 || len(tlsInfo.State.VerifiedChains[0]) == 0 {
		return "", fmt.Errorf("no verified chains")
	}

	cert := tlsInfo.State.VerifiedChains[0][0]
	if len(cert.Subject.OrganizationalUnit) == 0 || cert.Subject.OrganizationalUnit[0] != "agent" {
		return "", fmt.Errorf("invalid certificate OU")
	}

	return cert.Subject.CommonName, nil
}
