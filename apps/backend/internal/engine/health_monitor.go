package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/shared/pkg/docker"
)

const (
	defaultMonitorInterval = 30 * time.Second
	dbFetchRetries         = 3
	dbFetchRetryDelay      = 2 * time.Second
)

type HealthMonitor struct {
	docker     *docker.Client
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
	dockerClient *docker.Client,
	appRepo domain.AppRepository,
	notifier Notifier,
	logger *slog.Logger,
) *HealthMonitor {
	return &HealthMonitor{
		docker:     dockerClient,
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
	var apps []domain.App
	var err error
	for attempt := 0; attempt < dbFetchRetries; attempt++ {
		apps, err = m.appRepo.FindAll()
		if err == nil {
			break
		}
		if attempt < dbFetchRetries-1 {
			m.logger.Warn("Retrying fetch apps after transient error", "attempt", attempt+1, "error", err)
			select {
			case <-ctx.Done():
				return
			case <-m.stopCh:
				return
			case <-time.After(dbFetchRetryDelay):
			}
		}
	}
	if err != nil {
		m.logger.Error("Failed to fetch apps for health check after retries", "error", err)
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

func (m *HealthMonitor) CheckApp(ctx context.Context, appName string) *docker.ContainerHealth {
	health, err := m.docker.InspectContainer(ctx, appName)
	if err != nil {
		m.logger.Debug("Failed to inspect container", "appName", appName, "error", err)
		return &docker.ContainerHealth{
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
