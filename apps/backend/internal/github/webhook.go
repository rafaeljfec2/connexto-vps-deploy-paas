package github

import (
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
)

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

type WebhookHandler struct {
	appFinder         AppFinder
	deploymentCreator DeploymentCreator
	webhookSecret     string
	logger            *slog.Logger
}

func NewWebhookHandler(
	appFinder AppFinder,
	deploymentCreator DeploymentCreator,
	webhookSecret string,
	logger *slog.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		appFinder:         appFinder,
		deploymentCreator: deploymentCreator,
		webhookSecret:     webhookSecret,
		logger:            logger,
	}
}

func (h *WebhookHandler) Register(app *fiber.App) {
	v1 := app.Group("/paas-deploy/v1")
	webhooks := v1.Group("/webhooks")
	webhooks.Post("/github", h.HandleWebhook)
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

	logger := h.logger.With(
		slog.String("delivery_id", deliveryID),
		slog.String("event", event),
	)

	if event == EventPing {
		logger.Info("received ping event")
		return response.OK(c, map[string]string{"message": "pong"})
	}

	if event != EventPush {
		logger.Info("ignoring unsupported event")
		return response.OK(c, map[string]string{"message": "event ignored"})
	}

	body := c.Body()

	if !ValidateSignature(body, signature, h.webhookSecret) {
		logger.Warn("invalid webhook signature")
		return response.Unauthorized(c, "invalid signature")
	}

	var pushEvent PushEvent
	if err := json.Unmarshal(body, &pushEvent); err != nil {
		logger.Error("failed to parse push event", slog.String("error", err.Error()))
		return response.BadRequest(c, "invalid payload")
	}

	return h.handlePushEvent(c, logger, &pushEvent)
}

func (h *WebhookHandler) handlePushEvent(c *fiber.Ctx, logger *slog.Logger, event *PushEvent) error {
	if event.Repository == nil {
		logger.Warn("push event missing repository data")
		return response.OK(c, map[string]string{"message": "missing repository"})
	}

	branch := extractBranch(event.Ref)
	if branch == "" {
		logger.Info("ignoring non-branch push", slog.String("ref", event.Ref))
		return response.OK(c, map[string]string{"message": "non-branch push ignored"})
	}

	logger = logger.With(
		slog.String("repo", event.Repository.FullName),
		slog.String("branch", branch),
		slog.String("commit", event.After),
	)

	if event.Deleted {
		logger.Info("ignoring branch deletion event")
		return response.OK(c, map[string]string{"message": "branch deletion ignored"})
	}

	repoURLs := getRepoURLVariants(event.Repository)

	var app *domain.App
	var err error

	for _, repoURL := range repoURLs {
		app, err = h.appFinder.FindByRepoURL(repoURL)
		if err == nil && app != nil {
			break
		}
		if !errors.Is(err, domain.ErrNotFound) && err != nil {
			logger.Error("error finding app", slog.String("error", err.Error()))
			return response.InternalError(c)
		}
	}

	if app == nil {
		logger.Info("no app registered for repository")
		return response.OK(c, map[string]string{"message": "repository not registered"})
	}

	if app.Branch != branch {
		logger.Info("push to non-tracked branch",
			slog.String("app_branch", app.Branch),
		)
		return response.OK(c, map[string]string{"message": "branch not tracked"})
	}

	pending, err := h.deploymentCreator.FindPendingByAppID(app.ID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		logger.Error("error checking pending deployments", slog.String("error", err.Error()))
		return response.InternalError(c)
	}
	if pending != nil {
		logger.Info("deployment already pending for app", slog.String("app_id", app.ID))
		return response.OK(c, map[string]string{"message": "deployment already pending"})
	}

	commitMessage := "Push from GitHub"
	if event.HeadCommit != nil && event.HeadCommit.Message != "" {
		commitMessage = truncateString(event.HeadCommit.Message, 200)
	}

	input := domain.CreateDeploymentInput{
		AppID:         app.ID,
		CommitSHA:     event.After,
		CommitMessage: commitMessage,
	}

	deployment, err := h.deploymentCreator.Create(input)
	if err != nil {
		logger.Error("failed to create deployment", slog.String("error", err.Error()))
		return response.InternalError(c)
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
