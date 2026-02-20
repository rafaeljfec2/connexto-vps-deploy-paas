package handler

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/engine"
	"github.com/paasdeploy/backend/internal/response"
	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
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
	appRepo     domain.AppRepository
	serverRepo  domain.ServerRepository
	engine      *engine.Engine
	agentClient *agentclient.AgentClient
	agentPort   int
	logger      *slog.Logger
}

func NewContainerHealthHandler(
	appRepo domain.AppRepository,
	serverRepo domain.ServerRepository,
	eng *engine.Engine,
	agentClient *agentclient.AgentClient,
	agentPort int,
	logger *slog.Logger,
) *ContainerHealthHandler {
	return &ContainerHealthHandler{
		appRepo:     appRepo,
		serverRepo:  serverRepo,
		engine:      eng,
		agentClient: agentClient,
		agentPort:   agentPort,
		logger:      logger.With("handler", "container_health"),
	}
}

func (h *ContainerHealthHandler) isRemoteApp(app *domain.App) bool {
	return app.ServerID != nil && *app.ServerID != ""
}

func (h *ContainerHealthHandler) resolveServerHost(app *domain.App) (string, error) {
	if app.ServerID == nil {
		return "", fmt.Errorf("app has no server")
	}
	server, err := h.serverRepo.FindByID(*app.ServerID)
	if err != nil {
		return "", fmt.Errorf("server not found: %w", err)
	}
	return server.Host, nil
}

func (h *ContainerHealthHandler) Register(app fiber.Router) {
	v1 := app.Group(APIPrefix)
	v1.Get("/apps/:id/health", h.GetAppHealth)
	v1.Get("/apps/:id/container/logs", h.GetContainerLogs)
	v1.Get("/apps/:id/container/stats", h.GetContainerStats)
}

func (h *ContainerHealthHandler) GetAppHealth(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := EnsureAppOwnership(c, h.appRepo, id); err != nil {
		return err
	}

	app, err := h.appRepo.FindByID(id)
	if err != nil {
		return response.NotFound(c, MsgAppNotFound)
	}

	if h.isRemoteApp(app) {
		return h.getRemoteAppHealth(c, app)
	}

	health := h.engine.GetAppHealth(c.Context(), app.Name)

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

func (h *ContainerHealthHandler) getRemoteAppHealth(c *fiber.Ctx, app *domain.App) error {
	host, err := h.resolveServerHost(app)
	if err != nil {
		h.logger.Error("failed to resolve server for health check", "app_id", app.ID, "error", err)
		return response.OK(c, ContainerHealthResponse{
			Name:   app.Name,
			Status: "unknown",
			Health: "none",
		})
	}

	var result ContainerStatsResponse
	var hasStats bool
	err = h.agentClient.GetContainerStats(c.Context(), host, h.agentPort, app.Name, func(stats *pb.ContainerStats) {
		hasStats = true
		result = ContainerStatsResponse{
			CPUPercent:  stats.CpuPercent,
			MemoryUsage: stats.MemoryUsageBytes,
			MemoryLimit: stats.MemoryLimitBytes,
		}
	})

	if err != nil || !hasStats {
		h.logger.Debug("remote container not reachable for health", "app_id", app.ID, "error", err)
		return response.OK(c, ContainerHealthResponse{
			Name:   app.Name,
			Status: "not_found",
			Health: "none",
		})
	}

	health := "none"
	status := "running"
	if result.MemoryUsage > 0 {
		health = "healthy"
	}

	return response.OK(c, ContainerHealthResponse{
		Name:   app.Name,
		Status: status,
		Health: health,
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

	if err := EnsureAppOwnership(c, h.appRepo, id); err != nil {
		return err
	}

	app, err := h.appRepo.FindByID(id)
	if err != nil {
		return response.NotFound(c, MsgAppNotFound)
	}

	if h.isRemoteApp(app) {
		return h.getRemoteContainerLogs(c, app, tail, follow)
	}

	if follow {
		return h.streamContainerLogs(c, c.Context(), app.Name)
	}

	logs, err := h.engine.ContainerLogs(c.Context(), app.Name, tail)
	if err != nil {
		return response.OK(c, ContainerLogsResponse{Logs: ""})
	}

	return response.OK(c, ContainerLogsResponse{Logs: logs})
}

func (h *ContainerHealthHandler) getRemoteContainerLogs(c *fiber.Ctx, app *domain.App, tail int, follow bool) error {
	host, err := h.resolveServerHost(app)
	if err != nil {
		return response.OK(c, ContainerLogsResponse{Logs: ""})
	}

	if follow {
		return h.streamRemoteContainerLogs(c, app, host)
	}

	var logs string
	err = h.agentClient.GetContainerLogs(c.Context(), host, h.agentPort, app.Name, tail, false, func(entry *pb.ContainerLogEntry) {
		logs += entry.Message + "\n"
	})
	if err != nil {
		h.logger.Error("failed to get remote logs", "app_id", app.ID, "error", err)
		return response.OK(c, ContainerLogsResponse{Logs: ""})
	}
	return response.OK(c, ContainerLogsResponse{Logs: logs})
}

func (h *ContainerHealthHandler) streamRemoteContainerLogs(c *fiber.Ctx, app *domain.App, host string) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		ctx := c.Context()
		err := h.agentClient.GetContainerLogs(ctx, host, h.agentPort, app.Name, 100, true, func(entry *pb.ContainerLogEntry) {
			fmt.Fprintf(w, "data: %s\n\n", entry.Message)
			_ = w.Flush()
		})
		if err != nil {
			h.logger.Debug("Remote log stream ended", "app_id", app.ID, "error", err)
		}
	}))

	return nil
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

	if err := EnsureAppOwnership(c, h.appRepo, id); err != nil {
		return err
	}

	app, err := h.appRepo.FindByID(id)
	if err != nil {
		return response.NotFound(c, MsgAppNotFound)
	}

	if h.isRemoteApp(app) {
		return h.getRemoteContainerStats(c, app)
	}

	stats, err := h.engine.ContainerStats(c.Context(), app.Name)
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

func (h *ContainerHealthHandler) getRemoteContainerStats(c *fiber.Ctx, app *domain.App) error {
	host, err := h.resolveServerHost(app)
	if err != nil {
		return response.OK(c, ContainerStatsResponse{})
	}

	var result ContainerStatsResponse
	err = h.agentClient.GetContainerStats(c.Context(), host, h.agentPort, app.Name, func(stats *pb.ContainerStats) {
		result = ContainerStatsResponse{
			CPUPercent:  stats.CpuPercent,
			MemoryUsage: stats.MemoryUsageBytes,
			MemoryLimit: stats.MemoryLimitBytes,
			NetworkRx:   stats.NetworkRxBytes,
			NetworkTx:   stats.NetworkTxBytes,
		}
		if stats.MemoryLimitBytes > 0 {
			result.MemoryPercent = float64(stats.MemoryUsageBytes) / float64(stats.MemoryLimitBytes) * 100
		}
	})
	if err != nil {
		h.logger.Error("failed to get remote stats", "app_id", app.ID, "error", err)
		return response.OK(c, ContainerStatsResponse{})
	}

	return response.OK(c, result)
}
