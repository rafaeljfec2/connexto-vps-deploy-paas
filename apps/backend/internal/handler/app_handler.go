package handler

import (
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
	api := app.Group("/api")

	apps := api.Group("/apps")
	apps.Get("/", h.ListApps)
	apps.Post("/", h.CreateApp)
	apps.Get("/:id", h.GetApp)
	apps.Delete("/:id", h.DeleteApp)
	apps.Get("/:id/deployments", h.ListDeployments)
	apps.Post("/:id/redeploy", h.TriggerRedeploy)
	apps.Post("/:id/rollback", h.TriggerRollback)
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

	app, err := h.appService.CreateApp(input)
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

	if err := h.appService.DeleteApp(id); err != nil {
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
	default:
		return response.InternalError(c)
	}
}
