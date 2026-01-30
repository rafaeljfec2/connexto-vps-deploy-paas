package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/paasdeploy/backend/internal/domain"
)

const (
	defaultMonitorInterval = 30 * time.Second
)

type HealthMonitor struct {
	docker     *DockerClient
	appRepo    domain.AppRepository
	notifier   Notifier
	logger     *slog.Logger
	interval   time.Duration
	lastStatus map[string]string
	mu         sync.RWMutex
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

func NewHealthMonitor(
	docker *DockerClient,
	appRepo domain.AppRepository,
	notifier Notifier,
	logger *slog.Logger,
) *HealthMonitor {
	return &HealthMonitor{
		docker:     docker,
		appRepo:    appRepo,
		notifier:   notifier,
		logger:     logger.With("component", "health_monitor"),
		interval:   defaultMonitorInterval,
		lastStatus: make(map[string]string),
		stopCh:     make(chan struct{}),
	}
}

func (m *HealthMonitor) Start(ctx context.Context) {
	m.logger.Info("Starting health monitor", "interval", m.interval)

	m.wg.Add(1)
	go m.run(ctx)
}

func (m *HealthMonitor) Stop() {
	m.logger.Info("Stopping health monitor")
	close(m.stopCh)
	m.wg.Wait()
	m.logger.Info("Health monitor stopped")
}

func (m *HealthMonitor) run(ctx context.Context) {
	defer m.wg.Done()

	m.checkAllApps(ctx)

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkAllApps(ctx)
		}
	}
}

func (m *HealthMonitor) checkAllApps(ctx context.Context) {
	apps, err := m.appRepo.FindAll()
	if err != nil {
		m.logger.Error("Failed to fetch apps for health check", "error", err)
		return
	}

	for _, app := range apps {
		if app.LastDeployedAt == nil {
			continue
		}

		health := m.CheckApp(ctx, app.Name)
		if health == nil {
			continue
		}

		statusKey := health.Status + "|" + health.Health
		m.mu.RLock()
		lastKey := m.lastStatus[app.ID]
		m.mu.RUnlock()

		if statusKey != lastKey {
			m.logger.Info("App health changed",
				"appId", app.ID,
				"appName", app.Name,
				"status", health.Status,
				"health", health.Health,
			)

			m.notifier.EmitHealth(app.ID, HealthStatus{
				Status:    health.Status,
				Health:    health.Health,
				StartedAt: health.StartedAt,
				Uptime:    health.Uptime,
			})

			m.mu.Lock()
			m.lastStatus[app.ID] = statusKey
			m.mu.Unlock()
		}
	}
}

func (m *HealthMonitor) CheckApp(ctx context.Context, appName string) *ContainerHealth {
	health, err := m.docker.InspectContainer(ctx, appName)
	if err != nil {
		m.logger.Debug("Failed to inspect container", "appName", appName, "error", err)
		return &ContainerHealth{
			Name:   appName,
			Status: "not_deployed",
			Health: "none",
		}
	}

	return health
}

func (m *HealthMonitor) ClearAppStatus(appID string) {
	m.mu.Lock()
	delete(m.lastStatus, appID)
	m.mu.Unlock()
}
