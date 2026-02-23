package handler

import (
	"fmt"
	"log/slog"
	"net/url"

	"github.com/gofiber/fiber/v2"

	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/shared/pkg/docker"
)

type ResourceHandler struct {
	docker      *docker.Client
	agentClient *agentclient.AgentClient
	serverRepo  domain.ServerRepository
	agentPort   int
	logger      *slog.Logger
}

type ResourceHandlerConfig struct {
	Docker      *docker.Client
	AgentClient *agentclient.AgentClient
	ServerRepo  domain.ServerRepository
	AgentPort   int
	Logger      *slog.Logger
}

func NewResourceHandler(cfg ResourceHandlerConfig) *ResourceHandler {
	return &ResourceHandler{
		docker:      cfg.Docker,
		agentClient: cfg.AgentClient,
		serverRepo:  cfg.ServerRepo,
		agentPort:   cfg.AgentPort,
		logger:      cfg.Logger,
	}
}

func (h *ResourceHandler) resolveServerHost(serverID string) (string, error) {
	server, err := h.serverRepo.FindByID(serverID)
	if err != nil {
		return "", fmt.Errorf("server not found: %w", err)
	}
	return server.Host, nil
}

func (h *ResourceHandler) Register(app fiber.Router) {
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
	serverID := c.Query("serverId", "")

	if err := RequireAdminForLocal(c, serverID); err != nil {
		return err
	}

	if serverID != "" {
		host, err := h.resolveServerHost(serverID)
		if err != nil {
			return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
		}
		networks, err := h.agentClient.ListNetworks(c.Context(), host, h.agentPort)
		if err != nil {
			h.logger.Error("failed to list remote networks", "error", err)
			return response.ServerError(c, fiber.StatusInternalServerError, "Failed to list networks")
		}
		names := make([]string, 0, len(networks))
		for _, n := range networks {
			names = append(names, n.Name)
		}
		return response.OK(c, names)
	}

	nets, err := h.docker.ListNetworks(c.Context())
	if err != nil {
		h.logger.Error("failed to list networks", "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to list networks")
	}
	return response.OK(c, nets)
}

func (h *ResourceHandler) CreateNetwork(c *fiber.Ctx) error {
	serverID := c.Query("serverId", "")

	if err := RequireAdminForLocal(c, serverID); err != nil {
		return err
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil || body.Name == "" {
		return response.BadRequest(c, "name is required")
	}

	if serverID != "" {
		host, err := h.resolveServerHost(serverID)
		if err != nil {
			return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
		}
		if err := h.agentClient.CreateNetwork(c.Context(), host, h.agentPort, body.Name); err != nil {
			h.logger.Error("failed to create remote network", "name", body.Name, "error", err)
			return response.ServerError(c, fiber.StatusInternalServerError, "Failed to create network")
		}
		return response.Created(c, map[string]string{"name": body.Name})
	}

	if err := h.docker.EnsureNetwork(c.Context(), body.Name); err != nil {
		h.logger.Error("failed to create network", "name", body.Name, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to create network")
	}
	return response.Created(c, map[string]string{"name": body.Name})
}

func (h *ResourceHandler) RemoveNetwork(c *fiber.Ctx) error {
	serverID := c.Query("serverId", "")

	if err := RequireAdminForLocal(c, serverID); err != nil {
		return err
	}

	name, err := url.PathUnescape(c.Params("name"))
	if err != nil || name == "" {
		return response.BadRequest(c, "invalid network name")
	}

	if serverID != "" {
		host, err := h.resolveServerHost(serverID)
		if err != nil {
			return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
		}
		if err := h.agentClient.RemoveNetwork(c.Context(), host, h.agentPort, name); err != nil {
			h.logger.Error("failed to remove remote network", "name", name, "error", err)
			return response.ServerError(c, fiber.StatusInternalServerError, "Failed to remove network")
		}
		return response.NoContent(c)
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
	serverID := c.Query("serverId", "")

	if err := RequireAdminForLocal(c, serverID); err != nil {
		return err
	}

	if serverID != "" {
		host, err := h.resolveServerHost(serverID)
		if err != nil {
			return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
		}
		volumes, err := h.agentClient.ListVolumes(c.Context(), host, h.agentPort)
		if err != nil {
			h.logger.Error("failed to list remote volumes", "error", err)
			return response.ServerError(c, fiber.StatusInternalServerError, "Failed to list volumes")
		}
		names := make([]string, 0, len(volumes))
		for _, v := range volumes {
			names = append(names, v.Name)
		}
		return response.OK(c, names)
	}

	vols, err := h.docker.ListVolumes(c.Context())
	if err != nil {
		h.logger.Error("failed to list volumes", "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to list volumes")
	}
	return response.OK(c, vols)
}

func (h *ResourceHandler) CreateVolume(c *fiber.Ctx) error {
	serverID := c.Query("serverId", "")

	if err := RequireAdminForLocal(c, serverID); err != nil {
		return err
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil || body.Name == "" {
		return response.BadRequest(c, "name is required")
	}

	if serverID != "" {
		host, err := h.resolveServerHost(serverID)
		if err != nil {
			return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
		}
		if err := h.agentClient.CreateVolume(c.Context(), host, h.agentPort, body.Name); err != nil {
			h.logger.Error("failed to create remote volume", "name", body.Name, "error", err)
			return response.ServerError(c, fiber.StatusInternalServerError, "Failed to create volume")
		}
		return response.Created(c, map[string]string{"name": body.Name})
	}

	if err := h.docker.CreateVolume(c.Context(), body.Name); err != nil {
		h.logger.Error("failed to create volume", "name", body.Name, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to create volume")
	}
	return response.Created(c, map[string]string{"name": body.Name})
}

func (h *ResourceHandler) RemoveVolume(c *fiber.Ctx) error {
	serverID := c.Query("serverId", "")

	if err := RequireAdminForLocal(c, serverID); err != nil {
		return err
	}

	name, err := url.PathUnescape(c.Params("name"))
	if err != nil || name == "" {
		return response.BadRequest(c, "invalid volume name")
	}

	if serverID != "" {
		host, err := h.resolveServerHost(serverID)
		if err != nil {
			return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
		}
		if err := h.agentClient.RemoveVolume(c.Context(), host, h.agentPort, name); err != nil {
			h.logger.Error("failed to remove remote volume", "name", name, "error", err)
			return response.ServerError(c, fiber.StatusInternalServerError, "Failed to remove volume")
		}
		return response.NoContent(c)
	}

	if err := h.docker.RemoveVolume(c.Context(), name); err != nil {
		h.logger.Error("failed to remove volume", "name", name, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to remove volume")
	}
	return response.NoContent(c)
}

