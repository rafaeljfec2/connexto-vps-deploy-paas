package handler

import (
	"log/slog"
	"net/url"

	"github.com/gofiber/fiber/v2"

	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/shared/pkg/docker"
)

type ResourceHandler struct {
	docker *docker.Client
	logger *slog.Logger
}

func NewResourceHandler(docker *docker.Client, logger *slog.Logger) *ResourceHandler {
	return &ResourceHandler{docker: docker, logger: logger}
}

func (h *ResourceHandler) Register(app *fiber.App) {
	v1 := app.Group(APIPrefix)
	v1.Get("/networks", h.ListNetworks)
	v1.Post("/networks", h.CreateNetwork)
	v1.Delete("/networks/:name", h.RemoveNetwork)
	v1.Post("/containers/:id/networks", h.ConnectContainerNetwork)
	v1.Delete("/containers/:id/networks/:name", h.DisconnectContainerNetwork)

	v1.Get("/volumes", h.ListVolumes)
	v1.Post("/volumes", h.CreateVolume)
	v1.Delete("/volumes/:name", h.RemoveVolume)
}

func (h *ResourceHandler) ListNetworks(c *fiber.Ctx) error {
	nets, err := h.docker.ListNetworks(c.Context())
	if err != nil {
		h.logger.Error("failed to list networks", "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to list networks")
	}
	return response.OK(c, nets)
}

func (h *ResourceHandler) CreateNetwork(c *fiber.Ctx) error {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil || body.Name == "" {
		return response.BadRequest(c, "name is required")
	}
	if err := h.docker.EnsureNetwork(c.Context(), body.Name); err != nil {
		h.logger.Error("failed to create network", "name", body.Name, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to create network")
	}
	return response.Created(c, map[string]string{"name": body.Name})
}

func (h *ResourceHandler) RemoveNetwork(c *fiber.Ctx) error {
	name, err := url.PathUnescape(c.Params("name"))
	if err != nil || name == "" {
		return response.BadRequest(c, "invalid network name")
	}
	if err := h.docker.RemoveNetwork(c.Context(), name); err != nil {
		h.logger.Error("failed to remove network", "name", name, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to remove network")
	}
	return response.NoContent(c)
}

func (h *ResourceHandler) ConnectContainerNetwork(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Network string `json:"network"`
	}
	if err := c.BodyParser(&body); err != nil || body.Network == "" {
		return response.BadRequest(c, "network is required")
	}
	if err := h.docker.EnsureNetwork(c.Context(), body.Network); err != nil {
		h.logger.Error("failed to ensure network", "network", body.Network, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to ensure network")
	}
	if err := h.docker.ConnectToNetwork(c.Context(), id, body.Network); err != nil {
		h.logger.Error("failed to connect container to network", "container", id, "network", body.Network, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to connect container to network")
	}
	return response.OK(c, map[string]string{"connected": body.Network})
}

func (h *ResourceHandler) DisconnectContainerNetwork(c *fiber.Ctx) error {
	id := c.Params("id")
	name, err := url.PathUnescape(c.Params("name"))
	if err != nil || name == "" {
		return response.BadRequest(c, "invalid network name")
	}
	if err := h.docker.DisconnectFromNetwork(c.Context(), id, name); err != nil {
		h.logger.Error("failed to disconnect container from network", "container", id, "network", name, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to disconnect container from network")
	}
	return response.NoContent(c)
}

func (h *ResourceHandler) ListVolumes(c *fiber.Ctx) error {
	vols, err := h.docker.ListVolumes(c.Context())
	if err != nil {
		h.logger.Error("failed to list volumes", "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to list volumes")
	}
	return response.OK(c, vols)
}

func (h *ResourceHandler) CreateVolume(c *fiber.Ctx) error {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil || body.Name == "" {
		return response.BadRequest(c, "name is required")
	}
	if err := h.docker.CreateVolume(c.Context(), body.Name); err != nil {
		h.logger.Error("failed to create volume", "name", body.Name, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to create volume")
	}
	return response.Created(c, map[string]string{"name": body.Name})
}

func (h *ResourceHandler) RemoveVolume(c *fiber.Ctx) error {
	name, err := url.PathUnescape(c.Params("name"))
	if err != nil || name == "" {
		return response.BadRequest(c, "invalid volume name")
	}
	if err := h.docker.RemoveVolume(c.Context(), name); err != nil {
		h.logger.Error("failed to remove volume", "name", name, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to remove volume")
	}
	return response.NoContent(c)
}

