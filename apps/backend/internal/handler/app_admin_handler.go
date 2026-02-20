package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/engine"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/shared/pkg/compose"
)

type AppAdminHandler struct {
	appRepo          domain.AppRepository
	serverRepo       domain.ServerRepository
	customDomainRepo domain.CustomDomainRepository
	envVarRepo       domain.EnvVarRepository
	engine           *engine.Engine
	agentClient      *agentclient.AgentClient
	agentPort        int
	dataDir          string
	logger           *slog.Logger
}

type AppAdminHandlerConfig struct {
	AppRepo          domain.AppRepository
	ServerRepo       domain.ServerRepository
	CustomDomainRepo domain.CustomDomainRepository
	EnvVarRepo       domain.EnvVarRepository
	Engine           *engine.Engine
	AgentClient      *agentclient.AgentClient
	AgentPort        int
	DataDir          string
	Logger           *slog.Logger
}

func NewAppAdminHandler(cfg AppAdminHandlerConfig) *AppAdminHandler {
	return &AppAdminHandler{
		appRepo:          cfg.AppRepo,
		serverRepo:       cfg.ServerRepo,
		customDomainRepo: cfg.CustomDomainRepo,
		envVarRepo:       cfg.EnvVarRepo,
		engine:           cfg.Engine,
		agentClient:      cfg.AgentClient,
		agentPort:        cfg.AgentPort,
		dataDir:          cfg.DataDir,
		logger:           cfg.Logger.With("handler", "app_admin"),
	}
}

func (h *AppAdminHandler) Register(app fiber.Router) {
	v1 := app.Group(APIPrefix)

	apps := v1.Group("/apps")
	apps.Get("/:id/url", h.GetAppURL)
	apps.Get("/:id/config", h.GetAppConfig)
	apps.Post("/:id/container/restart", h.RestartContainer)
	apps.Post("/:id/container/stop", h.StopContainer)
	apps.Post("/:id/container/start", h.StartContainer)
	apps.Patch("/:id", h.UpdateApp)
}

func (h *AppAdminHandler) requireAppForUser(c *fiber.Ctx) (*domain.App, error) {
	user := GetUserFromContext(c)
	if user == nil {
		return nil, response.Unauthorized(c, MsgNotAuthenticated)
	}

	id := c.Params("id")
	app, err := h.appRepo.FindByIDAndUserID(id, user.ID)
	if err != nil {
		return nil, response.NotFound(c, MsgAppNotFound)
	}
	return app, nil
}

type AppURLResponse struct {
	URL      string `json:"url"`
	Port     int    `json:"port"`
	HostPort int    `json:"hostPort"`
}

func (h *AppAdminHandler) GetAppURL(c *fiber.Ctx) error {
	app, err := h.requireAppForUser(c)
	if err != nil {
		return err
	}

	if h.isRemoteApp(app) {
		return h.getRemoteAppURL(c, app)
	}

	config, err := h.readAppConfig(app.ID, app.Workdir)
	if err != nil {
		return response.OK(c, AppURLResponse{
			URL:      "",
			Port:     0,
			HostPort: 0,
		})
	}

	port := config.Port
	hostPort := config.HostPort
	if hostPort == 0 {
		hostPort = port
	}

	url := fmt.Sprintf("http://localhost:%d", hostPort)

	return response.OK(c, AppURLResponse{
		URL:      url,
		Port:     port,
		HostPort: hostPort,
	})
}

func (h *AppAdminHandler) getRemoteAppURL(c *fiber.Ctx, app *domain.App) error {
	port := h.resolveAppPort(app.ID)

	domains, _ := h.customDomainRepo.FindByAppID(c.Context(), app.ID)
	if len(domains) > 0 {
		url := fmt.Sprintf("https://%s", domains[0].Domain)
		if domains[0].PathPrefix != "" && domains[0].PathPrefix != "/" {
			url += domains[0].PathPrefix
		}
		return response.OK(c, AppURLResponse{
			URL:      url,
			Port:     port,
			HostPort: port,
		})
	}

	return response.OK(c, AppURLResponse{
		URL:      "",
		Port:     port,
		HostPort: port,
	})
}

type AppConfigResponse struct {
	Name        string            `json:"name"`
	Port        int               `json:"port"`
	HostPort    int               `json:"hostPort"`
	Healthcheck HealthcheckConfig `json:"healthcheck"`
	Resources   ResourcesConfig   `json:"resources"`
	Domains     []string          `json:"domains"`
}

type HealthcheckConfig struct {
	Path        string `json:"path"`
	Interval    string `json:"interval"`
	Timeout     string `json:"timeout"`
	Retries     int    `json:"retries"`
	StartPeriod string `json:"startPeriod"`
}

type ResourcesConfig struct {
	Memory string `json:"memory"`
	CPU    string `json:"cpu"`
}

func (h *AppAdminHandler) GetAppConfig(c *fiber.Ctx) error {
	app, err := h.requireAppForUser(c)
	if err != nil {
		return err
	}

	if h.isRemoteApp(app) {
		return h.getRemoteAppConfig(c, app)
	}

	config, err := h.readAppConfig(app.ID, app.Workdir)
	if err != nil {
		return response.NotFound(c, "config not found - app may not be deployed yet")
	}

	return response.OK(c, AppConfigResponse{
		Name:     config.Name,
		Port:     config.Port,
		HostPort: config.HostPort,
		Healthcheck: HealthcheckConfig{
			Path:        config.Healthcheck.Path,
			Interval:    config.Healthcheck.Interval,
			Timeout:     config.Healthcheck.Timeout,
			Retries:     config.Healthcheck.Retries,
			StartPeriod: config.Healthcheck.StartPeriod,
		},
		Resources: ResourcesConfig{
			Memory: config.Resources.Memory,
			CPU:    config.Resources.CPU,
		},
		Domains: config.Domains,
	})
}

func (h *AppAdminHandler) getRemoteAppConfig(c *fiber.Ctx, app *domain.App) error {
	defaults := &compose.Config{}
	compose.ApplyDefaults(defaults)

	port := h.resolveAppPort(app.ID)

	var domainNames []string
	domains, _ := h.customDomainRepo.FindByAppID(c.Context(), app.ID)
	for _, d := range domains {
		domainNames = append(domainNames, d.Domain)
	}

	return response.OK(c, AppConfigResponse{
		Name:     app.Name,
		Port:     port,
		HostPort: port,
		Healthcheck: HealthcheckConfig{
			Path:        defaults.Healthcheck.Path,
			Interval:    defaults.Healthcheck.Interval,
			Timeout:     defaults.Healthcheck.Timeout,
			Retries:     defaults.Healthcheck.Retries,
			StartPeriod: defaults.Healthcheck.StartPeriod,
		},
		Resources: ResourcesConfig{
			Memory: defaults.Resources.Memory,
			CPU:    defaults.Resources.CPU,
		},
		Domains: domainNames,
	})
}

func (h *AppAdminHandler) resolveAppPort(appID string) int {
	if h.envVarRepo != nil {
		vars, err := h.envVarRepo.FindByAppID(appID)
		if err == nil {
			for _, v := range vars {
				if v.Key == "PORT" {
					if parsed, err := strconv.Atoi(v.Value); err == nil && parsed > 0 {
						return parsed
					}
				}
			}
		}
	}

	defaults := &compose.Config{}
	compose.ApplyDefaults(defaults)
	return defaults.Port
}

type ContainerActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type containerAction struct {
	name    string
	do      func(context.Context, string) error
	success string
	fail    string
}

func (h *AppAdminHandler) isRemoteApp(app *domain.App) bool {
	return app.ServerID != nil && *app.ServerID != ""
}

func (h *AppAdminHandler) resolveServerHost(app *domain.App) (string, error) {
	if app.ServerID == nil {
		return "", fmt.Errorf("app has no server")
	}
	server, err := h.serverRepo.FindByID(*app.ServerID)
	if err != nil {
		return "", fmt.Errorf("server not found: %w", err)
	}
	return server.Host, nil
}

func (h *AppAdminHandler) executeContainerAction(c *fiber.Ctx, action containerAction) error {
	app, err := h.requireAppForUser(c)
	if err != nil {
		return err
	}

	var execErr error
	if h.isRemoteApp(app) {
		host, hostErr := h.resolveServerHost(app)
		if hostErr != nil {
			execErr = hostErr
		} else {
			switch action.name {
			case "restart":
				execErr = h.agentClient.RestartContainer(c.Context(), host, h.agentPort, app.Name)
			case "stop":
				execErr = h.agentClient.StopContainer(c.Context(), host, h.agentPort, app.Name)
			case "start":
				execErr = h.agentClient.StartContainer(c.Context(), host, h.agentPort, app.Name)
			default:
				execErr = fmt.Errorf("unknown action: %s", action.name)
			}
		}
	} else {
		execErr = action.do(c.Context(), app.Name)
	}

	if execErr != nil {
		h.logger.Error("Failed to "+action.name+" container", "appId", app.ID, "appName", app.Name, "error", execErr)
		return response.OK(c, ContainerActionResponse{
			Success: false,
			Message: action.fail,
		})
	}

	return response.OK(c, ContainerActionResponse{
		Success: true,
		Message: action.success,
	})
}

func (h *AppAdminHandler) RestartContainer(c *fiber.Ctx) error {
	return h.executeContainerAction(c, containerAction{
		name:    "restart",
		do:      h.engine.RestartContainer,
		success: "Container restarted successfully",
		fail:    "Failed to restart container",
	})
}

func (h *AppAdminHandler) StopContainer(c *fiber.Ctx) error {
	return h.executeContainerAction(c, containerAction{
		name:    "stop",
		do:      h.engine.StopContainer,
		success: "Container stopped successfully",
		fail:    "Failed to stop container",
	})
}

func (h *AppAdminHandler) StartContainer(c *fiber.Ctx) error {
	return h.executeContainerAction(c, containerAction{
		name:    "start",
		do:      h.engine.StartContainer,
		success: "Container started successfully",
		fail:    "Failed to start container",
	})
}

type UpdateAppInput struct {
	Branch  *string `json:"branch,omitempty"`
	Workdir *string `json:"workdir,omitempty"`
}

func (h *AppAdminHandler) UpdateApp(c *fiber.Ctx) error {
	app, err := h.requireAppForUser(c)
	if err != nil {
		return err
	}

	var input UpdateAppInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, MsgInvalidRequestBody)
	}

	updateInput := domain.UpdateAppInput{}
	if input.Branch != nil {
		updateInput.Branch = input.Branch
	}
	if input.Workdir != nil {
		updateInput.Workdir = input.Workdir
	}

	updatedApp, err := h.appRepo.Update(app.ID, updateInput)
	if err != nil {
		return response.InternalError(c)
	}

	return response.OK(c, updatedApp)
}

type paasDeployConfig struct {
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
	HostPort  int               `json:"hostPort,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Resources struct {
		Memory string `json:"memory"`
		CPU    string `json:"cpu"`
	} `json:"resources"`
	Domains []string `json:"domains,omitempty"`
}

func (h *AppAdminHandler) readAppConfig(appID, workdir string) (*paasDeployConfig, error) {
	repoDir := filepath.Join(h.dataDir, appID)

	var appDir string
	if workdir == "" || workdir == "." {
		appDir = repoDir
	} else {
		appDir = filepath.Join(repoDir, workdir)
	}

	configPath := filepath.Join(appDir, "paasdeploy.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read paasdeploy.json: %w", err)
	}

	var config paasDeployConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid paasdeploy.json: %w", err)
	}

	if config.Port == 0 {
		config.Port = 8080
	}

	return &config, nil
}
