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
	FindAllByRepoURL(repoURL string) ([]domain.App, error)
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

	h.logger.Info("webhook POST received",
		slog.String("event", event),
		slog.String("delivery_id", deliveryID),
		slog.String("remote_ip", c.IP()),
	)

	logger := h.logger.With(
		slog.String("delivery_id", deliveryID),
		slog.String("event", event),
	)

	if event == EventPing {
		logger.Info("received ping event")
		h.savePayloadAsync(deliveryID, event, body, "pong", nil)
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

func (h *WebhookHandler) savePayloadAsync(deliveryID, eventType string, payload []byte, outcome string, errMsg *string) {
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

	apps, err := h.findAppsByRepository(event.Repository, logger)
	if err != nil {
		errStr := err.Error()
		h.savePayload(c.Context(), deliveryID, eventType, body, "error", &errStr)
		return response.InternalError(c)
	}

	if len(apps) == 0 {
		logger.Info("no app registered for repository")
		h.savePayload(c.Context(), deliveryID, eventType, body, "ignored", strPtr("repository not registered"))
		return response.OK(c, map[string]string{"message": "repository not registered"})
	}

	changedFiles := extractChangedFiles(event.Commits)
	hasChangedFiles := len(changedFiles) > 0

	branchApps := filterAppsByBranch(apps, branch)
	if len(branchApps) == 0 {
		logger.Info("push to non-tracked branch for all apps", slog.String("branch", branch))
		h.savePayload(c.Context(), deliveryID, eventType, body, "ignored", strPtr("branch not tracked"))
		return response.OK(c, map[string]string{"message": "branch not tracked"})
	}

	otherWorkdirs := collectNonRootWorkdirs(branchApps)

	var deployments []fiber.Map
	for i := range branchApps {
		app := &branchApps[i]
		appLogger := logger.With(slog.String("app_id", app.ID), slog.String("app_name", app.Name))

		if hasChangedFiles && len(branchApps) > 1 && !shouldDeployApp(app, changedFiles, otherWorkdirs) {
			appLogger.Info("skipping deploy: no changed files match workdir", slog.String("workdir", app.Workdir))
			continue
		}

		result := h.tryCreateDeployment(appLogger, app, event)
		if result != nil {
			deployments = append(deployments, result)
		}
	}

	if len(deployments) == 0 {
		h.savePayload(c.Context(), deliveryID, eventType, body, "ignored", strPtr("no apps affected by changed files"))
		return response.OK(c, map[string]string{"message": "no apps affected by changed files"})
	}

	h.savePayload(c.Context(), deliveryID, eventType, body, "deployment_queued", nil)
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"success": true,
		"data":    deployments,
		"meta":    fiber.Map{"traceId": c.Locals("traceId")},
	})
}

func (h *WebhookHandler) findAppsByRepository(repo *Repository, logger *slog.Logger) ([]domain.App, error) {
	repoURLs := getRepoURLVariants(repo)

	for _, repoURL := range repoURLs {
		apps, err := h.appFinder.FindAllByRepoURL(repoURL)
		if err != nil {
			logger.Error("error finding apps", slog.String("error", err.Error()))
			return nil, err
		}
		if len(apps) > 0 {
			return apps, nil
		}
	}

	return nil, nil
}

func (h *WebhookHandler) tryCreateDeployment(logger *slog.Logger, app *domain.App, event *PushEvent) fiber.Map {
	commitMessage := getCommitMessage(event)

	if commitMessageSkipsDeploy(commitMessage) {
		logger.Info("skipping deployment: commit message contains [skip ci]")
		return nil
	}

	hasPending, err := h.hasPendingDeployment(app.ID, logger)
	if err != nil {
		logger.Error("error checking pending deployments", slog.String("error", err.Error()))
		return nil
	}
	if hasPending {
		logger.Info("deployment already pending for app")
		return nil
	}

	input := domain.CreateDeploymentInput{
		AppID:         app.ID,
		CommitSHA:     event.After,
		CommitMessage: commitMessage,
	}

	deployment, err := h.deploymentCreator.Create(input)
	if err != nil {
		logger.Error("failed to create deployment", slog.String("error", err.Error()))
		return nil
	}

	if h.deployAudit != nil {
		h.deployAudit.LogDeployStarted(context.Background(), deployment.ID, app.ID, app.Name, event.After)
	}

	logger.Info("deployment queued", slog.String("deployment_id", deployment.ID))

	return fiber.Map{
		"deploymentId": deployment.ID,
		"appId":        app.ID,
		"appName":      app.Name,
		"commitSha":    event.After,
	}
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

func commitMessageSkipsDeploy(msg string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(msg)), "[skip ci]")
}

func extractChangedFiles(commits []Commit) []string {
	seen := make(map[string]struct{})
	var files []string
	for _, c := range commits {
		files = appendUnique(files, seen, c.Added)
		files = appendUnique(files, seen, c.Modified)
		files = appendUnique(files, seen, c.Removed)
	}
	return files
}

func appendUnique(files []string, seen map[string]struct{}, items []string) []string {
	for _, f := range items {
		if _, ok := seen[f]; !ok {
			seen[f] = struct{}{}
			files = append(files, f)
		}
	}
	return files
}

func filterAppsByBranch(apps []domain.App, branch string) []domain.App {
	var result []domain.App
	for _, app := range apps {
		if app.Branch == branch {
			result = append(result, app)
		}
	}
	return result
}

func collectNonRootWorkdirs(apps []domain.App) []string {
	var workdirs []string
	for _, app := range apps {
		wd := normalizeWorkdir(app.Workdir)
		if wd != "" {
			workdirs = append(workdirs, wd)
		}
	}
	return workdirs
}

func normalizeWorkdir(workdir string) string {
	wd := strings.TrimPrefix(workdir, "./")
	wd = strings.Trim(wd, "/")
	if wd == "." || wd == "" {
		return ""
	}
	return wd
}

func shouldDeployApp(app *domain.App, changedFiles []string, otherWorkdirs []string) bool {
	appWorkdir := normalizeWorkdir(app.Workdir)

	if appWorkdir == "" {
		return hasFilesOutsideWorkdirs(changedFiles, otherWorkdirs)
	}

	prefix := appWorkdir + "/"
	for _, f := range changedFiles {
		if strings.HasPrefix(f, prefix) || f == appWorkdir {
			return true
		}
	}
	return false
}

func hasFilesOutsideWorkdirs(changedFiles []string, workdirs []string) bool {
	for _, f := range changedFiles {
		belongsToOther := false
		for _, wd := range workdirs {
			if strings.HasPrefix(f, wd+"/") || f == wd {
				belongsToOther = true
				break
			}
		}
		if !belongsToOther {
			return true
		}
	}
	return false
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
