package middleware

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/crypto"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/requestctx"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/backend/internal/service"
)

const (
	tokenContextKey  = "pat"
	bearerHeaderKey  = "Authorization"
	bearerTokenType  = "Bearer "
	mcpClientHeader  = "X-MCP-Client"
	mcpClientContext = "mcpClient"
)

type AuthMiddleware struct {
	sessionRepo       domain.SessionRepository
	userRepo          domain.UserRepository
	patService        *service.PersonalAccessTokenService
	logger            *slog.Logger
	sessionCookieName string
}

type AuthMiddlewareConfig struct {
	SessionRepo       domain.SessionRepository
	UserRepo          domain.UserRepository
	PATService        *service.PersonalAccessTokenService
	Logger            *slog.Logger
	SessionCookieName string
}

func NewAuthMiddleware(cfg AuthMiddlewareConfig) *AuthMiddleware {
	return &AuthMiddleware{
		sessionRepo:       cfg.SessionRepo,
		userRepo:          cfg.UserRepo,
		patService:        cfg.PATService,
		logger:            cfg.Logger,
		sessionCookieName: cfg.SessionCookieName,
	}
}

func (m *AuthMiddleware) Require() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if ok, err := m.tryBearer(c); ok {
			return err
		} else if err != nil {
			return err
		}

		return m.requireSession(c)
	}
}

func (m *AuthMiddleware) Optional() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if ok, err := m.tryBearer(c); ok {
			return err
		} else if err != nil {
			return c.Next()
		}

		return m.optionalSession(c)
	}
}

func (m *AuthMiddleware) tryBearer(c *fiber.Ctx) (handled bool, err error) {
	if m.patService == nil {
		return false, nil
	}
	header := c.Get(bearerHeaderKey)
	if header == "" || !strings.HasPrefix(header, bearerTokenType) {
		return false, nil
	}

	plaintext := strings.TrimSpace(strings.TrimPrefix(header, bearerTokenType))
	if plaintext == "" {
		return true, response.Unauthorized(c, "invalid bearer token")
	}

	token, authErr := m.patService.Authenticate(c.Context(), plaintext)
	if authErr != nil {
		m.logger.Debug("bearer token rejected", "error", authErr)
		switch {
		case errors.Is(authErr, service.ErrMalformedTokenString):
			return true, response.Unauthorized(c, "malformed bearer token")
		case errors.Is(authErr, domain.ErrTokenExpired):
			return true, response.Unauthorized(c, "token expired")
		case errors.Is(authErr, domain.ErrTokenRevoked):
			return true, response.Unauthorized(c, "token revoked")
		case errors.Is(authErr, domain.ErrNotFound):
			return true, response.Unauthorized(c, "invalid token")
		default:
			return true, response.Unauthorized(c, "authentication failed")
		}
	}

	user, err := m.userRepo.FindByID(c.Context(), token.UserID)
	if err != nil {
		m.logger.Error("user not found for valid token", "error", err, "user_id", token.UserID, "token_id", token.ID)
		return true, response.Unauthorized(c, "user not found")
	}

	requestctx.SetUserInContext(c, user)
	c.Locals(tokenContextKey, token)
	if client := c.Get(mcpClientHeader); client != "" {
		c.Locals(mcpClientContext, client)
	}

	tokenID := token.ID
	go func() {
		defer func() {
			if r := recover(); r != nil {
				m.logger.Error("touch last used panic", "tokenId", tokenID, "panic", r)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		m.patService.TouchLastUsed(ctx, tokenID)
	}()

	return true, c.Next()
}

func (m *AuthMiddleware) requireSession(c *fiber.Ctx) error {
	sessionToken := c.Cookies(m.sessionCookieName)
	if sessionToken == "" {
		return response.Unauthorized(c, "authentication required")
	}

	tokenHash := crypto.HashSessionToken(sessionToken)

	session, err := m.sessionRepo.FindByTokenHash(c.Context(), tokenHash)
	if err != nil {
		m.logger.Debug("session not found", "error", err)
		return response.Unauthorized(c, "invalid or expired session")
	}

	user, err := m.userRepo.FindByID(c.Context(), session.UserID)
	if err != nil {
		m.logger.Error("user not found for valid session", "error", err, "user_id", session.UserID)
		return response.Unauthorized(c, "user not found")
	}

	requestctx.SetUserInContext(c, user)
	return c.Next()
}

func (m *AuthMiddleware) optionalSession(c *fiber.Ctx) error {
	sessionToken := c.Cookies(m.sessionCookieName)
	if sessionToken == "" {
		return c.Next()
	}

	tokenHash := crypto.HashSessionToken(sessionToken)

	session, err := m.sessionRepo.FindByTokenHash(c.Context(), tokenHash)
	if err != nil {
		return c.Next()
	}

	user, err := m.userRepo.FindByID(c.Context(), session.UserID)
	if err != nil {
		return c.Next()
	}

	requestctx.SetUserInContext(c, user)
	return c.Next()
}

func GetTokenFromContext(c *fiber.Ctx) *domain.PersonalAccessToken {
	token, ok := c.Locals(tokenContextKey).(*domain.PersonalAccessToken)
	if !ok {
		return nil
	}
	return token
}

// RequireScope enforces a PAT scope when the request is authenticated via Bearer token.
// IMPORTANT: this middleware MUST be installed AFTER AuthMiddleware.Require(); when used
// in isolation it is a no-op for non-PAT (session) requests by design, but skipping
// Require() entirely would allow unauthenticated requests through.
func RequireScope(scope string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := GetTokenFromContext(c)
		if token == nil {
			return c.Next()
		}
		if !token.HasScope(scope) {
			return response.Forbidden(c, "token missing required scope: "+scope)
		}
		return c.Next()
	}
}

// DenyPAT rejects any request authenticated via Bearer PAT. Use it on endpoints that
// must only be reachable by interactive session users (e.g. PAT lifecycle endpoints
// to prevent privilege escalation by a compromised token).
func DenyPAT(reason string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if GetTokenFromContext(c) != nil {
			return response.Forbidden(c, reason)
		}
		return c.Next()
	}
}
