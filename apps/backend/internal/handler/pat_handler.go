package handler

import (
	"errors"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/middleware"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/backend/internal/service"
)

const denyPATOnTokensReason = "personal access tokens cannot manage other tokens; use a session login"

type PATHandler struct {
	service *service.PersonalAccessTokenService
}

func NewPATHandler(svc *service.PersonalAccessTokenService) *PATHandler {
	return &PATHandler{service: svc}
}

func (h *PATHandler) Register(router fiber.Router) {
	tokens := router.Group("/tokens", middleware.DenyPAT(denyPATOnTokensReason))
	tokens.Get("/", h.List)
	tokens.Post("/", h.Create)
	tokens.Delete("/:id", h.Revoke)
}

type TokenResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	TokenPrefix string     `json:"tokenPrefix"`
	Scopes      []string   `json:"scopes"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	RevokedAt   *time.Time `json:"revokedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type CreateTokenRequest struct {
	Name      string     `json:"name"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type CreateTokenResponse struct {
	Token          TokenResponse `json:"token"`
	PlaintextToken string        `json:"plaintextToken"`
}

type ListTokensResponse struct {
	Tokens []TokenResponse `json:"tokens"`
}

func (h *PATHandler) List(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	tokens, err := h.service.List(c.Context(), user.ID)
	if err != nil {
		slog.Default().Error("pat list failed", "user_id", user.ID, "error", err)
		return response.InternalError(c)
	}

	resp := ListTokensResponse{Tokens: make([]TokenResponse, len(tokens))}
	for i, t := range tokens {
		resp.Tokens[i] = toTokenResponse(&t)
	}
	return response.OK(c, resp)
}

func (h *PATHandler) Create(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	var req CreateTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, MsgInvalidRequestBody)
	}

	result, err := h.service.Create(c.Context(), service.CreateTokenInput{
		UserID:    user.ID,
		UserRole:  user.Role,
		Name:      req.Name,
		Scopes:    req.Scopes,
		ExpiresAt: req.ExpiresAt,
	})
	if err != nil {
		return mapPATError(c, err)
	}

	return response.Created(c, CreateTokenResponse{
		Token:          toTokenResponse(result.Token),
		PlaintextToken: result.PlaintextToken,
	})
}

func (h *PATHandler) Revoke(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c, "token id is required")
	}

	if err := h.service.Revoke(c.Context(), id, user.ID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "token not found")
		}
		slog.Default().Error("pat revoke failed", "user_id", user.ID, "token_id", id, "error", err)
		return response.InternalError(c)
	}

	return response.NoContent(c)
}

func toTokenResponse(t *domain.PersonalAccessToken) TokenResponse {
	return TokenResponse{
		ID:          t.ID,
		Name:        t.Name,
		TokenPrefix: t.TokenPrefix,
		Scopes:      t.Scopes,
		LastUsedAt:  t.LastUsedAt,
		ExpiresAt:   t.ExpiresAt,
		RevokedAt:   t.RevokedAt,
		CreatedAt:   t.CreatedAt,
	}
}

func mapPATError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidTokenName):
		return response.BadRequest(c, "token name must be between 3 and 120 characters")
	case errors.Is(err, service.ErrNoScopesProvided):
		return response.BadRequest(c, "at least one scope is required")
	case errors.Is(err, service.ErrInvalidScope):
		return response.BadRequestWithDetails(c, "invalid scope", map[string]any{
			"validScopes": domain.AllScopes,
		})
	case errors.Is(err, service.ErrScopeNotAllowed):
		return response.Forbidden(c, "scope not allowed for your user role")
	case errors.Is(err, service.ErrExpiryOutOfRange):
		return response.BadRequest(c, "expiry must be between 1 hour and 365 days in the future")
	default:
		slog.Default().Error("pat create failed", "error", err)
		return response.InternalError(c)
	}
}
