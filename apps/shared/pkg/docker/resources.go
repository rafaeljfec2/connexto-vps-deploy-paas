package docker

import (
	"context"
	"fmt"
	"strings"
)

func (d *Client) ListNetworks(ctx context.Context) ([]string, error) {
	d.executor.SetTimeout(30 * 1e9)
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

func (d *Client) RemoveNetwork(ctx context.Context, network string) error {
	d.executor.SetTimeout(30 * 1e9)
	_, err := d.executor.Run(ctx, "docker", "network", "rm", network)
	if err != nil {
		return fmt.Errorf("failed to remove network: %w", err)
	}
	return nil
}

func (d *Client) DisconnectFromNetwork(ctx context.Context, containerName, networkName string) error {
	d.executor.SetTimeout(30 * 1e9)
	_, err := d.executor.RunQuiet(ctx, "docker", "network", "disconnect", networkName, containerName)
	if err != nil {
		return fmt.Errorf("failed to disconnect container from network: %w", err)
	}
	return nil
}

func (d *Client) ListVolumes(ctx context.Context) ([]string, error) {
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

func (d *Client) CreateVolume(ctx context.Context, name string) error {
	d.executor.SetTimeout(30 * 1e9)
	_, err := d.executor.Run(ctx, "docker", "volume", "create", name)
	if err != nil {
		return fmt.Errorf("failed to create volume: %w", err)
	}
	return nil
}

func (d *Client) RemoveVolume(ctx context.Context, name string) error {
	d.executor.SetTimeout(30 * 1e9)
	_, err := d.executor.Run(ctx, "docker", "volume", "rm", name)
	if err != nil {
		return fmt.Errorf("failed to remove volume: %w", err)
	}
	return nil
}
