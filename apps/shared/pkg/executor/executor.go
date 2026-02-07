package executor

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

type Executor struct {
	workDir string
	timeout time.Duration
	logger  *slog.Logger
}

func New(workDir string, timeout time.Duration, logger *slog.Logger) *Executor {
	return &Executor{
		workDir: workDir,
		timeout: timeout,
		logger:  logger,
	}
}

func (e *Executor) Run(ctx context.Context, name string, args ...string) (*Result, error) {
	return e.run(ctx, true, name, args...)
}

func (e *Executor) RunQuiet(ctx context.Context, name string, args ...string) (*Result, error) {
	return e.run(ctx, false, name, args...)
}

func (e *Executor) run(ctx context.Context, logErrors bool, name string, args ...string) (*Result, error) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = e.workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	e.logger.Debug("Executing command",
		"command", name,
		"args", args,
		"workDir", e.workDir,
	)

	err := cmd.Run()

	result := &Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(start),
	}

	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	if ctx.Err() == context.DeadlineExceeded {
		return result, fmt.Errorf("command timed out after %v", e.timeout)
	}

	if err != nil {
		if logErrors {
			e.logger.Error("Command failed",
				"command", name,
				"args", args,
				"exitCode", result.ExitCode,
				"stderr", result.Stderr,
				"duration", result.Duration,
			)
		}
		return result, fmt.Errorf("command failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	e.logger.Debug("Command completed",
		"command", name,
		"exitCode", result.ExitCode,
		"duration", result.Duration,
	)

	return result, nil
}

func (e *Executor) RunWithStreaming(ctx context.Context, output chan<- string, name string, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = e.workDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	e.logger.Debug("Executing command with streaming",
		"command", name,
		"args", args,
		"workDir", e.workDir,
	)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		streamOutput(ctx, stdout, output)
	}()

	go func() {
		defer wg.Done()
		streamOutput(ctx, stderr, output)
	}()

	done := make(chan error)
	go func() {
		wg.Wait()
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		wg.Wait()
		e.logger.Error("Command timed out",
			"command", name,
			"args", args,
			"timeout", e.timeout,
		)
		return fmt.Errorf("command timed out after %v", e.timeout)
	case err := <-done:
		if err != nil {
			e.logger.Error("Command failed",
				"command", name,
				"args", args,
				"error", err,
			)
			return fmt.Errorf("command failed: %w", err)
		}
		return nil
	}
}

func streamOutput(ctx context.Context, reader io.Reader, output chan<- string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		line := scanner.Text()
		select {
		case output <- line:
		case <-ctx.Done():
			return
		default:
		}
	}
}

func (e *Executor) SetWorkDir(workDir string) {
	e.workDir = workDir
}

func (e *Executor) SetTimeout(timeout time.Duration) {
	e.timeout = timeout
}

func SanitizePath(path string) string {
	path = strings.ReplaceAll(path, "..", "")
	path = strings.ReplaceAll(path, "~", "")
	path = strings.TrimPrefix(path, "/")

	dangerous := []string{";", "&", "|", "$", "`", "(", ")", "{", "}", "[", "]", "<", ">", "\\", "\n", "\r"}
	for _, char := range dangerous {
		path = strings.ReplaceAll(path, char, "")
	}

	return path
}
