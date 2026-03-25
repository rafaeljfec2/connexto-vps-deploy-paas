package agentclient

import (
	"crypto/tls"
	"fmt"
	"time"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"github.com/paasdeploy/backend/internal/pki"
)

const defaultAgentPort = 50052

type AgentClient struct {
	pool    *connPool
	timeout time.Duration
}

func NewAgentClient(ca *pki.CertificateAuthority, timeout time.Duration, insecureSkipVerify bool) (*AgentClient, error) {
	cert, err := ca.GenerateClientCert("paasdeploy-backend")
	if err != nil {
		return nil, fmt.Errorf("generate backend client cert: %w", err)
	}
	parsed, err := tls.X509KeyPair(cert.CertPEM, cert.KeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse backend client cert: %w", err)
	}
	return &AgentClient{
		pool:    newConnPool(ca.GetCACertPEM(), &parsed, insecureSkipVerify),
		timeout: timeout,
	}, nil
}

func (c *AgentClient) Close() {
	c.pool.Close()
}

func (c *AgentClient) client(host string, port int) (pb.AgentServiceClient, error) {
	return c.pool.getOrDial(host, port)
}
