package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/paasdeploy/backend/internal/domain"
)

const (
	defaultStatsInterval = 3 * time.Second
)

type StatsMonitor struct {
	docker   *DockerClient
	appRepo  domain.AppRepository
	notifier Notifier
	logger   *slog.Logger
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func NewStatsMonitor(
	docker *DockerClient,
	appRepo domain.AppRepository,
	notifier Notifier,
	logger *slog.Logger,
) *StatsMonitor {
	return &StatsMonitor{
		docker:   docker,
		appRepo:  appRepo,
		notifier: notifier,
		logger:   logger.With("component", "stats_monitor"),
		interval: defaultStatsInterval,
		stopCh:   make(chan struct{}),
	}
}

func (m *StatsMonitor) Start(ctx context.Context) {
	m.logger.Info("Starting stats monitor", "interval", m.interval)

	m.wg.Add(1)
	go m.run(ctx)
}

func (m *StatsMonitor) Stop() {
	m.logger.Info("Stopping stats monitor")
	close(m.stopCh)
	m.wg.Wait()
	m.logger.Info("Stats monitor stopped")
}

func (m *StatsMonitor) run(ctx context.Context) {
	defer m.wg.Done()

	m.collectAllStats(ctx)

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.collectAllStats(ctx)
		}
	}
}

func (m *StatsMonitor) collectAllStats(ctx context.Context) {
	apps, err := m.appRepo.FindAll()
	if err != nil {
		m.logger.Error("Failed to fetch apps for stats collection", "error", err)
		return
	}

	for _, app := range apps {
		if app.LastDeployedAt == nil {
			continue
		}

		stats, err := m.docker.ContainerStats(ctx, app.Name)
		if err != nil {
			m.logger.Debug("Failed to get container stats", "appName", app.Name, "error", err)
			continue
		}

		m.notifier.EmitStats(app.ID, StatsData{
			CPUPercent:    stats.CPUPercent,
			MemoryUsage:   stats.MemoryUsage,
			MemoryLimit:   stats.MemoryLimit,
			MemoryPercent: stats.MemoryPercent,
			NetworkRx:     stats.NetworkRx,
			NetworkTx:     stats.NetworkTx,
			PIDs:          stats.PIDs,
		})
	}
}
