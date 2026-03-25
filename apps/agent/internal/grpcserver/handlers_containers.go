package grpcserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"github.com/paasdeploy/shared/pkg/docker"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultLogTail      = 100
	streamLogBuffer     = 256
	statsStreamInterval = 5 * time.Second
)

func (s *AgentService) ListContainers(ctx context.Context, req *pb.ListContainersRequest) (*pb.ListContainersResponse, error) {
	containers, err := s.docker.ListContainers(ctx, req.All)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	pbContainers := make([]*pb.ContainerInfo, 0, len(containers))
	for _, c := range containers {
		if req.AppId != nil && *req.AppId != "" {
			appLabel := c.Labels[docker.LabelPaasDeployApp]
			if appLabel != *req.AppId && c.Name != *req.AppId {
				continue
			}
		}

		created, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", c.Created)

		ports := make([]*pb.PortBinding, 0, len(c.Ports))
		for _, p := range c.Ports {
			ports = append(ports, &pb.PortBinding{
				ContainerPort: int32(p.PrivatePort),
				HostPort:      int32(p.PublicPort),
				Protocol:      p.Type,
			})
		}

		mounts := make([]*pb.ContainerMount, 0, len(c.Mounts))
		for _, m := range c.Mounts {
			mounts = append(mounts, &pb.ContainerMount{
				Type:        m.Type,
				Source:      m.Source,
				Destination: m.Destination,
				ReadOnly:    m.ReadOnly,
			})
		}

		pbContainers = append(pbContainers, &pb.ContainerInfo{
			Id:        c.ID,
			Name:      c.Name,
			Image:     c.Image,
			State:     c.State,
			Status:    c.Status,
			CreatedAt: timestamppb.New(created),
			Labels:    c.Labels,
			Ports:     ports,
			Networks:  c.Networks,
			Mounts:    mounts,
		})
	}

	return &pb.ListContainersResponse{Containers: pbContainers}, nil
}

func (s *AgentService) GetContainerLogs(req *pb.ContainerLogsRequest, stream pb.AgentService_GetContainerLogsServer) error {
	ctx := stream.Context()
	tail := int(req.Tail)
	if tail <= 0 {
		tail = defaultLogTail
	}

	if !req.Follow {
		return s.sendStaticLogs(ctx, req.ContainerId, tail, stream)
	}
	return s.streamFollowLogs(ctx, req.ContainerId, stream)
}

func (s *AgentService) sendStaticLogs(ctx context.Context, containerID string, tail int, stream pb.AgentService_GetContainerLogsServer) error {
	logs, err := s.docker.ContainerLogs(ctx, containerID, tail)
	if err != nil {
		return fmt.Errorf("failed to get container logs: %w", err)
	}
	for _, line := range strings.Split(logs, "\n") {
		if err := sendLogLine(line, stream); err != nil {
			return err
		}
	}
	return nil
}

func (s *AgentService) streamFollowLogs(ctx context.Context, containerID string, stream pb.AgentService_GetContainerLogsServer) error {
	output := make(chan string, streamLogBuffer)
	errCh := make(chan error, 1)

	go func() {
		errCh <- s.docker.StreamContainerLogs(ctx, containerID, output)
	}()

	for line := range output {
		if err := sendLogLine(line, stream); err != nil {
			return err
		}
	}

	if err := <-errCh; err != nil && ctx.Err() == nil {
		return err
	}
	return nil
}

func sendLogLine(line string, stream pb.AgentService_GetContainerLogsServer) error {
	if line == "" {
		return nil
	}
	ts, msg := parseLogTimestamp(line)
	return stream.Send(&pb.ContainerLogEntry{
		Timestamp: ts,
		Stream:    "stdout",
		Message:   msg,
	})
}

func parseLogTimestamp(line string) (*timestamppb.Timestamp, string) {
	if len(line) > 30 && (line[4] == '-' || line[0] == '2') {
		spaceIdx := strings.Index(line, " ")
		if spaceIdx > 0 && spaceIdx < 35 {
			if t, err := time.Parse(time.RFC3339Nano, line[:spaceIdx]); err == nil {
				return timestamppb.New(t), line[spaceIdx+1:]
			}
		}
	}
	return timestamppb.Now(), line
}

func (s *AgentService) GetContainerStats(req *pb.ContainerStatsRequest, stream pb.AgentService_GetContainerStatsServer) error {
	ctx := stream.Context()

	stats, err := s.docker.ContainerStats(ctx, req.ContainerId)
	if err != nil {
		return fmt.Errorf("failed to get container stats: %w", err)
	}

	if err := stream.Send(&pb.ContainerStats{
		Timestamp:        timestamppb.Now(),
		CpuPercent:       stats.CPUPercent,
		MemoryUsageBytes: stats.MemoryUsage,
		MemoryLimitBytes: stats.MemoryLimit,
		NetworkRxBytes:   stats.NetworkRx,
		NetworkTxBytes:   stats.NetworkTx,
	}); err != nil {
		return err
	}

	if !req.Stream {
		return nil
	}

	ticker := time.NewTicker(statsStreamInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			stats, err := s.docker.ContainerStats(ctx, req.ContainerId)
			if err != nil {
				s.logger.Warn("Failed to get container stats", "container", req.ContainerId, "error", err)
				continue
			}
			if err := stream.Send(&pb.ContainerStats{
				Timestamp:        timestamppb.Now(),
				CpuPercent:       stats.CPUPercent,
				MemoryUsageBytes: stats.MemoryUsage,
				MemoryLimitBytes: stats.MemoryLimit,
				NetworkRxBytes:   stats.NetworkRx,
				NetworkTxBytes:   stats.NetworkTx,
			}); err != nil {
				return err
			}
		}
	}
}

func (s *AgentService) RestartContainer(ctx context.Context, req *pb.RestartContainerRequest) (*pb.RestartContainerResponse, error) {
	if err := s.docker.RestartContainer(ctx, req.ContainerId); err != nil {
		return &pb.RestartContainerResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.RestartContainerResponse{Success: true, Message: "Container restarted"}, nil
}

func (s *AgentService) StopContainer(ctx context.Context, req *pb.StopContainerRequest) (*pb.StopContainerResponse, error) {
	if err := s.docker.StopContainer(ctx, req.ContainerId); err != nil {
		return &pb.StopContainerResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.StopContainerResponse{Success: true, Message: "Container stopped"}, nil
}

func (s *AgentService) StartContainer(ctx context.Context, req *pb.StartContainerRequest) (*pb.StartContainerResponse, error) {
	if err := s.docker.StartContainer(ctx, req.ContainerId); err != nil {
		return &pb.StartContainerResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.StartContainerResponse{Success: true, Message: "Container started"}, nil
}

func (s *AgentService) RemoveContainer(ctx context.Context, req *pb.RemoveContainerRequest) (*pb.RemoveContainerResponse, error) {
	if err := s.docker.RemoveContainer(ctx, req.ContainerId, req.Force); err != nil {
		return &pb.RemoveContainerResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.RemoveContainerResponse{Success: true, Message: "Container removed"}, nil
}

func (s *AgentService) CreateContainerFromTemplate(ctx context.Context, req *pb.CreateContainerFromTemplateRequest) (*pb.CreateContainerFromTemplateResponse, error) {
	opts := docker.CreateContainerOptions{
		Name:          req.Name,
		Image:         req.Image,
		Env:           req.Env,
		Network:       req.Network,
		RestartPolicy: req.RestartPolicy,
		Command:       req.Command,
	}

	for _, p := range req.Ports {
		opts.Ports = append(opts.Ports, docker.PortMapping{
			HostPort:      int(p.HostPort),
			ContainerPort: int(p.ContainerPort),
			Protocol:      p.Protocol,
		})
	}

	for _, v := range req.Volumes {
		opts.Volumes = append(opts.Volumes, docker.VolumeMapping{
			HostPath:      v.HostPath,
			ContainerPath: v.ContainerPath,
			ReadOnly:      v.ReadOnly,
		})
	}

	containerID, err := s.docker.CreateContainer(ctx, opts)
	if err != nil {
		s.logger.Error("Failed to create container from template", "name", req.Name, "image", req.Image, "error", err)
		return &pb.CreateContainerFromTemplateResponse{Success: false, Message: err.Error()}, nil
	}

	s.logger.Info("Container created from template", "id", containerID, "name", req.Name, "image", req.Image)
	return &pb.CreateContainerFromTemplateResponse{
		Success:     true,
		ContainerId: containerID,
		Message:     "Container created from template",
	}, nil
}
