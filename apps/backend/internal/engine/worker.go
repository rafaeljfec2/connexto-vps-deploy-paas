package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/service"
	"github.com/paasdeploy/shared/pkg/compose"
	"github.com/paasdeploy/shared/pkg/docker"
	"github.com/paasdeploy/shared/pkg/git"
	"github.com/paasdeploy/shared/pkg/health"
)

const (
	outputChannelBuffer   = 100
	healthCheckStartDelay = 15 * time.Second
)

type GitTokenProvider interface {
	GetToken(ctx context.Context, repoURL string) (string, error)
}

type WorkerDeps struct {
	Git              *git.Client
	Docker           *docker.Client
	Health           *health.Checker
	Notifier         Notifier
	Dispatcher       *Dispatcher
	EnvVarRepo       domain.EnvVarRepository
	CustomDomainRepo domain.CustomDomainRepository
	ServerRepo       domain.ServerRepository
	AgentClient      *agentclient.AgentClient
	AgentPort        int
	GitTokenProvider GitTokenProvider
	AuditService     *service.AuditService
	Logger           *slog.Logger
}

type Worker struct {
	id           int
	dataDir      string
	deps         WorkerDeps
	deployConfig *compose.Config
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
		"serverID", app.ServerID,
	)

	if app.ServerID != nil && *app.ServerID != "" {
		return w.runRemoteDeploy(ctx, deploy, app)
	}

	return w.runLocalDeploy(ctx, deploy, app)
}

func (w *Worker) runRemoteDeploy(ctx context.Context, deploy *domain.Deployment, app *domain.App) error {
	w.deps.Notifier.EmitDeployRunning(deploy.ID, app.ID)
	w.log(deploy.ID, app.ID, "Starting remote deployment for %s on server %s", app.Name, *app.ServerID)

	if w.deps.ServerRepo == nil || w.deps.AgentClient == nil {
		return w.fail(deploy, app, fmt.Errorf("remote deploy not available: server repository or agent client not configured"))
	}

	server, err := w.deps.ServerRepo.FindByID(*app.ServerID)
	if err != nil {
		return w.fail(deploy, app, fmt.Errorf("failed to find server %s: %w", *app.ServerID, err))
	}

	if server.Status != domain.ServerStatusOnline {
		return w.fail(deploy, app, fmt.Errorf("server %s is not online (status: %s)", server.Name, server.Status))
	}

	if err := w.loadEnvVars(app.ID); err != nil {
		w.appEnvVars = nil
		w.deps.Logger.Warn("Failed to load env vars for remote deploy", "error", err, "appId", app.ID)
	}

	token := w.getGitToken(ctx, app.RepositoryURL)

	domainRoutes := w.collectDomainRoutes(ctx, app.ID)
	var domains []string
	for _, d := range domainRoutes {
		domains = append(domains, d.Domain)
	}

	defaults := &compose.Config{}
	compose.ApplyDefaults(defaults)

	appPort := defaults.Port
	if portStr, ok := w.appEnvVars["PORT"]; ok {
		if parsed, err := strconv.Atoi(portStr); err == nil && parsed > 0 {
			appPort = parsed
		}
	}

	req := &pb.DeployRequest{
		DeploymentId: deploy.ID,
		AppId:        app.ID,
		AppName:      app.Name,
		Git: &pb.GitConfig{
			RepositoryUrl: app.RepositoryURL,
			Branch:        app.Branch,
			CommitSha:     deploy.CommitSHA,
			Workdir:       app.Workdir,
		},
		Build: &pb.BuildConfig{
			Dockerfile: defaults.Build.Dockerfile,
			Context:    defaults.Build.Context,
		},
		Runtime: &pb.RuntimeConfig{
			Port:    int32(appPort),
			Domains: domains,
			Resources: &pb.ResourceLimits{
				Memory: defaults.Resources.Memory,
				Cpu:    defaults.Resources.CPU,
			},
		},
		HealthCheck: &pb.HealthCheckConfig{
			Path:        defaults.Healthcheck.Path,
			Interval:    defaults.Healthcheck.Interval,
			Timeout:     defaults.Healthcheck.Timeout,
			Retries:     int32(defaults.Healthcheck.Retries),
			StartPeriod: defaults.Healthcheck.StartPeriod,
		},
		EnvVars: w.appEnvVars,
	}

	if token != "" {
		req.Git.Auth = &pb.GitConfig_AccessToken{AccessToken: token}
	}

	if deploy.PreviousImageTag != "" {
		req.RollbackImage = &deploy.PreviousImageTag
	}

	agentPort := w.deps.AgentPort
	if agentPort == 0 {
		agentPort = 50052
	}

	w.log(deploy.ID, app.ID, "Dispatching deploy to agent at %s:%d", server.Host, agentPort)

	onLog := func(entry *pb.DeployLogEntry) {
		prefix := formatLogStage(entry.Stage)
		w.log(deploy.ID, app.ID, "%s %s", prefix, entry.Message)
	}

	resp, err := w.deps.AgentClient.ExecuteDeployWithLogs(ctx, server.Host, agentPort, req, onLog)
	if err != nil {
		return w.fail(deploy, app, fmt.Errorf("remote deploy RPC failed: %w", err))
	}

	if !resp.Success {
		errMsg := resp.Message
		if resp.Error != nil {
			errMsg = fmt.Sprintf("[%s] %s: %s", resp.Error.Stage, resp.Error.Code, resp.Error.Message)
		}
		return w.fail(deploy, app, fmt.Errorf("remote deploy failed: %s", errMsg))
	}

	imageTag := ""
	if resp.Result != nil {
		imageTag = resp.Result.ImageTag
	}

	w.log(deploy.ID, app.ID, "Remote deployment completed successfully")
	return w.success(deploy, app, imageTag)
}

func (w *Worker) runLocalDeploy(ctx context.Context, deploy *domain.Deployment, app *domain.App) error {
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
	cfg, err := compose.LoadConfig(appDir)
	if err != nil {
		return err
	}

	if err := compose.ValidateDockerfile(appDir, cfg); err != nil {
		return err
	}

	w.deployConfig = cfg
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

	if err := w.deps.Docker.EnsureNetwork(ctx, docker.DefaultNetworkName); err != nil {
		return fmt.Errorf("failed to ensure network: %w", err)
	}

	if err := w.deps.Docker.RemoveContainer(ctx, app.Name, true); err != nil {
		if !strings.Contains(err.Error(), "No such container") {
			w.deps.Logger.Warn("Failed to remove existing container", "appName", app.Name, "error", err)
		}
	}

	imageTag := w.deps.Docker.GetImageTag(app.Name, deploy.CommitSHA)
	domainRoutes := w.collectDomainRoutes(ctx, app.ID)

	if err := compose.WriteComposeFile(appDir, compose.GenerateParams{
		AppName:  app.Name,
		ImageTag: imageTag,
		Config:   w.deployConfig,
		Domains:  domainRoutes,
		EnvVars:  w.appEnvVars,
	}); err != nil {
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

func (w *Worker) collectDomainRoutes(ctx context.Context, appID string) []compose.DomainRoute {
	var domainRoutes []compose.DomainRoute

	if w.deployConfig != nil {
		for _, d := range w.deployConfig.Domains {
			domainRoutes = append(domainRoutes, compose.DomainRoute{Domain: d})
		}
	}

	if w.deps.CustomDomainRepo != nil {
		customDomains, err := w.deps.CustomDomainRepo.FindByAppID(ctx, appID)
		if err == nil {
			for _, d := range customDomains {
				domainRoutes = append(domainRoutes, compose.DomainRoute{
					Domain:     d.Domain,
					PathPrefix: d.PathPrefix,
				})
			}
		}
	}

	return domainRoutes
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

	var healthURL string
	selfID, _ := w.deps.Docker.GetCurrentContainerID(ctx)
	useHostPort := w.deployConfig.HostPort > 0 && selfID == ""
	if useHostPort {
		healthURL = fmt.Sprintf("http://127.0.0.1:%d%s", w.deployConfig.HostPort, w.deployConfig.Healthcheck.Path)
		w.log(deploy.ID, app.ID, "Health check URL (via host port): %s", healthURL)
	} else {
		containerIP, err := w.deps.Docker.GetContainerIP(ctx, app.Name, docker.DefaultNetworkName)
		if err != nil {
			return fmt.Errorf("failed to get container IP: %w", err)
		}
		healthURL = fmt.Sprintf("http://%s:%d%s", containerIP, w.deployConfig.Port, w.deployConfig.Healthcheck.Path)
		w.log(deploy.ID, app.ID, "Health check URL: %s", healthURL)
	}

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

func formatLogStage(stage pb.DeployStage) string {
	switch stage {
	case pb.DeployStage_DEPLOY_STAGE_GIT_SYNC:
		return "[git]"
	case pb.DeployStage_DEPLOY_STAGE_BUILD:
		return "[build]"
	case pb.DeployStage_DEPLOY_STAGE_DEPLOY:
		return "[deploy]"
	case pb.DeployStage_DEPLOY_STAGE_HEALTH_CHECK:
		return "[health]"
	case pb.DeployStage_DEPLOY_STAGE_CLEANUP:
		return "[cleanup]"
	case pb.DeployStage_DEPLOY_STAGE_ROLLBACK:
		return "[rollback]"
	case pb.DeployStage_DEPLOY_STAGE_COMPLETE:
		return "[complete]"
	default:
		return "[agent]"
	}
}

func truncateSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}
