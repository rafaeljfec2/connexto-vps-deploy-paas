package engine

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/config"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/service"
	"github.com/paasdeploy/shared/pkg/compose"
	"github.com/paasdeploy/shared/pkg/docker"
	"github.com/paasdeploy/shared/pkg/git"
	"github.com/paasdeploy/shared/pkg/health"
	"github.com/paasdeploy/shared/pkg/lock"
)

type Engine struct {
	cfg              *config.Config
	db               *sql.DB
	dispatcher       *Dispatcher
	notifier         *ChannelNotifier
	healthMonitor    *HealthMonitor
	statsMonitor     *StatsMonitor
	docker           *docker.Client
	locker           *lock.Locker
	workers          []*Worker
	logger           *slog.Logger
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	running          bool
	mu               sync.Mutex
	customDomainRepo domain.CustomDomainRepository
	envVarRepo       domain.EnvVarRepository
}

type Params struct {
	Cfg              *config.Config
	DB               *sql.DB
	AppRepo          domain.AppRepository
	EnvVarRepo       domain.EnvVarRepository
	CustomDomainRepo domain.CustomDomainRepository
	ServerRepo       domain.ServerRepository
	AgentClient      *agentclient.AgentClient
	GitTokenProvider GitTokenProvider
	AuditService     *service.AuditService
	Logger           *slog.Logger
}

func New(p Params) *Engine {
	ctx, cancel := context.WithCancel(context.Background())

	queue := NewQueue(p.DB)
	lk := lock.New(p.Cfg.Deploy.DataDir)
	notifier := NewChannelNotifier(1000)
	dispatcher := NewDispatcher(queue, lk, p.Logger)
	dockerClient := docker.NewClient(p.Cfg.Deploy.DataDir, p.Cfg.Docker.Registry, p.Logger)
	healthMonitor := NewHealthMonitor(dockerClient, p.AppRepo, notifier, p.Logger)
	statsMonitor := NewStatsMonitor(dockerClient, p.AppRepo, notifier, p.Logger)

	engine := &Engine{
		cfg:              p.Cfg,
		db:               p.DB,
		dispatcher:       dispatcher,
		notifier:         notifier,
		healthMonitor:    healthMonitor,
		statsMonitor:     statsMonitor,
		docker:           dockerClient,
		locker:           lk,
		logger:           p.Logger.With("component", "engine"),
		ctx:              ctx,
		cancel:           cancel,
		customDomainRepo: p.CustomDomainRepo,
		envVarRepo:       p.EnvVarRepo,
	}

	deps := WorkerDeps{
		Git:              git.NewClient(p.Cfg.Deploy.DataDir, p.Logger),
		Docker:           dockerClient,
		Health:           health.NewChecker(p.Cfg.Deploy.HealthCheckTimeout, p.Cfg.Deploy.HealthCheckRetries, 5*time.Second, p.Logger),
		Notifier:         notifier,
		Dispatcher:       dispatcher,
		EnvVarRepo:       p.EnvVarRepo,
		CustomDomainRepo: p.CustomDomainRepo,
		ServerRepo:       p.ServerRepo,
		AgentClient:      p.AgentClient,
		AgentPort:        p.Cfg.GRPC.AgentPort,
		GitTokenProvider: p.GitTokenProvider,
		AuditService:     p.AuditService,
		Logger:           p.Logger,
	}

	for i := 0; i < p.Cfg.Deploy.Workers; i++ {
		worker := NewWorker(i, p.Cfg.Deploy.DataDir, deps)
		engine.workers = append(engine.workers, worker)
	}

	return engine
}

func (e *Engine) Start() error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return nil
	}
	e.running = true
	e.mu.Unlock()

	e.logger.Info("Starting deploy engine", "workers", len(e.workers))

	e.recoverFromRestart()

	if err := e.docker.EnsureNetwork(e.ctx, docker.DefaultNetworkName); err != nil {
		e.logger.Error("Failed to ensure network", "network", docker.DefaultNetworkName, "error", err)
	}

	if containerID, err := e.docker.GetCurrentContainerID(e.ctx); err == nil {
		if err := e.docker.ConnectToNetwork(e.ctx, containerID, docker.DefaultNetworkName); err != nil {
			e.logger.Warn("Failed to connect to network (may already be connected)", "network", docker.DefaultNetworkName, "error", err)
		}
	} else {
		e.logger.Debug("Not running in container or could not detect container ID", "error", err)
	}

	e.healthMonitor.Start(e.ctx)
	e.statsMonitor.Start(e.ctx)

	for _, worker := range e.workers {
		e.wg.Add(1)
		go e.runWorkerLoop(worker)
	}

	return nil
}

func (e *Engine) Stop() {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return
	}
	e.running = false
	e.mu.Unlock()

	e.logger.Info("Stopping deploy engine...")
	e.healthMonitor.Stop()
	e.statsMonitor.Stop()
	e.cancel()
	e.wg.Wait()
	e.notifier.Close()
	e.logger.Info("Deploy engine stopped")
}

func (e *Engine) recoverFromRestart() {
	e.logger.Info("Recovering from restart, cleaning up stale state...")

	if err := e.locker.CleanupStale(); err != nil {
		e.logger.Error("Failed to cleanup stale locks", "error", err)
	} else {
		e.logger.Info("Cleaned up stale locks")
	}

	query := `
		UPDATE deployments 
		SET status = 'failed', 
		    error_message = 'Deployment interrupted by server restart', 
		    finished_at = NOW() 
		WHERE status = 'running'
	`
	result, err := e.db.Exec(query)
	if err != nil {
		e.logger.Error("Failed to reset running deployments", "error", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		e.logger.Info("Reset interrupted deployments", "count", rowsAffected)
	}
}

func (e *Engine) runWorkerLoop(worker *Worker) {
	defer e.wg.Done()

	e.logger.Info("Worker started", "workerId", worker.id)

	for {
		select {
		case <-e.ctx.Done():
			e.logger.Info("Worker shutting down", "workerId", worker.id)
			return
		default:
		}

		deploy, app, err := e.dispatcher.Next(e.ctx)
		if err != nil {
			e.logger.Error("Failed to get next deployment", "error", err, "workerId", worker.id)
			time.Sleep(e.dispatcher.PollTime())
			continue
		}

		if deploy == nil {
			time.Sleep(e.dispatcher.PollTime())
			continue
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					e.logger.Error("Worker panic recovered", "error", r, "workerId", worker.id)
					e.dispatcher.MarkFailed(deploy.ID, "worker panic")
				}
				e.dispatcher.Release(app.ID)
			}()

			ctx, cancel := context.WithTimeout(e.ctx, e.cfg.Deploy.Timeout)
			defer cancel()

			if err := worker.Run(ctx, deploy, app); err != nil {
				e.logger.Error("Deployment failed",
					"deployId", deploy.ID,
					"appId", app.ID,
					"error", err,
					"workerId", worker.id,
				)
			}
		}()
	}
}

func (e *Engine) Events() <-chan DeployEvent {
	return e.notifier.Events()
}

func (e *Engine) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

func (e *Engine) WorkerCount() int {
	return len(e.workers)
}

func (e *Engine) Notifier() Notifier {
	return e.notifier
}

func (e *Engine) GetAppHealth(ctx context.Context, appName string) *docker.ContainerHealth {
	return e.healthMonitor.CheckApp(ctx, appName)
}

func (e *Engine) RestartContainer(ctx context.Context, containerName string) error {
	return e.docker.RestartContainer(ctx, containerName)
}

func (e *Engine) StopContainer(ctx context.Context, containerName string) error {
	return e.docker.StopContainer(ctx, containerName)
}

func (e *Engine) StartContainer(ctx context.Context, containerName string) error {
	return e.docker.StartContainer(ctx, containerName)
}

func (e *Engine) ContainerLogs(ctx context.Context, containerName string, tail int) (string, error) {
	return e.docker.ContainerLogs(ctx, containerName, tail)
}

func (e *Engine) StreamContainerLogs(ctx context.Context, containerName string, output chan<- string) error {
	return e.docker.StreamContainerLogs(ctx, containerName, output)
}

func (e *Engine) ContainerStats(ctx context.Context, containerName string) (*docker.ContainerStats, error) {
	return e.docker.ContainerStats(ctx, containerName)
}

func (e *Engine) Docker() *docker.Client {
	return e.docker
}

func (e *Engine) UpdateContainerDomains(ctx context.Context, app *domain.App) error {
	if app == nil {
		return fmt.Errorf("app is required")
	}
	e.logger.Info("Updating container domains", "app_id", app.ID, "app_name", app.Name)

	appDir := e.resolveAppDir(app)

	deployConfig, err := compose.LoadConfig(appDir)
	if err != nil {
		return err
	}

	currentImage, err := e.getCurrentContainerImage(ctx, app.Name)
	if err != nil {
		return err
	}

	allDomains := e.collectAllDomains(ctx, app.ID, deployConfig.Domains)
	envVars := e.collectEnvVars(app.ID)

	params := compose.GenerateParams{
		AppName:  app.Name,
		ImageTag: currentImage,
		Config:   deployConfig,
		Domains:  allDomains,
		EnvVars:  envVars,
	}

	if err := e.writeAndApplyCompose(ctx, appDir, app.ID, compose.GenerateContent(params)); err != nil {
		return err
	}

	e.logger.Info("Container updated with new domains", "app_id", app.ID, "domains", allDomains)
	return nil
}

func (e *Engine) resolveAppDir(app *domain.App) string {
	repoDir := filepath.Join(e.cfg.Deploy.DataDir, app.ID)
	if app.Workdir == "" || app.Workdir == "." {
		return repoDir
	}
	return filepath.Join(repoDir, app.Workdir)
}

func (e *Engine) getCurrentContainerImage(ctx context.Context, containerName string) (string, error) {
	containerHealth, err := e.docker.InspectContainer(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}
	if containerHealth == nil {
		return "", fmt.Errorf("container not found - app may not have been deployed yet")
	}
	return containerHealth.Image, nil
}

func (e *Engine) collectAllDomains(ctx context.Context, appID string, configDomains []string) []compose.DomainRoute {
	var domainRoutes []compose.DomainRoute
	for _, d := range configDomains {
		domainRoutes = append(domainRoutes, compose.DomainRoute{Domain: d})
	}

	if e.customDomainRepo == nil {
		return domainRoutes
	}

	customDomains, err := e.customDomainRepo.FindByAppID(ctx, appID)
	if err != nil {
		return domainRoutes
	}

	for _, d := range customDomains {
		domainRoutes = append(domainRoutes, compose.DomainRoute{
			Domain:     d.Domain,
			PathPrefix: d.PathPrefix,
		})
	}
	return domainRoutes
}

func (e *Engine) collectEnvVars(appID string) map[string]string {
	if e.envVarRepo == nil {
		return nil
	}

	vars, err := e.envVarRepo.FindByAppID(appID)
	if err != nil {
		return nil
	}

	envVars := make(map[string]string, len(vars))
	for _, v := range vars {
		envVars[v.Key] = v.Value
	}
	return envVars
}

func (e *Engine) writeAndApplyCompose(ctx context.Context, appDir, projectName, content string) error {
	composePath := filepath.Join(appDir, "docker-compose.yml")

	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write docker-compose.yml: %w", err)
	}

	if err := e.docker.ComposeUp(ctx, appDir, projectName, nil); err != nil {
		return fmt.Errorf("failed to update container: %w", err)
	}

	return nil
}

