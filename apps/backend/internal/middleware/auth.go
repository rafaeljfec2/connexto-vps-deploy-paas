package middleware

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/crypto"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/handler"
	"github.com/paasdeploy/backend/internal/response"
)

type AuthMiddleware struct {
	sessionRepo       domain.SessionRepository
	userRepo          domain.UserRepository
	logger            *slog.Logger
	sessionCookieName string
}

type AuthMiddlewareConfig struct {
	SessionRepo       domain.SessionRepository
	UserRepo          domain.UserRepository
	Logger            *slog.Logger
	SessionCookieName string
}

func NewAuthMiddleware(cfg AuthMiddlewareConfig) *AuthMiddleware {
	return &AuthMiddleware{
		sessionRepo:       cfg.SessionRepo,
		userRepo:          cfg.UserRepo,
		logger:            cfg.Logger,
		sessionCookieName: cfg.SessionCookieName,
	}
}

func (m *AuthMiddleware) Require() fiber.Handler {
	return func(c *fiber.Ctx) error {
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

		handler.SetUserInContext(c, user)

		return c.Next()
	}
}

func (m *AuthMiddleware) Optional() fiber.Handler {
	return func(c *fiber.Ctx) error {
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

		handler.SetUserInContext(c, user)

		return c.Next()
	}
}
