package handler

import (
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
)

type ContainerSSLHandler struct {
	agentClient *agentclient.AgentClient
	serverRepo  domain.ServerRepository
	agentPort   int
	logger      *slog.Logger
}

type ContainerSSLHandlerConfig struct {
	AgentClient *agentclient.AgentClient
	ServerRepo  domain.ServerRepository
	AgentPort   int
	Logger      *slog.Logger
}

func NewContainerSSLHandler(cfg ContainerSSLHandlerConfig) *ContainerSSLHandler {
	return &ContainerSSLHandler{
		agentClient: cfg.AgentClient,
		serverRepo:  cfg.ServerRepo,
		agentPort:   cfg.AgentPort,
		logger:      cfg.Logger,
	}
}

func (h *ContainerSSLHandler) Register(app fiber.Router) {
	v1 := app.Group(APIPrefix)
	v1.Post("/containers/:id/ssl", h.ConfigureSSL)
	v1.Get("/containers/:id/ssl", h.GetSSLStatus)
}

type ConfigureSSLRequest struct {
	ServerID     string `json:"serverId"`
	DatabaseType string `json:"databaseType"`
	DatabaseUser string `json:"databaseUser"`
	DatabaseName string `json:"databaseName"`
}

type SSLStatusResponse struct {
	SSLEnabled       bool   `json:"sslEnabled"`
	TLSVersion       string `json:"tlsVersion,omitempty"`
	Cipher           string `json:"cipher,omitempty"`
	CertificateExpiry string `json:"certificateExpiry,omitempty"`
	ConnectionString string `json:"connectionString,omitempty"`
}

func (h *ContainerSSLHandler) ConfigureSSL(c *fiber.Ctx) error {
	containerID := c.Params("id")

	var req ConfigureSSLRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.ServerID == "" {
		return response.BadRequest(c, "serverId is required")
	}

	if req.DatabaseType == "" {
		req.DatabaseType = "postgresql"
	}

	if req.DatabaseUser == "" {
		return response.BadRequest(c, "databaseUser is required")
	}

	if req.DatabaseName == "" {
		return response.BadRequest(c, "databaseName is required")
	}

	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	server, err := h.serverRepo.FindByIDForUser(req.ServerID, user.ID)
	if err != nil {
		return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
	}

	grpcReq := &pb.ConfigureContainerSSLRequest{
		ContainerId:  containerID,
		DatabaseType: req.DatabaseType,
		DatabaseUser: req.DatabaseUser,
		DatabaseName: req.DatabaseName,
	}

	resp, err := h.agentClient.ConfigureContainerSSL(c.Context(), server.Host, h.agentPort, grpcReq)
	if err != nil {
		h.logger.Error("Failed to configure container SSL", "containerId", containerID, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to configure SSL")
	}

	if !resp.Success {
		return response.ServerError(c, fiber.StatusInternalServerError, fmt.Sprintf("SSL configuration failed: %s", resp.Message))
	}

	result := SSLStatusResponse{
		SSLEnabled:        resp.SslEnabled,
		TLSVersion:        resp.TlsVersion,
		CertificateExpiry: resp.CertificateExpiry,
	}

	return response.OK(c, result)
}

func (h *ContainerSSLHandler) GetSSLStatus(c *fiber.Ctx) error {
	containerID := c.Params("id")
	serverID := c.Query("serverId", "")
	dbType := c.Query("databaseType", "postgresql")
	dbUser := c.Query("databaseUser", "")
	dbName := c.Query("databaseName", "")

	if serverID == "" || dbUser == "" || dbName == "" {
		return response.BadRequest(c, "serverId, databaseUser and databaseName are required")
	}

	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	server, err := h.serverRepo.FindByIDForUser(serverID, user.ID)
	if err != nil {
		return response.ServerError(c, fiber.StatusInternalServerError, MsgServerNotFound)
	}

	grpcReq := &pb.GetContainerSSLStatusRequest{
		ContainerId:  containerID,
		DatabaseType: dbType,
		DatabaseUser: dbUser,
		DatabaseName: dbName,
	}

	resp, err := h.agentClient.GetContainerSSLStatus(c.Context(), server.Host, h.agentPort, grpcReq)
	if err != nil {
		h.logger.Error("Failed to get container SSL status", "containerId", containerID, "error", err)
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to get SSL status")
	}

	result := SSLStatusResponse{
		SSLEnabled:        resp.SslEnabled,
		TLSVersion:        resp.TlsVersion,
		Cipher:            resp.Cipher,
		CertificateExpiry: resp.CertificateExpiry,
	}

	if resp.SslEnabled {
		port := h.resolveContainerPort(c, containerID, serverID, user.ID)
		result.ConnectionString = fmt.Sprintf(
			"postgresql://%s@%s:%d/%s?sslmode=require",
			dbUser, server.Host, port, dbName,
		)
	}

	return response.OK(c, result)
}

func (h *ContainerSSLHandler) resolveContainerPort(c *fiber.Ctx, containerID, serverID, userID string) int {
	server, err := h.serverRepo.FindByIDForUser(serverID, userID)
	if err != nil {
		return 5432
	}

	containers, err := h.agentClient.ListContainers(c.Context(), server.Host, h.agentPort, true, "")
	if err != nil {
		return 5432
	}

	for _, container := range containers {
		if container.Id == containerID || container.Name == containerID {
			for _, port := range container.Ports {
				if port.ContainerPort == 5432 && port.HostPort > 0 {
					return int(port.HostPort)
				}
			}
		}
	}

	return 5432
}
