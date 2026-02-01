package engine

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/paasdeploy/backend/internal/config"
	"github.com/paasdeploy/backend/internal/domain"
)

type Engine struct {
	cfg              *config.Config
	db               *sql.DB
	dispatcher       *Dispatcher
	notifier         *ChannelNotifier
	healthMonitor    *HealthMonitor
	statsMonitor     *StatsMonitor
	docker           *DockerClient
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

func New(cfg *config.Config, db *sql.DB, appRepo domain.AppRepository, envVarRepo domain.EnvVarRepository, customDomainRepo domain.CustomDomainRepository, logger *slog.Logger) *Engine {
	ctx, cancel := context.WithCancel(context.Background())

	queue := NewQueue(db)
	locker := NewLocker(cfg.Deploy.DataDir)
	notifier := NewChannelNotifier(1000)
	dispatcher := NewDispatcher(queue, locker, logger)
	docker := NewDockerClient(cfg.Deploy.DataDir, cfg.Docker.Registry, logger)
	healthMonitor := NewHealthMonitor(docker, appRepo, notifier, logger)
	statsMonitor := NewStatsMonitor(docker, appRepo, notifier, logger)

	engine := &Engine{
		cfg:              cfg,
		db:               db,
		dispatcher:       dispatcher,
		notifier:         notifier,
		healthMonitor:    healthMonitor,
		statsMonitor:     statsMonitor,
		docker:           docker,
		logger:           logger.With("component", "engine"),
		ctx:              ctx,
		cancel:           cancel,
		customDomainRepo: customDomainRepo,
		envVarRepo:       envVarRepo,
	}

	deps := WorkerDeps{
		Git:              NewGitClient(cfg.Deploy.DataDir, logger),
		Docker:           docker,
		Health:           NewHealthChecker(cfg.Deploy.HealthCheckTimeout, cfg.Deploy.HealthCheckRetries, 5*time.Second, logger),
		Notifier:         notifier,
		Dispatcher:       dispatcher,
		EnvVarRepo:       envVarRepo,
		CustomDomainRepo: customDomainRepo,
		Logger:           logger,
	}

	for i := 0; i < cfg.Deploy.Workers; i++ {
		worker := NewWorker(i, cfg.Deploy.DataDir, deps)
		engine.workers = append(engine.workers, worker)
	}

	return engine
}

const defaultNetworkName = "paasdeploy"

func (e *Engine) Start() error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return nil
	}
	e.running = true
	e.mu.Unlock()

	e.logger.Info("Starting deploy engine", "workers", len(e.workers))

	if err := e.docker.EnsureNetwork(e.ctx, defaultNetworkName); err != nil {
		e.logger.Error("Failed to ensure network", "network", defaultNetworkName, "error", err)
	}

	if containerID, err := e.docker.GetCurrentContainerID(e.ctx); err == nil {
		if err := e.docker.ConnectToNetwork(e.ctx, containerID, defaultNetworkName); err != nil {
			e.logger.Warn("Failed to connect to network (may already be connected)", "network", defaultNetworkName, "error", err)
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

func (e *Engine) GetAppHealth(ctx context.Context, appName string) *ContainerHealth {
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

func (e *Engine) ContainerStats(ctx context.Context, containerName string) (*ContainerStats, error) {
	return e.docker.ContainerStats(ctx, containerName)
}

func (e *Engine) Docker() *DockerClient {
	return e.docker
}

func (e *Engine) UpdateContainerDomains(ctx context.Context, app *domain.App) error {
	e.logger.Info("Updating container domains", "app_id", app.ID, "app_name", app.Name)

	appDir := e.resolveAppDir(app)

	deployConfig, err := e.loadDeployConfig(appDir)
	if err != nil {
		return err
	}

	currentImage, err := e.getCurrentContainerImage(ctx, app.Name)
	if err != nil {
		return err
	}

	allDomains := e.collectAllDomains(ctx, app.ID, deployConfig.Domains)
	envVars := e.collectEnvVars(app.ID)

	composeContent := e.generateComposeContent(app.Name, currentImage, deployConfig, allDomains, envVars)

	if err := e.writeAndApplyCompose(ctx, appDir, composeContent); err != nil {
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

func (e *Engine) loadDeployConfig(appDir string) (*PaasDeployConfig, error) {
	configPath := filepath.Join(appDir, "paasdeploy.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("paasdeploy.json not found - app may not have been deployed yet")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read paasdeploy.json: %w", err)
	}

	var config PaasDeployConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse paasdeploy.json: %w", err)
	}

	return &config, nil
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

func (e *Engine) collectAllDomains(ctx context.Context, appID string, configDomains []string) []string {
	allDomains := make([]string, len(configDomains))
	copy(allDomains, configDomains)

	if e.customDomainRepo == nil {
		return allDomains
	}

	customDomains, err := e.customDomainRepo.FindByAppID(ctx, appID)
	if err != nil {
		return allDomains
	}

	for _, d := range customDomains {
		allDomains = append(allDomains, d.Domain)
	}
	return allDomains
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

func (e *Engine) generateComposeContent(appName, image string, cfg *PaasDeployConfig, domains []string, envVars map[string]string) string {
	envYAML := e.buildEnvVarsYAML(cfg, envVars)
	labels := buildLabelsYAML(appName, domains, cfg.Port)
	portMapping := buildPortMapping(cfg.HostPort, cfg.Port)

	return fmt.Sprintf("services:\n"+
		"  %s:\n"+
		"    image: %s\n"+
		"    container_name: %s\n"+
		"    restart: unless-stopped\n"+
		"    ports:\n"+
		"      - \"%s\"\n"+
		"%s"+
		"%s"+
		"    healthcheck:\n"+
		"      test: [\"CMD\", \"wget\", \"-q\", \"--spider\", \"http://127.0.0.1:%d%s\"]\n"+
		"      interval: %s\n"+
		"      timeout: %s\n"+
		"      retries: %d\n"+
		"      start_period: %s\n"+
		"    deploy:\n"+
		"      resources:\n"+
		"        limits:\n"+
		"          memory: %s\n"+
		"          cpus: '%s'\n"+
		"    networks:\n"+
		"      - paasdeploy\n\n"+
		"networks:\n"+
		"  paasdeploy:\n"+
		"    external: true\n",
		appName, image, appName, portMapping, envYAML, labels,
		cfg.Port, cfg.Healthcheck.Path,
		cfg.Healthcheck.Interval, cfg.Healthcheck.Timeout,
		cfg.Healthcheck.Retries, cfg.Healthcheck.StartPeriod,
		cfg.Resources.Memory, cfg.Resources.CPU,
	)
}

func (e *Engine) writeAndApplyCompose(ctx context.Context, appDir, content string) error {
	composePath := filepath.Join(appDir, "docker-compose.yml")

	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write docker-compose.yml: %w", err)
	}

	if err := e.docker.ComposeUp(ctx, appDir, nil); err != nil {
		return fmt.Errorf("failed to update container: %w", err)
	}

	return nil
}

func (e *Engine) buildEnvVarsYAML(cfg *PaasDeployConfig, appEnvVars map[string]string) string {
	allEnvVars := make(map[string]string)

	for k, v := range cfg.Env {
		allEnvVars[k] = v
	}

	for k, v := range appEnvVars {
		allEnvVars[k] = v
	}

	if len(allEnvVars) == 0 {
		return ""
	}

	envVars := "    environment:\n"
	for k, v := range allEnvVars {
		envVars += fmt.Sprintf("      - %s=%s\n", k, v)
	}
	return envVars
}
