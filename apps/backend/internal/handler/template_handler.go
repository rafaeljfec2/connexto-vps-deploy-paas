package handler

import (
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/shared/pkg/docker"
)

type TemplateHandler struct {
	docker      *docker.Client
	agentClient *agentclient.AgentClient
	serverRepo  domain.ServerRepository
	agentPort   int
	logger      *slog.Logger
}

type TemplateHandlerConfig struct {
	Docker      *docker.Client
	AgentClient *agentclient.AgentClient
	ServerRepo  domain.ServerRepository
	AgentPort   int
	Logger      *slog.Logger
}

func NewTemplateHandler(cfg TemplateHandlerConfig) *TemplateHandler {
	return &TemplateHandler{
		docker:      cfg.Docker,
		agentClient: cfg.AgentClient,
		serverRepo:  cfg.ServerRepo,
		agentPort:   cfg.AgentPort,
		logger:      cfg.Logger,
	}
}

func (h *TemplateHandler) Register(app fiber.Router) {
	v1 := app.Group(APIPrefix)
	v1.Get("/templates", h.ListTemplates)
	v1.Get("/templates/:id", h.GetTemplate)
	v1.Post("/templates/:id/deploy", h.DeployTemplate)
}

func (h *TemplateHandler) ListTemplates(c *fiber.Ctx) error {
	category := c.Query("category", "")
	allTemplates := getTemplates()

	if category == "" {
		return response.OK(c, allTemplates)
	}

	filtered := make([]Template, 0)
	for _, t := range allTemplates {
		if t.Category == category {
			filtered = append(filtered, t)
		}
	}

	return response.OK(c, filtered)
}

func (h *TemplateHandler) GetTemplate(c *fiber.Ctx) error {
	id := c.Params("id")

	template := findTemplate(id)
	if template == nil {
		return response.NotFound(c, "Template not found")
	}

	return response.OK(c, template)
}

type DeployTemplateRequest struct {
	Name          string               `json:"name"`
	Env           map[string]string    `json:"env,omitempty"`
	Ports         []PortMappingRequest `json:"ports,omitempty"`
	Network       string               `json:"network,omitempty"`
	RestartPolicy string               `json:"restartPolicy,omitempty"`
}

func (h *TemplateHandler) DeployTemplate(c *fiber.Ctx) error {
	id := c.Params("id")
	serverID := c.Query("serverId", "")

	if err := RequireAdminForLocal(c, serverID); err != nil {
		return err
	}

	template := findTemplate(id)
	if template == nil {
		return response.NotFound(c, "Template not found")
	}

	var req DeployTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if serverID != "" {
		return h.deployTemplateRemote(c, serverID, template, req)
	}

	return h.deployTemplateLocal(c, id, template, req)
}

func (h *TemplateHandler) deployTemplateLocal(c *fiber.Ctx, templateID string, template *Template, req DeployTemplateRequest) error {
	opts := h.buildContainerOptions(template, req)

	containerID, err := h.docker.CreateContainer(c.Context(), opts)
	if err != nil {
		h.logger.Error("Failed to deploy template", "template", templateID, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to deploy template")
	}

	container, err := h.docker.GetContainerDetails(c.Context(), containerID)
	if err != nil {
		return response.Created(c, map[string]string{"id": containerID, "message": "Container created from template"})
	}

	return response.Created(c, map[string]interface{}{
		"id":        container.ID,
		"name":      container.Name,
		"image":     container.Image,
		"state":     container.State,
		"ipAddress": container.IPAddress,
		"message":   "Container created from template",
	})
}

func (h *TemplateHandler) deployTemplateRemote(c *fiber.Ctx, serverID string, template *Template, req DeployTemplateRequest) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	server, err := h.serverRepo.FindByIDForUser(serverID, user.ID)
	if err != nil {
		h.logger.Error("Failed to resolve server for template deploy", "serverId", serverID, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
	}

	grpcReq := h.buildGRPCCreateContainerRequest(template, req)

	resp, err := h.agentClient.CreateContainerFromTemplate(c.Context(), server.Host, h.agentPort, grpcReq)
	if err != nil {
		h.logger.Error("Failed to deploy template on remote agent", "serverId", serverID, "template", template.ID, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to deploy template on remote server")
	}

	if !resp.Success {
		return response.ServerError(c, fiber.StatusInternalServerError, fmt.Sprintf("Remote template deploy failed: %s", resp.Message))
	}

	return response.Created(c, map[string]interface{}{
		"id":      resp.ContainerId,
		"message": "Container created from template on remote server",
	})
}

func (h *TemplateHandler) buildGRPCCreateContainerRequest(template *Template, req DeployTemplateRequest) *pb.CreateContainerFromTemplateRequest {
	name := req.Name
	if name == "" {
		name = template.ID
	}

	restartPolicy := req.RestartPolicy
	if restartPolicy == "" {
		restartPolicy = "unless-stopped"
	}

	grpcReq := &pb.CreateContainerFromTemplateRequest{
		Name:          name,
		Image:         template.Image,
		Env:           req.Env,
		Network:       req.Network,
		RestartPolicy: restartPolicy,
	}

	grpcReq.Ports = h.buildGRPCPortMappings(template, req.Ports)

	for _, v := range template.Volumes {
		grpcReq.Volumes = append(grpcReq.Volumes, &pb.CreateContainerVolumeMapping{
			ContainerPath: v,
		})
	}

	return grpcReq
}

func (h *TemplateHandler) buildContainerOptions(template *Template, req DeployTemplateRequest) docker.CreateContainerOptions {
	name := req.Name
	if name == "" {
		name = template.ID
	}

	restartPolicy := req.RestartPolicy
	if restartPolicy == "" {
		restartPolicy = "unless-stopped"
	}

	opts := docker.CreateContainerOptions{
		Name:          name,
		Image:         template.Image,
		Env:           req.Env,
		Network:       req.Network,
		RestartPolicy: restartPolicy,
	}

	opts.Ports = h.buildPortMappings(template, req.Ports)

	return opts
}

func (h *TemplateHandler) buildPortMappings(template *Template, requestPorts []PortMappingRequest) []docker.PortMapping {
	if len(requestPorts) > 0 {
		ports := make([]docker.PortMapping, 0, len(requestPorts))
		for _, p := range requestPorts {
			ports = append(ports, docker.PortMapping{
				HostPort:      p.HostPort,
				ContainerPort: p.ContainerPort,
				Protocol:      p.Protocol,
			})
		}
		return ports
	}

	ports := make([]docker.PortMapping, 0, len(template.Ports))
	for i, port := range template.Ports {
		ports = append(ports, docker.PortMapping{
			HostPort:      port + i,
			ContainerPort: port,
			Protocol:      "tcp",
		})
	}
	return ports
}

func (h *TemplateHandler) buildGRPCPortMappings(template *Template, requestPorts []PortMappingRequest) []*pb.CreateContainerPortMapping {
	if len(requestPorts) > 0 {
		ports := make([]*pb.CreateContainerPortMapping, 0, len(requestPorts))
		for _, p := range requestPorts {
			protocol := p.Protocol
			if protocol == "" {
				protocol = "tcp"
			}
			ports = append(ports, &pb.CreateContainerPortMapping{
				HostPort:      int32(p.HostPort),
				ContainerPort: int32(p.ContainerPort),
				Protocol:      protocol,
			})
		}
		return ports
	}

	ports := make([]*pb.CreateContainerPortMapping, 0, len(template.Ports))
	for i, port := range template.Ports {
		ports = append(ports, &pb.CreateContainerPortMapping{
			HostPort:      int32(port + i),
			ContainerPort: int32(port),
			Protocol:      "tcp",
		})
	}
	return ports
}
