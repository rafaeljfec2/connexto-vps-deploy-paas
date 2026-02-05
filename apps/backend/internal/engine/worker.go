package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/service"
)

const (
	outputChannelBuffer   = 100
	healthCheckStartDelay = 15 * time.Second
	defaultAppPort        = 8080
)

type DomainRoute struct {
	Domain     string
	PathPrefix string
}

type PaasDeployConfig struct {
	Name    string `json:"name"`
	Runtime string `json:"runtime,omitempty"`
	Build   struct {
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
	HostPort  int               `json:"hostPort,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Resources struct {
		Memory string `json:"memory"`
		CPU    string `json:"cpu"`
	} `json:"resources"`
	Domains []string `json:"domains,omitempty"`
}

type GitTokenProvider interface {
	GetToken(ctx context.Context, repoURL string) (string, error)
}

type WorkerDeps struct {
	Git              *GitClient
	Docker           *DockerClient
	Health           *HealthChecker
	Notifier         Notifier
	Dispatcher       *Dispatcher
	EnvVarRepo       domain.EnvVarRepository
	CustomDomainRepo domain.CustomDomainRepository
	GitTokenProvider GitTokenProvider
	AuditService     *service.AuditService
	Logger           *slog.Logger
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
		w.appEnvVars = nil
		w.deps.Logger.Warn("Failed to load env vars", "error", err, "appId", app.ID)
		w.log(deploy.ID, app.ID, "Warning: could not load env vars from configuration (deploy will use paasdeploy.json env only)")
	} else {
		w.log(deploy.ID, app.ID, "Loaded %d environment variable(s) from configuration", len(w.appEnvVars))
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
	var appDir string
	if workdir == "" || workdir == "." {
		appDir = repoDir
	} else {
		appDir = filepath.Join(repoDir, workdir)
	}
	w.deps.Logger.Info("Calculated appDir", "repoDir", repoDir, "workdir", workdir, "appDir", appDir)
	return appDir
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

	token := w.getGitToken(ctx, app.RepositoryURL)

	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		w.log(deploy.ID, app.ID, "Cloning repository: %s", app.RepositoryURL)
		if err := os.MkdirAll(filepath.Dir(repoDir), 0755); err != nil {
			return err
		}

		if err := w.deps.Git.CloneWithToken(ctx, app.RepositoryURL, repoDir, token); err != nil {
			return err
		}
	}

	w.log(deploy.ID, app.ID, "Fetching updates...")
	if err := w.deps.Git.SyncWithToken(ctx, repoDir, deploy.CommitSHA, app.RepositoryURL, token); err != nil {
		return err
	}

	sha, err := w.deps.Git.GetCurrentCommitSHA(ctx, repoDir)
	if err != nil {
		return err
	}
	w.log(deploy.ID, app.ID, "Checked out commit: %s", truncateSHA(sha))

	return nil
}

func (w *Worker) getGitToken(ctx context.Context, repoURL string) string {
	if w.deps.GitTokenProvider == nil {
		return ""
	}

	token, err := w.deps.GitTokenProvider.GetToken(ctx, repoURL)
	if err != nil {
		w.deps.Logger.Warn("Failed to get git token, will try without authentication", "error", err)
		return ""
	}

	return token
}

func (w *Worker) loadConfig(appDir string) error {
	configPath := filepath.Join(appDir, "paasdeploy.json")

	w.deps.Logger.Info("Looking for paasdeploy.json", "configPath", configPath, "appDir", appDir)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		w.deps.Logger.Error("paasdeploy.json not found", "configPath", configPath, "appDir", appDir)
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

	applyConfigDefaults(&config)

	dockerfilePath := filepath.Join(appDir, config.Build.Dockerfile)
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("Dockerfile not found at %s - this file is required for deployment", config.Build.Dockerfile)
	}

	w.deployConfig = &config
	return nil
}

func applyConfigDefaults(config *PaasDeployConfig) {
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

	if err := w.deps.Docker.EnsureNetwork(ctx, defaultNetworkName); err != nil {
		return fmt.Errorf("failed to ensure network: %w", err)
	}

	if err := w.deps.Docker.RemoveContainer(ctx, app.Name, true); err != nil {
		if !strings.Contains(err.Error(), "No such container") {
			w.deps.Logger.Warn("Failed to remove existing container", "appName", app.Name, "error", err)
		}
	}

	imageTag := w.deps.Docker.GetImageTag(app.Name, deploy.CommitSHA)

	if err := w.generateComposeFile(ctx, appDir, app.Name, app.ID, imageTag); err != nil {
		return fmt.Errorf("failed to generate docker-compose.yml: %w", err)
	}

	output := make(chan string, outputChannelBuffer)
	go func() {
		for line := range output {
			w.log(deploy.ID, app.ID, "[deploy] %s", line)
		}
	}()

	err := w.deps.Docker.ComposeUp(ctx, appDir, app.ID, output)
	close(output)

	if err != nil {
		return err
	}

	w.log(deploy.ID, app.ID, "Container deployed successfully")
	return nil
}

func (w *Worker) generateComposeFile(ctx context.Context, appDir, appName, appID, imageTag string) error {
	cfg := w.deployConfig

	var domainRoutes []DomainRoute
	for _, d := range cfg.Domains {
		domainRoutes = append(domainRoutes, DomainRoute{Domain: d})
	}

	if w.deps.CustomDomainRepo != nil {
		customDomains, err := w.deps.CustomDomainRepo.FindByAppID(ctx, appID)
		if err == nil {
			for _, d := range customDomains {
				domainRoutes = append(domainRoutes, DomainRoute{
					Domain:     d.Domain,
					PathPrefix: d.PathPrefix,
				})
			}
		}
	}

	envVars := w.buildEnvVarsYAML(cfg)
	labels := buildLabelsYAML(appName, domainRoutes, cfg.Port)
	portMapping := buildPortMapping(cfg.HostPort, cfg.Port)

	healthCmd := buildHealthCheckCommand(cfg.Runtime, cfg.Port, cfg.Healthcheck.Path)

	composeContent := fmt.Sprintf("services:\n"+
		"  %s:\n"+
		"    image: %s\n"+
		"    container_name: %s\n"+
		"    restart: unless-stopped\n"+
		"    ports:\n"+
		"      - \"%s\"\n"+
		"%s"+
		"%s"+
		"    healthcheck:\n"+
		"      test:\n"+
		"        - CMD-SHELL\n"+
		"        - %s\n"+
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
		portMapping,
		envVars,
		labels,
		healthCmd,
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

func (w *Worker) buildEnvVarsYAML(cfg *PaasDeployConfig) string {
	allEnvVars := make(map[string]string)

	for k, v := range cfg.Env {
		allEnvVars[k] = v
	}

	if w.appEnvVars != nil {
		for k, v := range w.appEnvVars {
			allEnvVars[k] = v
		}
	}

	if len(allEnvVars) == 0 {
		return ""
	}

	envVars := "    environment:\n"
	for k, v := range allEnvVars {
		escapedValue := escapeEnvValue(v)
		envVars += fmt.Sprintf("      - %s=%s\n", k, escapedValue)
	}
	return envVars
}

func escapeEnvValue(value string) string {
	escaped := strings.ReplaceAll(value, "$", "$$")
	return escaped
}

func buildHealthCheckCommand(runtime string, port int, path string) string {
	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)

	switch strings.ToLower(runtime) {
	case "node", "nodejs", "node.js":
		return fmt.Sprintf(
			`node -e 'const h=require("http");h.get("%s",(r)=>process.exit(r.statusCode>=200&&r.statusCode<400?0:1)).on("error",()=>process.exit(1))'`,
			url,
		)
	case "python", "python3":
		return fmt.Sprintf(
			`python3 -c 'import urllib.request,sys;urllib.request.urlopen("%s");sys.exit(0)' 2>/dev/null || python -c 'import urllib.request,sys;urllib.request.urlopen("%s");sys.exit(0)'`,
			url, url,
		)
	case "go", "golang":
		return fmt.Sprintf("curl -sf %s || wget -q --spider %s || exit 1", url, url)
	case "ruby":
		return fmt.Sprintf(
			`ruby -e 'require "net/http";exit(Net::HTTP.get_response(URI("%s")).is_a?(Net::HTTPSuccess)?0:1)'`,
			url,
		)
	case "php":
		return fmt.Sprintf(
			`php -r 'exit(file_get_contents("%s")!==false?0:1);'`,
			url,
		)
	default:
		return fmt.Sprintf("curl -sf %s || wget -q --spider %s || exit 1", url, url)
	}
}

func buildLabelsYAML(appName string, domains []DomainRoute, port int) string {
	if len(domains) > 0 {
		var labels strings.Builder
		labels.WriteString("    labels:\n")
		labels.WriteString(fmt.Sprintf("      - \"paasdeploy.app=%s\"\n", appName))
		labels.WriteString("      - \"traefik.enable=true\"\n")
		labels.WriteString("      - \"traefik.docker.network=paasdeploy\"\n")
		labels.WriteString(fmt.Sprintf("      - \"traefik.http.services.%s.loadbalancer.server.port=%d\"\n", appName, port))

		for i, d := range domains {
			routerName := appName
			if i > 0 {
				routerName = fmt.Sprintf("%s-%d", appName, i)
			}

			priority := 1
			if d.PathPrefix != "" {
				priority = 100 + len(d.PathPrefix)
			}

			rule := fmt.Sprintf("Host(`%s`)", d.Domain)
			if d.PathPrefix != "" {
				rule = fmt.Sprintf("Host(`%s`) && PathPrefix(`%s`)", d.Domain, d.PathPrefix)
			}

			labels.WriteString(fmt.Sprintf("      - \"traefik.http.routers.%s.rule=%s\"\n", routerName, rule))
			labels.WriteString(fmt.Sprintf("      - \"traefik.http.routers.%s.priority=%d\"\n", routerName, priority))
			labels.WriteString(fmt.Sprintf("      - \"traefik.http.routers.%s.tls=true\"\n", routerName))
			labels.WriteString(fmt.Sprintf("      - \"traefik.http.routers.%s.tls.certresolver=letsencrypt\"\n", routerName))
			labels.WriteString(fmt.Sprintf("      - \"traefik.http.routers.%s.service=%s\"\n", routerName, appName))
		}

		return labels.String()
	}

	return fmt.Sprintf("    labels:\n"+
		"      - \"paasdeploy.app=%s\"\n"+
		"      - \"traefik.enable=true\"\n"+
		"      - \"traefik.docker.network=paasdeploy\"\n"+
		"      - \"traefik.http.routers.%s.rule=Host(`%s.localhost`)\"\n"+
		"      - \"traefik.http.services.%s.loadbalancer.server.port=%d\"\n",
		appName, appName, appName, appName, port)
}

func buildPortMapping(hostPort, port int) string {
	if hostPort > 0 {
		return fmt.Sprintf("%d:%d", hostPort, port)
	}
	return fmt.Sprintf("%d", port)
}

func (w *Worker) checkHealth(ctx context.Context, deploy *domain.Deployment, app *domain.App) error {
	w.log(deploy.ID, app.ID, "Performing health check...")

	startDelay := healthCheckStartDelay
	if w.deployConfig.Healthcheck.StartPeriod != "" {
		if parsed, err := time.ParseDuration(w.deployConfig.Healthcheck.StartPeriod); err == nil && parsed > startDelay {
			startDelay = parsed
		}
	}

	w.log(deploy.ID, app.ID, "Waiting %s for container to be ready...", startDelay)
	time.Sleep(startDelay)

	containerIP, err := w.deps.Docker.GetContainerIP(ctx, app.Name, defaultNetworkName)
	if err != nil {
		return fmt.Errorf("failed to get container IP: %w", err)
	}

	healthURL := fmt.Sprintf("http://%s:%d%s", containerIP, w.deployConfig.Port, w.deployConfig.Healthcheck.Path)
	w.log(deploy.ID, app.ID, "Health check URL: %s", healthURL)

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

	if err := w.deps.Docker.ComposeDown(ctx, repoDir, app.ID); err != nil {
		w.deps.Logger.Warn("Failed to stop containers during rollback", "deployId", deploy.ID, "appName", app.Name, "error", err)
		return err
	}

	w.deps.Logger.Info("Rollback completed", "deployId", deploy.ID, "appName", app.Name, "previousImage", deploy.PreviousImageTag)
	return nil
}

func (w *Worker) success(deploy *domain.Deployment, app *domain.App, imageTag string) error {
	w.log(deploy.ID, app.ID, "Deployment completed successfully")

	if w.deps.AuditService != nil {
		auditCtx := service.AuditContext{}
		w.deps.AuditService.LogDeploySuccess(context.Background(), auditCtx, deploy.ID, app.ID, app.Name)
	}

	if err := w.deps.Dispatcher.MarkSuccess(deploy.ID, imageTag); err != nil {
		w.deps.Logger.Error("Failed to mark deploy as success", "error", err)
	}

	if err := w.deps.Dispatcher.UpdateAppLastDeployedAt(app.ID); err != nil {
		w.deps.Logger.Error("Failed to update app last deployed at", "error", err)
	}

	if w.deployConfig != nil && w.deployConfig.Runtime != "" {
		if err := w.deps.Dispatcher.UpdateAppRuntime(app.ID, w.deployConfig.Runtime); err != nil {
			w.deps.Logger.Error("Failed to update app runtime", "error", err)
		}
	}

	go w.cleanupOldImages(deploy)

	w.deps.Notifier.EmitDeploySuccess(deploy.ID, app.ID)

	return nil
}

func (w *Worker) cleanupOldImages(deploy *domain.Deployment) {
	if deploy.PreviousImageTag == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	w.deps.Logger.Info("Cleaning up old Docker image",
		"deploy_id", deploy.ID,
		"previous_image", deploy.PreviousImageTag,
	)

	if err := w.deps.Docker.RemoveImage(ctx, deploy.PreviousImageTag); err != nil {
		w.deps.Logger.Warn("Failed to remove previous image", "error", err)
	}

	if err := w.deps.Docker.PruneUnusedImages(ctx); err != nil {
		w.deps.Logger.Warn("Failed to prune unused images", "error", err)
	}
}

func (w *Worker) fail(deploy *domain.Deployment, app *domain.App, err error) error {
	w.log(deploy.ID, app.ID, "Deployment failed: %s", err.Error())

	if w.deps.AuditService != nil {
		auditCtx := service.AuditContext{}
		w.deps.AuditService.LogDeployFailed(context.Background(), auditCtx, deploy.ID, app.ID, app.Name, err.Error())
	}

	if markErr := w.deps.Dispatcher.MarkFailed(deploy.ID, err.Error()); markErr != nil {
		w.deps.Logger.Error("Failed to mark deploy as failed", "error", markErr)
	}

	w.deps.Notifier.EmitDeployFailed(deploy.ID, app.ID, err.Error())

	return err
}

func (w *Worker) log(deployID, appID, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("15:04:05")
	logLine := fmt.Sprintf("[%s] %s", timestamp, message)

	w.deps.Dispatcher.AppendLogs(deployID, logLine+"\n")
	w.deps.Notifier.EmitLog(deployID, appID, logLine)
	w.deps.Logger.Info(message, "deployId", deployID, "appId", appID)
}

func truncateSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}
