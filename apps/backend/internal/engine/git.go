package engine

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"
)

type GitClient struct {
	executor *Executor
	logger   *slog.Logger
}

func NewGitClient(baseDir string, logger *slog.Logger) *GitClient {
	executor := NewExecutor(baseDir, 5*time.Minute, logger)
	return &GitClient{
		executor: executor,
		logger:   logger,
	}
}

func (g *GitClient) Clone(ctx context.Context, repoURL, targetDir string) error {
	g.logger.Info("Cloning repository", "url", repoURL, "target", targetDir)

	g.executor.SetWorkDir(filepath.Dir(targetDir))

	_, err := g.executor.Run(ctx, "git", "clone", "--depth", "1", repoURL, filepath.Base(targetDir))
	if err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	return nil
}

func (g *GitClient) Fetch(ctx context.Context, repoDir string) error {
	g.logger.Info("Fetching updates", "dir", repoDir)

	g.executor.SetWorkDir(repoDir)

	_, err := g.executor.Run(ctx, "git", "fetch", "origin")
	if err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}

	return nil
}

func (g *GitClient) ResetHard(ctx context.Context, repoDir, commitSHA string) error {
	g.logger.Info("Resetting to commit", "dir", repoDir, "commit", commitSHA)

	g.executor.SetWorkDir(repoDir)

	target := commitSHA
	if commitSHA == "" || commitSHA == "HEAD" {
		target = "origin/HEAD"
	}

	_, err := g.executor.Run(ctx, "git", "reset", "--hard", target)
	if err != nil {
		return fmt.Errorf("git reset failed: %w", err)
	}

	return nil
}

func (g *GitClient) GetCurrentCommitSHA(ctx context.Context, repoDir string) (string, error) {
	g.executor.SetWorkDir(repoDir)

	result, err := g.executor.Run(ctx, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get commit SHA: %w", err)
	}

	return strings.TrimSpace(result.Stdout), nil
}

func (g *GitClient) GetCommitMessage(ctx context.Context, repoDir string) (string, error) {
	g.executor.SetWorkDir(repoDir)

	result, err := g.executor.Run(ctx, "git", "log", "-1", "--pretty=%s")
	if err != nil {
		return "", fmt.Errorf("failed to get commit message: %w", err)
	}

	return strings.TrimSpace(result.Stdout), nil
}

func (g *GitClient) Sync(ctx context.Context, repoDir, commitSHA string) error {
	if err := g.Fetch(ctx, repoDir); err != nil {
		return err
	}

	if err := g.ResetHard(ctx, repoDir, commitSHA); err != nil {
		return err
	}

	return nil
}

func (g *GitClient) GetBranch(ctx context.Context, repoDir string) (string, error) {
	g.executor.SetWorkDir(repoDir)

	result, err := g.executor.Run(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get branch: %w", err)
	}

	return strings.TrimSpace(result.Stdout), nil
}

func (g *GitClient) CheckoutBranch(ctx context.Context, repoDir, branch string) error {
	g.logger.Info("Checking out branch", "dir", repoDir, "branch", branch)

	g.executor.SetWorkDir(repoDir)

	_, err := g.executor.Run(ctx, "git", "checkout", branch)
	if err != nil {
		return fmt.Errorf("git checkout failed: %w", err)
	}

	return nil
}
