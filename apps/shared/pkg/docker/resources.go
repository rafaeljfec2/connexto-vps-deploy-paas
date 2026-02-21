package docker

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (d *Client) ListNetworks(ctx context.Context) ([]string, error) {
	result, err := d.executor.RunQuietWithTimeout(ctx, 30*time.Second, "docker", "network", "ls", "--format", "{{.Name}}")
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

func (d *Client) ListVolumes(ctx context.Context) ([]string, error) {
	result, err := d.executor.RunQuietWithTimeout(ctx, 30*time.Second, "docker", "volume", "ls", "--format", "{{.Name}}")
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
