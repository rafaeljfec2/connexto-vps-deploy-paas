package grpcserver

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/paasdeploy/agent/internal/deploy"
	"github.com/paasdeploy/agent/internal/sysinfo"
	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"github.com/paasdeploy/shared/pkg/docker"
	"github.com/paasdeploy/shared/pkg/executor"
)

const logStreamBuffer = 512

type Config struct {
	Port     int
	CertPath string
	KeyPath  string
}

type Server struct {
	cfg        Config
	grpcServer *grpc.Server
	logger     *slog.Logger
}

func resolveDataDir() string {
	if dir := os.Getenv("DEPLOY_DATA_DIR"); dir != "" {
		return dir
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".paasdeploy", "apps")
	}
	return filepath.Join(os.TempDir(), "paasdeploy", "apps")
}

func New(cfg Config, logger *slog.Logger) (*Server, error) {
	if cfg.Port == 0 || cfg.CertPath == "" || cfg.KeyPath == "" {
		return nil, fmt.Errorf("missing grpc server config")
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.NoClientCert,
		MinVersion:   tls.VersionTLS13,
	}

	grpcServer := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsConfig)),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
			Time:              10 * time.Second,
			Timeout:           5 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	registry := os.Getenv("DOCKER_REGISTRY")
	dockerClient := docker.NewClient(resolveDataDir(), registry, logger)

	agentService := &AgentService{
		deployExecutor: deploy.NewExecutor(logger),
		docker:         dockerClient,
		executor:       executor.New("", 2*time.Minute, logger),
		logger:         logger.With("component", "agent-service"),
	}
	pb.RegisterAgentServiceServer(grpcServer, agentService)

	return &Server{
		cfg:        cfg,
		grpcServer: grpcServer,
		logger:     logger.With("component", "agent-grpc"),
	}, nil
}

func (s *Server) Start() error {
	addr := net.JoinHostPort("0.0.0.0", fmt.Sprintf("%d", s.cfg.Port))
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.logger.Info("Agent gRPC listening", "address", addr)
	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}

type AgentService struct {
	pb.UnimplementedAgentServiceServer
	deployExecutor *deploy.Executor
	docker         *docker.Client
	executor       *executor.Executor
	logStreams     sync.Map
	logger         *slog.Logger
}

func (s *AgentService) ExecuteDeploy(ctx context.Context, req *pb.DeployRequest) (*pb.DeployResponse, error) {
	logFn := s.buildLogFunc(req.DeploymentId)
	resp := s.deployExecutor.Execute(ctx, req, logFn)
	s.closeLogStream(req.DeploymentId)
	return resp, nil
}

func (s *AgentService) StreamDeployLogs(sub *pb.DeployLogSubscription, stream pb.AgentService_StreamDeployLogsServer) error {
	ch := make(chan *pb.DeployLogEntry, logStreamBuffer)
	s.logStreams.Store(sub.DeploymentId, ch)
	defer s.logStreams.Delete(sub.DeploymentId)

	s.logger.Info("Deploy log stream opened", "deploymentId", sub.DeploymentId)

	for entry := range ch {
		if err := stream.Send(entry); err != nil {
			s.logger.Warn("Failed to send deploy log entry", "deploymentId", sub.DeploymentId, "error", err)
			return err
		}
	}

	s.logger.Info("Deploy log stream closed", "deploymentId", sub.DeploymentId)
	return nil
}

func (s *AgentService) buildLogFunc(deploymentID string) deploy.LogFunc {
	return func(stage pb.DeployStage, level pb.DeployLogLevel, message string) {
		val, ok := s.logStreams.Load(deploymentID)
		if !ok {
			return
		}
		ch := val.(chan *pb.DeployLogEntry)
		entry := &pb.DeployLogEntry{
			DeploymentId: deploymentID,
			Timestamp:    timestamppb.Now(),
			Level:        level,
			Stage:        stage,
			Message:      message,
		}
		select {
		case ch <- entry:
		default:
			s.logger.Debug("Deploy log channel full, dropping entry", "deploymentId", deploymentID)
		}
	}
}

func (s *AgentService) closeLogStream(deploymentID string) {
	val, ok := s.logStreams.LoadAndDelete(deploymentID)
	if !ok {
		return
	}
	close(val.(chan *pb.DeployLogEntry))
}

func (s *AgentService) GetSystemInfo(ctx context.Context, _ *emptypb.Empty) (*pb.SystemInfo, error) {
	return sysinfo.GetSystemInfo()
}

func (s *AgentService) GetSystemMetrics(ctx context.Context, _ *emptypb.Empty) (*pb.SystemMetrics, error) {
	return sysinfo.GetSystemMetrics()
}

func (s *AgentService) ListContainers(ctx context.Context, req *pb.ListContainersRequest) (*pb.ListContainersResponse, error) {
	containers, err := s.docker.ListContainers(ctx, req.All)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	pbContainers := make([]*pb.ContainerInfo, 0, len(containers))
	for _, c := range containers {
		if req.AppId != nil && *req.AppId != "" {
			appLabel := c.Labels["paasdeploy.app"]
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

		pbContainers = append(pbContainers, &pb.ContainerInfo{
			Id:        c.ID,
			Name:      c.Name,
			Image:     c.Image,
			State:     c.State,
			Status:    c.Status,
			CreatedAt: timestamppb.New(created),
			Labels:    c.Labels,
			Ports:     ports,
		})
	}

	return &pb.ListContainersResponse{Containers: pbContainers}, nil
}

func (s *AgentService) GetContainerLogs(req *pb.ContainerLogsRequest, stream pb.AgentService_GetContainerLogsServer) error {
	ctx := stream.Context()
	tail := int(req.Tail)
	if tail <= 0 {
		tail = 100
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
	output := make(chan string, 256)
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

	ticker := time.NewTicker(5 * time.Second)
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
		return &pb.RestartContainerResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &pb.RestartContainerResponse{
		Success: true,
		Message: "Container restarted",
	}, nil
}

func (s *AgentService) StopContainer(ctx context.Context, req *pb.StopContainerRequest) (*pb.StopContainerResponse, error) {
	if err := s.docker.StopContainer(ctx, req.ContainerId); err != nil {
		return &pb.StopContainerResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &pb.StopContainerResponse{
		Success: true,
		Message: "Container stopped",
	}, nil
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

func (s *AgentService) StartContainer(ctx context.Context, req *pb.StartContainerRequest) (*pb.StartContainerResponse, error) {
	if err := s.docker.StartContainer(ctx, req.ContainerId); err != nil {
		return &pb.StartContainerResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &pb.StartContainerResponse{
		Success: true,
		Message: "Container started",
	}, nil
}

func (s *AgentService) ListImages(ctx context.Context, req *pb.ListImagesRequest) (*pb.ListImagesResponse, error) {
	images, err := s.docker.ListImages(ctx, req.All)
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	pbImages := make([]*pb.ImageInfo, 0, len(images))
	for _, img := range images {
		pbImages = append(pbImages, &pb.ImageInfo{
			Id:         img.ID,
			Repository: img.Repository,
			Tag:        img.Tag,
			Size:       img.Size,
			Created:    img.Created,
			Dangling:   img.Dangling,
		})
	}

	return &pb.ListImagesResponse{Images: pbImages}, nil
}

func (s *AgentService) RemoveImage(ctx context.Context, req *pb.RemoveImageRequest) (*pb.RemoveImageResponse, error) {
	if err := s.docker.RemoveImageByID(ctx, req.ImageId, req.Force); err != nil {
		return &pb.RemoveImageResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &pb.RemoveImageResponse{
		Success: true,
		Message: "Image removed",
	}, nil
}

func (s *AgentService) PruneImages(ctx context.Context, _ *pb.PruneImagesRequest) (*pb.PruneImagesResponse, error) {
	result, err := s.docker.PruneImages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prune images: %w", err)
	}
	return &pb.PruneImagesResponse{
		ImagesRemoved:       int32(result.ImagesDeleted),
		SpaceReclaimedBytes: result.SpaceReclaimed,
	}, nil
}

func (s *AgentService) ListNetworks(ctx context.Context, _ *pb.ListNetworksRequest) (*pb.ListNetworksResponse, error) {
	networks, err := s.docker.ListNetworks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	pbNetworks := make([]*pb.NetworkInfo, 0, len(networks))
	for _, n := range networks {
		pbNetworks = append(pbNetworks, &pb.NetworkInfo{Name: n})
	}

	return &pb.ListNetworksResponse{Networks: pbNetworks}, nil
}

func (s *AgentService) CreateNetwork(ctx context.Context, req *pb.CreateNetworkRequest) (*pb.CreateNetworkResponse, error) {
	if err := s.docker.EnsureNetwork(ctx, req.Name); err != nil {
		return &pb.CreateNetworkResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &pb.CreateNetworkResponse{
		Success: true,
		Message: "Network created",
	}, nil
}

func (s *AgentService) RemoveNetwork(ctx context.Context, req *pb.RemoveNetworkRequest) (*pb.RemoveNetworkResponse, error) {
	if err := s.docker.RemoveNetwork(ctx, req.Name); err != nil {
		return &pb.RemoveNetworkResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &pb.RemoveNetworkResponse{
		Success: true,
		Message: "Network removed",
	}, nil
}

func (s *AgentService) ListVolumes(ctx context.Context, _ *pb.ListVolumesRequest) (*pb.ListVolumesResponse, error) {
	volumes, err := s.docker.ListVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	pbVolumes := make([]*pb.VolumeInfo, 0, len(volumes))
	for _, v := range volumes {
		pbVolumes = append(pbVolumes, &pb.VolumeInfo{Name: v})
	}

	return &pb.ListVolumesResponse{Volumes: pbVolumes}, nil
}

func (s *AgentService) CreateVolume(ctx context.Context, req *pb.CreateVolumeRequest) (*pb.CreateVolumeResponse, error) {
	if err := s.docker.CreateVolume(ctx, req.Name); err != nil {
		return &pb.CreateVolumeResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &pb.CreateVolumeResponse{
		Success: true,
		Message: "Volume created",
	}, nil
}

func (s *AgentService) UpdateDomains(ctx context.Context, req *pb.UpdateDomainsRequest) (*pb.UpdateDomainsResponse, error) {
	return s.deployExecutor.UpdateDomains(ctx, req)
}

func (s *AgentService) RemoveVolume(ctx context.Context, req *pb.RemoveVolumeRequest) (*pb.RemoveVolumeResponse, error) {
	if err := s.docker.RemoveVolume(ctx, req.Name); err != nil {
		return &pb.RemoveVolumeResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &pb.RemoveVolumeResponse{
		Success: true,
		Message: "Volume removed",
	}, nil
}

func (s *AgentService) RemoveContainer(ctx context.Context, req *pb.RemoveContainerRequest) (*pb.RemoveContainerResponse, error) {
	if err := s.docker.RemoveContainer(ctx, req.ContainerId, req.Force); err != nil {
		return &pb.RemoveContainerResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &pb.RemoveContainerResponse{
		Success: true,
		Message: "Container removed",
	}, nil
}

const pushUpdateChunkLimit = 512 * 1024 * 1024

type pushUpdateResult struct {
	version      string
	totalSize    int64
	expectedSize int64
}

func (s *AgentService) receiveBinaryChunks(stream grpc.ClientStreamingServer[pb.UpdateBinaryChunk, pb.UpdateBinaryResponse], tmpPath string) (*pushUpdateResult, error) {
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	result := &pushUpdateResult{}
	first := true

	for {
		chunk, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			return nil, fmt.Errorf("receive chunk: %w", recvErr)
		}

		if first {
			result.version = chunk.Version
			result.expectedSize = chunk.TotalSize
			first = false
			s.logger.Info("receiving binary update", "version", result.version, "expectedSize", result.expectedSize)
		}

		n, writeErr := f.Write(chunk.Data)
		if writeErr != nil {
			return nil, fmt.Errorf("write chunk: %w", writeErr)
		}
		result.totalSize += int64(n)

		if result.totalSize > pushUpdateChunkLimit {
			return nil, fmt.Errorf("binary exceeds size limit")
		}
	}

	return result, nil
}

func replaceBinaryFile(execPath, tmpPath string) error {
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	if err := os.Remove(execPath); err != nil {
		return fmt.Errorf("remove current binary: %w", err)
	}
	if err := os.Rename(tmpPath, execPath); err != nil {
		return fmt.Errorf("rename new binary: %w", err)
	}
	return nil
}

func (s *AgentService) PushUpdate(stream grpc.ClientStreamingServer[pb.UpdateBinaryChunk, pb.UpdateBinaryResponse]) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	tmpPath := filepath.Join(filepath.Dir(execPath), "agent.new")

	result, err := s.receiveBinaryChunks(stream, tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return err
	}

	if result.totalSize == 0 {
		os.Remove(tmpPath)
		return fmt.Errorf("received empty binary")
	}

	if result.expectedSize > 0 && result.totalSize != result.expectedSize {
		os.Remove(tmpPath)
		return fmt.Errorf("size mismatch: expected %d, got %d", result.expectedSize, result.totalSize)
	}

	if err := replaceBinaryFile(execPath, tmpPath); err != nil {
		os.Remove(tmpPath)
		return err
	}

	s.logger.Info("binary update received, restarting", "version", result.version, "bytes", result.totalSize)

	if sendErr := stream.SendAndClose(&pb.UpdateBinaryResponse{
		Success: true,
		Message: fmt.Sprintf("update to %s received, restarting", result.version),
	}); sendErr != nil {
		return sendErr
	}

	go func() {
		time.Sleep(500 * time.Millisecond)
		_ = unix.Exec(execPath, os.Args, os.Environ())
	}()

	return nil
}

const execPTYBufSize = 4096

type execSession struct {
	stream pb.AgentService_ExecContainerServer
	ptmx   *os.File
	sendMu sync.Mutex
	done   chan struct{}
	logger *slog.Logger
}

func (es *execSession) sendOutput(data []byte) error {
	es.sendMu.Lock()
	defer es.sendMu.Unlock()
	return es.stream.Send(&pb.ExecOutput{
		Payload: &pb.ExecOutput_Data{Data: data},
	})
}

func (es *execSession) sendExitCode(code int) {
	es.sendMu.Lock()
	defer es.sendMu.Unlock()
	_ = es.stream.Send(&pb.ExecOutput{
		Payload: &pb.ExecOutput_ExitCode{ExitCode: int32(code)},
	})
}

func (es *execSession) readLoop(wg *sync.WaitGroup) {
	defer wg.Done()
	buf := make([]byte, execPTYBufSize)
	for {
		select {
		case <-es.done:
			return
		default:
		}
		n, readErr := es.ptmx.Read(buf)
		if n > 0 {
			out := make([]byte, n)
			copy(out, buf[:n])
			if sendErr := es.sendOutput(out); sendErr != nil {
				return
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				es.logger.Debug("exec: pty read error", "error", readErr)
			}
			return
		}
	}
}

func (es *execSession) writeLoop(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		in, recvErr := es.stream.Recv()
		if recvErr != nil {
			return
		}
		es.handleInput(in)
	}
}

func (es *execSession) handleInput(in *pb.ExecInput) {
	switch p := in.Payload.(type) {
	case *pb.ExecInput_Data:
		_, _ = es.ptmx.Write(p.Data)
	case *pb.ExecInput_Resize:
		if p.Resize.Cols > 0 && p.Resize.Rows > 0 {
			_ = pty.Setsize(es.ptmx, &pty.Winsize{
				Cols: uint16(p.Resize.Cols),
				Rows: uint16(p.Resize.Rows),
			})
		}
	}
}

func parseExecStartRequest(stream pb.AgentService_ExecContainerServer) (containerID, shell string, cols, rows uint16, err error) {
	msg, err := stream.Recv()
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("exec: failed to receive start message: %w", err)
	}
	startReq := msg.GetStart()
	if startReq == nil {
		return "", "", 0, 0, fmt.Errorf("exec: first message must be ExecStartRequest")
	}
	containerID = startReq.ContainerId
	shell = startReq.Shell
	if shell == "" {
		shell = "sh"
	}
	cols = uint16(startReq.Cols)
	rows = uint16(startReq.Rows)
	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}
	return containerID, shell, cols, rows, nil
}

func (s *AgentService) ExecContainer(stream pb.AgentService_ExecContainerServer) error {
	containerID, shell, cols, rows, err := parseExecStartRequest(stream)
	if err != nil {
		return err
	}

	cmd := exec.Command("docker", "exec", "-it", containerID, shell)
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		s.logger.Error("exec: failed to start pty", "error", err, "container", containerID)
		return fmt.Errorf("exec: failed to start docker exec: %w", err)
	}
	defer ptmx.Close()

	session := &execSession{
		stream: stream,
		ptmx:   ptmx,
		done:   make(chan struct{}),
		logger: s.logger,
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go session.readLoop(&wg)
	go session.writeLoop(&wg)

	exitCode := 0
	if waitErr := cmd.Wait(); waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	close(session.done)
	session.sendExitCode(exitCode)

	wg.Wait()
	return nil
}
