package handler

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/engine"
	"github.com/paasdeploy/backend/internal/response"
)

type ContainerHealthResponse struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Health    string `json:"health"`
	StartedAt string `json:"startedAt,omitempty"`
	Uptime    string `json:"uptime,omitempty"`
}

type ContainerHealthHandler struct {
	appRepo domain.AppRepository
	engine  *engine.Engine
}

func NewContainerHealthHandler(appRepo domain.AppRepository, eng *engine.Engine) *ContainerHealthHandler {
	return &ContainerHealthHandler{
		appRepo: appRepo,
		engine:  eng,
	}
}

func (h *ContainerHealthHandler) Register(app *fiber.App) {
	v1 := app.Group("/paas-deploy/v1")
	v1.Get("/apps/:id/health", h.GetAppHealth)
}

func (h *ContainerHealthHandler) GetAppHealth(c *fiber.Ctx) error {
	id := c.Params("id")

	app, err := h.appRepo.FindByID(id)
	if err != nil {
		return response.NotFound(c, "app not found")
	}

	ctx := context.Background()
	health := h.engine.GetAppHealth(ctx, app.Name)

	if health == nil {
		return response.OK(c, ContainerHealthResponse{
			Name:   app.Name,
			Status: "unknown",
			Health: "none",
		})
	}

	return response.OK(c, ContainerHealthResponse{
		Name:      health.Name,
		Status:    health.Status,
		Health:    health.Health,
		StartedAt: health.StartedAt,
		Uptime:    health.Uptime,
	})
}
