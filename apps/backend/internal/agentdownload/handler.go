package agentdownload

import (
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"

	"github.com/paasdeploy/backend/internal/response"
)

type Handler struct {
	tokenStore *TokenStore
	binaryPath string
	logger     *slog.Logger
}

func NewHandler(tokenStore *TokenStore, binaryPath string, logger *slog.Logger) *Handler {
	return &Handler{
		tokenStore: tokenStore,
		binaryPath: binaryPath,
		logger:     logger.With("handler", "agent_download"),
	}
}

func (h *Handler) ServeBinary(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return response.BadRequest(c, "missing token")
	}
	if !h.tokenStore.Validate(token) {
		h.logger.Warn("agent download token rejected", "ip", c.IP())
		return response.Unauthorized(c, "invalid or expired token")
	}
	if h.binaryPath == "" {
		h.logger.Warn("agent binary path not configured")
		return response.ServerError(c, fiber.StatusServiceUnavailable, "agent binary not available")
	}

	info, err := os.Stat(h.binaryPath)
	if err != nil {
		h.logger.Error("agent binary not found", "path", h.binaryPath, "error", err)
		return response.InternalError(c)
	}

	data, err := os.ReadFile(h.binaryPath)
	if err != nil {
		h.logger.Error("failed to read agent binary", "path", h.binaryPath, "error", err)
		return response.InternalError(c)
	}
	h.logger.Info("serving agent binary", "size", info.Size(), "ip", c.IP())
	c.Set("Content-Type", "application/octet-stream")
	c.Set("Content-Disposition", "attachment; filename=agent")
	return c.Send(data)
}
