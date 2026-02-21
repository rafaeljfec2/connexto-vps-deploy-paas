package cleaner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/paasdeploy/shared/pkg/executor"
)

type Cleaner struct {
	baseDir  string
	executor *executor.Executor
	logger   *slog.Logger
}

func New(baseDir string, logger *slog.Logger) *Cleaner {
	return &Cleaner{
		baseDir:  baseDir,
		executor: executor.New(baseDir, 2*time.Minute, logger),
		logger:   logger,
	}
}

func (c *Cleaner) CleanApp(ctx context.Context, appID, appName string) error {
	c.logger.Info("Starting app cleanup", "appID", appID, "appName", appName)

	if err := c.stopAndRemoveContainer(ctx, appName); err != nil {
		c.logger.Warn("Failed to stop/remove container", "appName", appName, "error", err)
	}

	if err := c.removeImages(ctx, appName); err != nil {
		c.logger.Warn("Failed to remove images", "appName", appName, "error", err)
	}

	if err := c.removeFiles(appID); err != nil {
		c.logger.Warn("Failed to remove files", "appID", appID, "error", err)
	}

	c.logger.Info("App cleanup completed", "appID", appID, "appName", appName)
	return nil
}

func (c *Cleaner) stopAndRemoveContainer(ctx context.Context, appName string) error {
	c.logger.Info("Stopping container", "containerName", appName)
	_, _ = c.executor.RunWithTimeout(ctx, 2*time.Minute, "docker", "stop", appName)

	c.logger.Info("Removing container with volumes", "containerName", appName)
	_, err := c.executor.RunWithTimeout(ctx, 2*time.Minute, "docker", "rm", "-f", "-v", appName)
	if err != nil {
		return fmt.Errorf("docker rm failed: %w", err)
	}

	return nil
}

func (c *Cleaner) removeImages(ctx context.Context, appName string) error {
	imagePattern := fmt.Sprintf("paasdeploy/%s", appName)

	result, err := c.executor.RunWithTimeout(ctx, 1*time.Minute, "docker", "images", "--filter", fmt.Sprintf("reference=%s*", imagePattern), "-q")
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	if result.Stdout == "" {
		c.logger.Debug("No images found to remove", "appName", appName)
		return nil
	}

	_, err = c.executor.RunWithTimeout(ctx, 1*time.Minute, "docker", "rmi", "-f", result.Stdout)
	if err != nil {
		return fmt.Errorf("failed to remove images: %w", err)
	}

	return nil
}

func (c *Cleaner) removeFiles(appID string) error {
	appDir := filepath.Join(c.baseDir, appID)

	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		c.logger.Debug("App directory does not exist, skipping file removal", "appID", appID, "dir", appDir)
		return nil
	}

	if err := os.RemoveAll(appDir); err != nil {
		return fmt.Errorf("failed to remove app directory: %w", err)
	}

	c.logger.Info("Removed app directory", "appID", appID, "dir", appDir)
	return nil
}
