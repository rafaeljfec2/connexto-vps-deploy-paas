package ghclient

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
)

type WebhookPayloadStore interface {
	SavePayload(ctx context.Context, deliveryID, eventType, provider string, payload []byte, outcome string, errMsg *string) error
}

const (
	HeaderGitHubEvent     = "X-GitHub-Event"
	HeaderGitHubSignature = "X-Hub-Signature-256"
	HeaderGitHubDelivery  = "X-GitHub-Delivery"

	EventPush = "push"
	EventPing = "ping"

	RefPrefix = "refs/heads/"
)

type AppFinder interface {
	FindByRepoURL(repoURL string) (*domain.App, error)
}

type DeploymentCreator interface {
	Create(input domain.CreateDeploymentInput) (*domain.Deployment, error)
	FindPendingByAppID(appID string) (*domain.Deployment, error)
}

type DeployAuditLogger interface {
	LogDeployStarted(ctx context.Context, deployID, appID, appName, commitSHA string)
}

type WebhookHandler struct {
	appFinder         AppFinder
	deploymentCreator DeploymentCreator
	deployAudit       DeployAuditLogger
	payloadStore      WebhookPayloadStore
	webhookSecret     string
	logger            *slog.Logger
}

func NewWebhookHandler(
	appFinder AppFinder,
	deploymentCreator DeploymentCreator,
	deployAudit DeployAuditLogger,
	payloadStore WebhookPayloadStore,
	webhookSecret string,
	logger *slog.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		appFinder:         appFinder,
		deploymentCreator: deploymentCreator,
		deployAudit:       deployAudit,
		payloadStore:      payloadStore,
		webhookSecret:     webhookSecret,
		logger:            logger,
	}
}

func (h *WebhookHandler) Register(app *fiber.App) {
	v1 := app.Group("/paas-deploy/v1")
	webhooks := v1.Group("/webhooks")
	webhooks.All("/github", h.handleWebhookRoute)
}

func (h *WebhookHandler) handleWebhookRoute(c *fiber.Ctx) error {
	switch c.Method() {
	case fiber.MethodGet:
		return h.HandleWebhookHealth(c)
	case fiber.MethodPost:
		return h.HandleWebhook(c)
	default:
		return fiber.NewError(fiber.StatusMethodNotAllowed,
			"Use GET for health check, POST for webhook delivery (ping/push)")
	}
}

func (h *WebhookHandler) HandleWebhookHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"ok":      true,
		"message": "webhook endpoint ready - POST with X-GitHub-Event for ping/push",
	})
}

// HandleWebhook godoc
//
//	@Summary		Recebe eventos do GitHub
//	@Description	Endpoint que recebe push events do GitHub para disparar deploys automaticos
//	@Tags			webhooks
//	@Accept			json
//	@Produce		json
//	@Param			X-GitHub-Event		header	string		true	"Tipo do evento (push, ping)"
//	@Param			X-Hub-Signature-256	header	string		true	"Assinatura HMAC-SHA256"
//	@Param			X-GitHub-Delivery	header	string		true	"ID unico da entrega"
//	@Param			payload				body	PushEvent	true	"Payload do evento"
//	@Success		200					{object}	map[string]string	"Evento processado (ping ou branch ignorado)"
//	@Success		202					{object}	map[string]string	"Deploy iniciado"
//	@Failure		400					{object}	map[string]string	"Payload invalido"
//	@Failure		401					{object}	map[string]string	"Assinatura invalida"
//	@Router			/webhooks/github [post]
func (h *WebhookHandler) HandleWebhook(c *fiber.Ctx) error {
	deliveryID := c.Get(HeaderGitHubDelivery)
	event := c.Get(HeaderGitHubEvent)
	signature := c.Get(HeaderGitHubSignature)
	body := c.Body()

	logger := h.logger.With(
		slog.String("delivery_id", deliveryID),
		slog.String("event", event),
	)

	if event == EventPing {
		logger.Info("received ping event")
		h.savePayloadAsync(c.Context(), deliveryID, event, body, "pong", nil)
		return response.OK(c, map[string]string{"message": "pong"})
	}

	if event != EventPush {
		logger.Info("ignoring unsupported event")
		h.savePayload(c.Context(), deliveryID, event, body, "ignored", nil)
		return response.OK(c, map[string]string{"message": "event ignored"})
	}

	h.savePayload(c.Context(), deliveryID, event, body, "received", nil)

	if !ValidateSignature(body, signature, h.webhookSecret) {
		logger.Warn("invalid webhook signature")
		errMsg := "invalid signature"
		h.savePayload(c.Context(), deliveryID, event, body, "invalid_signature", &errMsg)
		return response.Unauthorized(c, "invalid signature")
	}

	var pushEvent PushEvent
	if err := json.Unmarshal(body, &pushEvent); err != nil {
		logger.Error("failed to parse push event", slog.String("error", err.Error()))
		errStr := err.Error()
		h.savePayload(c.Context(), deliveryID, event, body, "parse_error", &errStr)
		return response.BadRequest(c, "invalid payload")
	}

	return h.handlePushEvent(c, logger, &pushEvent, deliveryID, event, body)
}

func (h *WebhookHandler) savePayload(ctx context.Context, deliveryID, eventType string, payload []byte, outcome string, errMsg *string) {
	if h.payloadStore == nil {
		return
	}
	if deliveryID == "" {
		deliveryID = "unknown-" + uuid.New().String()
	}
	if err := h.payloadStore.SavePayload(ctx, deliveryID, eventType, "github", payload, outcome, errMsg); err != nil {
		h.logger.Warn("failed to save webhook payload", "delivery_id", deliveryID, "error", err)
	}
}

func (h *WebhookHandler) savePayloadAsync(ctx context.Context, deliveryID, eventType string, payload []byte, outcome string, errMsg *string) {
	if h.payloadStore == nil {
		return
	}
	bodyCopy := make([]byte, len(payload))
	copy(bodyCopy, payload)
	var errMsgCopy *string
	if errMsg != nil {
		s := *errMsg
		errMsgCopy = &s
	}
	go func() {
		h.savePayload(context.Background(), deliveryID, eventType, bodyCopy, outcome, errMsgCopy)
	}()
}

func (h *WebhookHandler) handlePushEvent(c *fiber.Ctx, logger *slog.Logger, event *PushEvent, deliveryID, eventType string, body []byte) error {
	if event.Repository == nil {
		logger.Warn("push event missing repository data")
		outcome := "missing_repository"
		h.savePayload(c.Context(), deliveryID, eventType, body, outcome, nil)
		return response.OK(c, map[string]string{"message": "missing repository"})
	}

	branch := extractBranch(event.Ref)
	if branch == "" {
		logger.Info("ignoring non-branch push", slog.String("ref", event.Ref))
		h.savePayload(c.Context(), deliveryID, eventType, body, "ignored", strPtr("non-branch push"))
		return response.OK(c, map[string]string{"message": "non-branch push ignored"})
	}

	logger = logger.With(
		slog.String("repo", event.Repository.FullName),
		slog.String("branch", branch),
		slog.String("commit", event.After),
	)

	if event.Deleted {
		logger.Info("ignoring branch deletion event")
		h.savePayload(c.Context(), deliveryID, eventType, body, "ignored", strPtr("branch deleted"))
		return response.OK(c, map[string]string{"message": "branch deletion ignored"})
	}

	app, err := h.findAppByRepository(event.Repository, logger)
	if err != nil {
		errStr := err.Error()
		h.savePayload(c.Context(), deliveryID, eventType, body, "error", &errStr)
		return response.InternalError(c)
	}

	if app == nil {
		logger.Info("no app registered for repository")
		h.savePayload(c.Context(), deliveryID, eventType, body, "ignored", strPtr("repository not registered"))
		return response.OK(c, map[string]string{"message": "repository not registered"})
	}

	if app.Branch != branch {
		logger.Info("push to non-tracked branch", slog.String("app_branch", app.Branch))
		h.savePayload(c.Context(), deliveryID, eventType, body, "ignored", strPtr("branch not tracked"))
		return response.OK(c, map[string]string{"message": "branch not tracked"})
	}

	if hasPending, err := h.hasPendingDeployment(app.ID, logger); err != nil {
		errStr := err.Error()
		h.savePayload(c.Context(), deliveryID, eventType, body, "error", &errStr)
		return response.InternalError(c)
	} else if hasPending {
		logger.Info("deployment already pending for app", slog.String("app_id", app.ID))
		h.savePayload(c.Context(), deliveryID, eventType, body, "ignored", strPtr("deployment pending"))
		return response.OK(c, map[string]string{"message": "deployment already pending"})
	}

	return h.createDeployment(c, logger, app, event, deliveryID, eventType, body)
}

func (h *WebhookHandler) findAppByRepository(repo *Repository, logger *slog.Logger) (*domain.App, error) {
	repoURLs := getRepoURLVariants(repo)

	for _, repoURL := range repoURLs {
		app, err := h.appFinder.FindByRepoURL(repoURL)
		if err == nil && app != nil {
			return app, nil
		}
		if !errors.Is(err, domain.ErrNotFound) && err != nil {
			logger.Error("error finding app", slog.String("error", err.Error()))
			return nil, err
		}
	}

	return nil, nil
}

func (h *WebhookHandler) hasPendingDeployment(appID string, logger *slog.Logger) (bool, error) {
	pending, err := h.deploymentCreator.FindPendingByAppID(appID)
	if errors.Is(err, domain.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		logger.Error("error checking pending deployments", slog.String("error", err.Error()))
		return false, err
	}
	return pending != nil, nil
}

func strPtr(s string) *string {
	return &s
}

func (h *WebhookHandler) createDeployment(c *fiber.Ctx, logger *slog.Logger, app *domain.App, event *PushEvent, deliveryID, eventType string, body []byte) error {
	commitMessage := getCommitMessage(event)

	input := domain.CreateDeploymentInput{
		AppID:         app.ID,
		CommitSHA:     event.After,
		CommitMessage: commitMessage,
	}

	deployment, err := h.deploymentCreator.Create(input)
	if err != nil {
		logger.Error("failed to create deployment", slog.String("error", err.Error()))
		errStr := err.Error()
		h.savePayload(c.Context(), deliveryID, eventType, body, "error", &errStr)
		return response.InternalError(c)
	}

	h.savePayload(c.Context(), deliveryID, eventType, body, "deployment_queued", nil)

	if h.deployAudit != nil {
		h.deployAudit.LogDeployStarted(context.Background(), deployment.ID, app.ID, app.Name, event.After)
	}

	logger.Info("deployment queued",
		slog.String("app_id", app.ID),
		slog.String("app_name", app.Name),
		slog.String("deployment_id", deployment.ID),
	)

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"message":      "deployment queued",
			"deploymentId": deployment.ID,
			"appId":        app.ID,
			"commitSha":    event.After,
		},
		"meta": fiber.Map{
			"traceId": c.Locals("traceId"),
		},
	})
}

func getCommitMessage(event *PushEvent) string {
	if event.HeadCommit != nil && event.HeadCommit.Message != "" {
		return truncateString(event.HeadCommit.Message, 200)
	}
	return "Push from GitHub"
}

func extractBranch(ref string) string {
	if !strings.HasPrefix(ref, RefPrefix) {
		return ""
	}
	return strings.TrimPrefix(ref, RefPrefix)
}

func getRepoURLVariants(repo *Repository) []string {
	variants := make([]string, 0, 4)

	if repo.CloneURL != "" {
		variants = append(variants, repo.CloneURL)
		variants = append(variants, strings.TrimSuffix(repo.CloneURL, ".git"))
	}

	if repo.SSHURL != "" {
		variants = append(variants, repo.SSHURL)
	}

	if repo.HTMLURL != "" {
		variants = append(variants, repo.HTMLURL)
	}

	return variants
}

func truncateString(s string, maxLen int) string {
	firstLine := strings.Split(s, "\n")[0]
	if len(firstLine) <= maxLen {
		return firstLine
	}
	return firstLine[:maxLen-3] + "..."
}
