package grpcclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.AgentServiceClient
}

func New(serverAddr, caCertPath, certPath, keyPath string) (*Client, error) {
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      certPool,
		MinVersion:   tls.VersionTLS13,
	}

	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		client: pb.NewAgentServiceClient(conn),
	}, nil
}

func (c *Client) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	return c.client.Register(ctx, req)
}

func (c *Client) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	return c.client.Heartbeat(ctx, req)
}

func (c *Client) Close() error {
	return c.conn.Close()
}
