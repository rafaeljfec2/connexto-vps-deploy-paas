package handler

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/crypto"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/provisioner"
	"github.com/paasdeploy/backend/internal/response"
)

type ServerHandler struct {
	serverRepo     domain.ServerRepository
	tokenEncryptor *crypto.TokenEncryptor
	provisioner    *provisioner.SSHProvisioner
	healthChecker  *agentclient.HealthChecker
	agentPort      int
	logger         *slog.Logger
}

func NewServerHandler(
	serverRepo domain.ServerRepository,
	tokenEncryptor *crypto.TokenEncryptor,
	prov *provisioner.SSHProvisioner,
	healthChecker *agentclient.HealthChecker,
	agentPort int,
	logger *slog.Logger,
) *ServerHandler {
	return &ServerHandler{
		serverRepo:     serverRepo,
		tokenEncryptor: tokenEncryptor,
		provisioner:    prov,
		healthChecker:  healthChecker,
		agentPort:      agentPort,
		logger:         logger.With("handler", "server"),
	}
}

func (h *ServerHandler) Register(app fiber.Router) {
	v1 := app.Group(APIPrefix)
	servers := v1.Group("/servers")
	servers.Get("/", h.List)
	servers.Post("/", h.Create)
	servers.Get("/:id", h.Get)
	servers.Put("/:id", h.Update)
	servers.Delete("/:id", h.Delete)
	servers.Post("/:id/provision", h.Provision)
	servers.Get("/:id/health", h.HealthCheck)
}

type ServerResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Host            string  `json:"host"`
	SSHPort         int     `json:"sshPort"`
	SSHUser         string  `json:"sshUser"`
	Status          string  `json:"status"`
	AgentVersion    *string `json:"agentVersion,omitempty"`
	LastHeartbeatAt *string `json:"lastHeartbeatAt,omitempty"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

type ServerHealthResponse struct {
	Status    string `json:"status"`
	LatencyMs int64  `json:"latencyMs"`
}

func toServerResponse(s *domain.Server) ServerResponse {
	resp := ServerResponse{
		ID:        s.ID,
		Name:      s.Name,
		Host:      s.Host,
		SSHPort:   s.SSHPort,
		SSHUser:   s.SSHUser,
		Status:    string(s.Status),
		CreatedAt: s.CreatedAt.Format(DateTimeFormatISO8601),
		UpdatedAt: s.UpdatedAt.Format(DateTimeFormatISO8601),
	}
	if s.AgentVersion != nil {
		resp.AgentVersion = s.AgentVersion
	}
	if s.LastHeartbeatAt != nil {
		t := s.LastHeartbeatAt.Format(DateTimeFormatISO8601)
		resp.LastHeartbeatAt = &t
	}
	return resp
}

type CreateServerRequest struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	SSHPort     int    `json:"sshPort"`
	SSHUser     string `json:"sshUser"`
	SSHKey      string `json:"sshKey"`
	SSHPassword string `json:"sshPassword"`
}

type UpdateServerRequest struct {
	Name        *string `json:"name,omitempty"`
	Host        *string `json:"host,omitempty"`
	SSHPort     *int    `json:"sshPort,omitempty"`
	SSHUser     *string `json:"sshUser,omitempty"`
	SSHKey      *string `json:"sshKey,omitempty"`
	SSHPassword *string `json:"sshPassword,omitempty"`
}

func encryptCredential(encryptor *crypto.TokenEncryptor, plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	if encryptor != nil {
		return encryptor.Encrypt(plain)
	}
	return plain, nil
}

func applyUpdateSSHCredentials(encryptor *crypto.TokenEncryptor, req *UpdateServerRequest, input *domain.UpdateServerInput) error {
	if req.SSHKey != nil && *req.SSHKey != "" {
		encrypted, err := encryptCredential(encryptor, *req.SSHKey)
		if err != nil {
			return err
		}
		input.SSHKeyEncrypted = &encrypted
	}
	if req.SSHPassword != nil && *req.SSHPassword != "" {
		encrypted, err := encryptCredential(encryptor, *req.SSHPassword)
		if err != nil {
			return err
		}
		input.SSHPasswordEncrypted = &encrypted
	}
	return nil
}

func (h *ServerHandler) List(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	servers, err := h.serverRepo.FindAll()
	if err != nil {
		h.logger.Error("failed to list servers", "error", err)
		return response.InternalError(c)
	}

	resp := make([]ServerResponse, len(servers))
	for i := range servers {
		resp[i] = toServerResponse(&servers[i])
	}
	return response.OK(c, resp)
}

func (h *ServerHandler) Create(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	var req CreateServerRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, MsgInvalidRequestBody)
	}

	if req.Name == "" || req.Host == "" || req.SSHUser == "" {
		return response.BadRequest(c, "name, host and sshUser are required")
	}
	if req.SSHKey == "" && req.SSHPassword == "" {
		return response.BadRequest(c, "provide sshKey or sshPassword")
	}

	sshKeyEncrypted, err := encryptCredential(h.tokenEncryptor, req.SSHKey)
	if err != nil {
		h.logger.Error("failed to encrypt ssh key", "error", err)
		return response.InternalError(c)
	}
	sshPasswordEncrypted, err := encryptCredential(h.tokenEncryptor, req.SSHPassword)
	if err != nil {
		h.logger.Error("failed to encrypt ssh password", "error", err)
		return response.InternalError(c)
	}

	input := domain.CreateServerInput{
		Name:                 req.Name,
		Host:                 req.Host,
		SSHPort:              req.SSHPort,
		SSHUser:              req.SSHUser,
		SSHKeyEncrypted:      sshKeyEncrypted,
		SSHPasswordEncrypted: sshPasswordEncrypted,
	}

	server, err := h.serverRepo.Create(input)
	if err != nil {
		return HandleDomainError(c, err)
	}

	return response.Created(c, toServerResponse(server))
}

func (h *ServerHandler) Get(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	id := c.Params("id")
	server, err := h.serverRepo.FindByID(id)
	if err != nil {
		return HandleNotFoundOrInternal(c, err, MsgServerNotFound)
	}

	return response.OK(c, toServerResponse(server))
}

func (h *ServerHandler) Update(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	id := c.Params("id")
	_, err := h.serverRepo.FindByID(id)
	if err != nil {
		return HandleNotFoundOrInternal(c, err, MsgServerNotFound)
	}

	var req UpdateServerRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, MsgInvalidRequestBody)
	}

	input := domain.UpdateServerInput{
		Name:    req.Name,
		Host:    req.Host,
		SSHPort: req.SSHPort,
		SSHUser: req.SSHUser,
	}
	if err := applyUpdateSSHCredentials(h.tokenEncryptor, &req, &input); err != nil {
		h.logger.Error("failed to encrypt ssh credentials", "error", err)
		return response.InternalError(c)
	}

	server, err := h.serverRepo.Update(id, input)
	if err != nil {
		return HandleDomainError(c, err)
	}

	return response.OK(c, toServerResponse(server))
}

func (h *ServerHandler) Delete(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	id := c.Params("id")
	if err := h.serverRepo.Delete(id); err != nil {
		return HandleNotFoundOrInternal(c, err, MsgServerNotFound)
	}

	return response.NoContent(c)
}

func (h *ServerHandler) Provision(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	id := c.Params("id")
	server, err := h.serverRepo.FindByID(id)
	if err != nil {
		return HandleNotFoundOrInternal(c, err, MsgServerNotFound)
	}

	if h.provisioner == nil {
		return response.BadRequest(c, "provisioning not available: GRPC and PKI must be configured")
	}

	sshKey, sshPassword, err := h.decryptProvisionCredentials(server)
	if err != nil {
		return response.InternalError(c)
	}

	if sshKey == "" && sshPassword == "" {
		return response.BadRequest(c, "server has no ssh credentials")
	}

	status := domain.ServerStatusProvisioning
	_, _ = h.serverRepo.Update(id, domain.UpdateServerInput{Status: &status})

	if err := h.provisioner.Provision(server, sshKey, sshPassword); err != nil {
		h.logProvisionFailure(c, id, err)
		errStatus := domain.ServerStatusError
		_, _ = h.serverRepo.Update(id, domain.UpdateServerInput{Status: &errStatus})
		return response.BadRequest(c, "provision failed: "+err.Error())
	}

	onlineStatus := domain.ServerStatusOnline
	_, _ = h.serverRepo.Update(id, domain.UpdateServerInput{Status: &onlineStatus})
	return response.OK(c, map[string]string{"message": "provision completed"})
}

func (h *ServerHandler) HealthCheck(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	id := c.Params("id")
	server, err := h.serverRepo.FindByID(id)
	if err != nil {
		return HandleNotFoundOrInternal(c, err, MsgServerNotFound)
	}

	if h.healthChecker == nil || h.agentPort == 0 {
		return response.BadRequest(c, "health check not available")
	}

	latency, err := h.healthChecker.Check(context.Background(), server.Host, h.agentPort)
	if err != nil {
		return response.BadRequest(c, "health check failed: "+err.Error())
	}

	return response.OK(c, ServerHealthResponse{
		Status:    "ok",
		LatencyMs: latency.Milliseconds(),
	})
}

func (h *ServerHandler) decryptProvisionCredentials(server *domain.Server) (sshKey, sshPassword string, err error) {
	sshKey = server.SSHKeyEncrypted
	if h.tokenEncryptor != nil && sshKey != "" {
		sshKey, err = h.tokenEncryptor.Decrypt(server.SSHKeyEncrypted)
		if err != nil {
			h.logger.Error("failed to decrypt ssh key", "error", err)
			return "", "", err
		}
	}
	sshPassword = server.SSHPasswordEncrypted
	if h.tokenEncryptor != nil && sshPassword != "" {
		sshPassword, err = h.tokenEncryptor.Decrypt(server.SSHPasswordEncrypted)
		if err != nil {
			h.logger.Error("failed to decrypt ssh password", "error", err)
			return "", "", err
		}
	}
	return sshKey, sshPassword, nil
}

func (h *ServerHandler) logProvisionFailure(c *fiber.Ctx, serverID string, err error) {
	attrs := []any{"serverId", serverID, "error", err}
	if tid, ok := c.Locals("traceId").(string); ok && tid != "" {
		attrs = append(attrs, "traceId", tid)
	}
	h.logger.Error("provision failed", attrs...)
}
