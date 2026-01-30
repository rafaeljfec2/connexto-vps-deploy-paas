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
	Name  string `json:"name"`
	Build struct {
		Type       string            `json:"type"`
		Dockerfile string            `json:"dockerfile"`
		Context    string            `json:"context"`
		Args       map[string]string `json:"args,omitempty"`
		Target     string            `json:"target,omitempty"`
	} `json:"build"`
	Healthcheck struct {
		Path        string `json:"path"`
		Interval    string `json:"interval"`
		Timeout     string `json:"timeout"`
		Retries     int    `json:"retries"`
		StartPeriod string `json:"startPeriod"`
	} `json:"healthcheck"`
	Port      int               `json:"port"`
	Env       map[string]string `json:"env,omitempty"`
	Resources struct {
		Memory string `json:"memory"`
		CPU    string `json:"cpu"`
	} `json:"resources"`
	Domains []string `json:"domains,omitempty"`
}

type WorkerDeps struct {
	Git        *GitClient
	Docker     *DockerClient
	Health     *HealthChecker
	Notifier   Notifier
	Dispatcher *Dispatcher
	EnvVarRepo domain.EnvVarRepository
	Logger     *slog.Logger
}

type Worker struct {
	id           int
	dataDir      string
	deps         WorkerDeps
	deployConfig *PaasDeployConfig
	appEnvVars   map[string]string
}

func NewWorker(id int, dataDir string, deps WorkerDeps) *Worker {
	return &Worker{
		id:      id,
		dataDir: dataDir,
		deps:    deps,
	}
}

func (w *Worker) Run(ctx context.Context, deploy *domain.Deployment, app *domain.App) error {
	w.deps.Logger.Info("Starting deployment",
		"deployId", deploy.ID,
		"appName", app.Name,
		"commitSha", deploy.CommitSHA,
		"workdir", app.Workdir,
	)

	w.deps.Notifier.EmitDeployRunning(deploy.ID, app.ID)
	w.log(deploy.ID, app.ID, "Starting deployment for %s", app.Name)

	if err := w.loadEnvVars(app.ID); err != nil {
		w.deps.Logger.Warn("Failed to load env vars", "error", err)
	}

	repoDir := filepath.Join(w.dataDir, app.ID)
	appDir := w.getAppDir(repoDir, app.Workdir)

	if err := w.syncGit(ctx, deploy, app, repoDir); err != nil {
		return w.fail(deploy, app, fmt.Errorf("git sync failed: %w", err))
	}

	if err := w.loadConfig(appDir); err != nil {
		return w.fail(deploy, app, fmt.Errorf("failed to load paasdeploy.json: %w", err))
	}

	imageTag := w.deps.Docker.GetImageTag(app.Name, deploy.CommitSHA)

	if err := w.buildDocker(ctx, deploy, app, appDir, imageTag); err != nil {
		return w.fail(deploy, app, fmt.Errorf("docker build failed: %w", err))
	}

	if err := w.deployContainer(ctx, deploy, app, appDir); err != nil {
		w.log(deploy.ID, app.ID, "Deploy failed, attempting rollback...")
		if rollbackErr := w.rollback(ctx, deploy, app, appDir); rollbackErr != nil {
			w.deps.Logger.Error("Rollback failed", "error", rollbackErr)
		}
		return w.fail(deploy, app, fmt.Errorf("container deploy failed: %w", err))
	}

	if err := w.checkHealth(ctx, deploy, app); err != nil {
		w.log(deploy.ID, app.ID, "Health check failed, attempting rollback...")
		if rollbackErr := w.rollback(ctx, deploy, app, appDir); rollbackErr != nil {
			w.deps.Logger.Error("Rollback failed", "error", rollbackErr)
		}
		return w.fail(deploy, app, fmt.Errorf("health check failed: %w", err))
	}

	return w.success(deploy, app, imageTag)
}

func (w *Worker) getAppDir(repoDir, workdir string) string {
	if workdir == "" || workdir == "." {
		return repoDir
	}
	return filepath.Join(repoDir, workdir)
}

func (w *Worker) loadEnvVars(appID string) error {
	vars, err := w.deps.EnvVarRepo.FindByAppID(appID)
	if err != nil {
		return err
	}

	w.appEnvVars = make(map[string]string)
	for _, v := range vars {
		w.appEnvVars[v.Key] = v.Value
	}

	return nil
}

func (w *Worker) syncGit(ctx context.Context, deploy *domain.Deployment, app *domain.App, repoDir string) error {
	w.log(deploy.ID, app.ID, "Syncing repository...")

	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		w.log(deploy.ID, app.ID, "Cloning repository: %s", app.RepositoryURL)
		if err := os.MkdirAll(filepath.Dir(repoDir), 0755); err != nil {
			return err
		}
		if err := w.deps.Git.Clone(ctx, app.RepositoryURL, repoDir); err != nil {
			return err
		}
	}

	w.log(deploy.ID, app.ID, "Fetching updates...")
	if err := w.deps.Git.Sync(ctx, repoDir, deploy.CommitSHA); err != nil {
		return err
	}

	sha, err := w.deps.Git.GetCurrentCommitSHA(ctx, repoDir)
	if err != nil {
		return err
	}
	w.log(deploy.ID, app.ID, "Checked out commit: %s", sha[:12])

	return nil
}

func (w *Worker) loadConfig(appDir string) error {
	configPath := filepath.Join(appDir, "paasdeploy.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("paasdeploy.json not found in repository - this file is required for deployment")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read paasdeploy.json: %w", err)
	}

	var config PaasDeployConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid paasdeploy.json: %w", err)
	}

	if config.Name == "" {
		return fmt.Errorf("paasdeploy.json: 'name' field is required")
	}

	if config.Port == 0 {
		config.Port = defaultAppPort
	}
	if config.Build.Type == "" {
		config.Build.Type = "dockerfile"
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
	if config.Healthcheck.Interval == "" {
		config.Healthcheck.Interval = "30s"
	}
	if config.Healthcheck.Timeout == "" {
		config.Healthcheck.Timeout = "5s"
	}
	if config.Healthcheck.Retries == 0 {
		config.Healthcheck.Retries = 3
	}
	if config.Healthcheck.StartPeriod == "" {
		config.Healthcheck.StartPeriod = "10s"
	}
	if config.Resources.Memory == "" {
		config.Resources.Memory = "512m"
	}
	if config.Resources.CPU == "" {
		config.Resources.CPU = "0.5"
	}

	dockerfilePath := filepath.Join(appDir, config.Build.Dockerfile)
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("Dockerfile not found at %s - this file is required for deployment", config.Build.Dockerfile)
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

	err := w.deps.Docker.Build(ctx, buildContext, dockerfile, imageTag, output)
	close(output)

	if err != nil {
		return err
	}

	w.log(deploy.ID, app.ID, "Docker image built successfully")
	return nil
}

func (w *Worker) deployContainer(ctx context.Context, deploy *domain.Deployment, app *domain.App, appDir string) error {
	w.log(deploy.ID, app.ID, "Deploying container...")

	imageTag := w.deps.Docker.GetImageTag(app.Name, deploy.CommitSHA)

	if err := w.generateComposeFile(appDir, app.Name, imageTag); err != nil {
		return fmt.Errorf("failed to generate docker-compose.yml: %w", err)
	}

	output := make(chan string, outputChannelBuffer)
	go func() {
		for line := range output {
			w.log(deploy.ID, app.ID, "[deploy] %s", line)
		}
	}()

	err := w.deps.Docker.ComposeUp(ctx, appDir, output)
	close(output)

	if err != nil {
		return err
	}

	w.log(deploy.ID, app.ID, "Container deployed successfully")
	return nil
}

func (w *Worker) generateComposeFile(appDir, appName, imageTag string) error {
	cfg := w.deployConfig

	allEnvVars := make(map[string]string)

	for k, v := range cfg.Env {
		allEnvVars[k] = v
	}

	if w.appEnvVars != nil {
		for k, v := range w.appEnvVars {
			allEnvVars[k] = v
		}
	}

	var envVars string
	if len(allEnvVars) > 0 {
		envVars = "    environment:\n"
		for k, v := range allEnvVars {
			envVars += fmt.Sprintf("      - %s=%s\n", k, v)
		}
	}

	var labels string
	if len(cfg.Domains) > 0 {
		hosts := ""
		for i, domain := range cfg.Domains {
			if i > 0 {
				hosts += " || "
			}
			hosts += fmt.Sprintf("Host(`%s`)", domain)
		}
		labels = fmt.Sprintf("    labels:\n"+
			"      - \"traefik.enable=true\"\n"+
			"      - \"traefik.http.routers.%s.rule=%s\"\n"+
			"      - \"traefik.http.routers.%s.tls=true\"\n"+
			"      - \"traefik.http.routers.%s.tls.certresolver=letsencrypt\"\n"+
			"      - \"traefik.http.services.%s.loadbalancer.server.port=%d\"\n",
			appName, hosts, appName, appName, appName, cfg.Port)
	} else {
		labels = fmt.Sprintf("    labels:\n"+
			"      - \"traefik.enable=true\"\n"+
			"      - \"traefik.http.routers.%s.rule=Host(`%s.localhost`)\"\n"+
			"      - \"traefik.http.services.%s.loadbalancer.server.port=%d\"\n",
			appName, appName, appName, cfg.Port)
	}

	composeContent := fmt.Sprintf("version: \"3.8\"\n\n"+
		"services:\n"+
		"  %s:\n"+
		"    image: %s\n"+
		"    container_name: %s\n"+
		"    restart: unless-stopped\n"+
		"    ports:\n"+
		"      - \"%d\"\n"+
		"%s"+
		"%s"+
		"    healthcheck:\n"+
		"      test: [\"CMD\", \"wget\", \"-q\", \"--spider\", \"http://localhost:%d%s\"]\n"+
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
		appName,
		imageTag,
		appName,
		cfg.Port,
		envVars,
		labels,
		cfg.Port,
		cfg.Healthcheck.Path,
		cfg.Healthcheck.Interval,
		cfg.Healthcheck.Timeout,
		cfg.Healthcheck.Retries,
		cfg.Healthcheck.StartPeriod,
		cfg.Resources.Memory,
		cfg.Resources.CPU,
	)

	composePath := filepath.Join(appDir, "docker-compose.yml")
	return os.WriteFile(composePath, []byte(composeContent), 0644)
}

func (w *Worker) checkHealth(ctx context.Context, deploy *domain.Deployment, app *domain.App) error {
	w.log(deploy.ID, app.ID, "Performing health check...")

	healthURL := fmt.Sprintf("http://localhost:%d%s", w.deployConfig.Port, w.deployConfig.Healthcheck.Path)

	time.Sleep(healthCheckStartDelay)

	if err := w.deps.Health.CheckWithBackoff(ctx, healthURL); err != nil {
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

	if err := w.deps.Docker.ComposeDown(ctx, repoDir); err != nil {
		w.deps.Logger.Warn("Failed to stop containers", "error", err)
	}

	return nil
}

func (w *Worker) success(deploy *domain.Deployment, app *domain.App, imageTag string) error {
	w.log(deploy.ID, app.ID, "Deployment completed successfully")

	if err := w.deps.Dispatcher.MarkSuccess(deploy.ID, imageTag); err != nil {
		w.deps.Logger.Error("Failed to mark deploy as success", "error", err)
	}

	if err := w.deps.Dispatcher.UpdateAppLastDeployedAt(app.ID); err != nil {
		w.deps.Logger.Error("Failed to update app last deployed at", "error", err)
	}

	w.deps.Notifier.EmitDeploySuccess(deploy.ID, app.ID)

	return nil
}

func (w *Worker) fail(deploy *domain.Deployment, app *domain.App, err error) error {
	w.log(deploy.ID, app.ID, "Deployment failed: %s", err.Error())

	if markErr := w.deps.Dispatcher.MarkFailed(deploy.ID, err.Error()); markErr != nil {
		w.deps.Logger.Error("Failed to mark deploy as failed", "error", markErr)
	}

	w.deps.Notifier.EmitDeployFailed(deploy.ID, app.ID, err.Error())

	return err
}

func (w *Worker) log(deployID, appID, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("15:04:05")
	logLine := fmt.Sprintf("[%s] %s\n", timestamp, message)

	w.deps.Dispatcher.AppendLogs(deployID, logLine)
	w.deps.Notifier.EmitLog(deployID, appID, message)
	w.deps.Logger.Info(message, "deployId", deployID, "appId", appID)
}
