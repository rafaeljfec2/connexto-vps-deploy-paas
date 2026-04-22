package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/shared/pkg/docker"
)

type HealthcheckResponse struct {
	Success    bool     `json:"success"`
	ExitCode   int32    `json:"exitCode"`
	Stdout     string   `json:"stdout"`
	Stderr     string   `json:"stderr"`
	Command    []string `json:"command"`
	DurationMs int64    `json:"durationMs"`
}

const msgFailedRunHealthcheck = "Failed to run healthcheck"

func mapAgentHealthcheckResponse(resp *pb.RunContainerHealthcheckResponse) HealthcheckResponse {
	return HealthcheckResponse{
		Success:    resp.Success,
		ExitCode:   resp.ExitCode,
		Stdout:     resp.Stdout,
		Stderr:     resp.Stderr,
		Command:    resp.Command,
		DurationMs: resp.DurationMs,
	}
}

func mapLocalHealthcheckResult(result *docker.HealthcheckResult) HealthcheckResponse {
	return HealthcheckResponse{
		Success:    result.ExitCode == 0,
		ExitCode:   int32(result.ExitCode),
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
		Command:    result.Command,
		DurationMs: result.DurationMs,
	}
}

func (h *ContainerHandler) RunContainerHealthcheck(c *fiber.Ctx) error {
	id := c.Params("id")
	serverID := c.Query("serverId", "")

	if err := RequireAdminForLocal(c, serverID); err != nil {
		return err
	}

	if id == "" {
		return response.BadRequest(c, "container id is required")
	}

	if serverID != "" {
		return h.runRemoteHealthcheck(c, serverID, id)
	}

	return h.runLocalHealthcheck(c, id)
}

func (h *ContainerHandler) runRemoteHealthcheck(c *fiber.Ctx, serverID, containerID string) error {
	host, err := h.resolveServerHost(serverID, GetUserFromContext(c).ID)
	if err != nil {
		return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
	}

	resp, err := h.agentClient.RunContainerHealthcheck(c.Context(), host, h.agentPort, containerID)
	if err != nil {
		h.logger.Error("Failed to run healthcheck on remote container", "id", containerID, "serverId", serverID, "error", err)
		if isUnimplementedAgentErr(err) {
			return response.Conflict(c, msgAgentOutdated)
		}
		return response.ServerError(c, fiber.StatusInternalServerError, msgFailedRunHealthcheck)
	}

	if resp.NotConfigured {
		return response.ServerError(c, fiber.StatusUnprocessableEntity, resp.Message)
	}

	return response.OK(c, mapAgentHealthcheckResponse(resp))
}

func (h *ContainerHandler) runLocalHealthcheck(c *fiber.Ctx, containerID string) error {
	result, err := h.docker.RunHealthcheck(c.Context(), containerID)
	if err != nil {
		var notConfigured *docker.HealthcheckNotConfiguredError
		if errors.As(err, &notConfigured) {
			return response.ServerError(c, fiber.StatusUnprocessableEntity, notConfigured.Error())
		}
		h.logger.Error("Failed to run healthcheck", "id", containerID, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, msgFailedRunHealthcheck)
	}

	return response.OK(c, mapLocalHealthcheckResult(result))
}
