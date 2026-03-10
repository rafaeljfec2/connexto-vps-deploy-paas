package git

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/paasdeploy/shared/pkg/executor"
)

type Client struct {
	executor *executor.Executor
	logger   *slog.Logger
}

func NewClient(baseDir string, logger *slog.Logger) *Client {
	exec := executor.New(baseDir, 5*time.Minute, logger)
	return &Client{
		executor: exec,
		logger:   logger,
	}
}

func (g *Client) Clone(ctx context.Context, repoURL, targetDir string) error {
	return g.CloneWithToken(ctx, repoURL, targetDir, "")
}

func (g *Client) CloneWithToken(ctx context.Context, repoURL, targetDir, token string) error {
	g.logger.Info("Cloning repository", "url", repoURL, "target", targetDir, "authenticated", token != "")

	cloneURL := repoURL
	if token != "" {
		authenticatedURL, err := InjectTokenIntoURL(repoURL, token)
		if err != nil {
			return fmt.Errorf("failed to create authenticated URL: %w", err)
		}
		cloneURL = authenticatedURL
	}

	g.executor.SetWorkDir(filepath.Dir(targetDir))

	_, err := g.executor.Run(ctx, "git", "clone", "--depth", "1", cloneURL, filepath.Base(targetDir))
	if err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	return nil
}

func InjectTokenIntoURL(repoURL, token string) (string, error) {
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return "", err
	}

	parsed.User = url.UserPassword("x-access-token", token)
	return parsed.String(), nil
}

func (g *Client) Fetch(ctx context.Context, repoDir string) error {
	return g.FetchWithToken(ctx, repoDir, "", "")
}

func (g *Client) FetchWithToken(ctx context.Context, repoDir, repoURL, token string) error {
	g.logger.Info("Fetching updates", "dir", repoDir, "authenticated", token != "")

	g.executor.SetWorkDir(repoDir)

	if token != "" && repoURL != "" {
		authenticatedURL, err := InjectTokenIntoURL(repoURL, token)
		if err != nil {
			return fmt.Errorf("failed to create authenticated URL: %w", err)
		}
		_, err = g.executor.Run(ctx, "git", "remote", "set-url", "origin", authenticatedURL)
		if err != nil {
			g.logger.Warn("Failed to update remote URL with token", "error", err)
		}
	}

	_, err := g.executor.Run(ctx, "git", "fetch", "origin")
	if err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}

	if token != "" && repoURL != "" {
		_, _ = g.executor.Run(ctx, "git", "remote", "set-url", "origin", repoURL)
	}

	return nil
}

func (g *Client) ResetHard(ctx context.Context, repoDir, commitSHA string) error {
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

func (g *Client) GetCurrentCommitSHA(ctx context.Context, repoDir string) (string, error) {
	g.executor.SetWorkDir(repoDir)

	result, err := g.executor.Run(ctx, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get commit SHA: %w", err)
	}

	return strings.TrimSpace(result.Stdout), nil
}

func (g *Client) GetCommitMessage(ctx context.Context, repoDir string) (string, error) {
	g.executor.SetWorkDir(repoDir)

	result, err := g.executor.Run(ctx, "git", "log", "-1", "--pretty=%s")
	if err != nil {
		return "", fmt.Errorf("failed to get commit message: %w", err)
	}

	return strings.TrimSpace(result.Stdout), nil
}

func (g *Client) Sync(ctx context.Context, repoDir, commitSHA string) error {
	return g.SyncWithToken(ctx, repoDir, commitSHA, "", "")
}

func (g *Client) SyncWithToken(ctx context.Context, repoDir, commitSHA, repoURL, token string) error {
	if err := g.FetchWithToken(ctx, repoDir, repoURL, token); err != nil {
		return err
	}

	if err := g.ResetHard(ctx, repoDir, commitSHA); err != nil {
		g.logger.Warn("Reset failed, attempting unshallow fetch", "commit", commitSHA, "error", err)
		if unshallowErr := g.unshallowFetch(ctx, repoDir, repoURL, token); unshallowErr != nil {
			return fmt.Errorf("git reset failed: %w (unshallow also failed: %v)", err, unshallowErr)
		}

		if retryErr := g.ResetHard(ctx, repoDir, commitSHA); retryErr != nil {
			return retryErr
		}
	}

	return nil
}

func (g *Client) unshallowFetch(ctx context.Context, repoDir, repoURL, token string) error {
	g.logger.Info("Performing unshallow fetch", "dir", repoDir)
	g.executor.SetWorkDir(repoDir)

	if token != "" && repoURL != "" {
		authenticatedURL, err := InjectTokenIntoURL(repoURL, token)
		if err != nil {
			return fmt.Errorf("failed to create authenticated URL: %w", err)
		}
		_, _ = g.executor.Run(ctx, "git", "remote", "set-url", "origin", authenticatedURL)
		defer func() {
			_, _ = g.executor.Run(ctx, "git", "remote", "set-url", "origin", repoURL)
		}()
	}

	_, err := g.executor.Run(ctx, "git", "fetch", "--unshallow", "origin")
	if err != nil {
		_, err = g.executor.Run(ctx, "git", "fetch", "origin")
	}
	return err
}

func (g *Client) GetBranch(ctx context.Context, repoDir string) (string, error) {
	g.executor.SetWorkDir(repoDir)

	result, err := g.executor.Run(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get branch: %w", err)
	}

	return strings.TrimSpace(result.Stdout), nil
}

func (g *Client) CheckoutBranch(ctx context.Context, repoDir, branch string) error {
	g.logger.Info("Checking out branch", "dir", repoDir, "branch", branch)

	g.executor.SetWorkDir(repoDir)

	_, err := g.executor.Run(ctx, "git", "checkout", branch)
	if err != nil {
		return fmt.Errorf("git checkout failed: %w", err)
	}

	return nil
}
