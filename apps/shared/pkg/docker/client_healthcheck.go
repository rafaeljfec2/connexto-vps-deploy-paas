package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	healthcheckCMD       = "CMD"
	healthcheckCMDShell  = "CMD-SHELL"
	healthcheckNone      = "NONE"
	defaultHealthTimeout = 30 * time.Second
)

type HealthcheckResult struct {
	ExitCode   int      `json:"exitCode"`
	Stdout     string   `json:"stdout"`
	Stderr     string   `json:"stderr"`
	Command    []string `json:"command"`
	DurationMs int64    `json:"durationMs"`
}

type HealthcheckNotConfiguredError struct {
	ContainerID string
}

func (e *HealthcheckNotConfiguredError) Error() string {
	return fmt.Sprintf("container %s does not have a healthcheck configured", e.ContainerID)
}

type containerHealthcheckConfig struct {
	Test    []string `json:"Test"`
	Timeout int64    `json:"Timeout"`
}

func (d *Client) RunHealthcheck(ctx context.Context, containerID string) (*HealthcheckResult, error) {
	cfg, err := d.inspectHealthcheckConfig(ctx, containerID)
	if err != nil {
		return nil, err
	}

	if !healthcheckIsConfigured(cfg.Test) {
		return nil, &HealthcheckNotConfiguredError{ContainerID: containerID}
	}

	execArgs, err := buildHealthcheckExecArgs(containerID, cfg.Test)
	if err != nil {
		return nil, err
	}

	timeout := healthcheckTimeout(cfg.Timeout)

	d.logger.Info("Running container healthcheck",
		"container", containerID,
		"command", cfg.Test,
		"timeout", timeout,
	)

	result, runErr := d.executor.RunQuietWithTimeout(ctx, timeout, "docker", execArgs...)
	if result == nil {
		return nil, fmt.Errorf("failed to execute healthcheck: %w", runErr)
	}

	stderr := result.Stderr
	exitCode := result.ExitCode
	if isHealthcheckTimeout(runErr) {
		stderr = appendTimeoutMessage(stderr, timeout)
		if exitCode == 0 {
			exitCode = -1
		}
	}

	return &HealthcheckResult{
		ExitCode:   exitCode,
		Stdout:     result.Stdout,
		Stderr:     stderr,
		Command:    healthcheckUserCommand(cfg.Test),
		DurationMs: result.Duration.Milliseconds(),
	}, nil
}

func appendTimeoutMessage(stderr string, timeout time.Duration) string {
	timeoutMsg := fmt.Sprintf("healthcheck timed out after %s", timeout)
	if stderr == "" {
		return timeoutMsg
	}
	return stderr + "\n" + timeoutMsg
}

func isHealthcheckTimeout(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "timed out")
}

func (d *Client) inspectHealthcheckConfig(ctx context.Context, containerID string) (*containerHealthcheckConfig, error) {
	result, err := d.executor.RunQuietWithTimeout(ctx, 30*time.Second,
		"docker", "inspect", formatFlag, "{{json .Config.Healthcheck}}", containerID)
	if err != nil {
		if result != nil && strings.Contains(strings.ToLower(result.Stderr), errNoSuchContainer) {
			return nil, fmt.Errorf("container %s not found", containerID)
		}
		return nil, fmt.Errorf("failed to inspect container healthcheck: %w", err)
	}

	raw := strings.TrimSpace(result.Stdout)
	if raw == "" || raw == "null" {
		return &containerHealthcheckConfig{}, nil
	}

	var cfg containerHealthcheckConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse healthcheck config: %w", err)
	}

	return &cfg, nil
}

func healthcheckIsConfigured(test []string) bool {
	if len(test) == 0 {
		return false
	}
	if len(test) == 1 && strings.EqualFold(test[0], healthcheckNone) {
		return false
	}
	return true
}

func buildHealthcheckExecArgs(containerID string, test []string) ([]string, error) {
	if len(test) == 0 {
		return nil, fmt.Errorf("empty healthcheck command")
	}

	mode := test[0]
	args := []string{"exec", containerID}

	switch {
	case strings.EqualFold(mode, healthcheckCMD):
		if len(test) < 2 {
			return nil, fmt.Errorf("CMD healthcheck requires at least one argument")
		}
		args = append(args, test[1:]...)
	case strings.EqualFold(mode, healthcheckCMDShell):
		if len(test) < 2 {
			return nil, fmt.Errorf("CMD-SHELL healthcheck requires a script")
		}
		args = append(args, "sh", "-c", test[1])
	default:
		args = append(args, test...)
	}

	return args, nil
}

func healthcheckUserCommand(test []string) []string {
	if len(test) == 0 {
		return nil
	}
	mode := test[0]
	if strings.EqualFold(mode, healthcheckCMD) || strings.EqualFold(mode, healthcheckCMDShell) {
		return test[1:]
	}
	return test
}

func healthcheckTimeout(nanos int64) time.Duration {
	if nanos <= 0 {
		return defaultHealthTimeout
	}
	return time.Duration(nanos)
}
