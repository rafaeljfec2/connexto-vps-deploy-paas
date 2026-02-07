package agentclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"github.com/paasdeploy/backend/internal/pki"
)

const defaultAgentPort = 50052

type AgentClient struct {
	rootCA                 []byte
	timeout                time.Duration
	insecureSkipVerify     bool
}

func NewAgentClient(ca *pki.CertificateAuthority, timeout time.Duration, insecureSkipVerify bool) *AgentClient {
	return &AgentClient{
		rootCA:             ca.GetCACertPEM(),
		timeout:            timeout,
		insecureSkipVerify: insecureSkipVerify,
	}
}

func (c *AgentClient) dial(host string, port int) (pb.AgentServiceClient, func(), error) {
	if port == 0 {
		port = defaultAgentPort
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	tlsConfig := &tls.Config{
		ServerName:         host,
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: c.insecureSkipVerify,
	}
	if !c.insecureSkipVerify {
		roots := x509.NewCertPool()
		if ok := roots.AppendCertsFromPEM(c.rootCA); !ok {
			return nil, nil, fmt.Errorf("invalid CA cert")
		}
		tlsConfig.RootCAs = roots
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return nil, nil, fmt.Errorf("grpc dial: %w", err)
	}
	client := pb.NewAgentServiceClient(conn)
	return client, func() { conn.Close() }, nil
}

func (c *AgentClient) GetSystemInfo(ctx context.Context, host string, port int) (*pb.SystemInfo, error) {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return client.GetSystemInfo(ctx, &emptypb.Empty{})
}

func (c *AgentClient) GetSystemMetrics(ctx context.Context, host string, port int) (*pb.SystemMetrics, error) {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return client.GetSystemMetrics(ctx, &emptypb.Empty{})
}

func (c *AgentClient) ExecuteDeploy(ctx context.Context, host string, port int, req *pb.DeployRequest) (*pb.DeployResponse, error) {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent: %w", err)
	}
	defer cleanup()

	return client.ExecuteDeploy(ctx, req)
}
