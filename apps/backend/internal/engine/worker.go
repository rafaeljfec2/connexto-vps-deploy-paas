package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/paasdeploy/backend/internal/domain"
)

const (
	outputChannelBuffer   = 100
	healthCheckStartDelay = 5 * time.Second
	defaultAppPort        = 8080
)

type PaasDeployConfig struct {
	Name   string `json:"name"`
	Build  struct {
		Type       string `json:"type"`
		Dockerfile string `json:"dockerfile"`
		Context    string `json:"context"`
	} `json:"build"`
	Healthcheck struct {
		Path     string `json:"path"`
		Interval string `json:"interval"`
		Timeout  string `json:"timeout"`
	} `json:"healthcheck"`
	Port int `json:"port"`
}

type Worker struct {
	id           int
	dataDir      string
	git          *GitClient
	docker       *DockerClient
	health       *HealthChecker
	notifier     Notifier
	dispatcher   *Dispatcher
	logger       *slog.Logger
	deployConfig *PaasDeployConfig
}

func NewWorker(
	id int,
	dataDir string,
	git *GitClient,
	docker *DockerClient,
	health *HealthChecker,
	notifier Notifier,
	dispatcher *Dispatcher,
	logger *slog.Logger,
) *Worker {
	return &Worker{
		id:         id,
		dataDir:    dataDir,
		git:        git,
		docker:     docker,
		health:     health,
		notifier:   notifier,
		dispatcher: dispatcher,
		logger:     logger.With("workerId", id),
	}
}

func (w *Worker) Run(ctx context.Context, deploy *domain.Deployment, app *domain.App) error {
	w.logger.Info("Starting deployment",
		"deployId", deploy.ID,
		"appName", app.Name,
		"commitSha", deploy.CommitSHA,
	)

	w.notifier.EmitDeployRunning(deploy.ID, app.ID)
	w.log(deploy.ID, app.ID, "Starting deployment for %s", app.Name)

	repoDir := filepath.Join(w.dataDir, app.ID)

	if err := w.syncGit(ctx, deploy, app, repoDir); err != nil {
		return w.fail(deploy, app, fmt.Errorf("git sync failed: %w", err))
	}

	if err := w.loadConfig(repoDir); err != nil {
		return w.fail(deploy, app, fmt.Errorf("failed to load paasdeploy.json: %w", err))
	}

	imageTag := w.docker.GetImageTag(app.Name, deploy.CommitSHA)

	if err := w.buildDocker(ctx, deploy, app, repoDir, imageTag); err != nil {
		return w.fail(deploy, app, fmt.Errorf("docker build failed: %w", err))
	}

	if err := w.deployContainer(ctx, deploy, app, repoDir); err != nil {
		w.log(deploy.ID, app.ID, "Deploy failed, attempting rollback...")
		if rollbackErr := w.rollback(ctx, deploy, app, repoDir); rollbackErr != nil {
			w.logger.Error("Rollback failed", "error", rollbackErr)
		}
		return w.fail(deploy, app, fmt.Errorf("container deploy failed: %w", err))
	}

	if err := w.checkHealth(ctx, deploy, app); err != nil {
		w.log(deploy.ID, app.ID, "Health check failed, attempting rollback...")
		if rollbackErr := w.rollback(ctx, deploy, app, repoDir); rollbackErr != nil {
			w.logger.Error("Rollback failed", "error", rollbackErr)
		}
		return w.fail(deploy, app, fmt.Errorf("health check failed: %w", err))
	}

	return w.success(deploy, app, imageTag)
}

func (w *Worker) syncGit(ctx context.Context, deploy *domain.Deployment, app *domain.App, repoDir string) error {
	w.log(deploy.ID, app.ID, "Syncing repository...")

	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		w.log(deploy.ID, app.ID, "Cloning repository: %s", app.RepositoryURL)
		if err := os.MkdirAll(filepath.Dir(repoDir), 0755); err != nil {
			return err
		}
		if err := w.git.Clone(ctx, app.RepositoryURL, repoDir); err != nil {
			return err
		}
	}

	w.log(deploy.ID, app.ID, "Fetching updates...")
	if err := w.git.Sync(ctx, repoDir, deploy.CommitSHA); err != nil {
		return err
	}

	sha, err := w.git.GetCurrentCommitSHA(ctx, repoDir)
	if err != nil {
		return err
	}
	w.log(deploy.ID, app.ID, "Checked out commit: %s", sha[:12])

	return nil
}

func (w *Worker) loadConfig(repoDir string) error {
	configPath := filepath.Join(repoDir, "paasdeploy.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		w.deployConfig = &PaasDeployConfig{
			Port: defaultAppPort,
		}
		w.deployConfig.Build.Type = "dockerfile"
		w.deployConfig.Build.Dockerfile = "./Dockerfile"
		w.deployConfig.Build.Context = "."
		w.deployConfig.Healthcheck.Path = "/health"
		return nil
	}

	var config PaasDeployConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	if config.Port == 0 {
		config.Port = defaultAppPort
	}
	if config.Build.Dockerfile == "" {
		config.Build.Dockerfile = "./Dockerfile"
	}
	if config.Build.Context == "" {
		config.Build.Context = "."
	}
	if config.Healthcheck.Path == "" {
		config.Healthcheck.Path = "/health"
	}

	w.deployConfig = &config
	return nil
}

func (w *Worker) buildDocker(ctx context.Context, deploy *domain.Deployment, app *domain.App, repoDir, imageTag string) error {
	w.log(deploy.ID, app.ID, "Building Docker image: %s", imageTag)

	buildContext := filepath.Join(repoDir, w.deployConfig.Build.Context)
	dockerfile := filepath.Join(repoDir, w.deployConfig.Build.Dockerfile)

	output := make(chan string, outputChannelBuffer)
	go func() {
		for line := range output {
			w.log(deploy.ID, app.ID, "[build] %s", line)
		}
	}()

	err := w.docker.Build(ctx, buildContext, dockerfile, imageTag, output)
	close(output)

	if err != nil {
		return err
	}

	w.log(deploy.ID, app.ID, "Docker image built successfully")
	return nil
}

func (w *Worker) deployContainer(ctx context.Context, deploy *domain.Deployment, app *domain.App, repoDir string) error {
	w.log(deploy.ID, app.ID, "Deploying container...")

	output := make(chan string, outputChannelBuffer)
	go func() {
		for line := range output {
			w.log(deploy.ID, app.ID, "[deploy] %s", line)
		}
	}()

	err := w.docker.ComposeUp(ctx, repoDir, output)
	close(output)

	if err != nil {
		return err
	}

	w.log(deploy.ID, app.ID, "Container deployed successfully")
	return nil
}

func (w *Worker) checkHealth(ctx context.Context, deploy *domain.Deployment, app *domain.App) error {
	w.log(deploy.ID, app.ID, "Performing health check...")

	healthURL := fmt.Sprintf("http://localhost:%d%s", w.deployConfig.Port, w.deployConfig.Healthcheck.Path)

	time.Sleep(healthCheckStartDelay)

	if err := w.health.CheckWithBackoff(ctx, healthURL); err != nil {
		return err
	}

	w.log(deploy.ID, app.ID, "Health check passed")
	return nil
}

func (w *Worker) rollback(ctx context.Context, deploy *domain.Deployment, app *domain.App, repoDir string) error {
	if deploy.PreviousImageTag == "" {
		w.log(deploy.ID, app.ID, "No previous image to rollback to")
		return nil
	}

	w.log(deploy.ID, app.ID, "Rolling back to: %s", deploy.PreviousImageTag)

	if err := w.docker.ComposeDown(ctx, repoDir); err != nil {
		w.logger.Warn("Failed to stop containers", "error", err)
	}

	return nil
}

func (w *Worker) success(deploy *domain.Deployment, app *domain.App, imageTag string) error {
	w.log(deploy.ID, app.ID, "Deployment completed successfully")

	if err := w.dispatcher.MarkSuccess(deploy.ID, imageTag); err != nil {
		w.logger.Error("Failed to mark deploy as success", "error", err)
	}

	if err := w.dispatcher.UpdateAppLastDeployedAt(app.ID); err != nil {
		w.logger.Error("Failed to update app last deployed at", "error", err)
	}

	w.notifier.EmitDeploySuccess(deploy.ID, app.ID)

	return nil
}

func (w *Worker) fail(deploy *domain.Deployment, app *domain.App, err error) error {
	w.log(deploy.ID, app.ID, "Deployment failed: %s", err.Error())

	if markErr := w.dispatcher.MarkFailed(deploy.ID, err.Error()); markErr != nil {
		w.logger.Error("Failed to mark deploy as failed", "error", markErr)
	}

	w.notifier.EmitDeployFailed(deploy.ID, app.ID, err.Error())

	return err
}

func (w *Worker) log(deployID, appID, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("15:04:05")
	logLine := fmt.Sprintf("[%s] %s\n", timestamp, message)

	w.dispatcher.AppendLogs(deployID, logLine)
	w.notifier.EmitLog(deployID, appID, message)
	w.logger.Info(message, "deployId", deployID, "appId", appID)
}
