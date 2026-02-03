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
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/paasdeploy/backend/internal/pki"
)

type HealthChecker struct {
	rootCA  []byte
	timeout time.Duration
}

func NewHealthChecker(ca *pki.CertificateAuthority, timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		rootCA:  ca.GetCACertPEM(),
		timeout: timeout,
	}
}

func (c *HealthChecker) Check(ctx context.Context, host string, port int) (time.Duration, error) {
	if host == "" || port == 0 {
		return 0, fmt.Errorf("invalid agent address")
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	roots := x509.NewCertPool()
	if ok := roots.AppendCertsFromPEM(c.rootCA); !ok {
		return 0, fmt.Errorf("invalid CA cert")
	}
	tlsConfig := &tls.Config{
		RootCAs:    roots,
		ServerName: host,
		MinVersion: tls.VersionTLS13,
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	start := time.Now()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)), grpc.WithBlock())
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	client := grpc_health_v1.NewHealthClient(conn)
	if _, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{}); err != nil {
		return 0, err
	}
	return time.Since(start), nil
}
