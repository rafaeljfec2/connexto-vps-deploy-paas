package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/paasdeploy/backend/internal/sysinfo"
)

const defaultSystemStatsInterval = 10 * time.Second

type SystemStatsMonitor struct {
	emitter  BroadcastEmitter
	logger   *slog.Logger
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func NewSystemStatsMonitor(emitter BroadcastEmitter, logger *slog.Logger) *SystemStatsMonitor {
	return &SystemStatsMonitor{
		emitter:  emitter,
		logger:   logger.With("component", "system_stats_monitor"),
		interval: defaultSystemStatsInterval,
		stopCh:   make(chan struct{}),
	}
}

func (m *SystemStatsMonitor) Start(ctx context.Context) {
	m.logger.Info("Starting system stats monitor", "interval", m.interval)
	m.wg.Add(1)
	go m.run(ctx)
}

func (m *SystemStatsMonitor) Stop() {
	m.logger.Info("Stopping system stats monitor")
	close(m.stopCh)
	m.wg.Wait()
	m.logger.Info("System stats monitor stopped")
}

func (m *SystemStatsMonitor) run(ctx context.Context) {
	defer m.wg.Done()

	m.collect()

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.collect()
		}
	}
}

func (m *SystemStatsMonitor) collect() {
	stats := sysinfo.GetStats()
	m.emitter.EmitSystemStats(SystemStatsPayload{
		SystemInfo:    stats.SystemInfo,
		SystemMetrics: stats.SystemMetrics,
	})
}
