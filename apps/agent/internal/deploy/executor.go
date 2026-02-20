package deploy

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"github.com/paasdeploy/shared/pkg/compose"
	"github.com/paasdeploy/shared/pkg/docker"
	"github.com/paasdeploy/shared/pkg/git"
	"github.com/paasdeploy/shared/pkg/health"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultHealthTimeout  = 2 * time.Minute
	defaultHealthRetries  = 10
	defaultHealthInterval = 5 * time.Second
	healthCheckStartDelay = 15 * time.Second
	logChannelBuffer      = 256
)

type LogFunc func(stage pb.DeployStage, level pb.DeployLogLevel, message string)

type Executor struct {
	dataDir string
	git     *git.Client
	docker  *docker.Client
	health  *health.Checker
	logger  *slog.Logger
}

func resolveDataDir() string {
	if dir := os.Getenv("DEPLOY_DATA_DIR"); dir != "" {
		return dir
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".paasdeploy", "apps")
	}
	return filepath.Join(os.TempDir(), "paasdeploy", "apps")
}

func NewExecutor(logger *slog.Logger) *Executor {
	dataDir := resolveDataDir()

	registry := os.Getenv("DOCKER_REGISTRY")

	return &Executor{
		dataDir: dataDir,
		git:     git.NewClient(dataDir, logger),
		docker:  docker.NewClient(dataDir, registry, logger),
		health:  health.NewChecker(defaultHealthTimeout, defaultHealthRetries, defaultHealthInterval, logger),
		logger:  logger.With("component", "deploy-executor"),
	}
}

func (e *Executor) Execute(ctx context.Context, req *pb.DeployRequest, logFn LogFunc) *pb.DeployResponse {
	startedAt := time.Now()
	e.logger.Info("Starting deployment",
		"deploymentId", req.DeploymentId,
		"appName", req.AppName,
		"commitSha", req.Git.GetCommitSha(),
	)

	emit := e.emitter(logFn)

	cfg := e.buildConfig(req)
	repoDir := filepath.Join(e.dataDir, req.AppId)
	appDir := e.resolveAppDir(repoDir, req.Git.GetWorkdir())

	emit(pb.DeployStage_DEPLOY_STAGE_GIT_SYNC, pb.DeployLogLevel_DEPLOY_LOG_LEVEL_INFO, "Syncing repository...")
	if err := e.syncGit(ctx, req, repoDir); err != nil {
		emit(pb.DeployStage_DEPLOY_STAGE_GIT_SYNC, pb.DeployLogLevel_DEPLOY_LOG_LEVEL_ERROR, err.Error())
		return e.failResponse(req, pb.DeployErrorCode_DEPLOY_ERROR_GIT_CLONE_FAILED, "git_sync", err, startedAt)
	}
	emit(pb.DeployStage_DEPLOY_STAGE_GIT_SYNC, pb.DeployLogLevel_DEPLOY_LOG_LEVEL_INFO, "Repository synced successfully")

	imageTag := e.docker.GetImageTag(req.AppName, req.Git.GetCommitSha())

	emit(pb.DeployStage_DEPLOY_STAGE_BUILD, pb.DeployLogLevel_DEPLOY_LOG_LEVEL_INFO, fmt.Sprintf("Building image %s", imageTag))
	if err := e.buildImage(ctx, req, appDir, imageTag, logFn); err != nil {
		emit(pb.DeployStage_DEPLOY_STAGE_BUILD, pb.DeployLogLevel_DEPLOY_LOG_LEVEL_ERROR, err.Error())
		return e.failResponse(req, pb.DeployErrorCode_DEPLOY_ERROR_BUILD_FAILED, "build", err, startedAt)
	}
	emit(pb.DeployStage_DEPLOY_STAGE_BUILD, pb.DeployLogLevel_DEPLOY_LOG_LEVEL_INFO, "Image built successfully")

	emit(pb.DeployStage_DEPLOY_STAGE_DEPLOY, pb.DeployLogLevel_DEPLOY_LOG_LEVEL_INFO, "Deploying container...")
	if err := e.deployContainer(ctx, req, cfg, appDir, imageTag, logFn); err != nil {
		emit(pb.DeployStage_DEPLOY_STAGE_DEPLOY, pb.DeployLogLevel_DEPLOY_LOG_LEVEL_ERROR, err.Error())
		return e.failResponse(req, pb.DeployErrorCode_DEPLOY_ERROR_CONTAINER_START_FAILED, "deploy", err, startedAt)
	}
	emit(pb.DeployStage_DEPLOY_STAGE_DEPLOY, pb.DeployLogLevel_DEPLOY_LOG_LEVEL_INFO, "Container deployed successfully")

	emit(pb.DeployStage_DEPLOY_STAGE_HEALTH_CHECK, pb.DeployLogLevel_DEPLOY_LOG_LEVEL_INFO, "Running health check...")
	if err := e.checkHealth(ctx, req, cfg); err != nil {
		emit(pb.DeployStage_DEPLOY_STAGE_HEALTH_CHECK, pb.DeployLogLevel_DEPLOY_LOG_LEVEL_ERROR, err.Error())
		e.rollback(ctx, req, appDir)
		return e.failResponse(req, pb.DeployErrorCode_DEPLOY_ERROR_HEALTH_CHECK_FAILED, "health_check", err, startedAt)
	}
	emit(pb.DeployStage_DEPLOY_STAGE_HEALTH_CHECK, pb.DeployLogLevel_DEPLOY_LOG_LEVEL_INFO, "Health check passed")

	go e.cleanupOldImages(imageTag)

	completedAt := time.Now()
	emit(pb.DeployStage_DEPLOY_STAGE_COMPLETE, pb.DeployLogLevel_DEPLOY_LOG_LEVEL_INFO,
		fmt.Sprintf("Deployment completed in %s", completedAt.Sub(startedAt).Round(time.Millisecond)))

	return &pb.DeployResponse{
		Success: true,
		Message: "deployment completed successfully",
		Result: &pb.DeployResult{
			ImageTag:    imageTag,
			StartedAt:   timestamppb.New(startedAt),
			CompletedAt: timestamppb.New(completedAt),
			ExposedPort: int32(cfg.Port),
		},
	}
}

func (e *Executor) emitter(logFn LogFunc) func(pb.DeployStage, pb.DeployLogLevel, string) {
	return func(stage pb.DeployStage, level pb.DeployLogLevel, msg string) {
		if logFn != nil {
			logFn(stage, level, msg)
		}
	}
}

func (e *Executor) buildConfig(req *pb.DeployRequest) *compose.Config {
	cfg := &compose.Config{}
	cfg.Name = req.AppName

	if req.Build != nil {
		cfg.Build.Dockerfile = req.Build.Dockerfile
		cfg.Build.Context = req.Build.Context
		cfg.Build.Args = req.Build.Args
		cfg.Build.Target = req.Build.Target
	}

	if req.Runtime != nil {
		cfg.Port = int(req.Runtime.Port)
		if req.Runtime.HostPort != nil {
			cfg.HostPort = int(*req.Runtime.HostPort)
		}
		if req.Runtime.Resources != nil {
			cfg.Resources.Memory = req.Runtime.Resources.Memory
			cfg.Resources.CPU = req.Runtime.Resources.Cpu
		}
		cfg.Domains = req.Runtime.Domains
	}

	if req.HealthCheck != nil {
		cfg.Healthcheck.Path = req.HealthCheck.Path
		cfg.Healthcheck.Interval = req.HealthCheck.Interval
		cfg.Healthcheck.Timeout = req.HealthCheck.Timeout
		cfg.Healthcheck.Retries = int(req.HealthCheck.Retries)
		cfg.Healthcheck.StartPeriod = req.HealthCheck.StartPeriod
	}

	compose.ApplyDefaults(cfg)
	return cfg
}

func (e *Executor) resolveAppDir(repoDir, workdir string) string {
	if workdir == "" || workdir == "." {
		return repoDir
	}
	return filepath.Join(repoDir, workdir)
}

func (e *Executor) syncGit(ctx context.Context, req *pb.DeployRequest, repoDir string) error {
	gitCfg := req.Git
	if gitCfg == nil {
		return fmt.Errorf("git config is required")
	}

	token := ""
	if t, ok := gitCfg.Auth.(*pb.GitConfig_AccessToken); ok {
		token = t.AccessToken
	}

	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		e.logger.Info("Cloning repository", "url", gitCfg.RepositoryUrl, "target", repoDir)
		if err := os.MkdirAll(filepath.Dir(repoDir), 0755); err != nil {
			return fmt.Errorf("failed to create repo directory: %w", err)
		}
		if err := e.git.CloneWithToken(ctx, gitCfg.RepositoryUrl, repoDir, token); err != nil {
			return fmt.Errorf("clone failed: %w", err)
		}
	}

	e.logger.Info("Syncing repository", "commitSha", gitCfg.CommitSha)
	if err := e.git.SyncWithToken(ctx, repoDir, gitCfg.CommitSha, gitCfg.RepositoryUrl, token); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	return nil
}

func (e *Executor) buildImage(ctx context.Context, req *pb.DeployRequest, appDir, imageTag string, logFn LogFunc) error {
	dockerfile := "./Dockerfile"
	buildContext := "."

	var opts *docker.BuildOptions
	if req.Build != nil {
		if req.Build.Dockerfile != "" {
			dockerfile = req.Build.Dockerfile
		}
		if req.Build.Context != "" {
			buildContext = req.Build.Context
		}
		if len(req.Build.Args) > 0 || req.Build.Target != "" {
			opts = &docker.BuildOptions{
				BuildArgs: req.Build.Args,
				Target:    req.Build.Target,
			}
		}
	}

	fullContext := filepath.Join(appDir, buildContext)
	fullDockerfile := filepath.Join(appDir, dockerfile)

	e.logger.Info("Building Docker image", "imageTag", imageTag, "dockerfile", fullDockerfile)

	output := make(chan string, logChannelBuffer)
	go func() {
		for line := range output {
			if logFn != nil {
				logFn(pb.DeployStage_DEPLOY_STAGE_BUILD, pb.DeployLogLevel_DEPLOY_LOG_LEVEL_INFO, line)
			}
		}
	}()

	err := e.docker.BuildWithOptions(ctx, fullContext, fullDockerfile, imageTag, opts, output)
	close(output)
	return err
}

func (e *Executor) deployContainer(ctx context.Context, req *pb.DeployRequest, cfg *compose.Config, appDir, imageTag string, logFn LogFunc) error {
	if err := e.docker.EnsureNetwork(ctx, docker.DefaultNetworkName); err != nil {
		return fmt.Errorf("failed to ensure network: %w", err)
	}

	if err := e.docker.RemoveContainer(ctx, req.AppName, true); err != nil {
		e.logger.Warn("Failed to remove existing container", "appName", req.AppName, "error", err)
	}

	var domainRoutes []compose.DomainRoute
	for _, d := range cfg.Domains {
		domainRoutes = append(domainRoutes, compose.DomainRoute{Domain: d})
	}

	if err := compose.WriteComposeFile(appDir, compose.GenerateParams{
		AppName:  req.AppName,
		ImageTag: imageTag,
		Config:   cfg,
		Domains:  domainRoutes,
		EnvVars:  req.EnvVars,
	}); err != nil {
		return fmt.Errorf("failed to write docker-compose.yml: %w", err)
	}

	e.logger.Info("Starting container", "appName", req.AppName)

	output := make(chan string, logChannelBuffer)
	go func() {
		for line := range output {
			if logFn != nil {
				logFn(pb.DeployStage_DEPLOY_STAGE_DEPLOY, pb.DeployLogLevel_DEPLOY_LOG_LEVEL_INFO, line)
			}
		}
	}()

	err := e.docker.ComposeUp(ctx, appDir, req.AppId, output)
	close(output)
	return err
}

func (e *Executor) checkHealth(ctx context.Context, req *pb.DeployRequest, cfg *compose.Config) error {
	startDelay := healthCheckStartDelay
	if cfg.Healthcheck.StartPeriod != "" {
		if parsed, err := time.ParseDuration(cfg.Healthcheck.StartPeriod); err == nil && parsed > startDelay {
			startDelay = parsed
		}
	}

	e.logger.Info("Waiting for container startup", "delay", startDelay)
	time.Sleep(startDelay)

	containerIP, err := e.docker.GetContainerIP(ctx, req.AppName, docker.DefaultNetworkName)
	if err != nil {
		return fmt.Errorf("failed to get container IP: %w", err)
	}

	healthURL := fmt.Sprintf("http://%s:%d%s", containerIP, cfg.Port, cfg.Healthcheck.Path)
	e.logger.Info("Running health check", "url", healthURL)

	return e.health.CheckWithBackoff(ctx, healthURL)
}

func (e *Executor) rollback(ctx context.Context, req *pb.DeployRequest, appDir string) {
	if req.RollbackImage == nil || *req.RollbackImage == "" {
		e.logger.Info("No rollback image specified, skipping rollback")
		return
	}

	e.logger.Info("Rolling back deployment", "appName", req.AppName, "rollbackImage", *req.RollbackImage)
	if err := e.docker.ComposeDown(ctx, appDir, req.AppId); err != nil {
		e.logger.Error("Rollback compose down failed", "error", err)
	}
}

func (e *Executor) cleanupOldImages(currentTag string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	images, err := e.docker.ListImages(ctx, false)
	if err != nil {
		e.logger.Warn("Failed to list images for cleanup", "error", err)
		return
	}

	parts := strings.SplitN(currentTag, ":", 2)
	if len(parts) != 2 {
		return
	}
	currentRepo := parts[0]

	for _, img := range images {
		if img.Repository != currentRepo {
			continue
		}
		fullTag := img.Repository + ":" + img.Tag
		if fullTag == currentTag {
			continue
		}
		e.logger.Info("Removing old image", "tag", fullTag)
		_ = e.docker.RemoveImage(ctx, fullTag)
	}

	_ = e.docker.PruneUnusedImages(ctx)
}

func (e *Executor) UpdateDomains(ctx context.Context, req *pb.UpdateDomainsRequest) (*pb.UpdateDomainsResponse, error) {
	e.logger.Info("Updating container domains",
		"appId", req.AppId,
		"appName", req.AppName,
		"domains", len(req.Domains),
	)

	appDir := filepath.Join(e.dataDir, req.AppId)
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("app directory not found: %s", appDir)
	}

	containerHealth, err := e.docker.InspectContainer(ctx, req.AppName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}
	if containerHealth == nil {
		return nil, fmt.Errorf("container %s not found", req.AppName)
	}

	var domainRoutes []compose.DomainRoute
	for _, d := range req.Domains {
		domainRoutes = append(domainRoutes, compose.DomainRoute{
			Domain:     d.Domain,
			PathPrefix: d.PathPrefix,
		})
	}

	cfg := &compose.Config{}
	compose.ApplyDefaults(cfg)
	if req.Port > 0 {
		cfg.Port = int(req.Port)
	}

	content := compose.GenerateContent(compose.GenerateParams{
		AppName:  req.AppName,
		ImageTag: containerHealth.Image,
		Config:   cfg,
		Domains:  domainRoutes,
		EnvVars:  req.EnvVars,
	})

	composePath := filepath.Join(appDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write docker-compose.yml: %w", err)
	}

	if err := e.docker.ComposeUp(ctx, appDir, req.AppId, nil); err != nil {
		return nil, fmt.Errorf("failed to recreate container: %w", err)
	}

	e.logger.Info("Container domains updated successfully",
		"appId", req.AppId,
		"appName", req.AppName,
		"domains", len(req.Domains),
	)

	return &pb.UpdateDomainsResponse{
		Success: true,
		Message: "Container domains updated successfully",
	}, nil
}

func (e *Executor) failResponse(req *pb.DeployRequest, code pb.DeployErrorCode, stage string, err error, startedAt time.Time) *pb.DeployResponse {
	e.logger.Error("Deployment failed",
		"deploymentId", req.DeploymentId,
		"appName", req.AppName,
		"stage", stage,
		"error", err,
	)

	return &pb.DeployResponse{
		Success: false,
		Message: fmt.Sprintf("deployment failed at stage %s: %s", stage, err.Error()),
		Error: &pb.DeployError{
			Code:    code,
			Message: err.Error(),
			Stage:   stage,
		},
		Result: &pb.DeployResult{
			StartedAt:   timestamppb.New(startedAt),
			CompletedAt: timestamppb.New(time.Now()),
		},
	}
}
