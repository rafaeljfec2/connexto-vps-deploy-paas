package engine

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"
)

type DockerClient struct {
	executor *Executor
	logger   *slog.Logger
	registry string
}

func NewDockerClient(baseDir string, registry string, logger *slog.Logger) *DockerClient {
	executor := NewExecutor(baseDir, 15*time.Minute, logger)
	return &DockerClient{
		executor: executor,
		logger:   logger,
		registry: registry,
	}
}

func (d *DockerClient) Build(ctx context.Context, workDir, dockerfile, tag string, output chan<- string) error {
	d.logger.Info("Building Docker image", "workDir", workDir, "dockerfile", dockerfile, "tag", tag)

	d.executor.SetWorkDir(workDir)
	d.executor.SetTimeout(15 * time.Minute)

	args := []string{
		"build",
		"-t", tag,
		"-f", dockerfile,
		".",
	}

	if output != nil {
		return d.executor.RunWithStreaming(ctx, output, "docker", args...)
	}

	_, err := d.executor.Run(ctx, "docker", args...)
	if err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	return nil
}

func (d *DockerClient) ComposeUp(ctx context.Context, projectDir string, output chan<- string) error {
	d.logger.Info("Starting containers with docker compose", "dir", projectDir)

	d.executor.SetWorkDir(projectDir)
	d.executor.SetTimeout(5 * time.Minute)

	composeFile := filepath.Join(projectDir, "docker-compose.yml")

	args := []string{
		"compose",
		"-f", composeFile,
		"up",
		"-d",
		"--remove-orphans",
	}

	if output != nil {
		return d.executor.RunWithStreaming(ctx, output, "docker", args...)
	}

	_, err := d.executor.Run(ctx, "docker", args...)
	if err != nil {
		return fmt.Errorf("docker compose up failed: %w", err)
	}

	return nil
}

func (d *DockerClient) ComposeDown(ctx context.Context, projectDir string) error {
	d.logger.Info("Stopping containers with docker compose", "dir", projectDir)

	d.executor.SetWorkDir(projectDir)
	d.executor.SetTimeout(2 * time.Minute)

	composeFile := filepath.Join(projectDir, "docker-compose.yml")

	_, err := d.executor.Run(ctx, "docker", "compose", "-f", composeFile, "down")
	if err != nil {
		return fmt.Errorf("docker compose down failed: %w", err)
	}

	return nil
}

func (d *DockerClient) Pull(ctx context.Context, image string) error {
	d.logger.Info("Pulling Docker image", "image", image)

	d.executor.SetTimeout(10 * time.Minute)

	_, err := d.executor.Run(ctx, "docker", "pull", image)
	if err != nil {
		return fmt.Errorf("docker pull failed: %w", err)
	}

	return nil
}

func (d *DockerClient) Push(ctx context.Context, tag string) error {
	d.logger.Info("Pushing Docker image", "tag", tag)

	d.executor.SetTimeout(10 * time.Minute)

	_, err := d.executor.Run(ctx, "docker", "push", tag)
	if err != nil {
		return fmt.Errorf("docker push failed: %w", err)
	}

	return nil
}

func (d *DockerClient) Tag(ctx context.Context, source, target string) error {
	d.logger.Info("Tagging Docker image", "source", source, "target", target)

	_, err := d.executor.Run(ctx, "docker", "tag", source, target)
	if err != nil {
		return fmt.Errorf("docker tag failed: %w", err)
	}

	return nil
}

func (d *DockerClient) ImageExists(ctx context.Context, tag string) (bool, error) {
	result, err := d.executor.Run(ctx, "docker", "image", "inspect", tag)
	if err != nil {
		if strings.Contains(result.Stderr, "No such image") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (d *DockerClient) RemoveImage(ctx context.Context, tag string) error {
	_, err := d.executor.Run(ctx, "docker", "rmi", "-f", tag)
	if err != nil {
		return fmt.Errorf("docker rmi failed: %w", err)
	}
	return nil
}

func (d *DockerClient) EnsureNetwork(ctx context.Context, networkName string) error {
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

func (d *DockerClient) getImagePrefix() string {
	if d.registry != "" {
		return d.registry + "/paasdeploy"
	}
	return "paasdeploy"
}

func (d *DockerClient) GetImageTag(appName, commitSHA string) string {
	tag := commitSHA
	if len(tag) > 12 {
		tag = tag[:12]
	}
	return fmt.Sprintf("%s/%s:%s", d.getImagePrefix(), appName, tag)
}

func (d *DockerClient) GetLatestTag(appName string) string {
	return fmt.Sprintf("%s/%s:latest", d.getImagePrefix(), appName)
}

type ContainerHealth struct {
	Name      string
	Status    string
	Health    string
	StartedAt string
	Uptime    string
}

func (d *DockerClient) ContainerExists(ctx context.Context, containerName string) (bool, error) {
	d.executor.SetTimeout(30 * time.Second)

	result, err := d.executor.Run(ctx, "docker", "ps", "-a", "--filter", fmt.Sprintf("name=^%s$", containerName), "--format", "{{.Names}}")
	if err != nil {
		return false, fmt.Errorf("failed to check container: %w", err)
	}

	return strings.TrimSpace(result.Stdout) == containerName, nil
}

func (d *DockerClient) InspectContainer(ctx context.Context, containerName string) (*ContainerHealth, error) {
	d.executor.SetTimeout(30 * time.Second)

	format := "{{.State.Status}}|{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}|{{.State.StartedAt}}"
	result, err := d.executor.RunQuiet(ctx, "docker", "inspect", "--format", format, containerName)
	if err != nil {
		stderrLower := strings.ToLower(result.Stderr)
		if strings.Contains(stderrLower, "no such object") || strings.Contains(stderrLower, "no such container") {
			return &ContainerHealth{
				Name:   containerName,
				Status: "not_found",
				Health: "none",
			}, nil
		}
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(result.Stdout), "|")
	if len(parts) < 3 {
		return nil, fmt.Errorf("unexpected inspect output: %s", result.Stdout)
	}

	health := &ContainerHealth{
		Name:      containerName,
		Status:    parts[0],
		Health:    parts[1],
		StartedAt: parts[2],
	}

	if health.Status == "running" && health.StartedAt != "" {
		startTime, err := time.Parse(time.RFC3339Nano, health.StartedAt)
		if err == nil {
			health.Uptime = formatUptime(time.Since(startTime))
		}
	}

	return health, nil
}

func formatUptime(d time.Duration) string {
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
