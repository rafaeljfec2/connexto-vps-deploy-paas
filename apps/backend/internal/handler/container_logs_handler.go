package handler

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"github.com/paasdeploy/backend/internal/response"
)

type ContainerLogsResponseGeneral struct {
	Logs string `json:"logs"`
}

func (h *ContainerHandler) GetContainerLogs(c *fiber.Ctx) error {
	id := c.Params("id")
	tailStr := c.Query("tail", "100")
	follow := c.Query("follow", "false") == "true"
	serverID := c.Query("serverId", "")

	if err := RequireAdminForLocal(c, serverID); err != nil {
		return err
	}

	tail, err := strconv.Atoi(tailStr)
	if err != nil {
		tail = 100
	}

	since, err := ParseSinceQuery(c)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	if serverID != "" {
		return h.getRemoteContainerLogs(c, serverID, id, tail, follow, since)
	}

	if follow {
		return h.streamContainerLogs(c, c.Context(), id, since)
	}

	logs, err := h.docker.ContainerLogs(c.Context(), id, tail, since)
	if err != nil {
		h.logger.Error("Failed to get container logs", "id", id, "error", err)
		return response.OK(c, ContainerLogsResponseGeneral{Logs: ""})
	}

	return response.OK(c, ContainerLogsResponseGeneral{Logs: logs})
}

func (h *ContainerHandler) getRemoteContainerLogs(c *fiber.Ctx, serverID, containerID string, tail int, follow bool, since *time.Time) error {
	host, err := h.resolveServerHost(serverID, GetUserFromContext(c).ID)
	if err != nil {
		return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
	}

	if follow {
		return h.streamRemoteContainerLogs(c, host, containerID, since)
	}

	var logLines []string
	err = h.agentClient.GetContainerLogs(c.Context(), host, h.agentPort, containerID, tail, false, since, func(entry *pb.ContainerLogEntry) {
		logLines = append(logLines, entry.GetMessage())
	})
	if err != nil {
		h.logger.Error("Failed to get remote container logs", "id", containerID, "serverId", serverID, "error", err)
		return response.OK(c, ContainerLogsResponseGeneral{Logs: ""})
	}

	return response.OK(c, ContainerLogsResponseGeneral{Logs: strings.Join(logLines, "\n")})
}

func (h *ContainerHandler) streamRemoteContainerLogs(c *fiber.Ctx, host, containerID string, since *time.Time) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = h.agentClient.GetContainerLogs(ctx, host, h.agentPort, containerID, 100, true, since, func(entry *pb.ContainerLogEntry) {
				fmt.Fprintf(w, "data: %s\n\n", entry.GetMessage())
				_ = w.Flush()
			})
		}()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				fmt.Fprintf(w, ": keepalive\n\n")
				if err := w.Flush(); err != nil {
					cancel()
					return
				}
			case <-done:
				return
			}
		}
	}))

	return nil
}

func (h *ContainerHandler) streamContainerLogs(c *fiber.Ctx, ctx context.Context, containerID string, since *time.Time) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		output := make(chan string, 100)
		done := make(chan struct{})

		go func() {
			defer close(done)
			_ = h.docker.StreamContainerLogs(ctx, containerID, since, output)
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
