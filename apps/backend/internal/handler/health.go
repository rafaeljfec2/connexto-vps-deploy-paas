package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/response"
)

type HealthData struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

type HealthHandler struct {
	version string
}

func NewHealthHandler(version string) *HealthHandler {
	return &HealthHandler{
		version: version,
	}
}

func (h *HealthHandler) Register(app *fiber.App) {
	app.Get("/", h.Root)
	app.Get("/health", h.Health)
}

func (h *HealthHandler) Root(c *fiber.Ctx) error {
	return c.Redirect("/paas-deploy/v1/swagger/index.html", fiber.StatusMovedPermanently)
}

func (h *HealthHandler) Health(c *fiber.Ctx) error {
	return response.OK(c, HealthData{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
		Version:   h.version,
	})
}
