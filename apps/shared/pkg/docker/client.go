package docker

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/paasdeploy/shared/pkg/executor"
)

const (
	DefaultNetworkName = "paasdeploy"
	formatFlag         = "--format"
	errNoSuchContainer = "no such container"
)

type Client struct {
	executor        *executor.Executor
	logger          *slog.Logger
	registry        string
	buildxAvailable bool
}

func NewClient(baseDir string, registry string, logger *slog.Logger) *Client {
	exec := executor.New(baseDir, 15*time.Minute, logger)
	client := &Client{
		executor: exec,
		logger:   logger,
		registry: registry,
	}
	client.initBuildx()
	return client
}

func (d *Client) initBuildx() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := d.executor.Run(ctx, "docker", "buildx", "version")
	if err != nil {
		d.logger.Info("BuildKit (buildx) not available, using legacy builder")
		d.buildxAvailable = false
		return
	}

	_, err = d.executor.Run(ctx, "docker", "buildx", "inspect", "--builder", "default")
	if err == nil {
		d.buildxAvailable = true
		d.logger.Info("BuildKit available, using buildx with default builder")
		return
	}

	d.buildxAvailable = false
	d.logger.Info("Buildx default builder not available, using legacy builder")
}

type BuildOptions struct {
	BuildArgs map[string]string
	Target    string
}

func (d *Client) Build(ctx context.Context, workDir, dockerfile, tag string, output chan<- string) error {
	return d.BuildWithOptions(ctx, workDir, dockerfile, tag, nil, output)
}

func (d *Client) BuildWithOptions(ctx context.Context, workDir, dockerfile, tag string, opts *BuildOptions, output chan<- string) error {
	d.logger.Info("Building Docker image", "workDir", workDir, "dockerfile", dockerfile, "tag", tag)

	d.executor.SetWorkDir(workDir)
	d.executor.SetTimeout(15 * time.Minute)

	var args []string
	if d.buildxAvailable {
		args = []string{
			"buildx", "build",
			"--builder", "default",
			"--load",
			"-t", tag,
			"-f", dockerfile,
		}
	} else {
		args = []string{
			"build",
			"-t", tag,
			"-f", dockerfile,
		}
	}

	if opts != nil {
		for k, v := range opts.BuildArgs {
			args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
		}
		if opts.Target != "" {
			args = append(args, "--target", opts.Target)
		}
	}

	args = append(args, ".")

	if output != nil {
		err := d.executor.RunWithStreaming(ctx, output, "docker", args...)
		if err != nil {
			d.logger.Error("Docker build failed", "tag", tag, "workDir", workDir, "error", err)
			return fmt.Errorf("docker build failed: %w", err)
		}
		return nil
	}

	_, err := d.executor.Run(ctx, "docker", args...)
	if err != nil {
		d.logger.Error("Docker build failed", "tag", tag, "workDir", workDir, "error", err)
		return fmt.Errorf("docker build failed: %w", err)
	}

	return nil
}

func (d *Client) ComposeUp(ctx context.Context, projectDir, projectName string, output chan<- string) error {
	d.logger.Info("Starting containers with docker compose", "dir", projectDir)

	d.executor.SetWorkDir(projectDir)
	d.executor.SetTimeout(5 * time.Minute)

	composeFile := filepath.Join(projectDir, "docker-compose.yml")

	args := []string{
		"compose",
		"-f", composeFile,
		"-p", projectName,
		"up",
		"-d",
		"--force-recreate",
		"--remove-orphans",
	}

	if output != nil {
		err := d.executor.RunWithStreaming(ctx, output, "docker", args...)
		if err != nil {
			d.logger.Error("Docker compose up failed", "projectName", projectName, "dir", projectDir, "error", err)
			return fmt.Errorf("docker compose up failed: %w", err)
		}
		return nil
	}

	_, err := d.executor.Run(ctx, "docker", args...)
	if err != nil {
		d.logger.Error("Docker compose up failed", "projectName", projectName, "dir", projectDir, "error", err)
		return fmt.Errorf("docker compose up failed: %w", err)
	}

	return nil
}

func (d *Client) ComposeDown(ctx context.Context, projectDir, projectName string) error {
	d.logger.Info("Stopping containers with docker compose", "dir", projectDir)

	d.executor.SetWorkDir(projectDir)
	d.executor.SetTimeout(2 * time.Minute)

	composeFile := filepath.Join(projectDir, "docker-compose.yml")

	_, err := d.executor.Run(ctx, "docker", "compose", "-f", composeFile, "-p", projectName, "down")
	if err != nil {
		return fmt.Errorf("docker compose down failed: %w", err)
	}

	return nil
}

func (d *Client) Pull(ctx context.Context, image string) error {
	d.logger.Info("Pulling Docker image", "image", image)

	d.executor.SetTimeout(10 * time.Minute)

	_, err := d.executor.Run(ctx, "docker", "pull", image)
	if err != nil {
		return fmt.Errorf("docker pull failed: %w", err)
	}

	return nil
}

func (d *Client) Push(ctx context.Context, tag string) error {
	d.logger.Info("Pushing Docker image", "tag", tag)

	d.executor.SetTimeout(10 * time.Minute)

	_, err := d.executor.Run(ctx, "docker", "push", tag)
	if err != nil {
		return fmt.Errorf("docker push failed: %w", err)
	}

	return nil
}

func (d *Client) Tag(ctx context.Context, source, target string) error {
	d.logger.Info("Tagging Docker image", "source", source, "target", target)

	_, err := d.executor.Run(ctx, "docker", "tag", source, target)
	if err != nil {
		return fmt.Errorf("docker tag failed: %w", err)
	}

	return nil
}

func (d *Client) ImageExists(ctx context.Context, tag string) (bool, error) {
	result, err := d.executor.Run(ctx, "docker", "image", "inspect", tag)
	if err != nil {
		if strings.Contains(result.Stderr, "No such image") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (d *Client) RemoveImage(ctx context.Context, tag string) error {
	if tag == "" {
		return nil
	}

	d.logger.Info("Removing Docker image", "tag", tag)
	d.executor.SetTimeout(2 * time.Minute)

	_, err := d.executor.RunQuiet(ctx, "docker", "rmi", "-f", tag)
	if err != nil {
		d.logger.Debug("Failed to remove image (may not exist or in use)", "tag", tag)
		return nil
	}
	return nil
}

func (d *Client) EnsureNetwork(ctx context.Context, networkName string) error {
	d.logger.Info("Checking Docker network", "network", networkName)

	result, err := d.executor.Run(ctx, "docker", "network", "inspect", networkName)
	if err == nil {
		d.logger.Info("Network already exists", "network", networkName)
		return nil
	}

	if !strings.Contains(result.Stderr, "No such network") && !strings.Contains(result.Stderr, "network") {
		return fmt.Errorf("failed to inspect network: %w", err)
	}

	d.logger.Info("Creating Docker network", "network", networkName)
	_, err = d.executor.Run(ctx, "docker", "network", "create", networkName)
	if err != nil {
		return fmt.Errorf("failed to create network %s: %w", networkName, err)
	}

	d.logger.Info("Docker network created successfully", "network", networkName)
	return nil
}

func (d *Client) ConnectToNetwork(ctx context.Context, containerName, networkName string) error {
	result, err := d.executor.RunQuiet(ctx, "docker", "network", "connect", networkName, containerName)
	if err != nil {
		if strings.Contains(result.Stderr, "already exists") {
			d.logger.Debug("Container already connected to network", "container", containerName, "network", networkName)
			return nil
		}
		return fmt.Errorf("failed to connect container to network: %w", err)
	}
	d.logger.Info("Container connected to network", "container", containerName, "network", networkName)
	return nil
}

func (d *Client) GetCurrentContainerID(ctx context.Context) (string, error) {
	result, err := d.executor.RunQuiet(ctx, "cat", "/proc/self/cgroup")
	if err != nil {
		return "", fmt.Errorf("failed to read cgroup: %w", err)
	}

	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		if strings.Contains(line, "docker") {
			parts := strings.Split(line, "/")
			if len(parts) > 0 {
				containerID := parts[len(parts)-1]
				if len(containerID) >= 12 {
					return containerID[:12], nil
				}
			}
		}
	}

	result, err = d.executor.RunQuiet(ctx, "hostname")
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %w", err)
	}

	hostname := strings.TrimSpace(result.Stdout)
	if hostname != "" {
		return hostname, nil
	}

	return "", fmt.Errorf("could not determine container ID")
}

func (d *Client) getImagePrefix() string {
	if d.registry != "" {
		return d.registry + "/paasdeploy"
	}
	return "paasdeploy"
}

func (d *Client) GetImageTag(appName, commitSHA string) string {
	tag := commitSHA
	if len(tag) > 12 {
		tag = tag[:12]
	}
	return fmt.Sprintf("%s/%s:%s", d.getImagePrefix(), appName, tag)
}

func (d *Client) GetLatestTag(appName string) string {
	return fmt.Sprintf("%s/%s:latest", d.getImagePrefix(), appName)
}

type ContainerHealth struct {
	Name      string
	Status    string
	Health    string
	StartedAt string
	Uptime    string
	Image     string
}

func (d *Client) ContainerExists(ctx context.Context, containerName string) (bool, error) {
	d.executor.SetTimeout(30 * time.Second)

	result, err := d.executor.Run(ctx, "docker", "ps", "-a", "--filter", fmt.Sprintf("name=^%s$", containerName), formatFlag, "{{.Names}}")
	if err != nil {
		return false, fmt.Errorf("failed to check container: %w", err)
	}

	return strings.TrimSpace(result.Stdout) == containerName, nil
}

func (d *Client) InspectContainer(ctx context.Context, containerName string) (*ContainerHealth, error) {
	d.executor.SetTimeout(30 * time.Second)

	format := "{{.State.Status}}|{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}|{{.State.StartedAt}}|{{.Config.Image}}"
	result, err := d.executor.RunQuiet(ctx, "docker", "inspect", formatFlag, format, containerName)
	if err != nil {
		stderrLower := strings.ToLower(result.Stderr)
		if strings.Contains(stderrLower, "no such object") || strings.Contains(stderrLower, errNoSuchContainer) {
			return &ContainerHealth{
				Name:   containerName,
				Status: "not_found",
				Health: "none",
			}, nil
		}
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(result.Stdout), "|")
	if len(parts) < 4 {
		return nil, fmt.Errorf("unexpected inspect output: %s", result.Stdout)
	}

	health := &ContainerHealth{
		Name:      containerName,
		Status:    parts[0],
		Health:    parts[1],
		StartedAt: parts[2],
		Image:     parts[3],
	}

	if health.Status == "running" && health.StartedAt != "" {
		startTime, err := time.Parse(time.RFC3339Nano, health.StartedAt)
		if err == nil {
			health.Uptime = FormatUptime(time.Since(startTime))
		}
	}

	return health, nil
}

func FormatUptime(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func (d *Client) GetContainerIP(ctx context.Context, containerName, networkName string) (string, error) {
	d.executor.SetTimeout(30 * time.Second)

	format := fmt.Sprintf("{{.NetworkSettings.Networks.%s.IPAddress}}", networkName)
	result, err := d.executor.RunQuiet(ctx, "docker", "inspect", formatFlag, format, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to get container IP: %w", err)
	}

	ip := strings.TrimSpace(result.Stdout)
	if ip == "" {
		return "", fmt.Errorf("container %s has no IP in network %s", containerName, networkName)
	}

	return ip, nil
}

func (d *Client) IsCurrentContainer(ctx context.Context, containerID string) bool {
	selfID, err := d.GetCurrentContainerID(ctx)
	if err != nil {
		return false
	}
	if selfID == "" || containerID == "" {
		return false
	}
	shortSelf := selfID
	if len(shortSelf) > 12 {
		shortSelf = shortSelf[:12]
	}
	shortTarget := containerID
	if len(shortTarget) > 12 {
		shortTarget = shortTarget[:12]
	}
	return shortSelf == shortTarget
}

func (d *Client) RestartContainer(ctx context.Context, containerName string) error {
	d.logger.Info("Restarting container", "containerName", containerName)
	d.executor.SetTimeout(2 * time.Minute)

	if d.IsCurrentContainer(ctx, containerName) {
		return fmt.Errorf("cannot restart the current container from within itself - use SSH or host to restart")
	}

	_, err := d.executor.Run(ctx, "docker", "restart", containerName)
	if err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}

	return nil
}

func (d *Client) StopContainer(ctx context.Context, containerName string) error {
	d.logger.Info("Stopping container", "containerName", containerName)
	d.executor.SetTimeout(1 * time.Minute)

	if d.IsCurrentContainer(ctx, containerName) {
		return fmt.Errorf("cannot stop the current container from within itself - use SSH or host to stop")
	}

	_, err := d.executor.Run(ctx, "docker", "stop", containerName)
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	return nil
}

func (d *Client) StartContainer(ctx context.Context, containerName string) error {
	d.logger.Info("Starting container", "containerName", containerName)
	d.executor.SetTimeout(1 * time.Minute)

	_, err := d.executor.Run(ctx, "docker", "start", containerName)
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}

func (d *Client) ContainerLogs(ctx context.Context, containerName string, tail int) (string, error) {
	d.executor.SetTimeout(30 * time.Second)

	tailArg := "100"
	if tail > 0 {
		tailArg = fmt.Sprintf("%d", tail)
	}

	result, err := d.executor.RunQuiet(ctx, "docker", "logs", "--tail", tailArg, "--timestamps", containerName)
	if err != nil {
		stderrLower := strings.ToLower(result.Stderr)
		if strings.Contains(stderrLower, errNoSuchContainer) {
			return "", fmt.Errorf("container not found: %s", containerName)
		}
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}

	output := result.Stdout
	if result.Stderr != "" && !strings.Contains(strings.ToLower(result.Stderr), "error") {
		output = result.Stderr + output
	}

	return output, nil
}

func (d *Client) StreamContainerLogs(ctx context.Context, containerName string, output chan<- string) error {
	d.executor.SetTimeout(10 * time.Minute)

	return d.executor.RunWithStreaming(ctx, output, "docker", "logs", "-f", "--tail", "100", "--timestamps", containerName)
}

type ContainerStats struct {
	CPUPercent    float64 `json:"cpuPercent"`
	MemoryUsage   int64   `json:"memoryUsage"`
	MemoryLimit   int64   `json:"memoryLimit"`
	MemoryPercent float64 `json:"memoryPercent"`
	NetworkRx     int64   `json:"networkRx"`
	NetworkTx     int64   `json:"networkTx"`
	PIDs          int     `json:"pids"`
}

func (d *Client) ContainerStats(ctx context.Context, containerName string) (*ContainerStats, error) {
	d.executor.SetTimeout(30 * time.Second)

	format := "{{.CPUPerc}}|{{.MemUsage}}|{{.MemPerc}}|{{.NetIO}}|{{.PIDs}}"
	result, err := d.executor.RunQuiet(ctx, "docker", "stats", "--no-stream", formatFlag, format, containerName)
	if err != nil {
		stderrLower := strings.ToLower(result.Stderr)
		if strings.Contains(stderrLower, errNoSuchContainer) {
			return nil, fmt.Errorf("container not found: %s", containerName)
		}
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}

	output := strings.TrimSpace(result.Stdout)
	if output == "" {
		return nil, fmt.Errorf("no stats available for container: %s", containerName)
	}

	return parseContainerStats(output)
}

func parseContainerStats(output string) (*ContainerStats, error) {
	parts := strings.Split(output, "|")
	if len(parts) < 5 {
		return nil, fmt.Errorf("unexpected stats output: %s", output)
	}

	stats := &ContainerStats{}

	cpuStr := strings.TrimSuffix(parts[0], "%")
	fmt.Sscanf(cpuStr, "%f", &stats.CPUPercent)

	memParts := strings.Split(parts[1], " / ")
	if len(memParts) == 2 {
		stats.MemoryUsage = ParseMemoryValue(strings.TrimSpace(memParts[0]))
		stats.MemoryLimit = ParseMemoryValue(strings.TrimSpace(memParts[1]))
	}

	memPercStr := strings.TrimSuffix(parts[2], "%")
	fmt.Sscanf(memPercStr, "%f", &stats.MemoryPercent)

	netParts := strings.Split(parts[3], " / ")
	if len(netParts) == 2 {
		stats.NetworkRx = ParseNetworkValue(strings.TrimSpace(netParts[0]))
		stats.NetworkTx = ParseNetworkValue(strings.TrimSpace(netParts[1]))
	}

	fmt.Sscanf(parts[4], "%d", &stats.PIDs)

	return stats, nil
}

func ParseMemoryValue(s string) int64 {
	s = strings.ToUpper(s)
	var value float64
	var unit string
	fmt.Sscanf(s, "%f%s", &value, &unit)

	multiplier := int64(1)
	switch {
	case strings.HasPrefix(unit, "K"):
		multiplier = 1024
	case strings.HasPrefix(unit, "M"):
		multiplier = 1024 * 1024
	case strings.HasPrefix(unit, "G"):
		multiplier = 1024 * 1024 * 1024
	}

	return int64(value * float64(multiplier))
}

func ParseNetworkValue(s string) int64 {
	s = strings.ToUpper(s)
	var value float64
	var unit string
	fmt.Sscanf(s, "%f%s", &value, &unit)

	multiplier := int64(1)
	switch {
	case strings.HasPrefix(unit, "K"):
		multiplier = 1000
	case strings.HasPrefix(unit, "M"):
		multiplier = 1000 * 1000
	case strings.HasPrefix(unit, "G"):
		multiplier = 1000 * 1000 * 1000
	}

	return int64(value * float64(multiplier))
}

func (d *Client) PruneUnusedImages(ctx context.Context) error {
	d.logger.Info("Pruning unused Docker images")
	d.executor.SetTimeout(5 * time.Minute)

	_, err := d.executor.RunQuiet(ctx, "docker", "image", "prune", "-f")
	if err != nil {
		d.logger.Debug("Failed to prune images", "error", err)
		return nil
	}

	return nil
}

type ImageInfo struct {
	ID         string   `json:"id"`
	Repository string   `json:"repository"`
	Tag        string   `json:"tag"`
	Size       int64    `json:"size"`
	Created    string   `json:"created"`
	Containers int      `json:"containers"`
	Dangling   bool     `json:"dangling"`
	Labels     []string `json:"labels"`
}

func (d *Client) ListImages(ctx context.Context, all bool) ([]ImageInfo, error) {
	d.executor.SetTimeout(30 * time.Second)

	args := []string{"images", formatFlag, "{{.ID}}|{{.Repository}}|{{.Tag}}|{{.Size}}|{{.CreatedAt}}|{{.Containers}}"}
	if all {
		args = append(args, "-a")
	}

	result, err := d.executor.Run(ctx, "docker", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	return d.parseImageList(result.Stdout), nil
}

func (d *Client) parseImageList(output string) []ImageInfo {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	images := make([]ImageInfo, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 6 {
			continue
		}

		img := ImageInfo{
			ID:         parts[0],
			Repository: parts[1],
			Tag:        parts[2],
			Created:    parts[4],
		}

		img.Size = ParseImageSize(parts[3])
		img.Containers = parseImageContainers(parts[5])
		img.Dangling = parts[1] == "<none>" || parts[2] == "<none>"

		images = append(images, img)
	}

	return images
}

func ParseImageSize(sizeStr string) int64 {
	sizeStr = strings.ToUpper(sizeStr)
	var value float64
	var unit string
	fmt.Sscanf(sizeStr, "%f%s", &value, &unit)

	multiplier := int64(1)
	switch {
	case strings.HasPrefix(unit, "K"):
		multiplier = 1024
	case strings.HasPrefix(unit, "M"):
		multiplier = 1024 * 1024
	case strings.HasPrefix(unit, "G"):
		multiplier = 1024 * 1024 * 1024
	}

	return int64(value * float64(multiplier))
}

func parseImageContainers(containersStr string) int {
	containersStr = strings.TrimSpace(containersStr)
	if containersStr == "" || containersStr == "-" {
		return 0
	}
	containers, err := strconv.Atoi(containersStr)
	if err != nil {
		return 0
	}
	return containers
}

func (d *Client) RemoveImageByID(ctx context.Context, imageID string, force bool) error {
	d.logger.Info("Removing Docker image", "id", imageID, "force", force)
	d.executor.SetTimeout(2 * time.Minute)

	args := []string{"rmi"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, imageID)

	_, err := d.executor.Run(ctx, "docker", args...)
	if err != nil {
		return fmt.Errorf("failed to remove image: %w", err)
	}

	return nil
}

type PruneResult struct {
	ImagesDeleted  int
	SpaceReclaimed int64
}

func (d *Client) PruneImages(ctx context.Context) (*PruneResult, error) {
	d.logger.Info("Pruning all dangling Docker images")
	d.executor.SetTimeout(5 * time.Minute)

	result, err := d.executor.Run(ctx, "docker", "image", "prune", "-a", "-f")
	if err != nil {
		return nil, fmt.Errorf("failed to prune images: %w", err)
	}

	pruneResult := &PruneResult{}
	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Total reclaimed space") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				pruneResult.SpaceReclaimed = ParseImageSize(strings.TrimSpace(parts[1]))
			}
		}
		if strings.HasPrefix(line, "deleted:") || strings.HasPrefix(line, "Deleted:") {
			pruneResult.ImagesDeleted++
		}
	}

	return pruneResult, nil
}
