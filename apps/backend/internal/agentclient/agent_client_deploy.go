package agentclient

import (
	"context"
	"fmt"
	"io"
	"time"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

type DeployLogHandler func(entry *pb.DeployLogEntry)

func (c *AgentClient) ExecuteDeployWithLogs(
	ctx context.Context, host string, port int,
	req *pb.DeployRequest, onLog DeployLogHandler,
) (*pb.DeployResponse, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent: %w", err)
	}
	if onLog != nil {
		ready := make(chan struct{})
		go c.streamLogs(ctx, cl, req.DeploymentId, onLog, ready)
		select {
		case <-ready:
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return cl.ExecuteDeploy(ctx, req)
}

func (c *AgentClient) streamLogs(ctx context.Context, cl pb.AgentServiceClient, deploymentID string, onLog DeployLogHandler, ready chan<- struct{}) {
	stream, err := cl.StreamDeployLogs(ctx, &pb.DeployLogSubscription{DeploymentId: deploymentID})
	close(ready)
	if err != nil {
		return
	}
	for {
		entry, err := stream.Recv()
		if err == io.EOF || err != nil {
			return
		}
		onLog(entry)
	}
}

func (c *AgentClient) ExecuteDeploy(ctx context.Context, host string, port int, req *pb.DeployRequest) (*pb.DeployResponse, error) {
	return c.ExecuteDeployWithLogs(ctx, host, port, req, nil)
}
