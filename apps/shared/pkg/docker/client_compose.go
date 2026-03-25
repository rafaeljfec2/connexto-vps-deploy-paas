package docker

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
)

func (d *Client) ComposeUp(ctx context.Context, projectDir, projectName string, output chan<- string) error {
	d.logger.Info("Starting containers with docker compose", "dir", projectDir)

	d.executor.SetWorkDir(projectDir)

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
		err := d.executor.RunWithStreamingTimeout(ctx, 5*time.Minute, output, "docker", args...)
		if err != nil {
			d.logger.Error("Docker compose up failed", "projectName", projectName, "dir", projectDir, "error", err)
			return fmt.Errorf("docker compose up failed: %w", err)
		}
		return nil
	}

	_, err := d.executor.RunWithTimeout(ctx, 5*time.Minute, "docker", args...)
	if err != nil {
		d.logger.Error("Docker compose up failed", "projectName", projectName, "dir", projectDir, "error", err)
		return fmt.Errorf("docker compose up failed: %w", err)
	}

	return nil
}

func (d *Client) ComposeDown(ctx context.Context, projectDir, projectName string) error {
	d.logger.Info("Stopping containers with docker compose", "dir", projectDir)

	d.executor.SetWorkDir(projectDir)

	composeFile := filepath.Join(projectDir, "docker-compose.yml")

	_, err := d.executor.RunWithTimeout(ctx, 2*time.Minute, "docker", "compose", "-f", composeFile, "-p", projectName, "down")
	if err != nil {
		return fmt.Errorf("docker compose down failed: %w", err)
	}

	return nil
}
