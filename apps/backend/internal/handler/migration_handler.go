package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/migration"
	"github.com/paasdeploy/backend/internal/response"
)

type MigrationHandler struct {
	service *migration.MigrationService
	logger  *slog.Logger
}

func NewMigrationHandler(logger *slog.Logger) *MigrationHandler {
	return &MigrationHandler{
		service: migration.NewMigrationService(logger),
		logger:  logger.With("handler", "migration"),
	}
}

func (h *MigrationHandler) Register(app fiber.Router) {
	v1 := app.Group(APIPrefix)
	m := v1.Group("/migration")

	m.Get("/status", h.GetStatus)
	m.Post("/backup", h.CreateBackup)
	m.Post("/containers/stop", h.StopContainers)
	m.Post("/containers/start", h.StartContainers)
	m.Post("/proxy/stop-nginx", h.StopNginx)
	m.Get("/sites/:index/traefik", h.GetTraefikConfig)
	m.Post("/sites/:index/migrate", h.MigrateSite)
}

func (h *MigrationHandler) GetStatus(c *fiber.Ctx) error {
	status, err := h.service.GetStatus(c.Context())
	if err != nil {
		h.logger.Error("Failed to get migration status", "error", err)
		return response.InternalError(c)
	}

	return response.OK(c, status)
}

func (h *MigrationHandler) CreateBackup(c *fiber.Ctx) error {
	result, err := h.service.CreateBackup(c.Context())
	if err != nil {
		h.logger.Error("Failed to create backup", "error", err)
		return response.BadRequest(c, "Failed to create backup: "+err.Error())
	}

	h.logger.Info("Backup created", "path", result.Path)
	return response.OK(c, result)
}

type ContainerActionRequest struct {
	ContainerIDs []string `json:"containerIds"`
}

func (h *MigrationHandler) StopContainers(c *fiber.Ctx) error {
	var req ContainerActionRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if len(req.ContainerIDs) == 0 {
		return response.BadRequest(c, "No containers specified")
	}

	if err := h.service.StopContainers(c.Context(), req.ContainerIDs); err != nil {
		h.logger.Error("Failed to stop containers", "error", err)
		return response.BadRequest(c, "Failed to stop containers: "+err.Error())
	}

	h.logger.Info("Containers stopped", "count", len(req.ContainerIDs))
	return response.OK(c, fiber.Map{
		"message": "Containers stopped successfully",
		"stopped": req.ContainerIDs,
	})
}

func (h *MigrationHandler) StartContainers(c *fiber.Ctx) error {
	var req ContainerActionRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if len(req.ContainerIDs) == 0 {
		return response.BadRequest(c, "No containers specified")
	}

	if err := h.service.StartContainers(c.Context(), req.ContainerIDs); err != nil {
		h.logger.Error("Failed to start containers", "error", err)
		return response.BadRequest(c, "Failed to start containers: "+err.Error())
	}

	h.logger.Info("Containers started", "count", len(req.ContainerIDs))
	return response.OK(c, fiber.Map{
		"message": "Containers started successfully",
		"started": req.ContainerIDs,
	})
}

func (h *MigrationHandler) StopNginx(c *fiber.Ctx) error {
	if err := h.service.StopNginx(c.Context()); err != nil {
		h.logger.Error("Failed to stop nginx", "error", err)
		return response.BadRequest(c, "Failed to stop nginx: "+err.Error())
	}

	h.logger.Info("Nginx stopped and disabled")
	return response.OK(c, fiber.Map{
		"message": "Nginx stopped and disabled successfully",
	})
}

func (h *MigrationHandler) GetTraefikConfig(c *fiber.Ctx) error {
	index, err := c.ParamsInt("index")
	if err != nil {
		return response.BadRequest(c, "Invalid site index")
	}

	status, err := h.service.GetStatus(c.Context())
	if err != nil {
		return response.InternalError(c)
	}

	if index < 0 || index >= len(status.NginxSites) {
		return response.BadRequest(c, "Site index out of range")
	}

	site := status.NginxSites[index]
	configs := h.service.GetTraefikConfigs(site)
	yaml := h.service.GetTraefikLabelsYAML(site)

	return response.OK(c, fiber.Map{
		"site":    site.ServerNames,
		"configs": configs,
		"yaml":    yaml,
	})
}

type MigrateSiteRequest struct {
	ContainerID string `json:"containerId"`
}

func (h *MigrationHandler) MigrateSite(c *fiber.Ctx) error {
	index, err := c.ParamsInt("index")
	if err != nil {
		return response.BadRequest(c, "Invalid site index")
	}

	var req MigrateSiteRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.ContainerID == "" {
		return response.BadRequest(c, "Container ID is required")
	}

	status, err := h.service.GetStatus(c.Context())
	if err != nil {
		return response.InternalError(c)
	}

	if index < 0 || index >= len(status.NginxSites) {
		return response.BadRequest(c, "Site index out of range")
	}

	site := status.NginxSites[index]

	h.logger.Info("Starting migration", "site", site.ServerNames[0], "container", req.ContainerID)

	result, err := h.service.MigrateContainer(c.Context(), site, req.ContainerID)
	if err != nil {
		h.logger.Error("Migration failed", "error", err)
		return response.BadRequest(c, "Migration failed: "+err.Error())
	}

	h.logger.Info("Migration completed", "site", site.ServerNames[0], "newContainer", result.ContainerID)
	return response.OK(c, result)
}
