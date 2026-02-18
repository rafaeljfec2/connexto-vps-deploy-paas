package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/shared/pkg/docker"
)

type ImageHandler struct {
	docker *docker.Client
	logger *slog.Logger
}

const errListImages = "Failed to list images"

func NewImageHandler(docker *docker.Client, logger *slog.Logger) *ImageHandler {
	return &ImageHandler{
		docker: docker,
		logger: logger,
	}
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

	target := id
	if ref != "" {
		target = ref
	}

	if err := h.docker.RemoveImageByID(c.Context(), target, force); err != nil {
		h.logger.Error("Failed to remove image", "target", target, "force", force, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to remove image")
	}

	return response.NoContent(c)
}

type PruneResult struct {
	ImagesDeleted  int   `json:"imagesDeleted"`
	SpaceReclaimed int64 `json:"spaceReclaimed"`
}

func (h *ImageHandler) PruneImages(c *fiber.Ctx) error {
	result, err := h.docker.PruneImages(c.Context())
	if err != nil {
		h.logger.Error("Failed to prune images", "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to prune images")
	}

	return response.OK(c, PruneResult{
		ImagesDeleted:  result.ImagesDeleted,
		SpaceReclaimed: result.SpaceReclaimed,
	})
}
