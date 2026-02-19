package handler

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/shared/pkg/docker"
	"github.com/valyala/fasthttp"
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
}

type ContainerHandlerConfig struct {
	Docker      *docker.Client
	AgentClient *agentclient.AgentClient
	ServerRepo  domain.ServerRepository
	AgentPort   int
	Logger      *slog.Logger
}

func NewContainerHandler(cfg ContainerHandlerConfig) *ContainerHandler {
	return &ContainerHandler{
		docker:      cfg.Docker,
		agentClient: cfg.AgentClient,
		serverRepo:  cfg.ServerRepo,
		agentPort:   cfg.AgentPort,
		logger:      cfg.Logger,
	}
}

func (h *ContainerHandler) resolveServerHost(serverID string) (string, error) {
	server, err := h.serverRepo.FindByID(serverID)
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
	host, err := h.resolveServerHost(serverID)
	if err != nil {
		h.logger.Error("Failed to resolve server", "serverId", serverID, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Server not found")
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
		if _, ok := labels["paasdeploy.app"]; ok {
			isManaged = true
		}

		result = append(result, ContainerResponse{
			ID:                  ct.Id,
			Name:                ct.Name,
			Image:               ct.Image,
			State:               ct.State,
			Status:              ct.Status,
			Ports:               ports,
			Labels:              labels,
			Networks:            []string{},
			Mounts:              []ContainerMountResponse{},
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

type CreateContainerRequest struct {
	Name          string               `json:"name"`
	Image         string               `json:"image"`
	Ports         []PortMappingRequest `json:"ports,omitempty"`
	Env           map[string]string    `json:"env,omitempty"`
	Volumes       []VolumeMappingRequest `json:"volumes,omitempty"`
	Network       string               `json:"network,omitempty"`
	RestartPolicy string               `json:"restartPolicy,omitempty"`
	Command       []string             `json:"command,omitempty"`
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

	return response.Created(c, h.toContainerResponse(*container))
}

func (h *ContainerHandler) StartContainer(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.docker.StartContainer(c.Context(), id); err != nil {
		h.logger.Error("Failed to start container", "id", id, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to start container")
	}

	return response.OK(c, map[string]string{"message": "Container started", "id": id})
}

func (h *ContainerHandler) StopContainer(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.docker.StopContainer(c.Context(), id); err != nil {
		h.logger.Error("Failed to stop container", "id", id, "error", err)
		if isSelfContainerError(err) {
			return response.BadRequest(c, "Operation not allowed for this container")
		}
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to stop container")
	}

	return response.OK(c, map[string]string{"message": "Container stopped", "id": id})
}

func (h *ContainerHandler) RestartContainer(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.docker.RestartContainer(c.Context(), id); err != nil {
		h.logger.Error("Failed to restart container", "id", id, "error", err)
		if isSelfContainerError(err) {
			return response.BadRequest(c, "Operation not allowed for this container")
		}
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to restart container")
	}

	return response.OK(c, map[string]string{"message": "Container restarted", "id": id})
}

func (h *ContainerHandler) RemoveContainer(c *fiber.Ctx) error {
	id := c.Params("id")
	force := c.Query("force", "false") == "true"

	if err := h.docker.RemoveContainer(c.Context(), id, force); err != nil {
		h.logger.Error("Failed to remove container", "id", id, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to remove container")
	}

	return response.NoContent(c)
}

type ContainerLogsResponseGeneral struct {
	Logs string `json:"logs"`
}

func (h *ContainerHandler) GetContainerLogs(c *fiber.Ctx) error {
	id := c.Params("id")
	tailStr := c.Query("tail", "100")
	follow := c.Query("follow", "false") == "true"

	tail, err := strconv.Atoi(tailStr)
	if err != nil {
		tail = 100
	}

	if follow {
		return h.streamContainerLogs(c, c.Context(), id)
	}

	logs, err := h.docker.ContainerLogs(c.Context(), id, tail)
	if err != nil {
		h.logger.Error("Failed to get container logs", "id", id, "error", err)
		return response.OK(c, ContainerLogsResponseGeneral{Logs: ""})
	}

	return response.OK(c, ContainerLogsResponseGeneral{Logs: logs})
}

func (h *ContainerHandler) streamContainerLogs(c *fiber.Ctx, ctx context.Context, containerID string) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		output := make(chan string, 100)
		done := make(chan struct{})

		go func() {
			defer close(done)
			_ = h.docker.StreamContainerLogs(ctx, containerID, output)
		}()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case line, ok := <-output:
				if !ok {
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", line)
				if err := w.Flush(); err != nil {
					return
				}
			case <-ticker.C:
				fmt.Fprintf(w, ": keepalive\n\n")
				if err := w.Flush(); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}))

	return nil
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
	if _, ok := container.Labels["paasdeploy.app"]; ok {
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
