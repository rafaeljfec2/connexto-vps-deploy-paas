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

func (h *EnvVarHandler) Register(app fiber.Router) {
	v1 := app.Group(APIPrefix)
	apps := v1.Group("/apps")

	apps.Get("/:id/env", h.ListEnvVars)
	apps.Post("/:id/env", h.CreateEnvVar)
	apps.Put("/:id/env/bulk", h.BulkUpsertEnvVars)
	apps.Put("/:id/env/:varId", h.UpdateEnvVar)
	apps.Delete("/:id/env/:varId", h.DeleteEnvVar)
}

func (h *EnvVarHandler) ListEnvVars(c *fiber.Ctx) error {
	appID := c.Params("id")

	if err := EnsureAppOwnership(c, h.appRepo, appID); err != nil {
		return err
	}

	vars, err := h.envVarRepo.FindByAppID(appID)
	if err != nil {
		return response.InternalError(c)
	}

	return response.OK(c, ToEnvVarResponses(vars))
}

func (h *EnvVarHandler) CreateEnvVar(c *fiber.Ctx) error {
	appID := c.Params("id")

	if err := EnsureAppOwnership(c, h.appRepo, appID); err != nil {
		return err
	}

	var input domain.CreateEnvVarInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, MsgInvalidRequestBody)
	}

	if input.Key == "" {
		return response.BadRequest(c, MsgKeyRequired)
	}

	envVar, err := h.envVarRepo.Create(appID, input)
	if err != nil {
		return response.InternalError(c)
	}

	return response.Created(c, envVar.ToResponse())
}

func (h *EnvVarHandler) BulkUpsertEnvVars(c *fiber.Ctx) error {
	appID := c.Params("id")

	if err := EnsureAppOwnership(c, h.appRepo, appID); err != nil {
		return err
	}

	var input domain.BulkEnvVarInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, MsgInvalidRequestBody)
	}

	if len(input.Vars) == 0 {
		return response.BadRequest(c, MsgAtLeastOneVariable)
	}

	for _, v := range input.Vars {
		if v.Key == "" {
			return response.BadRequest(c, MsgAllVarsMustHaveKey)
		}
	}

	if err := h.envVarRepo.BulkUpsert(appID, input.Vars); err != nil {
		return response.InternalError(c)
	}

	vars, err := h.envVarRepo.FindByAppID(appID)
	if err != nil {
		return response.InternalError(c)
	}

	return response.OK(c, ToEnvVarResponses(vars))
}

func (h *EnvVarHandler) UpdateEnvVar(c *fiber.Ctx) error {
	appID := c.Params("id")
	if err := EnsureAppOwnership(c, h.appRepo, appID); err != nil {
		return err
	}

	varID := c.Params("varId")

	var input domain.UpdateEnvVarInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, MsgInvalidRequestBody)
	}

	envVar, err := h.envVarRepo.Update(varID, input)
	if err != nil {
		return HandleNotFoundOrInternal(c, err, MsgEnvVarNotFound)
	}

	return response.OK(c, envVar.ToResponse())
}

func (h *EnvVarHandler) DeleteEnvVar(c *fiber.Ctx) error {
	appID := c.Params("id")
	if err := EnsureAppOwnership(c, h.appRepo, appID); err != nil {
		return err
	}

	varID := c.Params("varId")

	if err := h.envVarRepo.Delete(varID); err != nil {
		return HandleNotFoundOrInternal(c, err, MsgEnvVarNotFound)
	}

	return response.OK(c, map[string]string{"message": MsgEnvVarDeleted})
}
