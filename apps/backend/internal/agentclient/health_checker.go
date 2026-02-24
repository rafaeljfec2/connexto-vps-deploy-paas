package agentclient

import (
	"context"
	"fmt"
	"time"

	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type HealthChecker struct {
	pool    *connPool
	timeout time.Duration
}

func NewHealthChecker(ac *AgentClient, timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		pool:    ac.pool,
		timeout: timeout,
	}
}

func (c *HealthChecker) Check(ctx context.Context, host string, port int) (time.Duration, error) {
	if host == "" || port == 0 {
		return 0, fmt.Errorf("invalid agent address")
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	start := time.Now()
	conn, err := c.pool.getOrDialConn(host, port)
	if err != nil {
		return 0, err
	}

	client := grpc_health_v1.NewHealthClient(conn)
	if _, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{}); err != nil {
		return 0, err
	}
	return time.Since(start), nil
}
