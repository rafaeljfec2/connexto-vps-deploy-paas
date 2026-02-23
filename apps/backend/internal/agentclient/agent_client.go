package agentclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"github.com/paasdeploy/backend/internal/pki"
)

const defaultAgentPort = 50052

type AgentClient struct {
	rootCA             []byte
	timeout            time.Duration
	insecureSkipVerify bool
}

func NewAgentClient(ca *pki.CertificateAuthority, timeout time.Duration, insecureSkipVerify bool) *AgentClient {
	return &AgentClient{
		rootCA:             ca.GetCACertPEM(),
		timeout:            timeout,
		insecureSkipVerify: insecureSkipVerify,
	}
}

func (c *AgentClient) dial(host string, port int) (pb.AgentServiceClient, func(), error) {
	if port == 0 {
		port = defaultAgentPort
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	tlsConfig := &tls.Config{
		ServerName:         host,
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: c.insecureSkipVerify,
	}
	if !c.insecureSkipVerify {
		roots := x509.NewCertPool()
		if ok := roots.AppendCertsFromPEM(c.rootCA); !ok {
			return nil, nil, fmt.Errorf("invalid CA cert")
		}
		tlsConfig.RootCAs = roots
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return nil, nil, fmt.Errorf("grpc dial: %w", err)
	}
	client := pb.NewAgentServiceClient(conn)
	return client, func() { conn.Close() }, nil
}

func (c *AgentClient) GetSystemInfo(ctx context.Context, host string, port int) (*pb.SystemInfo, error) {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return client.GetSystemInfo(ctx, &emptypb.Empty{})
}

func (c *AgentClient) GetSystemMetrics(ctx context.Context, host string, port int) (*pb.SystemMetrics, error) {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return client.GetSystemMetrics(ctx, &emptypb.Empty{})
}

type DeployLogHandler func(entry *pb.DeployLogEntry)

func (c *AgentClient) ExecuteDeployWithLogs(
	ctx context.Context,
	host string,
	port int,
	req *pb.DeployRequest,
	onLog DeployLogHandler,
) (*pb.DeployResponse, error) {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent: %w", err)
	}
	defer cleanup()

	if onLog != nil {
		go c.streamLogs(ctx, client, req.DeploymentId, onLog)
	}

	return client.ExecuteDeploy(ctx, req)
}

func (c *AgentClient) streamLogs(ctx context.Context, client pb.AgentServiceClient, deploymentID string, onLog DeployLogHandler) {
	stream, err := client.StreamDeployLogs(ctx, &pb.DeployLogSubscription{
		DeploymentId: deploymentID,
	})
	if err != nil {
		return
	}

	for {
		entry, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			return
		}
		onLog(entry)
	}
}

func (c *AgentClient) ExecuteDeploy(ctx context.Context, host string, port int, req *pb.DeployRequest) (*pb.DeployResponse, error) {
	return c.ExecuteDeployWithLogs(ctx, host, port, req, nil)
}

func (c *AgentClient) ListContainers(ctx context.Context, host string, port int, all bool, appID string) ([]*pb.ContainerInfo, error) {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := &pb.ListContainersRequest{All: all}
	if appID != "" {
		req.AppId = &appID
	}

	resp, err := client.ListContainers(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}
	return resp.Containers, nil
}

type ContainerLogHandler func(entry *pb.ContainerLogEntry)

func (c *AgentClient) GetContainerLogs(ctx context.Context, host string, port int, containerID string, tail int, follow bool, onLog ContainerLogHandler) error {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return err
	}
	defer cleanup()

	stream, err := client.GetContainerLogs(ctx, &pb.ContainerLogsRequest{
		ContainerId: containerID,
		Tail:        int32(tail),
		Follow:      follow,
		Timestamps:  true,
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

type ContainerStatsHandler func(stats *pb.ContainerStats)

func (c *AgentClient) GetContainerStats(ctx context.Context, host string, port int, containerID string, onStats ContainerStatsHandler) error {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return err
	}
	defer cleanup()

	stream, err := client.GetContainerStats(ctx, &pb.ContainerStatsRequest{
		ContainerId: containerID,
		Stream:      false,
	})
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
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := client.RestartContainer(ctx, &pb.RestartContainerRequest{ContainerId: containerID})
	if err != nil {
		return fmt.Errorf("restart container: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("restart container failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) StopContainer(ctx context.Context, host string, port int, containerID string) error {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := client.StopContainer(ctx, &pb.StopContainerRequest{ContainerId: containerID})
	if err != nil {
		return fmt.Errorf("stop container: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("stop container failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) StartContainer(ctx context.Context, host string, port int, containerID string) error {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := client.StartContainer(ctx, &pb.StartContainerRequest{ContainerId: containerID})
	if err != nil {
		return fmt.Errorf("start container: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("start container failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) GetDockerInfo(ctx context.Context, host string, port int) (*pb.DockerInfo, error) {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return client.GetDockerInfo(ctx, &emptypb.Empty{})
}

func (c *AgentClient) ListImages(ctx context.Context, host string, port int, all bool) ([]*pb.ImageInfo, error) {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := client.ListImages(ctx, &pb.ListImagesRequest{All: all})
	if err != nil {
		return nil, fmt.Errorf("list images: %w", err)
	}
	return resp.Images, nil
}

func (c *AgentClient) RemoveImage(ctx context.Context, host string, port int, imageID string, force bool) error {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := client.RemoveImage(ctx, &pb.RemoveImageRequest{ImageId: imageID, Force: force})
	if err != nil {
		return fmt.Errorf("remove image: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("remove image failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) PruneImages(ctx context.Context, host string, port int) (*pb.PruneImagesResponse, error) {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return client.PruneImages(ctx, &pb.PruneImagesRequest{})
}

func (c *AgentClient) ListNetworks(ctx context.Context, host string, port int) ([]*pb.NetworkInfo, error) {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := client.ListNetworks(ctx, &pb.ListNetworksRequest{})
	if err != nil {
		return nil, fmt.Errorf("list networks: %w", err)
	}
	return resp.Networks, nil
}

func (c *AgentClient) CreateNetwork(ctx context.Context, host string, port int, name string) error {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := client.CreateNetwork(ctx, &pb.CreateNetworkRequest{Name: name})
	if err != nil {
		return fmt.Errorf("create network: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("create network failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) RemoveNetwork(ctx context.Context, host string, port int, name string) error {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := client.RemoveNetwork(ctx, &pb.RemoveNetworkRequest{Name: name})
	if err != nil {
		return fmt.Errorf("remove network: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("remove network failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) ListVolumes(ctx context.Context, host string, port int) ([]*pb.VolumeInfo, error) {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := client.ListVolumes(ctx, &pb.ListVolumesRequest{})
	if err != nil {
		return nil, fmt.Errorf("list volumes: %w", err)
	}
	return resp.Volumes, nil
}

func (c *AgentClient) CreateVolume(ctx context.Context, host string, port int, name string) error {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := client.CreateVolume(ctx, &pb.CreateVolumeRequest{Name: name})
	if err != nil {
		return fmt.Errorf("create volume: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("create volume failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) UpdateDomains(ctx context.Context, host string, port int, req *pb.UpdateDomainsRequest) error {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := client.UpdateDomains(ctx, req)
	if err != nil {
		return fmt.Errorf("update domains: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("update domains failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) RemoveContainer(ctx context.Context, host string, port int, containerID string, force bool) error {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := client.RemoveContainer(ctx, &pb.RemoveContainerRequest{ContainerId: containerID, Force: force})
	if err != nil {
		return fmt.Errorf("remove container: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("remove container failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) RemoveVolume(ctx context.Context, host string, port int, name string) error {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := client.RemoveVolume(ctx, &pb.RemoveVolumeRequest{Name: name})
	if err != nil {
		return fmt.Errorf("remove volume: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("remove volume failed: %s", resp.Message)
	}
	return nil
}

type ExecContainerStream struct {
	Stream  pb.AgentService_ExecContainerClient
	Cleanup func()
}

func (c *AgentClient) ExecContainer(ctx context.Context, host string, port int) (*ExecContainerStream, error) {
	client, cleanup, err := c.dial(host, port)
	if err != nil {
		return nil, fmt.Errorf("exec container dial: %w", err)
	}

	stream, err := client.ExecContainer(ctx)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("exec container stream: %w", err)
	}

	return &ExecContainerStream{
		Stream:  stream,
		Cleanup: cleanup,
	}, nil
}
