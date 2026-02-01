package engine

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const dockerFormatArg = "--format"

type ContainerInfo struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	State     string            `json:"state"`
	Status    string            `json:"status"`
	Health    string            `json:"health"`
	Created   string            `json:"created"`
	IPAddress string            `json:"ipAddress"`
	Ports     []ContainerPort   `json:"ports"`
	Labels    map[string]string `json:"labels"`
}

type ContainerPort struct {
	PrivatePort int    `json:"privatePort"`
	PublicPort  int    `json:"publicPort,omitempty"`
	Type        string `json:"type"`
}

type CreateContainerOptions struct {
	Name          string            `json:"name"`
	Image         string            `json:"image"`
	Ports         []PortMapping     `json:"ports,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	Volumes       []VolumeMapping   `json:"volumes,omitempty"`
	Network       string            `json:"network,omitempty"`
	RestartPolicy string            `json:"restartPolicy,omitempty"`
	Command       []string          `json:"command,omitempty"`
}

type PortMapping struct {
	HostPort      int    `json:"hostPort"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol,omitempty"`
}

type VolumeMapping struct {
	HostPath      string `json:"hostPath"`
	ContainerPath string `json:"containerPath"`
	ReadOnly      bool   `json:"readOnly,omitempty"`
}

func (d *DockerClient) ListContainers(ctx context.Context, all bool) ([]ContainerInfo, error) {
	d.executor.SetTimeout(30 * time.Second)

	args := []string{"ps", dockerFormatArg, "{{.ID}}|{{.Names}}|{{.Image}}|{{.State}}|{{.Status}}|{{.Ports}}|{{.CreatedAt}}|{{.Labels}}"}
	if all {
		args = append(args, "-a")
	}

	result, err := d.executor.Run(ctx, "docker", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	return d.parseContainerList(ctx, result.Stdout), nil
}

func (d *DockerClient) parseContainerList(ctx context.Context, output string) []ContainerInfo {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	containers := make([]ContainerInfo, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		container := d.parseContainerLine(ctx, line)
		if container != nil {
			containers = append(containers, *container)
		}
	}

	return containers
}

func (d *DockerClient) parseContainerLine(ctx context.Context, line string) *ContainerInfo {
	parts := strings.Split(line, "|")
	if len(parts) < 7 {
		return nil
	}

	labelsStr := ""
	if len(parts) > 7 {
		labelsStr = parts[7]
	}

	container := &ContainerInfo{
		ID:      parts[0],
		Name:    parts[1],
		Image:   parts[2],
		State:   parts[3],
		Status:  parts[4],
		Created: parts[6],
		Labels:  parseLabels(labelsStr),
		Ports:   parsePorts(parts[5]),
	}

	container.Health, _ = d.getContainerHealth(ctx, container.ID)
	container.IPAddress, _ = d.getContainerIP(ctx, container.ID)

	return container
}

func (d *DockerClient) getContainerHealth(ctx context.Context, containerID string) (string, error) {
	format := "{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}"
	result, err := d.executor.RunQuiet(ctx, "docker", "inspect", dockerFormatArg, format, containerID)
	if err != nil {
		return "none", nil
	}
	return strings.TrimSpace(result.Stdout), nil
}

func (d *DockerClient) getContainerIP(ctx context.Context, containerID string) (string, error) {
	format := "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}"
	result, err := d.executor.RunQuiet(ctx, "docker", "inspect", dockerFormatArg, format, containerID)
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(result.Stdout), nil
}

func (d *DockerClient) CreateContainer(ctx context.Context, opts CreateContainerOptions) (string, error) {
	d.logger.Info("Creating container", "name", opts.Name, "image", opts.Image)
	d.executor.SetTimeout(5 * time.Minute)

	if err := d.Pull(ctx, opts.Image); err != nil {
		d.logger.Warn("Failed to pull image, trying to use local", "image", opts.Image, "error", err)
	}

	args := d.buildContainerArgs(opts)

	result, err := d.executor.Run(ctx, "docker", args...)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	containerID := strings.TrimSpace(result.Stdout)
	d.logger.Info("Container created", "id", containerID, "name", opts.Name)

	return containerID, nil
}

func (d *DockerClient) buildContainerArgs(opts CreateContainerOptions) []string {
	args := []string{"run", "-d"}

	if opts.Name != "" {
		args = append(args, "--name", opts.Name)
	}

	args = append(args, buildPortArgs(opts.Ports)...)
	args = append(args, buildEnvArgs(opts.Env)...)
	args = append(args, buildVolumeArgs(opts.Volumes)...)

	if opts.Network != "" {
		args = append(args, "--network", opts.Network)
	}

	if opts.RestartPolicy != "" {
		args = append(args, "--restart", opts.RestartPolicy)
	}

	args = append(args, opts.Image)
	args = append(args, opts.Command...)

	return args
}

func buildPortArgs(ports []PortMapping) []string {
	args := make([]string, 0, len(ports)*2)
	for _, p := range ports {
		protocol := p.Protocol
		if protocol == "" {
			protocol = "tcp"
		}
		args = append(args, "-p", fmt.Sprintf("%d:%d/%s", p.HostPort, p.ContainerPort, protocol))
	}
	return args
}

func buildEnvArgs(env map[string]string) []string {
	args := make([]string, 0, len(env)*2)
	for key, value := range env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}
	return args
}

func buildVolumeArgs(volumes []VolumeMapping) []string {
	args := make([]string, 0, len(volumes)*2)
	for _, v := range volumes {
		volumeArg := fmt.Sprintf("%s:%s", v.HostPath, v.ContainerPath)
		if v.ReadOnly {
			volumeArg += ":ro"
		}
		args = append(args, "-v", volumeArg)
	}
	return args
}

func (d *DockerClient) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	d.logger.Info("Removing container", "id", containerID, "force", force)
	d.executor.SetTimeout(1 * time.Minute)

	args := []string{"rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, containerID)

	_, err := d.executor.Run(ctx, "docker", args...)
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	return nil
}

func (d *DockerClient) GetContainerDetails(ctx context.Context, containerID string) (*ContainerInfo, error) {
	d.executor.SetTimeout(30 * time.Second)

	format := "{{.ID}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}|{{.State.StartedAt}}|{{range $k, $v := .Config.Labels}}{{$k}}={{$v}},{{end}}"
	result, err := d.executor.Run(ctx, "docker", "inspect", dockerFormatArg, format, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	return d.parseContainerDetails(ctx, containerID, result.Stdout)
}

func (d *DockerClient) parseContainerDetails(ctx context.Context, containerID, output string) (*ContainerInfo, error) {
	parts := strings.Split(strings.TrimSpace(output), "|")
	if len(parts) < 5 {
		return nil, fmt.Errorf("unexpected inspect output")
	}

	labelsStr := ""
	if len(parts) > 5 {
		labelsStr = strings.TrimSuffix(parts[5], ",")
	}

	container := &ContainerInfo{
		ID:     parts[0],
		Name:   strings.TrimPrefix(parts[1], "/"),
		Image:  parts[2],
		State:  parts[3],
		Labels: parseLabels(labelsStr),
	}

	container.Health, _ = d.getContainerHealth(ctx, containerID)
	container.IPAddress, _ = d.getContainerIP(ctx, containerID)

	portsResult, err := d.executor.RunQuiet(ctx, "docker", "port", containerID)
	if err == nil {
		container.Ports = parseDockerPortOutput(portsResult.Stdout)
	}

	return container, nil
}

func parseLabels(labelStr string) map[string]string {
	labels := make(map[string]string)
	if labelStr == "" {
		return labels
	}

	pairs := strings.Split(labelStr, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			labels[kv[0]] = kv[1]
		}
	}
	return labels
}

func parsePorts(portStr string) []ContainerPort {
	if portStr == "" {
		return []ContainerPort{}
	}

	mappings := strings.Split(portStr, ", ")
	ports := make([]ContainerPort, 0, len(mappings))

	for _, mapping := range mappings {
		if port := parsePortMapping(mapping); port != nil {
			ports = append(ports, *port)
		}
	}
	return ports
}

func parsePortMapping(mapping string) *ContainerPort {
	mapping = strings.TrimSpace(mapping)
	if mapping == "" {
		return nil
	}

	port := &ContainerPort{Type: "tcp"}

	if strings.HasSuffix(mapping, "/udp") {
		port.Type = "udp"
		mapping = strings.TrimSuffix(mapping, "/udp")
	} else {
		mapping = strings.TrimSuffix(mapping, "/tcp")
	}

	if strings.Contains(mapping, "->") {
		parts := strings.Split(mapping, "->")
		if len(parts) == 2 {
			port.PublicPort = parsePortNumber(strings.TrimSpace(parts[0]))
			port.PrivatePort = parsePortNumber(strings.TrimSpace(parts[1]))
		}
	} else {
		port.PrivatePort = parsePortNumber(mapping)
	}

	if port.PrivatePort == 0 && port.PublicPort == 0 {
		return nil
	}

	return port
}

func parsePortNumber(s string) int {
	if idx := strings.LastIndex(s, ":"); idx != -1 {
		s = s[idx+1:]
	}
	var port int
	fmt.Sscanf(s, "%d", &port)
	return port
}

func parseDockerPortOutput(output string) []ContainerPort {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	ports := make([]ContainerPort, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, " -> ")
		if len(parts) != 2 {
			continue
		}

		port := ContainerPort{Type: "tcp"}
		containerPart := parts[0]
		hostPart := parts[1]

		if strings.Contains(containerPart, "/udp") {
			port.Type = "udp"
		}
		fmt.Sscanf(containerPart, "%d", &port.PrivatePort)

		if idx := strings.LastIndex(hostPart, ":"); idx != -1 {
			fmt.Sscanf(hostPart[idx+1:], "%d", &port.PublicPort)
		}

		if port.PrivatePort > 0 {
			ports = append(ports, port)
		}
	}

	return ports
}
