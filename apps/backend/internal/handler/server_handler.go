package handler

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/crypto"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/provisioner"
	"github.com/paasdeploy/backend/internal/response"
)

var acmeEmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func validateAcmeEmail(email *string) error {
	if email == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*email)
	if trimmed == "" {
		return nil
	}
	if !acmeEmailRegex.MatchString(trimmed) {
		return domain.ErrInvalidInput
	}
	return nil
}

var agentStatsTimeout = 5 * time.Second

const (
	LatestAgentVersion = "0.6.0"
	msgProvisionFailed = "provision failed"
)

type UpdateAgentEnqueuer interface {
	EnqueueUpdateAgent(serverID string)
}

type ServerHandlerAgentDeps struct {
	HealthChecker        *agentclient.HealthChecker
	AgentClient          *agentclient.AgentClient
	AgentPort            int
	UpdateAgentEnqueuer  UpdateAgentEnqueuer
}

type ServerHandler struct {
	serverRepo          domain.ServerRepository
	tokenEncryptor       *crypto.TokenEncryptor
	provisioner          *provisioner.SSHProvisioner
	sseHandler           *SSEHandler
	healthChecker        *agentclient.HealthChecker
	agentClient          *agentclient.AgentClient
	agentPort            int
	updateAgentEnqueuer  UpdateAgentEnqueuer
	appService           AppsByServerLister
	logger               *slog.Logger
}

type AppsByServerLister interface {
	ListAppsByServerID(serverID, userID string) ([]domain.AppWithDeployment, error)
}

func NewServerHandler(
	serverRepo domain.ServerRepository,
	tokenEncryptor *crypto.TokenEncryptor,
	prov *provisioner.SSHProvisioner,
	sseHandler *SSEHandler,
	agentDeps ServerHandlerAgentDeps,
	appService AppsByServerLister,
	logger *slog.Logger,
) *ServerHandler {
	return &ServerHandler{
		serverRepo:         serverRepo,
		tokenEncryptor:     tokenEncryptor,
		provisioner:        prov,
		sseHandler:         sseHandler,
		healthChecker:      agentDeps.HealthChecker,
		agentClient:        agentDeps.AgentClient,
		agentPort:          agentDeps.AgentPort,
		updateAgentEnqueuer: agentDeps.UpdateAgentEnqueuer,
		appService:         appService,
		logger:             logger.With("handler", "server"),
	}
}

func (h *ServerHandler) Register(app fiber.Router) {
	v1 := app.Group(APIPrefix)
	servers := v1.Group("/servers")
	servers.Get("/", h.List)
	servers.Post("/", h.Create)
	servers.Get("/:id/stats", h.GetStats)
	servers.Get("/:id", h.Get)
	servers.Put("/:id", h.Update)
	servers.Delete("/:id", h.Delete)
	servers.Post("/:id/provision", h.Provision)
	servers.Post("/:id/update-agent", h.UpdateAgent)
	servers.Get("/:id/health", h.HealthCheck)
	servers.Get("/:id/apps", h.ListServerApps)
}

type ServerResponse struct {
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	Host                 string  `json:"host"`
	SSHPort              int     `json:"sshPort"`
	SSHUser              string  `json:"sshUser"`
	AcmeEmail            *string `json:"acmeEmail,omitempty"`
	Status               string  `json:"status"`
	AgentVersion         *string `json:"agentVersion,omitempty"`
	LatestAgentVersion   string  `json:"latestAgentVersion"`
	LastHeartbeatAt      *string `json:"lastHeartbeatAt,omitempty"`
	CreatedAt            string  `json:"createdAt"`
	UpdatedAt            string  `json:"updatedAt"`
}

type ServerHealthResponse struct {
	Status    string `json:"status"`
	LatencyMs int64  `json:"latencyMs"`
}


func toServerResponse(s *domain.Server) ServerResponse {
	resp := ServerResponse{
		ID:                 s.ID,
		Name:               s.Name,
		Host:               s.Host,
		SSHPort:            s.SSHPort,
		SSHUser:            s.SSHUser,
		AcmeEmail:          s.AcmeEmail,
		Status:             string(s.Status),
		LatestAgentVersion: LatestAgentVersion,
		CreatedAt:          s.CreatedAt.Format(DateTimeFormatISO8601),
		UpdatedAt:          s.UpdatedAt.Format(DateTimeFormatISO8601),
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
	Name        string  `json:"name"`
	Host        string  `json:"host"`
	SSHPort     int     `json:"sshPort"`
	SSHUser     string  `json:"sshUser"`
	SSHKey      string  `json:"sshKey"`
	SSHPassword string  `json:"sshPassword"`
	AcmeEmail   *string `json:"acmeEmail,omitempty"`
}

type UpdateServerRequest struct {
	Name        *string `json:"name,omitempty"`
	Host        *string `json:"host,omitempty"`
	SSHPort     *int    `json:"sshPort,omitempty"`
	SSHUser     *string `json:"sshUser,omitempty"`
	SSHKey      *string `json:"sshKey,omitempty"`
	SSHPassword *string `json:"sshPassword,omitempty"`
	AcmeEmail   *string `json:"acmeEmail,omitempty"`
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

func (h *ServerHandler) requireServerForUser(c *fiber.Ctx) (*domain.Server, *domain.User, error) {
	user := GetUserFromContext(c)
	if user == nil {
		return nil, nil, response.Unauthorized(c, MsgNotAuthenticated)
	}

	id := c.Params("id")
	server, err := h.serverRepo.FindByIDForUser(id, user.ID)
	if err != nil {
		return nil, nil, HandleNotFoundOrInternal(c, err, MsgServerNotFound)
	}
	return server, user, nil
}

func (h *ServerHandler) List(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	servers, err := h.serverRepo.FindAllByUserID(user.ID)
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

	if err := validateAcmeEmail(req.AcmeEmail); err != nil {
		return response.BadRequest(c, "invalid ACME email format")
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
		UserID:               user.ID,
		Name:                 req.Name,
		Host:                 req.Host,
		SSHPort:              req.SSHPort,
		SSHUser:              req.SSHUser,
		SSHKeyEncrypted:      sshKeyEncrypted,
		SSHPasswordEncrypted: sshPasswordEncrypted,
		AcmeEmail:            req.AcmeEmail,
	}

	server, err := h.serverRepo.Create(input)
	if err != nil {
		return HandleDomainError(c, err)
	}

	return response.Created(c, toServerResponse(server))
}

func (h *ServerHandler) Get(c *fiber.Ctx) error {
	server, _, err := h.requireServerForUser(c)
	if err != nil {
		return err
	}

	return response.OK(c, toServerResponse(server))
}

func (h *ServerHandler) GetStats(c *fiber.Ctx) error {
	server, _, err := h.requireServerForUser(c)
	if err != nil {
		return err
	}

	if h.agentClient == nil || h.agentPort == 0 {
		return response.ServerError(c, fiber.StatusServiceUnavailable, "agent stats not available")
	}

	ctx, cancel := context.WithTimeout(c.Context(), agentStatsTimeout)
	defer cancel()

	sysInfo, err := h.agentClient.GetSystemInfo(ctx, server.Host, h.agentPort)
	if err != nil {
		h.logger.Warn("get system info failed", "serverId", server.ID, "error", err)
		return response.ServerError(c, fiber.StatusServiceUnavailable, "agent unreachable; check if the agent is running and port 50052 is reachable")
	}

	sysMetrics, err := h.agentClient.GetSystemMetrics(ctx, server.Host, h.agentPort)
	if err != nil {
		h.logger.Warn("get system metrics failed", "serverId", server.ID, "error", err)
		return response.ServerError(c, fiber.StatusServiceUnavailable, "agent unreachable; check if the agent is running and port 50052 is reachable")
	}

	return response.OK(c, fiber.Map{
		"systemInfo":   sysInfo,
		"systemMetrics": sysMetrics,
	})
}

func (h *ServerHandler) Update(c *fiber.Ctx) error {
	server, _, err := h.requireServerForUser(c)
	if err != nil {
		return err
	}

	id := server.ID
	var req UpdateServerRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, MsgInvalidRequestBody)
	}

	if err := validateAcmeEmail(req.AcmeEmail); err != nil {
		return response.BadRequest(c, "invalid ACME email format")
	}

	input := domain.UpdateServerInput{
		Name:      req.Name,
		Host:      req.Host,
		SSHPort:   req.SSHPort,
		SSHUser:   req.SSHUser,
		AcmeEmail: req.AcmeEmail,
	}
	if err := applyUpdateSSHCredentials(h.tokenEncryptor, &req, &input); err != nil {
		h.logger.Error("failed to encrypt ssh credentials", "error", err)
		return response.InternalError(c)
	}

	updated, err := h.serverRepo.Update(id, input)
	if err != nil {
		return HandleDomainError(c, err)
	}

	return response.OK(c, toServerResponse(updated))
}

func (h *ServerHandler) Delete(c *fiber.Ctx) error {
	server, _, err := h.requireServerForUser(c)
	if err != nil {
		return err
	}

	id := server.ID

	if h.provisioner != nil {
		sshKey, sshPassword, decErr := h.decryptProvisionCredentials(server)
		if decErr == nil && (sshKey != "" || sshPassword != "") {
			if depErr := h.provisioner.Deprovision(server, sshKey, sshPassword); depErr != nil {
				h.logger.Warn("deprovision failed, deleting from db anyway",
					"serverId", id, "host", server.Host, "error", depErr)
			}
		}
	}

	if err := h.serverRepo.Delete(id); err != nil {
		return HandleNotFoundOrInternal(c, err, MsgServerNotFound)
	}

	return response.NoContent(c)
}

func (h *ServerHandler) Provision(c *fiber.Ctx) error {
	server, _, err := h.requireServerForUser(c)
	if err != nil {
		return err
	}

	id := server.ID

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

	progress := (*provisioner.ProvisionProgress)(nil)
	if h.sseHandler != nil {
		progress = &provisioner.ProvisionProgress{
			OnStep: func(step, status, message string) {
				h.sseHandler.EmitProvisionStep(id, step, status, message)
			},
			OnLog: func(message string) {
				h.sseHandler.EmitProvisionLog(id, message)
			},
		}
	}

	if err := h.provisioner.Provision(server, sshKey, sshPassword, progress); err != nil {
		userMsg := err.Error()
		if h.sseHandler != nil {
			h.sseHandler.EmitProvisionFailed(id, userMsg)
		}
		h.logProvisionFailure(c, id, err)
		errStatus := domain.ServerStatusError
		_, _ = h.serverRepo.Update(id, domain.UpdateServerInput{Status: &errStatus})
		return response.BadRequest(c, userMsg)
	}

	if h.sseHandler != nil {
		h.sseHandler.EmitProvisionCompleted(id)
	}
	onlineStatus := domain.ServerStatusOnline
	_, _ = h.serverRepo.Update(id, domain.UpdateServerInput{Status: &onlineStatus})
	return response.OK(c, map[string]string{"message": "provision completed"})
}

func (h *ServerHandler) HealthCheck(c *fiber.Ctx) error {
	server, _, err := h.requireServerForUser(c)
	if err != nil {
		return err
	}

	if h.healthChecker == nil || h.agentPort == 0 {
		return response.BadRequest(c, "health check not available")
	}

	latency, err := h.healthChecker.Check(context.Background(), server.Host, h.agentPort)
	if err != nil {
		h.logger.Error("health check failed", "serverId", server.ID, "error", err)
		return response.BadRequest(c, "health check failed")
	}

	return response.OK(c, ServerHealthResponse{
		Status:    "ok",
		LatencyMs: latency.Milliseconds(),
	})
}

func (h *ServerHandler) UpdateAgent(c *fiber.Ctx) error {
	server, _, err := h.requireServerForUser(c)
	if err != nil {
		return err
	}

	id := server.ID

	if h.updateAgentEnqueuer == nil {
		return response.ServerError(c, fiber.StatusServiceUnavailable, "update agent not available")
	}

	h.updateAgentEnqueuer.EnqueueUpdateAgent(id)

	if h.sseHandler != nil {
		h.sseHandler.EmitAgentUpdateEnqueued(id)
	}

	return response.Accepted(c, fiber.Map{
		"message": "update agent command enqueued; agent will receive it on next heartbeat",
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
	h.logger.Error(msgProvisionFailed, attrs...)
}

func (h *ServerHandler) ListServerApps(c *fiber.Ctx) error {
	server, user, err := h.requireServerForUser(c)
	if err != nil {
		return err
	}

	apps, err := h.appService.ListAppsByServerID(server.ID, user.ID)
	if err != nil {
		h.logger.Error("failed to list server apps", "serverId", server.ID, "error", err)
		return response.InternalError(c)
	}

	return response.OK(c, apps)
}
