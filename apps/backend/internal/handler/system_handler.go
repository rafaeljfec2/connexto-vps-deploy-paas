package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/backend/internal/sysinfo"
)

type SystemHandler struct{}

func NewSystemHandler() *SystemHandler {
	return &SystemHandler{}
}

func (h *SystemHandler) Register(app fiber.Router) {
	v1 := app.Group(APIPrefix)
	v1.Get("/system/stats", h.GetStats)
}

func (h *SystemHandler) GetStats(c *fiber.Ctx) error {
	return response.OK(c, sysinfo.GetStats())
}
