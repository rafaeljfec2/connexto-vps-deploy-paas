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

type ImageHandler struct {
	docker      *docker.Client
	agentClient *agentclient.AgentClient
	serverRepo  domain.ServerRepository
	agentPort   int
	logger      *slog.Logger
}

const errListImages = "Failed to list images"

type ImageHandlerConfig struct {
	Docker      *docker.Client
	AgentClient *agentclient.AgentClient
	ServerRepo  domain.ServerRepository
	AgentPort   int
	Logger      *slog.Logger
}

func NewImageHandler(cfg ImageHandlerConfig) *ImageHandler {
	return &ImageHandler{
		docker:      cfg.Docker,
		agentClient: cfg.AgentClient,
		serverRepo:  cfg.ServerRepo,
		agentPort:   cfg.AgentPort,
		logger:      cfg.Logger,
	}
}

func (h *ImageHandler) resolveServerHost(serverID string) (string, error) {
	server, err := h.serverRepo.FindByID(serverID)
	if err != nil {
		return "", fmt.Errorf("server not found: %w", err)
	}
	return server.Host, nil
}

func (h *ImageHandler) Register(app fiber.Router) {
	v1 := app.Group(APIPrefix)
	v1.Get("/images", h.ListImages)
	v1.Get("/images/dangling", h.ListDanglingImages)
	v1.Delete("/images/:id", h.RemoveImage)
	v1.Post("/images/prune", h.PruneImages)
}

type ImageResponse struct {
	ID         string   `json:"id"`
	Repository string   `json:"repository"`
	Tag        string   `json:"tag"`
	Size       int64    `json:"size"`
	Created    string   `json:"created"`
	Containers int      `json:"containers"`
	Dangling   bool     `json:"dangling"`
	Labels     []string `json:"labels"`
}

func (h *ImageHandler) ListImages(c *fiber.Ctx) error {
	serverID := c.Query("serverId", "")

	if err := RequireAdminForLocal(c, serverID); err != nil {
		return err
	}

	if serverID != "" {
		return h.listRemoteImages(c, serverID)
	}

	images, err := h.docker.ListImages(c.Context(), false)
	if err != nil {
		h.logger.Error(errListImages, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, errListImages)
	}

	result := make([]ImageResponse, len(images))
	for i, img := range images {
		result[i] = ImageResponse{
			ID:         img.ID,
			Repository: img.Repository,
			Tag:        img.Tag,
			Size:       img.Size,
			Created:    img.Created,
			Containers: img.Containers,
			Dangling:   img.Dangling,
			Labels:     img.Labels,
		}
	}

	return response.OK(c, result)
}

func (h *ImageHandler) listRemoteImages(c *fiber.Ctx, serverID string) error {
	host, err := h.resolveServerHost(serverID)
	if err != nil {
		return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
	}

	images, err := h.agentClient.ListImages(c.Context(), host, h.agentPort, false)
	if err != nil {
		h.logger.Error("Failed to list remote images", "serverId", serverID, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, errListImages)
	}

	result := make([]ImageResponse, 0, len(images))
	for _, img := range images {
		result = append(result, ImageResponse{
			ID:         img.Id,
			Repository: img.Repository,
			Tag:        img.Tag,
			Size:       img.Size,
			Created:    img.Created,
			Dangling:   img.Dangling,
		})
	}

	return response.OK(c, result)
}

func (h *ImageHandler) ListDanglingImages(c *fiber.Ctx) error {
	images, err := h.docker.ListImages(c.Context(), true)
	if err != nil {
		h.logger.Error("Failed to list dangling images", "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, errListImages)
	}

	danglingImages := make([]ImageResponse, 0)
	for _, img := range images {
		if img.Dangling {
			danglingImages = append(danglingImages, ImageResponse{
				ID:         img.ID,
				Repository: img.Repository,
				Tag:        img.Tag,
				Size:       img.Size,
				Created:    img.Created,
				Containers: img.Containers,
				Dangling:   img.Dangling,
				Labels:     img.Labels,
			})
		}
	}

	return response.OK(c, danglingImages)
}

func (h *ImageHandler) RemoveImage(c *fiber.Ctx) error {
	id := c.Params("id")
	ref := c.Query("ref", "")
	force := c.Query("force", "false") == "true"
	serverID := c.Query("serverId", "")

	if err := RequireAdminForLocal(c, serverID); err != nil {
		return err
	}

	target := id
	if ref != "" {
		target = ref
	}

	if serverID != "" {
		host, err := h.resolveServerHost(serverID)
		if err != nil {
			return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
		}
		if err := h.agentClient.RemoveImage(c.Context(), host, h.agentPort, target, force); err != nil {
			h.logger.Error("Failed to remove remote image", "target", target, "error", err)
			return h.imageRemoveError(c, err)
		}
		return response.NoContent(c)
	}

	if err := h.docker.RemoveImageByID(c.Context(), target, force); err != nil {
		h.logger.Error("Failed to remove image", "target", target, "force", force, "error", err)
		return h.imageRemoveError(c, err)
	}

	return response.NoContent(c)
}

func (h *ImageHandler) imageRemoveError(c *fiber.Ctx, err error) error {
	msg := err.Error()
	if strings.Contains(msg, "must force") || strings.Contains(msg, "is using") || strings.Contains(msg, "conflict") {
		return response.ServerError(c, fiber.StatusConflict, "Image is being used by a running container. Stop the container first or force removal.")
	}
	return response.ServerError(c, fiber.StatusInternalServerError, "Failed to remove image")
}

type PruneResult struct {
	ImagesDeleted  int   `json:"imagesDeleted"`
	SpaceReclaimed int64 `json:"spaceReclaimed"`
}

func (h *ImageHandler) PruneImages(c *fiber.Ctx) error {
	serverID := c.Query("serverId", "")

	if err := RequireAdminForLocal(c, serverID); err != nil {
		return err
	}

	if serverID != "" {
		host, err := h.resolveServerHost(serverID)
		if err != nil {
			return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
		}
		pruneResp, err := h.agentClient.PruneImages(c.Context(), host, h.agentPort)
		if err != nil {
			h.logger.Error("Failed to prune remote images", "serverId", serverID, "error", err)
			return response.ServerError(c, fiber.StatusInternalServerError, MsgFailedPruneImages)
		}
		return response.OK(c, PruneResult{
			ImagesDeleted:  int(pruneResp.ImagesRemoved),
			SpaceReclaimed: pruneResp.SpaceReclaimedBytes,
		})
	}

	result, err := h.docker.PruneImages(c.Context())
	if err != nil {
		h.logger.Error(MsgFailedPruneImages, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, MsgFailedPruneImages)
	}

	return response.OK(c, PruneResult{
		ImagesDeleted:  result.ImagesDeleted,
		SpaceReclaimed: result.SpaceReclaimed,
	})
}
