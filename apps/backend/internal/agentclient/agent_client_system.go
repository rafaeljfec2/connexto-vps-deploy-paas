package agentclient

import (
	"context"
	"fmt"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (c *AgentClient) GetSystemInfo(ctx context.Context, host string, port int) (*pb.SystemInfo, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return cl.GetSystemInfo(ctx, &emptypb.Empty{})
}

func (c *AgentClient) GetSystemMetrics(ctx context.Context, host string, port int) (*pb.SystemMetrics, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return cl.GetSystemMetrics(ctx, &emptypb.Empty{})
}

func (c *AgentClient) GetDockerInfo(ctx context.Context, host string, port int) (*pb.DockerInfo, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return cl.GetDockerInfo(ctx, &emptypb.Empty{})
}

func (c *AgentClient) GetCertificates(ctx context.Context, host string, port int) ([]*pb.CertificateInfo, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := cl.GetCertificates(ctx, &pb.GetCertificatesRequest{})
	if err != nil {
		return nil, err
	}
	return resp.Certificates, nil
}

func (c *AgentClient) ListNetworks(ctx context.Context, host string, port int) ([]*pb.NetworkInfo, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := cl.ListNetworks(ctx, &pb.ListNetworksRequest{})
	if err != nil {
		return nil, fmt.Errorf("list networks: %w", err)
	}
	return resp.Networks, nil
}

func (c *AgentClient) CreateNetwork(ctx context.Context, host string, port int, name string) error {
	cl, err := c.client(host, port)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := cl.CreateNetwork(ctx, &pb.CreateNetworkRequest{Name: name})
	if err != nil {
		return fmt.Errorf("create network: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("create network failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) RemoveNetwork(ctx context.Context, host string, port int, name string) error {
	cl, err := c.client(host, port)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := cl.RemoveNetwork(ctx, &pb.RemoveNetworkRequest{Name: name})
	if err != nil {
		return fmt.Errorf("remove network: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("remove network failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) ListVolumes(ctx context.Context, host string, port int) ([]*pb.VolumeInfo, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := cl.ListVolumes(ctx, &pb.ListVolumesRequest{})
	if err != nil {
		return nil, fmt.Errorf("list volumes: %w", err)
	}
	return resp.Volumes, nil
}

func (c *AgentClient) CreateVolume(ctx context.Context, host string, port int, name string) error {
	cl, err := c.client(host, port)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := cl.CreateVolume(ctx, &pb.CreateVolumeRequest{Name: name})
	if err != nil {
		return fmt.Errorf("create volume: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("create volume failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) RemoveVolume(ctx context.Context, host string, port int, name string) error {
	cl, err := c.client(host, port)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := cl.RemoveVolume(ctx, &pb.RemoveVolumeRequest{Name: name})
	if err != nil {
		return fmt.Errorf("remove volume: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("remove volume failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) PruneVolumes(ctx context.Context, host string, port int) (*pb.PruneVolumesResponse, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return cl.PruneVolumes(ctx, &pb.PruneVolumesRequest{})
}

func (c *AgentClient) UpdateDomains(ctx context.Context, host string, port int, req *pb.UpdateDomainsRequest) error {
	cl, err := c.client(host, port)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := cl.UpdateDomains(ctx, req)
	if err != nil {
		return fmt.Errorf("update domains: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("update domains failed: %s", resp.Message)
	}
	return nil
}
