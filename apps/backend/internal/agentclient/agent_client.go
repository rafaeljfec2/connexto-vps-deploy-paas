package agentclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"github.com/paasdeploy/backend/internal/pki"
)

const pushUpdateChunkSize = 256 * 1024

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

type ContainerLogHandler func(entry *pb.ContainerLogEntry)

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

type ContainerStatsHandler func(stats *pb.ContainerStats)

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

func (c *AgentClient) GetDockerInfo(ctx context.Context, host string, port int) (*pb.DockerInfo, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return cl.GetDockerInfo(ctx, &emptypb.Empty{})
}

func (c *AgentClient) ListImages(ctx context.Context, host string, port int, all bool) ([]*pb.ImageInfo, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := cl.ListImages(ctx, &pb.ListImagesRequest{All: all})
	if err != nil {
		return nil, fmt.Errorf("list images: %w", err)
	}
	return resp.Images, nil
}

func (c *AgentClient) RemoveImage(ctx context.Context, host string, port int, imageID string, force bool) error {
	cl, err := c.client(host, port)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := cl.RemoveImage(ctx, &pb.RemoveImageRequest{ImageId: imageID, Force: force})
	if err != nil {
		return fmt.Errorf("remove image: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("remove image failed: %s", resp.Message)
	}
	return nil
}

func (c *AgentClient) PruneImages(ctx context.Context, host string, port int) (*pb.PruneImagesResponse, error) {
	cl, err := c.client(host, port)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return cl.PruneImages(ctx, &pb.PruneImagesRequest{})
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

type ExecContainerStream struct {
	Stream pb.AgentService_ExecContainerClient
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

func (c *AgentClient) PushUpdate(ctx context.Context, host string, port int, binaryPath, version string) error {
	f, err := os.Open(binaryPath)
	if err != nil {
		return fmt.Errorf("open binary: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat binary: %w", err)
	}

	cl, err := c.client(host, port)
	if err != nil {
		return fmt.Errorf("push update dial: %w", err)
	}

	stream, err := cl.PushUpdate(ctx)
	if err != nil {
		return fmt.Errorf("push update stream: %w", err)
	}

	sendFailed, err := streamBinaryChunks(f, stream, version, info.Size())
	if err != nil {
		return err
	}

	return closePushUpdateStream(stream, sendFailed)
}

func streamBinaryChunks(
	f *os.File,
	stream pb.AgentService_PushUpdateClient,
	version string,
	totalSize int64,
) (bool, error) {
	buf := make([]byte, pushUpdateChunkSize)
	first := true

	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			chunk := &pb.UpdateBinaryChunk{Data: buf[:n]}
			if first {
				chunk.Version = version
				chunk.TotalSize = totalSize
				first = false
			}
			if failed, err := sendChunk(stream, chunk); err != nil {
				return false, err
			} else if failed {
				return true, nil
			}
		}
		if readErr == io.EOF {
			return false, nil
		}
		if readErr != nil {
			return false, fmt.Errorf("read binary: %w", readErr)
		}
	}
}

func sendChunk(stream pb.AgentService_PushUpdateClient, chunk *pb.UpdateBinaryChunk) (eofReceived bool, err error) {
	sendErr := stream.Send(chunk)
	if sendErr == nil {
		return false, nil
	}
	if sendErr == io.EOF {
		return true, nil
	}
	return false, fmt.Errorf("send chunk: %w", sendErr)
}

func closePushUpdateStream(stream pb.AgentService_PushUpdateClient, sendFailed bool) error {
	resp, err := stream.CloseAndRecv()
	if err != nil {
		if sendFailed {
			return fmt.Errorf("agent closed stream early (agent may not support gRPC push — try HTTPS mode): %w", err)
		}
		return fmt.Errorf("close stream: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("agent rejected update: %s", resp.Message)
	}
	return nil
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
