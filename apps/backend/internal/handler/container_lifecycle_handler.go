package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/paasdeploy/backend/internal/middleware"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/shared/pkg/docker"
)

type CreateContainerRequest struct {
	Name          string                 `json:"name"`
	Image         string                 `json:"image"`
	Ports         []PortMappingRequest   `json:"ports,omitempty"`
	Env           map[string]string      `json:"env,omitempty"`
	Volumes       []VolumeMappingRequest `json:"volumes,omitempty"`
	Network       string                 `json:"network,omitempty"`
	RestartPolicy string                 `json:"restartPolicy,omitempty"`
	Command       []string               `json:"command,omitempty"`
}

type PortMappingRequest struct {
	HostPort      int    `json:"hostPort"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol,omitempty"`
}

type VolumeMappingRequest struct {
	HostPath      string `json:"hostPath"`
	ContainerPath string `json:"containerPath"`
	ReadOnly      bool   `json:"readOnly,omitempty"`
}

func (h *ContainerHandler) CreateContainer(c *fiber.Ctx) error {
	var req CreateContainerRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Image == "" {
		return response.BadRequest(c, "Image is required")
	}

	opts := docker.CreateContainerOptions{
		Name:          req.Name,
		Image:         req.Image,
		Env:           req.Env,
		Network:       req.Network,
		RestartPolicy: req.RestartPolicy,
		Command:       req.Command,
	}

	for _, p := range req.Ports {
		opts.Ports = append(opts.Ports, docker.PortMapping{
			HostPort:      p.HostPort,
			ContainerPort: p.ContainerPort,
			Protocol:      p.Protocol,
		})
	}

	for _, v := range req.Volumes {
		opts.Volumes = append(opts.Volumes, docker.VolumeMapping{
			HostPath:      v.HostPath,
			ContainerPath: v.ContainerPath,
			ReadOnly:      v.ReadOnly,
		})
	}

	containerID, err := h.docker.CreateContainer(c.Context(), opts)
	if err != nil {
		h.logger.Error("Failed to create container", "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to create container")
	}

	container, err := h.docker.GetContainerDetails(c.Context(), containerID)
	if err != nil {
		return response.OK(c, map[string]string{"id": containerID, "message": "Container created"})
	}

	h.invalidateContainers()
	return response.Created(c, h.toContainerResponse(*container))
}

func (h *ContainerHandler) StartContainer(c *fiber.Ctx) error {
	id := c.Params("id")
	serverID := c.Query("serverId", "")

	if err := RequireAdminForLocal(c, serverID); err != nil {
		return err
	}

	if serverID != "" {
		host, err := h.resolveServerHost(serverID, GetUserFromContext(c).ID)
		if err != nil {
			return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
		}
		if err := h.agentClient.StartContainer(c.Context(), host, h.agentPort, id); err != nil {
			h.logger.Error("Failed to start remote container", "id", id, "serverId", serverID, "error", err)
			return response.ServerError(c, fiber.StatusInternalServerError, msgFailedStartContainer)
		}
		h.invalidateContainers()
		return response.OK(c, map[string]string{"message": "Container started", "id": id})
	}

	if err := h.docker.StartContainer(c.Context(), id); err != nil {
		h.logger.Error("Failed to start container", "id", id, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, msgFailedStartContainer)
	}

	h.invalidateContainers()
	return response.OK(c, map[string]string{"message": "Container started", "id": id})
}

func (h *ContainerHandler) StopContainer(c *fiber.Ctx) error {
	id := c.Params("id")
	serverID := c.Query("serverId", "")

	if err := RequireAdminForLocal(c, serverID); err != nil {
		return err
	}

	if serverID != "" {
		host, err := h.resolveServerHost(serverID, GetUserFromContext(c).ID)
		if err != nil {
			return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
		}
		if err := h.agentClient.StopContainer(c.Context(), host, h.agentPort, id); err != nil {
			h.logger.Error("Failed to stop remote container", "id", id, "serverId", serverID, "error", err)
			return response.ServerError(c, fiber.StatusInternalServerError, msgFailedStopContainer)
		}
		h.invalidateContainers()
		return response.OK(c, map[string]string{"message": "Container stopped", "id": id})
	}

	if err := h.docker.StopContainer(c.Context(), id); err != nil {
		h.logger.Error("Failed to stop container", "id", id, "error", err)
		if isSelfContainerError(err) {
			return response.BadRequest(c, "Operation not allowed for this container")
		}
		return response.ServerError(c, fiber.StatusInternalServerError, msgFailedStopContainer)
	}

	h.invalidateContainers()
	return response.OK(c, map[string]string{"message": "Container stopped", "id": id})
}

func (h *ContainerHandler) RestartContainer(c *fiber.Ctx) error {
	id := c.Params("id")
	serverID := c.Query("serverId", "")

	if err := RequireAdminForLocal(c, serverID); err != nil {
		return err
	}

	if serverID != "" {
		host, err := h.resolveServerHost(serverID, GetUserFromContext(c).ID)
		if err != nil {
			return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
		}
		if err := h.agentClient.RestartContainer(c.Context(), host, h.agentPort, id); err != nil {
			h.logger.Error("Failed to restart remote container", "id", id, "serverId", serverID, "error", err)
			return response.ServerError(c, fiber.StatusInternalServerError, msgFailedRestartContainer)
		}
		h.invalidateContainers()
		return response.OK(c, map[string]string{"message": "Container restarted", "id": id})
	}

	if err := h.docker.RestartContainer(c.Context(), id); err != nil {
		h.logger.Error("Failed to restart container", "id", id, "error", err)
		if isSelfContainerError(err) {
			return response.BadRequest(c, "Operation not allowed for this container")
		}
		return response.ServerError(c, fiber.StatusInternalServerError, msgFailedRestartContainer)
	}

	h.invalidateContainers()
	return response.OK(c, map[string]string{"message": "Container restarted", "id": id})
}

func (h *ContainerHandler) RemoveContainer(c *fiber.Ctx) error {
	id := c.Params("id")
	force := c.Query("force", "false") == "true"
	serverID := c.Query("serverId", "")

	if err := RequireAdminForLocal(c, serverID); err != nil {
		return err
	}

	report := response.DryRunReport{
		Action:      "containers.remove",
		Resource:    "container",
		ResourceID:  id,
		Description: "Would remove the Docker container",
		Effects: []string{
			"the container is stopped (if running) and removed from the host",
			"associated logs/state are lost",
		},
		Reversible: false,
		Metadata: map[string]any{
			"force":    force,
			"serverId": serverID,
		},
	}
	if abort, errEnforce := middleware.EnforceDestructive(c, report); abort || errEnforce != nil {
		return errEnforce
	}

	if serverID != "" {
		host, err := h.resolveServerHost(serverID, GetUserFromContext(c).ID)
		if err != nil {
			return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
		}
		if err := h.agentClient.RemoveContainer(c.Context(), host, h.agentPort, id, force); err != nil {
			h.logger.Error("Failed to remove remote container", "id", id, "serverId", serverID, "error", err)
			return response.ServerError(c, fiber.StatusInternalServerError, msgFailedRemoveContainer)
		}
		h.invalidateContainers()
		return response.NoContent(c)
	}

	if err := h.docker.RemoveContainer(c.Context(), id, force); err != nil {
		h.logger.Error("Failed to remove container", "id", id, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, msgFailedRemoveContainer)
	}

	h.invalidateContainers()
	return response.NoContent(c)
}
