package engine

import (
	"context"
	"fmt"
	"strings"
)

// ListNetworks returns names of docker networks
func (d *DockerClient) ListNetworks(ctx context.Context) ([]string, error) {
	d.executor.SetTimeout(30 * 1e9) // 30s
	result, err := d.executor.RunQuiet(ctx, "docker", "network", "ls", "--format", "{{.Name}}")
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}
	out := strings.TrimSpace(result.Stdout)
	if out == "" {
		return []string{}, nil
	}
	lines := strings.Split(out, "\n")
	return lines, nil
}

// RemoveNetwork removes a docker network by name
func (d *DockerClient) RemoveNetwork(ctx context.Context, network string) error {
	d.executor.SetTimeout(30 * 1e9)
	_, err := d.executor.Run(ctx, "docker", "network", "rm", network)
	if err != nil {
		return fmt.Errorf("failed to remove network: %w", err)
	}
	return nil
}

// DisconnectFromNetwork disconnects container from network
func (d *DockerClient) DisconnectFromNetwork(ctx context.Context, containerName, networkName string) error {
	d.executor.SetTimeout(30 * 1e9)
	_, err := d.executor.RunQuiet(ctx, "docker", "network", "disconnect", networkName, containerName)
	if err != nil {
		// ignore already not connected
		return fmt.Errorf("failed to disconnect container from network: %w", err)
	}
	return nil
}

// ListVolumes returns list of docker volumes (names)
func (d *DockerClient) ListVolumes(ctx context.Context) ([]string, error) {
	d.executor.SetTimeout(30 * 1e9)
	result, err := d.executor.RunQuiet(ctx, "docker", "volume", "ls", "--format", "{{.Name}}")
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}
	out := strings.TrimSpace(result.Stdout)
	if out == "" {
		return []string{}, nil
	}
	lines := strings.Split(out, "\n")
	return lines, nil
}

// CreateVolume creates a docker volume
func (d *DockerClient) CreateVolume(ctx context.Context, name string) error {
	d.executor.SetTimeout(30 * 1e9)
	_, err := d.executor.Run(ctx, "docker", "volume", "create", name)
	if err != nil {
		return fmt.Errorf("failed to create volume: %w", err)
	}
	return nil
}

// RemoveVolume removes a docker volume
func (d *DockerClient) RemoveVolume(ctx context.Context, name string) error {
	d.executor.SetTimeout(30 * 1e9)
	_, err := d.executor.Run(ctx, "docker", "volume", "rm", name)
	if err != nil {
		return fmt.Errorf("failed to remove volume: %w", err)
	}
	return nil
}

