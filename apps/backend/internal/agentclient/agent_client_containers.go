package agentclient

import (
	"context"
	"fmt"
	"io"
	"time"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

type ContainerLogHandler func(entry *pb.ContainerLogEntry)

type ContainerStatsHandler func(stats *pb.ContainerStats)

type ExecContainerStream struct {
	Stream pb.AgentService_ExecContainerClient
}

func (c *AgentClient) ListContainers(ctx context.Context, host string, port int, all bool, appID string) ([]*pb.ContainerInfo, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	req := &pb.ListContainersRequest{All: all}
	if appID != "" {
		req.AppId = &appID
	}
	resp, err := cl.ListContainers(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}
	return resp.Containers, nil
}

func (c *AgentClient) GetContainerLogs(ctx context.Context, host string, port int, containerID string, tail int, follow bool, onLog ContainerLogHandler) error {
	cl, err := c.client(host, port)
	if err != nil {
		return err
	}
	stream, err := cl.GetContainerLogs(ctx, &pb.ContainerLogsRequest{
		ContainerId: containerID, Tail: int32(tail), Follow: follow, Timestamps: true,
	})
	if err != nil {
		return fmt.Errorf("get container logs: %w", err)
	}
	for {
		entry, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		onLog(entry)
	}
}

func (c *AgentClient) GetContainerStats(ctx context.Context, host string, port int, containerID string, onStats ContainerStatsHandler) error {
	cl, err := c.client(host, port)
	if err != nil {
		return err
	}
	stream, err := cl.GetContainerStats(ctx, &pb.ContainerStatsRequest{ContainerId: containerID, Stream: false})
	if err != nil {
		return fmt.Errorf("get container stats: %w", err)
	}
	for {
		stats, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		onStats(stats)
	}
}

func (c *AgentClient) RestartContainer(ctx context.Context, host string, port int, containerID string) error {
	cl, err := c.client(host, port)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := cl.RestartContainer(ctx, &pb.RestartContainerRequest{ContainerId: containerID})
	if err != nil {
		return fmt.Errorf("restart container: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("restart container failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) StopContainer(ctx context.Context, host string, port int, containerID string) error {
	cl, err := c.client(host, port)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := cl.StopContainer(ctx, &pb.StopContainerRequest{ContainerId: containerID})
	if err != nil {
		return fmt.Errorf("stop container: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("stop container failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) StartContainer(ctx context.Context, host string, port int, containerID string) error {
	cl, err := c.client(host, port)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := cl.StartContainer(ctx, &pb.StartContainerRequest{ContainerId: containerID})
	if err != nil {
		return fmt.Errorf("start container: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("start container failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) RemoveContainer(ctx context.Context, host string, port int, containerID string, force bool) error {
	cl, err := c.client(host, port)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := cl.RemoveContainer(ctx, &pb.RemoveContainerRequest{ContainerId: containerID, Force: force})
	if err != nil {
		return fmt.Errorf("remove container: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("remove container failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) PruneContainers(ctx context.Context, host string, port int) (*pb.PruneContainersResponse, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return cl.PruneContainers(ctx, &pb.PruneContainersRequest{})
}

func (c *AgentClient) CreateContainerFromTemplate(ctx context.Context, host string, port int, req *pb.CreateContainerFromTemplateRequest) (*pb.CreateContainerFromTemplateResponse, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	return cl.CreateContainerFromTemplate(ctx, req)
}

func (c *AgentClient) ExecContainer(ctx context.Context, host string, port int) (*ExecContainerStream, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, fmt.Errorf("exec container dial: %w", err)
	}
	stream, err := cl.ExecContainer(ctx)
	if err != nil {
		return nil, fmt.Errorf("exec container stream: %w", err)
	}
	return &ExecContainerStream{Stream: stream}, nil
}
