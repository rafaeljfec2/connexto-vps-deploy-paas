package agent

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/paasdeploy/agent/internal/grpcclient"
	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
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
	if err := a.register(ctx); err != nil {
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

func (a *Agent) register(ctx context.Context) error {
	_, err := a.client.Register(ctx, &pb.RegisterRequest{
		AgentId:      a.cfg.ServerID,
		AgentVersion: "0.1.0",
		SystemInfo:   &pb.SystemInfo{},
		DockerInfo:   &pb.DockerInfo{},
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *Agent) heartbeat(ctx context.Context) error {
	_, err := a.client.Heartbeat(ctx, &pb.HeartbeatRequest{
		AgentId: a.cfg.ServerID,
		Status: &pb.AgentStatus{
			State:             pb.AgentState_AGENT_STATE_IDLE,
			ActiveDeployCount: 0,
			ContainerCount:    0,
		},
	})
	return err
}
