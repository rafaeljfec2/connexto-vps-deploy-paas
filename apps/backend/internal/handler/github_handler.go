package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/ghclient"
	"github.com/paasdeploy/backend/internal/response"
)


type GitHubHandler struct {
	appClient        *ghclient.AppClient
	installationRepo domain.InstallationRepository
	userRepo         domain.UserRepository
	logger           *slog.Logger
	webhookSecret    string
	appInstallURL    string
	setupURL         string
}

type GitHubHandlerConfig struct {
	AppClient        *ghclient.AppClient
	InstallationRepo domain.InstallationRepository
	UserRepo         domain.UserRepository
	Logger           *slog.Logger
	WebhookSecret    string
	AppInstallURL    string
	SetupURL         string
}

func NewGitHubHandler(cfg GitHubHandlerConfig) *GitHubHandler {
	return &GitHubHandler{
		appClient:        cfg.AppClient,
		installationRepo: cfg.InstallationRepo,
		userRepo:         cfg.UserRepo,
		logger:           cfg.Logger,
		webhookSecret:    cfg.WebhookSecret,
		appInstallURL:    cfg.AppInstallURL,
		setupURL:         cfg.SetupURL,
	}
}

func (h *GitHubHandler) Register(app *fiber.App) {
	app.Post("/paas-deploy/v1/webhooks/github/app", h.HandleInstallationWebhook)
}

func (h *GitHubHandler) RegisterProtected(app fiber.Router) {
	gh := app.Group("/api/github")
	gh.Get("/install", h.RedirectToInstall)
	gh.Get("/installations", h.ListInstallations)
	gh.Get("/repos", h.ListRepositories)
	gh.Get("/repos/:owner/:repo", h.GetRepository)
}

func (h *GitHubHandler) requireAuth(c *fiber.Ctx) (*domain.User, error) {
	user := GetUserFromContext(c)
	if user == nil {
		return nil, response.Unauthorized(c, MsgNotAuthenticated)
	}
	return user, nil
}

type installationResult struct {
	installation *domain.Installation
	needInstall  bool
}

func (h *GitHubHandler) findInstallation(c *fiber.Ctx, userID, installationIDParam string) (*installationResult, error) {
	if installationIDParam != "" {
		inst, err := h.installationRepo.FindByID(c.Context(), installationIDParam)
		if err != nil {
			return nil, err
		}
		return &installationResult{installation: inst}, nil
	}

	inst, err := h.installationRepo.FindDefaultInstallation(c.Context(), userID)
	if errors.Is(err, domain.ErrNotFound) {
		installations, listErr := h.installationRepo.FindUserInstallations(c.Context(), userID)
		if listErr != nil {
			return nil, listErr
		}
		if len(installations) == 0 {
			return &installationResult{needInstall: true}, nil
		}
		return &installationResult{installation: &installations[0]}, nil
	}
	if err != nil {
		return nil, err
	}
	return &installationResult{installation: inst}, nil
}

func convertRepo(repo ghclient.AppRepository) RepositoryResponse {
	return RepositoryResponse{
		ID:            repo.ID,
		Name:          repo.Name,
		FullName:      repo.FullName,
		Private:       repo.Private,
		Description:   repo.Description,
		HTMLURL:       repo.HTMLURL,
		CloneURL:      repo.CloneURL,
		DefaultBranch: repo.DefaultBranch,
		Language:      repo.Language,
		Owner: OwnerResponse{
			Login:     repo.Owner.Login,
			AvatarURL: repo.Owner.AvatarURL,
			Type:      repo.Owner.Type,
		},
	}
}

func convertRepos(repos []ghclient.AppRepository) []RepositoryResponse {
	resp := make([]RepositoryResponse, len(repos))
	for i, repo := range repos {
		resp[i] = convertRepo(repo)
	}
	return resp
}

func (h *GitHubHandler) needInstallResponse() ReposResponse {
	return ReposResponse{
		Repositories:   []RepositoryResponse{},
		NeedInstall:    true,
		InstallMessage: MsgInstallGitHubApp,
	}
}

func (h *GitHubHandler) RedirectToInstall(c *fiber.Ctx) error {
	return c.Redirect(h.appInstallURL, fiber.StatusTemporaryRedirect)
}

func (h *GitHubHandler) ListInstallations(c *fiber.Ctx) error {
	user, err := h.requireAuth(c)
	if err != nil {
		return err
	}

	installations, err := h.installationRepo.FindUserInstallations(c.Context(), user.ID)
	if err != nil {
		h.logger.Error("failed to list installations", "error", err, "user_id", user.ID)
		return response.InternalError(c)
	}

	resp := make([]InstallationResponse, len(installations))
	for i, inst := range installations {
		resp[i] = InstallationResponse{
			ID:                  inst.ID,
			InstallationID:      inst.InstallationID,
			AccountType:         inst.AccountType,
			AccountLogin:        inst.AccountLogin,
			RepositorySelection: inst.RepositorySelection,
		}
	}

	return response.OK(c, resp)
}

func (h *GitHubHandler) ListRepositories(c *fiber.Ctx) error {
	user, err := h.requireAuth(c)
	if err != nil {
		return err
	}

	result, err := h.findInstallation(c, user.ID, c.Query("installation_id"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.OK(c, h.needInstallResponse())
		}
		h.logger.Error("failed to find installation", "error", err, "user_id", user.ID)
		return response.InternalError(c)
	}

	if result.needInstall {
		return response.OK(c, h.needInstallResponse())
	}

	if h.appClient == nil {
		return response.InternalError(c)
	}

	repos, err := h.appClient.ListInstallationRepos(c.Context(), result.installation.InstallationID)
	if err != nil {
		h.logger.Error("failed to list repos from GitHub", "error", err, "installation_id", result.installation.InstallationID)
		return response.InternalError(c)
	}

	return response.OK(c, ReposResponse{
		Repositories: convertRepos(repos),
		NeedInstall:  false,
	})
}

func (h *GitHubHandler) GetRepository(c *fiber.Ctx) error {
	user, err := h.requireAuth(c)
	if err != nil {
		return err
	}

	owner := c.Params("owner")
	repo := c.Params("repo")

	if owner == "" || repo == "" {
		return response.BadRequest(c, MsgOwnerRepoRequired)
	}

	result, err := h.findInstallation(c, user.ID, "")
	if err != nil {
		h.logger.Error("failed to find installation", "error", err, "user_id", user.ID)
		return response.InternalError(c)
	}

	if result.needInstall {
		return response.NotFound(c, MsgNoGitHubInstallation)
	}

	if h.appClient == nil {
		return response.InternalError(c)
	}

	repoData, err := h.appClient.GetRepository(c.Context(), result.installation.InstallationID, owner, repo)
	if err != nil {
		h.logger.Error("failed to get repo from GitHub", "error", err, "owner", owner, "repo", repo)
		return response.NotFound(c, "repository not found or not accessible")
	}

	return response.OK(c, convertRepo(*repoData))
}

func (h *GitHubHandler) HandleInstallationWebhook(c *fiber.Ctx) error {
	signature := c.Get("X-Hub-Signature-256")
	if signature == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing signature"})
	}

	body := c.Body()

	if !h.verifySignature(body, signature) {
		h.logger.Warn("invalid webhook signature")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid signature"})
	}

	event := c.Get("X-GitHub-Event")
	h.logger.Info("received GitHub App webhook", "event", event)

	switch event {
	case "installation":
		return h.handleInstallationEvent(c, body)
	case "installation_repositories":
		return h.handleInstallationReposEvent(c, body)
	case "ping":
		return c.JSON(fiber.Map{"message": "pong"})
	default:
		return c.JSON(fiber.Map{"message": "event ignored"})
	}
}

func (h *GitHubHandler) handleInstallationCreated(ctx context.Context, inst InstallationPayloadData, sender SenderPayload) error {
	permissions := make(map[string]string)
	for k, v := range inst.Permissions {
		permissions[k] = v
	}

	installation, err := h.installationRepo.Create(ctx, domain.CreateInstallationInput{
		InstallationID:      inst.ID,
		AccountType:         inst.Account.Type,
		AccountID:           inst.Account.ID,
		AccountLogin:        inst.Account.Login,
		RepositorySelection: inst.RepositorySelection,
		Permissions:         permissions,
	})
	if err != nil {
		h.logger.Error("failed to create installation", "error", err, "installation_id", inst.ID)
		return err
	}
	h.logger.Info("installation created", "installation_id", inst.ID, "account", inst.Account.Login)

	h.linkUserToInstallation(ctx, sender.ID, installation.ID, inst.ID)

	return nil
}

func (h *GitHubHandler) linkUserToInstallation(ctx context.Context, senderGitHubID int64, installationUUID string, installationID int64) {
	if h.userRepo == nil {
		return
	}

	user, err := h.userRepo.FindByGitHubID(ctx, senderGitHubID)
	if err != nil || user == nil {
		return
	}

	if err := h.installationRepo.LinkUserToInstallation(ctx, user.ID, installationUUID, true); err != nil {
		h.logger.Error("failed to link user to installation", "error", err, "user_id", user.ID, "installation_id", installationUUID)
		return
	}

	h.logger.Info("user linked to installation", "user_id", user.ID, "github_login", user.GitHubLogin, "installation_id", installationID)
}

func (h *GitHubHandler) handleInstallationDeleted(ctx context.Context, installationID int64) {
	existing, err := h.installationRepo.FindByInstallationID(ctx, installationID)
	if err != nil {
		h.logger.Warn("installation not found for deletion", "installation_id", installationID, "error", err)
		return
	}

	if err := h.installationRepo.Delete(ctx, existing.ID); err != nil {
		h.logger.Error("failed to delete installation", "error", err, "installation_id", installationID)
		return
	}

	h.logger.Info("installation deleted", "installation_id", installationID)
}

func (h *GitHubHandler) handleInstallationSuspend(ctx context.Context, inst InstallationPayloadData, suspend bool) {
	existing, err := h.installationRepo.FindByInstallationID(ctx, inst.ID)
	if err != nil {
		h.logger.Warn("installation not found for suspend/unsuspend", "installation_id", inst.ID, "error", err)
		return
	}

	var suspendedAt *time.Time
	if suspend {
		suspendedAt = inst.SuspendedAt
	}

	if _, err := h.installationRepo.Update(ctx, existing.ID, domain.UpdateInstallationInput{
		SuspendedAt: suspendedAt,
	}); err != nil {
		action := "suspend"
		if !suspend {
			action = "unsuspend"
		}
		h.logger.Error("failed to "+action+" installation", "error", err, "installation_id", inst.ID)
		return
	}

	if suspend {
		h.logger.Info("installation suspended", "installation_id", inst.ID)
	} else {
		h.logger.Info("installation unsuspended", "installation_id", inst.ID)
	}
}

func (h *GitHubHandler) handleInstallationEvent(c *fiber.Ctx, body []byte) error {
	var payload InstallationEventPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error("failed to parse installation event", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}

	ctx := c.Context()

	switch payload.Action {
	case "created":
		if err := h.handleInstallationCreated(ctx, payload.Installation, payload.Sender); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to store installation"})
		}
	case "deleted":
		h.handleInstallationDeleted(ctx, payload.Installation.ID)
	case "suspend":
		h.handleInstallationSuspend(ctx, payload.Installation, true)
	case "unsuspend":
		h.handleInstallationSuspend(ctx, payload.Installation, false)
	}

	return c.JSON(fiber.Map{"message": "processed"})
}

func (h *GitHubHandler) handleInstallationReposEvent(c *fiber.Ctx, body []byte) error {
	var payload InstallationReposEventPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error("failed to parse installation_repositories event", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}

	h.logger.Info("installation repositories changed",
		"installation_id", payload.Installation.ID,
		"action", payload.Action,
		"added", len(payload.RepositoriesAdded),
		"removed", len(payload.RepositoriesRemoved),
	)

	return c.JSON(fiber.Map{"message": "processed"})
}

func (h *GitHubHandler) verifySignature(payload []byte, signature string) bool {
	if h.webhookSecret == "" {
		return false
	}

	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	sig, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(payload)
	expected := mac.Sum(nil)

	return hmac.Equal(sig, expected)
}

type InstallationResponse struct {
	ID                  string `json:"id"`
	InstallationID      int64  `json:"installationId"`
	AccountType         string `json:"accountType"`
	AccountLogin        string `json:"accountLogin"`
	RepositorySelection string `json:"repositorySelection"`
}

type ReposResponse struct {
	Repositories   []RepositoryResponse `json:"repositories"`
	NeedInstall    bool                 `json:"needInstall"`
	InstallMessage string               `json:"installMessage,omitempty"`
}

type RepositoryResponse struct {
	ID            int64         `json:"id"`
	Name          string        `json:"name"`
	FullName      string        `json:"fullName"`
	Private       bool          `json:"private"`
	Description   string        `json:"description"`
	HTMLURL       string        `json:"htmlUrl"`
	CloneURL      string        `json:"cloneUrl"`
	DefaultBranch string        `json:"defaultBranch"`
	Language      string        `json:"language"`
	Owner         OwnerResponse `json:"owner"`
}

type OwnerResponse struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatarUrl"`
	Type      string `json:"type"`
}

type InstallationEventPayload struct {
	Action       string                  `json:"action"`
	Installation InstallationPayloadData `json:"installation"`
	Sender       SenderPayload           `json:"sender"`
}

type InstallationPayloadData struct {
	ID                  int64             `json:"id"`
	Account             AccountPayload    `json:"account"`
	RepositorySelection string            `json:"repository_selection"`
	Permissions         map[string]string `json:"permissions"`
	Events              []string          `json:"events"`
	SuspendedAt         *time.Time        `json:"suspended_at"`
	SuspendedBy         *AccountPayload   `json:"suspended_by"`
}

type AccountPayload struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Type      string `json:"type"`
	AvatarURL string `json:"avatar_url"`
}

type SenderPayload struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Type  string `json:"type"`
}

type InstallationReposEventPayload struct {
	Action              string                  `json:"action"`
	Installation        InstallationPayloadData `json:"installation"`
	RepositoriesAdded   []RepoPayload           `json:"repositories_added"`
	RepositoriesRemoved []RepoPayload           `json:"repositories_removed"`
}

type RepoPayload struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Private  bool   `json:"private"`
}
