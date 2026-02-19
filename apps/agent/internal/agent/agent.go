package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/paasdeploy/agent/internal/grpcclient"
	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"golang.org/x/sys/unix"
)

const (
	Version           = "0.3.1"
	selfUpdateTimeout = 2 * time.Minute
	tempBinaryName    = "agent.new"
)

type Config struct {
	ServerAddr string
	ServerID   string
	CACertPath string
	CertPath   string
	KeyPath    string
}

type Agent struct {
	cfg    Config
	client *grpcclient.Client
	logger *slog.Logger
}

func New(cfg Config, logger *slog.Logger) (*Agent, error) {
	if cfg.ServerAddr == "" || cfg.ServerID == "" || cfg.CACertPath == "" || cfg.CertPath == "" || cfg.KeyPath == "" {
		return nil, errors.New("missing required config")
	}

	client, err := grpcclient.New(cfg.ServerAddr, cfg.CACertPath, cfg.CertPath, cfg.KeyPath)
	if err != nil {
		return nil, err
	}

	return &Agent{
		cfg:    cfg,
		client: client,
		logger: logger.With("component", "agent"),
	}, nil
}

func (a *Agent) Run(ctx context.Context) error {
	if err := a.registerWithRetry(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = a.client.Close()
			return ctx.Err()
		case <-ticker.C:
			if err := a.heartbeat(ctx); err != nil {
				a.logger.Error("heartbeat failed", "error", err)
			}
		}
	}
}

func (a *Agent) registerWithRetry(ctx context.Context) error {
	const maxBackoff = 30 * time.Second
	backoff := time.Second

	for {
		err := a.register(ctx)
		if err == nil {
			a.logger.Info("registered with backend")
			return nil
		}

		a.logger.Warn("register failed, retrying", "error", err, "retryIn", backoff)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (a *Agent) register(ctx context.Context) error {
	_, err := a.client.Register(ctx, &pb.RegisterRequest{
		AgentId:      a.cfg.ServerID,
		AgentVersion: Version,
		SystemInfo:   &pb.SystemInfo{},
		DockerInfo:   &pb.DockerInfo{},
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *Agent) heartbeat(ctx context.Context) error {
	resp, err := a.client.Heartbeat(ctx, &pb.HeartbeatRequest{
		AgentId:      a.cfg.ServerID,
		AgentVersion: Version,
		Status: &pb.AgentStatus{
			State:             pb.AgentState_AGENT_STATE_IDLE,
			ActiveDeployCount: 0,
			ContainerCount:    0,
		},
	})
	if err != nil {
		return err
	}
	a.processCommands(resp.GetCommands())
	return nil
}

func (a *Agent) processCommands(commands []*pb.AgentCommand) {
	for _, cmd := range commands {
		if cmd == nil {
			continue
		}
		switch cmd.Type {
		case pb.AgentCommandType_AGENT_COMMAND_UPDATE_AGENT:
			a.handleUpdateAgent(cmd.GetPayload())
		default:
			a.logger.Debug("ignoring unknown command", "type", cmd.Type)
		}
	}
}

func (a *Agent) handleUpdateAgent(payload string) {
	if payload == "" {
		a.logger.Info("update agent requested but no download URL")
		return
	}
	a.logger.Info("update agent requested", "url", payload)
	go a.runSelfUpdate(payload)
}

func (a *Agent) runSelfUpdate(downloadURL string) {
	a.logger.Info("starting self-update", "url", downloadURL)
	execPath, err := os.Executable()
	if err != nil {
		a.logger.Error("failed to get executable path", "error", err)
		return
	}
	a.logger.Info("current binary path", "path", execPath)

	if err := a.downloadAndReplace(downloadURL, execPath); err != nil {
		a.logger.Error("self-update failed", "error", err)
		return
	}
	a.logger.Info("self-update complete, restarting", "path", execPath)
	if err := unix.Exec(execPath, os.Args, os.Environ()); err != nil {
		a.logger.Error("exec failed", "error", err)
	}
}

func (a *Agent) downloadAndReplace(downloadURL, execPath string) error {
	dir := filepath.Dir(execPath)
	newPath := filepath.Join(dir, tempBinaryName)

	if err := a.downloadToFile(downloadURL, newPath); err != nil {
		return err
	}
	return a.replaceBinary(execPath, newPath)
}

func (a *Agent) downloadToFile(downloadURL, destPath string) error {
	a.logger.Info("downloading binary", "dest", destPath)
	client := &http.Client{Timeout: selfUpdateTimeout}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "PaasDeploy-Agent/"+Version)
	resp, err := client.Do(req)
	if err != nil {
		a.logger.Error("download request failed", "error", err.Error())
		return fmt.Errorf("download request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		a.logger.Error("download returned non-200", "status", resp.StatusCode, "body", string(body))
		return fmt.Errorf("download status: %d", resp.StatusCode)
	}
	a.logger.Info("download response received", "status", resp.StatusCode, "contentLength", resp.ContentLength)

	f, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	written, err := io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(destPath)
		return fmt.Errorf("write binary: %w", err)
	}
	a.logger.Info("binary downloaded", "bytes", written)
	if written == 0 {
		os.Remove(destPath)
		return fmt.Errorf("empty binary")
	}
	if err := os.Chmod(destPath, 0o755); err != nil {
		os.Remove(destPath)
		return fmt.Errorf("chmod: %w", err)
	}
	return nil
}

func (a *Agent) replaceBinary(execPath, newPath string) error {
	if err := os.Remove(execPath); err != nil {
		os.Remove(newPath)
		return fmt.Errorf("remove current binary: %w", err)
	}
	if err := os.Rename(newPath, execPath); err != nil {
		return fmt.Errorf("rename new binary: %w", err)
	}
	return nil
}
