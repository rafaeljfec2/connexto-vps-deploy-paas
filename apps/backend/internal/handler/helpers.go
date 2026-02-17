package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
)

func isKnownDomainError(err error) bool {
	return errors.Is(err, domain.ErrNotFound) ||
		errors.Is(err, domain.ErrAlreadyExists) ||
		errors.Is(err, domain.ErrInvalidInput) ||
		errors.Is(err, domain.ErrDeployInProgress) ||
		errors.Is(err, domain.ErrNoDeployAvailable) ||
		errors.Is(err, domain.ErrWebhookNotConfigured) ||
		errors.Is(err, domain.ErrForbidden)
}

func HandleDomainError(c *fiber.Ctx, err error) error {
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
	case errors.Is(err, domain.ErrForbidden):
		return response.Forbidden(c, err.Error())
	default:
		return response.InternalError(c)
	}
}

func HandleNotFoundOrInternal(c *fiber.Ctx, err error, notFoundMsg string) error {
	if errors.Is(err, domain.ErrNotFound) {
		return response.NotFound(c, notFoundMsg)
	}
	return response.InternalError(c)
}

func EnsureAppExists(c *fiber.Ctx, appRepo domain.AppRepository, appID string) error {
	_, err := appRepo.FindByID(appID)
	if err != nil {
		return HandleNotFoundOrInternal(c, err, MsgAppNotFound)
	}
	return nil
}

func EnsureAppOwnership(c *fiber.Ctx, appRepo domain.AppRepository, appID string) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}
	_, err := appRepo.FindByIDAndUserID(appID, user.ID)
	if err != nil {
		return HandleNotFoundOrInternal(c, err, MsgAppNotFound)
	}
	return nil
}

func ToEnvVarResponses(vars []domain.EnvVar) []domain.EnvVarResponse {
	responses := make([]domain.EnvVarResponse, len(vars))
	for i, v := range vars {
		responses[i] = v.ToResponse()
	}
	return responses
}
