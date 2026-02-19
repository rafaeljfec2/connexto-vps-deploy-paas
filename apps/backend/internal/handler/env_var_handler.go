package handler

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
)

type EnvVarHandler struct {
	envVarRepo domain.EnvVarRepository
	appRepo    domain.AppRepository
	logger     *slog.Logger
}

func NewEnvVarHandler(envVarRepo domain.EnvVarRepository, appRepo domain.AppRepository, logger *slog.Logger) *EnvVarHandler {
	return &EnvVarHandler{
		envVarRepo: envVarRepo,
		appRepo:    appRepo,
		logger:     logger,
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
		h.logger.Error("Failed to list env vars", "appId", appID, "error", err)
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
		if isDuplicateKeyError(err) {
			return response.BadRequest(c, "Environment variable '"+input.Key+"' already exists")
		}
		h.logger.Error("Failed to create env var", "appId", appID, "key", input.Key, "error", err)
		return response.InternalError(c)
	}

	return response.Created(c, envVar.ToResponse())
}

func isDuplicateKeyError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint")
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
		h.logger.Error("Failed to bulk upsert env vars", "appId", appID, "error", err)
		return response.InternalError(c)
	}

	vars, err := h.envVarRepo.FindByAppID(appID)
	if err != nil {
		h.logger.Error("Failed to list env vars after bulk upsert", "appId", appID, "error", err)
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
