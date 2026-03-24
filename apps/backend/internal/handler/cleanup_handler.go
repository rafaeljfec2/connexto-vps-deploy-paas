package handler

import (
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
)

const (
	maxCleanupLogLimit   = 200
	errCleanupServerNotFound = "Server not found"
)

type CleanupHandler struct {
	agentClient    *agentclient.AgentClient
	serverRepo     domain.ServerRepository
	cleanupLogRepo domain.CleanupLogRepository
	agentPort      int
	logger         *slog.Logger
}

type CleanupHandlerConfig struct {
	AgentClient    *agentclient.AgentClient
	ServerRepo     domain.ServerRepository
	CleanupLogRepo domain.CleanupLogRepository
	AgentPort      int
	Logger         *slog.Logger
}

func NewCleanupHandler(cfg CleanupHandlerConfig) *CleanupHandler {
	return &CleanupHandler{
		agentClient:    cfg.AgentClient,
		serverRepo:     cfg.ServerRepo,
		cleanupLogRepo: cfg.CleanupLogRepo,
		agentPort:      cfg.AgentPort,
		logger:         cfg.Logger,
	}
}

func (h *CleanupHandler) Register(app fiber.Router) {
	v1 := app.Group(APIPrefix)

	cleanup := v1.Group("/servers/:serverId/cleanup")
	cleanup.Post("/containers", h.PruneContainers)
	cleanup.Post("/volumes", h.PruneVolumes)
	cleanup.Get("/logs", h.ListCleanupLogs)
}

type PruneResponse struct {
	ItemsRemoved        int    `json:"itemsRemoved"`
	SpaceReclaimedBytes int64  `json:"spaceReclaimedBytes"`
	CleanupType         string `json:"cleanupType"`
}

func (h *CleanupHandler) PruneContainers(c *fiber.Ctx) error {
	serverID := c.Params("serverId")
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	server, err := h.serverRepo.FindByIDForUser(serverID, user.ID)
	if err != nil {
		return HandleNotFoundOrInternal(c, err, errCleanupServerNotFound)
	}

	resp, err := h.agentClient.PruneContainers(c.Context(), server.Host, h.agentPort)
	if err != nil {
		h.logCleanup(serverID, domain.CleanupTypeContainers, 0, 0, err.Error())
		h.logger.Error("Failed to prune containers", "serverId", serverID, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to prune containers")
	}

	h.logCleanup(serverID, domain.CleanupTypeContainers, int(resp.ContainersRemoved), resp.SpaceReclaimedBytes, "")

	return response.OK(c, PruneResponse{
		ItemsRemoved:        int(resp.ContainersRemoved),
		SpaceReclaimedBytes: resp.SpaceReclaimedBytes,
		CleanupType:         string(domain.CleanupTypeContainers),
	})
}

func (h *CleanupHandler) PruneVolumes(c *fiber.Ctx) error {
	serverID := c.Params("serverId")
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	server, err := h.serverRepo.FindByIDForUser(serverID, user.ID)
	if err != nil {
		return HandleNotFoundOrInternal(c, err, errCleanupServerNotFound)
	}

	resp, err := h.agentClient.PruneVolumes(c.Context(), server.Host, h.agentPort)
	if err != nil {
		h.logCleanup(serverID, domain.CleanupTypeVolumes, 0, 0, err.Error())
		h.logger.Error("Failed to prune volumes", "serverId", serverID, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to prune volumes")
	}

	h.logCleanup(serverID, domain.CleanupTypeVolumes, int(resp.VolumesRemoved), resp.SpaceReclaimedBytes, "")

	return response.OK(c, PruneResponse{
		ItemsRemoved:        int(resp.VolumesRemoved),
		SpaceReclaimedBytes: resp.SpaceReclaimedBytes,
		CleanupType:         string(domain.CleanupTypeVolumes),
	})
}

func (h *CleanupHandler) ListCleanupLogs(c *fiber.Ctx) error {
	serverID := c.Params("serverId")
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	if _, err := h.serverRepo.FindByIDForUser(serverID, user.ID); err != nil {
		return HandleNotFoundOrInternal(c, err, errCleanupServerNotFound)
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	if limit <= 0 {
		limit = 50
	}
	if limit > maxCleanupLogLimit {
		limit = maxCleanupLogLimit
	}
	if offset < 0 {
		offset = 0
	}

	logs, err := h.cleanupLogRepo.FindByServerID(serverID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list cleanup logs", "serverId", serverID, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to list cleanup logs")
	}

	return response.OK(c, logs)
}

func (h *CleanupHandler) logCleanup(serverID string, cleanupType domain.CleanupType, itemsRemoved int, spaceReclaimed int64, errMsg string) {
	status := domain.CleanupStatusSuccess
	if errMsg != "" {
		status = domain.CleanupStatusFailed
	}

	if _, err := h.cleanupLogRepo.Create(domain.CreateCleanupLogInput{
		ServerID:            serverID,
		CleanupType:         cleanupType,
		ItemsRemoved:        itemsRemoved,
		SpaceReclaimedBytes: spaceReclaimed,
		Trigger:             domain.CleanupTriggerManual,
		Status:              status,
		ErrorMessage:        errMsg,
	}); err != nil {
		h.logger.Error("Failed to save cleanup log", "serverID", serverID, "error", err)
	}
}
