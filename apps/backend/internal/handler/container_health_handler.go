package handler

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/engine"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/valyala/fasthttp"
)

type ContainerHealthResponse struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Health    string `json:"health"`
	StartedAt string `json:"startedAt,omitempty"`
	Uptime    string `json:"uptime,omitempty"`
}

type ContainerHealthHandler struct {
	appRepo domain.AppRepository
	engine  *engine.Engine
}

func NewContainerHealthHandler(appRepo domain.AppRepository, eng *engine.Engine) *ContainerHealthHandler {
	return &ContainerHealthHandler{
		appRepo: appRepo,
		engine:  eng,
	}
}

func (h *ContainerHealthHandler) Register(app *fiber.App) {
	v1 := app.Group("/paas-deploy/v1")
	v1.Get("/apps/:id/health", h.GetAppHealth)
	v1.Get("/apps/:id/container/logs", h.GetContainerLogs)
	v1.Get("/apps/:id/container/stats", h.GetContainerStats)
}

func (h *ContainerHealthHandler) GetAppHealth(c *fiber.Ctx) error {
	id := c.Params("id")

	app, err := h.appRepo.FindByID(id)
	if err != nil {
		return response.NotFound(c, "app not found")
	}

	ctx := context.Background()
	health := h.engine.GetAppHealth(ctx, app.Name)

	if health == nil {
		return response.OK(c, ContainerHealthResponse{
			Name:   app.Name,
			Status: "unknown",
			Health: "none",
		})
	}

	return response.OK(c, ContainerHealthResponse{
		Name:      health.Name,
		Status:    health.Status,
		Health:    health.Health,
		StartedAt: health.StartedAt,
		Uptime:    health.Uptime,
	})
}

type ContainerLogsResponse struct {
	Logs string `json:"logs"`
}

func (h *ContainerHealthHandler) GetContainerLogs(c *fiber.Ctx) error {
	id := c.Params("id")
	tailStr := c.Query("tail", "100")
	follow := c.Query("follow", "false") == "true"

	tail, err := strconv.Atoi(tailStr)
	if err != nil {
		tail = 100
	}

	app, err := h.appRepo.FindByID(id)
	if err != nil {
		return response.NotFound(c, "app not found")
	}

	ctx := context.Background()

	if follow {
		return h.streamContainerLogs(c, ctx, app.Name)
	}

	logs, err := h.engine.ContainerLogs(ctx, app.Name, tail)
	if err != nil {
		return response.OK(c, ContainerLogsResponse{Logs: ""})
	}

	return response.OK(c, ContainerLogsResponse{Logs: logs})
}

func (h *ContainerHealthHandler) streamContainerLogs(c *fiber.Ctx, ctx context.Context, containerName string) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		output := make(chan string, 100)
		done := make(chan struct{})

		go func() {
			defer close(done)
			_ = h.engine.StreamContainerLogs(ctx, containerName, output)
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

type ContainerStatsResponse struct {
	CPUPercent    float64 `json:"cpuPercent"`
	MemoryUsage   int64   `json:"memoryUsage"`
	MemoryLimit   int64   `json:"memoryLimit"`
	MemoryPercent float64 `json:"memoryPercent"`
	NetworkRx     int64   `json:"networkRx"`
	NetworkTx     int64   `json:"networkTx"`
	PIDs          int     `json:"pids"`
}

func (h *ContainerHealthHandler) GetContainerStats(c *fiber.Ctx) error {
	id := c.Params("id")

	app, err := h.appRepo.FindByID(id)
	if err != nil {
		return response.NotFound(c, "app not found")
	}

	ctx := context.Background()
	stats, err := h.engine.ContainerStats(ctx, app.Name)
	if err != nil {
		return response.OK(c, ContainerStatsResponse{})
	}

	return response.OK(c, ContainerStatsResponse{
		CPUPercent:    stats.CPUPercent,
		MemoryUsage:   stats.MemoryUsage,
		MemoryLimit:   stats.MemoryLimit,
		MemoryPercent: stats.MemoryPercent,
		NetworkRx:     stats.NetworkRx,
		NetworkTx:     stats.NetworkTx,
		PIDs:          stats.PIDs,
	})
}
