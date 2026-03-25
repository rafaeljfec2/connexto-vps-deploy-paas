package grpcserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/paasdeploy/agent/internal/sysinfo"
	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *AgentService) GetSystemInfo(ctx context.Context, _ *emptypb.Empty) (*pb.SystemInfo, error) {
	return sysinfo.GetSystemInfo()
}

func (s *AgentService) GetSystemMetrics(ctx context.Context, _ *emptypb.Empty) (*pb.SystemMetrics, error) {
	return sysinfo.GetSystemMetrics()
}

func (s *AgentService) GetDockerInfo(ctx context.Context, _ *emptypb.Empty) (*pb.DockerInfo, error) {
	result, err := s.executor.Run(ctx, "docker", "info", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to get docker info: %w", err)
	}

	var info struct {
		ServerVersion string `json:"ServerVersion"`
		Driver        string `json:"Driver"`
		Images        int64  `json:"Images"`
		Containers    int64  `json:"Containers"`
		Swarm         struct {
			LocalNodeState string `json:"LocalNodeState"`
		} `json:"Swarm"`
	}

	if err := json.Unmarshal([]byte(strings.TrimSpace(result.Stdout)), &info); err != nil {
		return nil, fmt.Errorf("failed to parse docker info: %w", err)
	}

	result, _ = s.executor.RunQuiet(ctx, "docker", "version", "--format", "{{.Client.APIVersion}}")
	apiVersion := strings.TrimSpace(result.Stdout)

	return &pb.DockerInfo{
		Version:         info.ServerVersion,
		ApiVersion:      apiVersion,
		StorageDriver:   info.Driver,
		ImagesCount:     info.Images,
		ContainersCount: info.Containers,
		SwarmActive:     info.Swarm.LocalNodeState == "active",
	}, nil
}

func (s *AgentService) ListNetworks(ctx context.Context, _ *pb.ListNetworksRequest) (*pb.ListNetworksResponse, error) {
	networks, err := s.docker.ListNetworks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	pbNetworks := make([]*pb.NetworkInfo, 0, len(networks))
	for _, n := range networks {
		pbNetworks = append(pbNetworks, &pb.NetworkInfo{
			Name:     n.Name,
			Id:       n.ID,
			Driver:   n.Driver,
			Scope:    n.Scope,
			Internal: n.Internal,
		})
	}

	return &pb.ListNetworksResponse{Networks: pbNetworks}, nil
}

func (s *AgentService) CreateNetwork(ctx context.Context, req *pb.CreateNetworkRequest) (*pb.CreateNetworkResponse, error) {
	if err := s.docker.EnsureNetwork(ctx, req.Name); err != nil {
		return &pb.CreateNetworkResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.CreateNetworkResponse{Success: true, Message: "Network created"}, nil
}

func (s *AgentService) RemoveNetwork(ctx context.Context, req *pb.RemoveNetworkRequest) (*pb.RemoveNetworkResponse, error) {
	if err := s.docker.RemoveNetwork(ctx, req.Name); err != nil {
		return &pb.RemoveNetworkResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.RemoveNetworkResponse{Success: true, Message: "Network removed"}, nil
}

func (s *AgentService) ListVolumes(ctx context.Context, _ *pb.ListVolumesRequest) (*pb.ListVolumesResponse, error) {
	volumes, err := s.docker.ListVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	pbVolumes := make([]*pb.VolumeInfo, 0, len(volumes))
	for _, v := range volumes {
		pbVolumes = append(pbVolumes, &pb.VolumeInfo{
			Name:       v.Name,
			Driver:     v.Driver,
			Mountpoint: v.Mountpoint,
		})
	}

	return &pb.ListVolumesResponse{Volumes: pbVolumes}, nil
}

func (s *AgentService) CreateVolume(ctx context.Context, req *pb.CreateVolumeRequest) (*pb.CreateVolumeResponse, error) {
	if err := s.docker.CreateVolume(ctx, req.Name); err != nil {
		return &pb.CreateVolumeResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.CreateVolumeResponse{Success: true, Message: "Volume created"}, nil
}

func (s *AgentService) RemoveVolume(ctx context.Context, req *pb.RemoveVolumeRequest) (*pb.RemoveVolumeResponse, error) {
	if err := s.docker.RemoveVolume(ctx, req.Name); err != nil {
		return &pb.RemoveVolumeResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.RemoveVolumeResponse{Success: true, Message: "Volume removed"}, nil
}

func (s *AgentService) UpdateDomains(ctx context.Context, req *pb.UpdateDomainsRequest) (*pb.UpdateDomainsResponse, error) {
	return s.deployExecutor.UpdateDomains(ctx, req)
}

func (s *AgentService) GetCertificates(ctx context.Context, _ *pb.GetCertificatesRequest) (*pb.GetCertificatesResponse, error) {
	certs, err := s.traefikClient.GetAllCertificatesStatus(ctx)
	if err != nil {
		s.logger.Warn("Failed to fetch certificates from Traefik", "error", err)
		return &pb.GetCertificatesResponse{}, nil
	}

	result := make([]*pb.CertificateInfo, 0, len(certs))
	for _, c := range certs {
		result = append(result, &pb.CertificateInfo{
			Domain: c.Domain,
			Status: c.Status,
			Issuer: c.Issuer,
			Error:  c.Error,
		})
	}

	return &pb.GetCertificatesResponse{Certificates: result}, nil
}
