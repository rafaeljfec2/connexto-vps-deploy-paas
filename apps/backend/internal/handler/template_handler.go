package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/shared/pkg/docker"
)

type TemplateHandler struct {
	docker *docker.Client
	logger *slog.Logger
}

func NewTemplateHandler(docker *docker.Client, logger *slog.Logger) *TemplateHandler {
	return &TemplateHandler{
		docker: docker,
		logger: logger,
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

	template := findTemplate(id)
	if template == nil {
		return response.NotFound(c, "Template not found")
	}

	var req DeployTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	opts := h.buildContainerOptions(template, req)

	containerID, err := h.docker.CreateContainer(c.Context(), opts)
	if err != nil {
		h.logger.Error("Failed to deploy template", "template", id, "error", err)
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
