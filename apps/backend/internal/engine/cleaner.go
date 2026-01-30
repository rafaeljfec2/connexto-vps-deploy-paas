package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type AppCleaner struct {
	baseDir  string
	executor *Executor
	logger   *slog.Logger
}

func NewAppCleaner(baseDir string, logger *slog.Logger) *AppCleaner {
	return &AppCleaner{
		baseDir:  baseDir,
		executor: NewExecutor(baseDir, 2*time.Minute, logger),
		logger:   logger,
	}
}

func (c *AppCleaner) CleanApp(ctx context.Context, appName string) error {
	c.logger.Info("Starting app cleanup", "appName", appName)

	if err := c.stopContainers(ctx, appName); err != nil {
		c.logger.Warn("Failed to stop containers", "appName", appName, "error", err)
	}

	if err := c.removeImages(ctx, appName); err != nil {
		c.logger.Warn("Failed to remove images", "appName", appName, "error", err)
	}

	if err := c.removeFiles(appName); err != nil {
		c.logger.Warn("Failed to remove files", "appName", appName, "error", err)
	}

	c.logger.Info("App cleanup completed", "appName", appName)
	return nil
}

func (c *AppCleaner) stopContainers(ctx context.Context, appName string) error {
	appDir := filepath.Join(c.baseDir, appName)
	composeFile := filepath.Join(appDir, "docker-compose.yml")

	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		c.logger.Debug("No docker-compose.yml found, skipping container stop", "appName", appName)
		return nil
	}

	c.executor.SetWorkDir(appDir)
	c.executor.SetTimeout(2 * time.Minute)

	_, err := c.executor.Run(ctx, "docker", "compose", "-f", composeFile, "down", "--remove-orphans", "-v")
	if err != nil {
		return fmt.Errorf("docker compose down failed: %w", err)
	}

	return nil
}

func (c *AppCleaner) removeImages(ctx context.Context, appName string) error {
	c.executor.SetTimeout(1 * time.Minute)

	imagePattern := fmt.Sprintf("paasdeploy/%s", appName)

	result, err := c.executor.Run(ctx, "docker", "images", "--filter", fmt.Sprintf("reference=%s*", imagePattern), "-q")
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	if result.Stdout == "" {
		c.logger.Debug("No images found to remove", "appName", appName)
		return nil
	}

	_, err = c.executor.Run(ctx, "docker", "rmi", "-f", result.Stdout)
	if err != nil {
		return fmt.Errorf("failed to remove images: %w", err)
	}

	return nil
}

func (c *AppCleaner) removeFiles(appName string) error {
	appDir := filepath.Join(c.baseDir, appName)

	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		c.logger.Debug("App directory does not exist, skipping file removal", "appName", appName, "dir", appDir)
		return nil
	}

	if err := os.RemoveAll(appDir); err != nil {
		return fmt.Errorf("failed to remove app directory: %w", err)
	}

	c.logger.Info("Removed app directory", "appName", appName, "dir", appDir)
	return nil
}
