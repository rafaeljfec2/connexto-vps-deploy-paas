package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
)

type EnvVarHandler struct {
	envVarRepo domain.EnvVarRepository
	appRepo    domain.AppRepository
}

func NewEnvVarHandler(envVarRepo domain.EnvVarRepository, appRepo domain.AppRepository) *EnvVarHandler {
	return &EnvVarHandler{
		envVarRepo: envVarRepo,
		appRepo:    appRepo,
	}
}

func (h *EnvVarHandler) Register(app *fiber.App) {
	v1 := app.Group("/paas-deploy/v1")
	apps := v1.Group("/apps")

	apps.Get("/:id/env", h.ListEnvVars)
	apps.Post("/:id/env", h.CreateEnvVar)
	apps.Put("/:id/env/bulk", h.BulkUpsertEnvVars)
	apps.Put("/:id/env/:varId", h.UpdateEnvVar)
	apps.Delete("/:id/env/:varId", h.DeleteEnvVar)
}

func (h *EnvVarHandler) ListEnvVars(c *fiber.Ctx) error {
	appID := c.Params("id")

	_, err := h.appRepo.FindByID(appID)
	if err != nil {
		if err == domain.ErrNotFound {
			return response.NotFound(c, "App not found")
		}
		return response.InternalError(c)
	}

	vars, err := h.envVarRepo.FindByAppID(appID)
	if err != nil {
		return response.InternalError(c)
	}

	responses := make([]domain.EnvVarResponse, len(vars))
	for i, v := range vars {
		responses[i] = v.ToResponse()
	}

	return response.OK(c, responses)
}

func (h *EnvVarHandler) CreateEnvVar(c *fiber.Ctx) error {
	appID := c.Params("id")

	_, err := h.appRepo.FindByID(appID)
	if err != nil {
		if err == domain.ErrNotFound {
			return response.NotFound(c, "App not found")
		}
		return response.InternalError(c)
	}

	var input domain.CreateEnvVarInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if input.Key == "" {
		return response.BadRequest(c, "Key is required")
	}

	envVar, err := h.envVarRepo.Create(appID, input)
	if err != nil {
		return response.InternalError(c)
	}

	return response.Created(c, envVar.ToResponse())
}

func (h *EnvVarHandler) BulkUpsertEnvVars(c *fiber.Ctx) error {
	appID := c.Params("id")

	_, err := h.appRepo.FindByID(appID)
	if err != nil {
		if err == domain.ErrNotFound {
			return response.NotFound(c, "App not found")
		}
		return response.InternalError(c)
	}

	var input domain.BulkEnvVarInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if len(input.Vars) == 0 {
		return response.BadRequest(c, "At least one variable is required")
	}

	for _, v := range input.Vars {
		if v.Key == "" {
			return response.BadRequest(c, "All variables must have a key")
		}
	}

	if err := h.envVarRepo.BulkUpsert(appID, input.Vars); err != nil {
		return response.InternalError(c)
	}

	vars, err := h.envVarRepo.FindByAppID(appID)
	if err != nil {
		return response.InternalError(c)
	}

	responses := make([]domain.EnvVarResponse, len(vars))
	for i, v := range vars {
		responses[i] = v.ToResponse()
	}

	return response.OK(c, responses)
}

func (h *EnvVarHandler) UpdateEnvVar(c *fiber.Ctx) error {
	varID := c.Params("varId")

	var input domain.UpdateEnvVarInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	envVar, err := h.envVarRepo.Update(varID, input)
	if err != nil {
		if err == domain.ErrNotFound {
			return response.NotFound(c, "Environment variable not found")
		}
		return response.InternalError(c)
	}

	return response.OK(c, envVar.ToResponse())
}

func (h *EnvVarHandler) DeleteEnvVar(c *fiber.Ctx) error {
	varID := c.Params("varId")

	if err := h.envVarRepo.Delete(varID); err != nil {
		if err == domain.ErrNotFound {
			return response.NotFound(c, "Environment variable not found")
		}
		return response.InternalError(c)
	}

	return response.OK(c, map[string]string{"message": "Environment variable deleted"})
}
