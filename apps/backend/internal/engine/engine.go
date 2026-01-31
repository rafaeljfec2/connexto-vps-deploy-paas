package engine

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"

	"github.com/paasdeploy/backend/internal/config"
	"github.com/paasdeploy/backend/internal/domain"
)

type Engine struct {
	cfg           *config.Config
	db            *sql.DB
	dispatcher    *Dispatcher
	notifier      *ChannelNotifier
	healthMonitor *HealthMonitor
	statsMonitor  *StatsMonitor
	docker        *DockerClient
	workers       []*Worker
	logger        *slog.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	running       bool
	mu            sync.Mutex
}

func New(cfg *config.Config, db *sql.DB, appRepo domain.AppRepository, envVarRepo domain.EnvVarRepository, logger *slog.Logger) *Engine {
	ctx, cancel := context.WithCancel(context.Background())

	queue := NewQueue(db)
	locker := NewLocker(cfg.Deploy.DataDir)
	notifier := NewChannelNotifier(1000)
	dispatcher := NewDispatcher(queue, locker, logger)
	docker := NewDockerClient(cfg.Deploy.DataDir, cfg.Docker.Registry, logger)
	healthMonitor := NewHealthMonitor(docker, appRepo, notifier, logger)
	statsMonitor := NewStatsMonitor(docker, appRepo, notifier, logger)

	engine := &Engine{
		cfg:           cfg,
		db:            db,
		dispatcher:    dispatcher,
		notifier:      notifier,
		healthMonitor: healthMonitor,
		statsMonitor:  statsMonitor,
		docker:        docker,
		logger:        logger.With("component", "engine"),
		ctx:           ctx,
		cancel:        cancel,
	}

	deps := WorkerDeps{
		Git:        NewGitClient(cfg.Deploy.DataDir, logger),
		Docker:     docker,
		Health:     NewHealthChecker(cfg.Deploy.HealthCheckTimeout, cfg.Deploy.HealthCheckRetries, 5*time.Second, logger),
		Notifier:   notifier,
		Dispatcher: dispatcher,
		EnvVarRepo: envVarRepo,
		Logger:     logger,
	}

	for i := 0; i < cfg.Deploy.Workers; i++ {
		worker := NewWorker(i, cfg.Deploy.DataDir, deps)
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
