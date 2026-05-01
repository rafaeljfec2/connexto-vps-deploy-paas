package handler

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
)

// ParseSinceQuery reads the optional ?since= query parameter and returns it
// as a *time.Time. The expected format is RFC3339 (e.g. "2026-04-30T19:00:00Z");
// duration shorthand is the MCP layer's responsibility to canonicalise before
// hitting the backend. Returns (nil, nil) when the param is absent — callers
// then forward nil downstream meaning "no time filter".
func ParseSinceQuery(c *fiber.Ctx) (*time.Time, error) {
	raw := c.Query("since", "")
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, fmt.Errorf("invalid since: must be RFC3339 (e.g. 2026-04-30T19:00:00Z)")
	}
	return &t, nil
}

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
		return response.NotFound(c, "resource not found")
	case errors.Is(err, domain.ErrAlreadyExists):
		return response.Conflict(c, "resource already exists")
	case errors.Is(err, domain.ErrInvalidInput):
		return response.BadRequest(c, "invalid input")
	case errors.Is(err, domain.ErrDeployInProgress):
		return response.Conflict(c, "deployment already in progress for this app")
	case errors.Is(err, domain.ErrNoDeployAvailable):
		return response.NotFound(c, "no deployment available for rollback")
	case errors.Is(err, domain.ErrWebhookNotConfigured):
		return response.BadRequest(c, "webhook management not configured")
	case errors.Is(err, domain.ErrForbidden):
		return response.Forbidden(c, "forbidden")
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

func RequireAdminForLocal(c *fiber.Ctx, serverID string) error {
	if serverID != "" {
		return nil
	}
	user := GetUserFromContext(c)
	if user != nil && user.IsAdmin() {
		return nil
	}
	return response.Forbidden(c, "local operations require admin role")
}

func ToEnvVarResponses(vars []domain.EnvVar) []domain.EnvVarResponse {
	responses := make([]domain.EnvVarResponse, len(vars))
	for i, v := range vars {
		responses[i] = v.ToResponse()
	}
	return responses
}
