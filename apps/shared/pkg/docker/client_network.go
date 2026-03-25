package docker

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type NetworkInfo struct {
	Name       string   `json:"name"`
	ID         string   `json:"id"`
	Driver     string   `json:"driver"`
	Scope      string   `json:"scope"`
	Internal   bool     `json:"internal"`
	Containers []string `json:"containers"`
}

type VolumeInfo struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint"`
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

func (d *Client) ListNetworks(ctx context.Context) ([]NetworkInfo, error) {
	result, err := d.executor.RunQuietWithTimeout(ctx, 30*time.Second,
		"docker", "network", "ls", "--format", "{{.ID}}|{{.Name}}|{{.Driver}}|{{.Scope}}|{{.Internal}}")
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}
	out := strings.TrimSpace(result.Stdout)
	if out == "" {
		return []NetworkInfo{}, nil
	}

	lines := strings.Split(out, "\n")
	networks := make([]NetworkInfo, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 3 {
			continue
		}
		net := NetworkInfo{
			ID:     parts[0],
			Name:   parts[1],
			Driver: parts[2],
		}
		if len(parts) > 3 {
			net.Scope = parts[3]
		}
		if len(parts) > 4 {
			net.Internal = parts[4] == "true"
		}
		networks = append(networks, net)
	}
	return networks, nil
}

func (d *Client) RemoveNetwork(ctx context.Context, network string) error {
	_, err := d.executor.RunWithTimeout(ctx, 30*time.Second, "docker", "network", "rm", network)
	if err != nil {
		return fmt.Errorf("failed to remove network: %w", err)
	}
	return nil
}

func (d *Client) DisconnectFromNetwork(ctx context.Context, containerName, networkName string) error {
	_, err := d.executor.RunQuietWithTimeout(ctx, 30*time.Second, "docker", "network", "disconnect", networkName, containerName)
	if err != nil {
		return fmt.Errorf("failed to disconnect container from network: %w", err)
	}
	return nil
}

func (d *Client) ListVolumes(ctx context.Context) ([]VolumeInfo, error) {
	result, err := d.executor.RunQuietWithTimeout(ctx, 30*time.Second,
		"docker", "volume", "ls", "--format", "{{.Name}}|{{.Driver}}|{{.Mountpoint}}")
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}
	out := strings.TrimSpace(result.Stdout)
	if out == "" {
		return []VolumeInfo{}, nil
	}

	lines := strings.Split(out, "\n")
	volumes := make([]VolumeInfo, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 1 || parts[0] == "" {
			continue
		}
		vol := VolumeInfo{Name: parts[0]}
		if len(parts) > 1 {
			vol.Driver = parts[1]
		}
		if len(parts) > 2 {
			vol.Mountpoint = parts[2]
		}
		volumes = append(volumes, vol)
	}
	return volumes, nil
}

func (d *Client) CreateVolume(ctx context.Context, name string) error {
	_, err := d.executor.RunWithTimeout(ctx, 30*time.Second, "docker", "volume", "create", name)
	if err != nil {
		return fmt.Errorf("failed to create volume: %w", err)
	}
	return nil
}

func (d *Client) RemoveVolume(ctx context.Context, name string) error {
	_, err := d.executor.RunWithTimeout(ctx, 30*time.Second, "docker", "volume", "rm", name)
	if err != nil {
		return fmt.Errorf("failed to remove volume: %w", err)
	}
	return nil
}
