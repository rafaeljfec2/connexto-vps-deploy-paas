package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/engine"
	"github.com/paasdeploy/backend/internal/response"
)

type AppAdminHandler struct {
	appRepo domain.AppRepository
	engine  *engine.Engine
	dataDir string
	logger  *slog.Logger
}

func NewAppAdminHandler(appRepo domain.AppRepository, eng *engine.Engine, dataDir string, logger *slog.Logger) *AppAdminHandler {
	return &AppAdminHandler{
		appRepo: appRepo,
		engine:  eng,
		dataDir: dataDir,
		logger:  logger.With("handler", "app_admin"),
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

func (h *AppAdminHandler) executeContainerAction(c *fiber.Ctx, action containerAction) error {
	app, err := h.requireAppForUser(c)
	if err != nil {
		return err
	}

	if err := action.do(c.Context(), app.Name); err != nil {
		h.logger.Error("Failed to "+action.name+" container", "appId", app.ID, "appName", app.Name, "error", err)
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
