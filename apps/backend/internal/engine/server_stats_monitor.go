package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/domain"
)

const (
	defaultServerStatsInterval = 10 * time.Second
	serverStatsFetchTimeout    = 5 * time.Second
)

type ServerStatsMonitor struct {
	serverRepo  domain.ServerRepository
	agentClient *agentclient.AgentClient
	agentPort   int
	emitter     BroadcastEmitter
	logger      *slog.Logger
	interval    time.Duration
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

func NewServerStatsMonitor(
	serverRepo domain.ServerRepository,
	agentClient *agentclient.AgentClient,
	agentPort int,
	emitter BroadcastEmitter,
	logger *slog.Logger,
) *ServerStatsMonitor {
	return &ServerStatsMonitor{
		serverRepo:  serverRepo,
		agentClient: agentClient,
		agentPort:   agentPort,
		emitter:     emitter,
		logger:      logger.With("component", "server_stats_monitor"),
		interval:    defaultServerStatsInterval,
		stopCh:      make(chan struct{}),
	}
}

func (m *ServerStatsMonitor) Start(ctx context.Context) {
	if m.agentClient == nil || m.agentPort == 0 {
		m.logger.Info("Server stats monitor disabled: no agent client configured")
		return
	}
	m.logger.Info("Starting server stats monitor", "interval", m.interval)
	m.wg.Add(1)
	go m.run(ctx)
}

func (m *ServerStatsMonitor) Stop() {
	m.logger.Info("Stopping server stats monitor")
	close(m.stopCh)
	m.wg.Wait()
	m.logger.Info("Server stats monitor stopped")
}

func (m *ServerStatsMonitor) run(ctx context.Context) {
	defer m.wg.Done()

	m.collectAll(ctx)

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.collectAll(ctx)
		}
	}
}

func (m *ServerStatsMonitor) collectAll(ctx context.Context) {
	servers, err := m.serverRepo.FindAll()
	if err != nil {
		m.logger.Error("Failed to fetch servers for stats", "error", err)
		return
	}

	var wg sync.WaitGroup
	for i := range servers {
		srv := servers[i]
		if srv.Status != domain.ServerStatusOnline {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			m.collectServer(ctx, &srv)
		}()
	}
	wg.Wait()
}

func (m *ServerStatsMonitor) collectServer(ctx context.Context, srv *domain.Server) {
	fetchCtx, cancel := context.WithTimeout(ctx, serverStatsFetchTimeout)
	defer cancel()

	sysInfo, err := m.agentClient.GetSystemInfo(fetchCtx, srv.Host, m.agentPort)
	if err != nil {
		m.logger.Debug("Failed to get system info from server", "serverId", srv.ID, "error", err)
		return
	}

	sysMetrics, err := m.agentClient.GetSystemMetrics(fetchCtx, srv.Host, m.agentPort)
	if err != nil {
		m.logger.Debug("Failed to get system metrics from server", "serverId", srv.ID, "error", err)
		return
	}

	m.emitter.EmitServerStats(srv.ID, SystemStatsPayload{
		SystemInfo:    sysInfo,
		SystemMetrics: sysMetrics,
	})
}
