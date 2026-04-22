package handler

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/shared/pkg/docker"
)

const (
	msgFailedStartContainer   = "Failed to start container"
	msgFailedStopContainer    = "Failed to stop container"
	msgFailedRestartContainer = "Failed to restart container"
	msgFailedRemoveContainer  = "Failed to remove container"
)

func isSelfContainerError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "cannot restart FlowDeploy backend") ||
		strings.Contains(msg, "cannot stop FlowDeploy backend")
}

type ContainerHandler struct {
	docker      *docker.Client
	agentClient *agentclient.AgentClient
	serverRepo  domain.ServerRepository
	agentPort   int
	logger      *slog.Logger
	sseHandler  *SSEHandler
}

type ContainerHandlerConfig struct {
	Docker      *docker.Client
	AgentClient *agentclient.AgentClient
	ServerRepo  domain.ServerRepository
	AgentPort   int
	Logger      *slog.Logger
	SSEHandler  *SSEHandler
}

func NewContainerHandler(cfg ContainerHandlerConfig) *ContainerHandler {
	return &ContainerHandler{
		docker:      cfg.Docker,
		agentClient: cfg.AgentClient,
		serverRepo:  cfg.ServerRepo,
		agentPort:   cfg.AgentPort,
		logger:      cfg.Logger,
		sseHandler:  cfg.SSEHandler,
	}
}

func (h *ContainerHandler) invalidateContainers() {
	if h.sseHandler != nil {
		h.sseHandler.EmitInvalidate("containers")
	}
}

func (h *ContainerHandler) resolveServerHost(serverID, userID string) (string, error) {
	server, err := h.serverRepo.FindByIDForUser(serverID, userID)
	if err != nil {
		return "", fmt.Errorf("server not found: %w", err)
	}
	return server.Host, nil
}

func (h *ContainerHandler) Register(app fiber.Router) {
	v1 := app.Group(APIPrefix)
	v1.Get("/containers", h.ListContainers)
	v1.Get("/containers/:id", h.GetContainer)
	v1.Post("/containers", h.CreateContainer)
	v1.Post("/containers/:id/start", h.StartContainer)
	v1.Post("/containers/:id/stop", h.StopContainer)
	v1.Post("/containers/:id/restart", h.RestartContainer)
	v1.Delete("/containers/:id", h.RemoveContainer)
	v1.Get("/containers/:id/logs", h.GetContainerLogs)
	v1.Post("/containers/:id/healthcheck", h.RunContainerHealthcheck)
}

type ContainerResponse struct {
	ID                  string                   `json:"id"`
	Name                string                   `json:"name"`
	Image               string                   `json:"image"`
	State               string                   `json:"state"`
	Status              string                   `json:"status"`
	Health              string                   `json:"health"`
	Created             string                   `json:"created"`
	IPAddress           string                   `json:"ipAddress"`
	Ports               []ContainerPortResponse  `json:"ports"`
	Labels              map[string]string        `json:"labels"`
	Networks            []string                 `json:"networks"`
	Mounts              []ContainerMountResponse `json:"mounts"`
	IsFlowDeployManaged bool                     `json:"isFlowDeployManaged"`
}

type ContainerMountResponse struct {
	Type        string `json:"type"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	ReadOnly    bool   `json:"readOnly"`
}

type ContainerPortResponse struct {
	PrivatePort int    `json:"privatePort"`
	PublicPort  int    `json:"publicPort,omitempty"`
	Type        string `json:"type"`
}

func (h *ContainerHandler) ListContainers(c *fiber.Ctx) error {
	all := c.Query("all", "true") == "true"
	serverID := c.Query("serverId", "")

	if err := RequireAdminForLocal(c, serverID); err != nil {
		return err
	}

	if serverID != "" {
		return h.listRemoteContainers(c, serverID, all)
	}

	containers, err := h.docker.ListContainers(c.Context(), all)
	if err != nil {
		h.logger.Error("Failed to list containers", "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to list containers")
	}

	result := make([]ContainerResponse, len(containers))
	for i, container := range containers {
		result[i] = h.toContainerResponse(container)
	}

	return response.OK(c, result)
}

func (h *ContainerHandler) listRemoteContainers(c *fiber.Ctx, serverID string, all bool) error {
	host, err := h.resolveServerHost(serverID, GetUserFromContext(c).ID)
	if err != nil {
		h.logger.Error("Failed to resolve server", "serverId", serverID, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
	}

	containers, err := h.agentClient.ListContainers(c.Context(), host, h.agentPort, all, "")
	if err != nil {
		h.logger.Error("Failed to list remote containers", "serverId", serverID, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to list containers from remote server")
	}

	result := make([]ContainerResponse, 0, len(containers))
	for _, ct := range containers {
		ports := make([]ContainerPortResponse, 0, len(ct.Ports))
		for _, p := range ct.Ports {
			ports = append(ports, ContainerPortResponse{
				PrivatePort: int(p.ContainerPort),
				PublicPort:  int(p.HostPort),
				Type:        p.Protocol,
			})
		}

		labels := ct.Labels
		if labels == nil {
			labels = map[string]string{}
		}

		isManaged := false
		if _, ok := labels[docker.LabelPaasDeployApp]; ok {
			isManaged = true
		}

		networks := ct.Networks
		if networks == nil {
			networks = []string{}
		}

		mounts := make([]ContainerMountResponse, 0, len(ct.Mounts))
		for _, m := range ct.Mounts {
			mounts = append(mounts, ContainerMountResponse{
				Type:        m.Type,
				Source:      m.Source,
				Destination: m.Destination,
				ReadOnly:    m.ReadOnly,
			})
		}

		result = append(result, ContainerResponse{
			ID:                  ct.Id,
			Name:                ct.Name,
			Image:               ct.Image,
			State:               ct.State,
			Status:              ct.Status,
			Ports:               ports,
			Labels:              labels,
			Networks:            networks,
			Mounts:              mounts,
			IsFlowDeployManaged: isManaged,
		})
	}

	return response.OK(c, result)
}

func (h *ContainerHandler) GetContainer(c *fiber.Ctx) error {
	id := c.Params("id")

	container, err := h.docker.GetContainerDetails(c.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get container", "id", id, "error", err)
		return response.NotFound(c, "Container not found")
	}

	return response.OK(c, h.toContainerResponse(*container))
}

func (h *ContainerHandler) toContainerResponse(container docker.ContainerInfo) ContainerResponse {
	ports := make([]ContainerPortResponse, len(container.Ports))
	for i, p := range container.Ports {
		ports[i] = ContainerPortResponse{
			PrivatePort: p.PrivatePort,
			PublicPort:  p.PublicPort,
			Type:        p.Type,
		}
	}

	mounts := make([]ContainerMountResponse, len(container.Mounts))
	for i, m := range container.Mounts {
		mounts[i] = ContainerMountResponse{
			Type:        m.Type,
			Source:      m.Source,
			Destination: m.Destination,
			ReadOnly:    m.ReadOnly,
		}
	}

	networks := container.Networks
	if networks == nil {
		networks = []string{}
	}

	isFlowDeployManaged := false
	if _, ok := container.Labels[docker.LabelPaasDeployApp]; ok {
		isFlowDeployManaged = true
	}
	if network, ok := container.Labels["traefik.docker.network"]; ok {
		if network == "paasdeploy" {
			isFlowDeployManaged = true
		}
	}
	if project, ok := container.Labels["com.docker.compose.project"]; ok {
		if project == "paasdeploy" || project == "flowdeploy" {
			isFlowDeployManaged = true
		}
	}

	return ContainerResponse{
		ID:                  container.ID,
		Name:                container.Name,
		Image:               container.Image,
		State:               container.State,
		Status:              container.Status,
		Health:              container.Health,
		Created:             container.Created,
		IPAddress:           container.IPAddress,
		Ports:               ports,
		Labels:              container.Labels,
		Networks:            networks,
		Mounts:              mounts,
		IsFlowDeployManaged: isFlowDeployManaged,
	}
}
