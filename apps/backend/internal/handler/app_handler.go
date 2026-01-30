package handler

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/backend/internal/service"
)

type AppHandler struct {
	appService *service.AppService
}

func NewAppHandler(appService *service.AppService) *AppHandler {
	return &AppHandler{
		appService: appService,
	}
}

func (h *AppHandler) Register(app *fiber.App) {
	v1 := app.Group("/paas-deploy/v1")

	apps := v1.Group("/apps")
	apps.Get("/", h.ListApps)
	apps.Post("/", h.CreateApp)
	apps.Get("/:id", h.GetApp)
	apps.Delete("/:id", h.DeleteApp)
	apps.Get("/:id/deployments", h.ListDeployments)
	apps.Post("/:id/redeploy", h.TriggerRedeploy)
	apps.Post("/:id/rollback", h.TriggerRollback)

	apps.Post("/:id/webhook", h.SetupWebhook)
	apps.Delete("/:id/webhook", h.RemoveWebhook)
	apps.Get("/:id/webhook/status", h.GetWebhookStatus)
}

func (h *AppHandler) ListApps(c *fiber.Ctx) error {
	apps, err := h.appService.ListApps()
	if err != nil {
		return h.handleError(c, err)
	}

	return response.OK(c, apps)
}

func (h *AppHandler) CreateApp(c *fiber.Ctx) error {
	var input domain.CreateAppInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	ctx := context.Background()
	app, err := h.appService.CreateApp(ctx, input)
	if err != nil {
		return h.handleError(c, err)
	}

	return response.Created(c, app)
}

func (h *AppHandler) GetApp(c *fiber.Ctx) error {
	id := c.Params("id")

	app, err := h.appService.GetApp(id)
	if err != nil {
		return h.handleError(c, err)
	}

	return response.OK(c, app)
}

func (h *AppHandler) DeleteApp(c *fiber.Ctx) error {
	id := c.Params("id")

	ctx := context.Background()
	if err := h.appService.DeleteApp(ctx, id); err != nil {
		return h.handleError(c, err)
	}

	return response.NoContent(c)
}

func (h *AppHandler) ListDeployments(c *fiber.Ctx) error {
	appID := c.Params("id")

	deployments, err := h.appService.ListDeployments(appID)
	if err != nil {
		return h.handleError(c, err)
	}

	return response.OK(c, deployments)
}

func (h *AppHandler) TriggerRedeploy(c *fiber.Ctx) error {
	appID := c.Params("id")

	var input struct {
		CommitSHA string `json:"commitSha,omitempty"`
	}
	_ = c.BodyParser(&input)

	deployment, err := h.appService.TriggerDeploy(appID, input.CommitSHA)
	if err != nil {
		return h.handleError(c, err)
	}

	return response.Created(c, deployment)
}

func (h *AppHandler) TriggerRollback(c *fiber.Ctx) error {
	appID := c.Params("id")

	deployment, err := h.appService.TriggerRollback(appID)
	if err != nil {
		return h.handleError(c, err)
	}

	return response.Created(c, deployment)
}

func (h *AppHandler) SetupWebhook(c *fiber.Ctx) error {
	id := c.Params("id")

	ctx := context.Background()
	result, err := h.appService.SetupWebhook(ctx, id)
	if err != nil {
		return h.handleError(c, err)
	}

	return response.OK(c, result)
}

func (h *AppHandler) RemoveWebhook(c *fiber.Ctx) error {
	id := c.Params("id")

	ctx := context.Background()
	if err := h.appService.RemoveWebhook(ctx, id); err != nil {
		return h.handleError(c, err)
	}

	return response.NoContent(c)
}

func (h *AppHandler) GetWebhookStatus(c *fiber.Ctx) error {
	id := c.Params("id")

	ctx := context.Background()
	status, err := h.appService.GetWebhookStatus(ctx, id)
	if err != nil {
		return h.handleError(c, err)
	}

	return response.OK(c, status)
}

func (h *AppHandler) handleError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return response.NotFound(c, err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		return response.Conflict(c, err.Error())
	case errors.Is(err, domain.ErrInvalidInput):
		return response.BadRequest(c, err.Error())
	case errors.Is(err, domain.ErrDeployInProgress):
		return response.Conflict(c, err.Error())
	case errors.Is(err, domain.ErrNoDeployAvailable):
		return response.NotFound(c, err.Error())
	case errors.Is(err, domain.ErrWebhookNotConfigured):
		return response.BadRequest(c, err.Error())
	default:
		return response.InternalError(c)
	}
}
