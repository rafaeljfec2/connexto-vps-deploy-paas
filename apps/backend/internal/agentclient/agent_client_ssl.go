package agentclient

import (
	"context"
	"fmt"
	"time"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

const sslConfigTimeout = 3 * time.Minute

func (c *AgentClient) ConfigureContainerSSL(ctx context.Context, host string, port int, req *pb.ConfigureContainerSSLRequest) (*pb.ConfigureContainerSSLResponse, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, sslConfigTimeout)
	defer cancel()
	resp, err := cl.ConfigureContainerSSL(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("configure container SSL: %w", err)
	}
	return resp, nil
}

func (c *AgentClient) GetContainerSSLStatus(ctx context.Context, host string, port int, req *pb.GetContainerSSLStatusRequest) (*pb.GetContainerSSLStatusResponse, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := cl.GetContainerSSLStatus(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get container SSL status: %w", err)
	}
	return resp, nil
}
