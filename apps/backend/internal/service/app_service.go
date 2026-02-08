package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/ghclient"
	"github.com/paasdeploy/backend/internal/webhook"
)

type AppCleaner interface {
	CleanApp(ctx context.Context, appID, appName string) error
}

type AppService struct {
	appRepo        domain.AppRepository
	deploymentRepo domain.DeploymentRepository
	envVarRepo     domain.EnvVarRepository
	webhookManager webhook.Manager
	appCleaner     AppCleaner
	logger         *slog.Logger
}

func NewAppService(
	appRepo domain.AppRepository,
	deploymentRepo domain.DeploymentRepository,
	envVarRepo domain.EnvVarRepository,
	webhookManager webhook.Manager,
	appCleaner AppCleaner,
	logger *slog.Logger,
) *AppService {
	return &AppService{
		appRepo:        appRepo,
		deploymentRepo: deploymentRepo,
		envVarRepo:     envVarRepo,
		webhookManager: webhookManager,
		appCleaner:     appCleaner,
		logger:         logger,
	}
}

func (s *AppService) ListApps() ([]domain.App, error) {
	apps, err := s.appRepo.FindAll()
	if err != nil {
		return nil, err
	}
	if apps == nil {
		apps = []domain.App{}
	}
	return apps, nil
}

func (s *AppService) ListAppsWithDeployments() ([]domain.AppWithDeployment, error) {
	apps, err := s.appRepo.FindAll()
	if err != nil {
		return nil, err
	}
	if len(apps) == 0 {
		return []domain.AppWithDeployment{}, nil
	}

	appIDs := make([]string, len(apps))
	for i, app := range apps {
		appIDs[i] = app.ID
	}

	deployments, err := s.deploymentRepo.FindMostRecentByAppIDs(appIDs)
	if err != nil {
		s.logger.Warn("failed to fetch deployments for apps", "error", err)
		deployments = make(map[string]*domain.Deployment)
	}

	result := make([]domain.AppWithDeployment, len(apps))
	for i, app := range apps {
		result[i] = domain.AppWithDeployment{App: app}
		if deploy := deployments[app.ID]; deploy != nil {
			result[i].LastDeployment = s.toDeploymentSummary(deploy)
		}
	}

	return result, nil
}

func (s *AppService) ListAppsByServerID(serverID string) ([]domain.AppWithDeployment, error) {
	apps, err := s.appRepo.FindByServerID(serverID)
	if err != nil {
		return nil, err
	}
	if len(apps) == 0 {
		return []domain.AppWithDeployment{}, nil
	}

	appIDs := make([]string, len(apps))
	for i, app := range apps {
		appIDs[i] = app.ID
	}

	deployments, err := s.deploymentRepo.FindMostRecentByAppIDs(appIDs)
	if err != nil {
		s.logger.Warn("failed to fetch deployments for server apps", "error", err)
		deployments = make(map[string]*domain.Deployment)
	}

	result := make([]domain.AppWithDeployment, len(apps))
	for i, app := range apps {
		result[i] = domain.AppWithDeployment{App: app}
		if deploy := deployments[app.ID]; deploy != nil {
			result[i].LastDeployment = s.toDeploymentSummary(deploy)
		}
	}

	return result, nil
}

func (s *AppService) toDeploymentSummary(d *domain.Deployment) *domain.DeploymentSummary {
	summary := &domain.DeploymentSummary{
		ID:            d.ID,
		Status:        d.Status,
		CommitSHA:     d.CommitSHA,
		CommitMessage: d.CommitMessage,
		StartedAt:     d.StartedAt,
		FinishedAt:    d.FinishedAt,
		Logs:          d.Logs,
	}

	if d.StartedAt != nil && d.FinishedAt != nil {
		duration := d.FinishedAt.Sub(*d.StartedAt).Milliseconds()
		summary.DurationMs = &duration
	}

	return summary
}

func (s *AppService) GetApp(id string) (*domain.App, error) {
	return s.appRepo.FindByID(id)
}

func (s *AppService) CreateApp(ctx context.Context, input domain.CreateAppInput) (*domain.App, error) {
	if err := s.validateCreateInput(input); err != nil {
		return nil, err
	}

	existing, err := s.appRepo.FindByName(input.Name)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrAlreadyExists
	}

	app, err := s.appRepo.Create(input)
	if err != nil {
		return nil, err
	}

	if s.webhookManager != nil {
		go s.setupWebhookAsync(ctx, app)
	}

	return app, nil
}

func (s *AppService) setupWebhookAsync(ctx context.Context, app *domain.App) {
	result, err := s.webhookManager.Setup(ctx, webhook.SetupInput{
		RepositoryURL: app.RepositoryURL,
	})
	if err != nil {
		s.logger.Error("failed to setup webhook",
			"app_id", app.ID,
			"app_name", app.Name,
			"error", err,
		)
		return
	}

	if result == nil {
		return
	}

	_, err = s.appRepo.Update(app.ID, domain.UpdateAppInput{
		WebhookID: &result.WebhookID,
	})
	if err != nil {
		s.logger.Error("failed to update app with webhook_id",
			"app_id", app.ID,
			"webhook_id", result.WebhookID,
			"error", err,
		)
	}

	s.logger.Info("webhook setup completed",
		"app_id", app.ID,
		"app_name", app.Name,
		"webhook_id", result.WebhookID,
	)
}

func (s *AppService) UpdateApp(id string, input domain.UpdateAppInput) (*domain.App, error) {
	return s.appRepo.Update(id, input)
}

func (s *AppService) DeleteApp(ctx context.Context, id string) error {
	app, err := s.appRepo.FindByID(id)
	if err != nil {
		return err
	}

	if s.webhookManager != nil && app.WebhookID != nil {
		go s.removeWebhookAsync(ctx, app)
	}

	if s.appCleaner != nil {
		go s.cleanAppAsync(ctx, app)
	}

	return s.appRepo.Delete(id)
}

func (s *AppService) cleanAppAsync(ctx context.Context, app *domain.App) {
	if err := s.appCleaner.CleanApp(ctx, app.ID, app.Name); err != nil {
		s.logger.Warn("failed to clean app resources",
			"app_id", app.ID,
			"app_name", app.Name,
			"error", err,
		)
	}
}

func (s *AppService) removeWebhookAsync(ctx context.Context, app *domain.App) {
	if app.WebhookID == nil {
		return
	}

	err := s.webhookManager.Remove(ctx, webhook.RemoveInput{
		RepositoryURL: app.RepositoryURL,
		WebhookID:     *app.WebhookID,
	})
	if err != nil {
		s.logger.Error("failed to remove webhook",
			"app_id", app.ID,
			"app_name", app.Name,
			"webhook_id", *app.WebhookID,
			"error", err,
		)
		return
	}

	s.logger.Info("webhook removed",
		"app_id", app.ID,
		"app_name", app.Name,
		"webhook_id", *app.WebhookID,
	)
}

func (s *AppService) PurgeApp(ctx context.Context, id string) error {
	app, err := s.appRepo.FindByID(id)
	if err != nil {
		return err
	}

	s.logger.Info("starting app purge",
		"app_id", app.ID,
		"app_name", app.Name,
	)

	if s.webhookManager != nil && app.WebhookID != nil {
		err := s.webhookManager.Remove(ctx, webhook.RemoveInput{
			RepositoryURL: app.RepositoryURL,
			WebhookID:     *app.WebhookID,
		})
		if err != nil {
			s.logger.Warn("failed to remove webhook during purge",
				"app_id", app.ID,
				"error", err,
			)
		}
	}

	if s.appCleaner != nil {
		if err := s.appCleaner.CleanApp(ctx, app.ID, app.Name); err != nil {
			s.logger.Warn("failed to clean app resources",
				"app_id", app.ID,
				"error", err,
			)
		}
	}

	if s.envVarRepo != nil {
		if err := s.envVarRepo.DeleteByAppID(app.ID); err != nil {
			s.logger.Warn("failed to delete env vars",
				"app_id", app.ID,
				"error", err,
			)
		}
	}

	if err := s.deploymentRepo.DeleteByAppID(app.ID); err != nil {
		s.logger.Warn("failed to delete deployments",
			"app_id", app.ID,
			"error", err,
		)
	}

	if err := s.appRepo.HardDelete(app.ID); err != nil {
		return err
	}

	s.logger.Info("app purge completed",
		"app_id", app.ID,
		"app_name", app.Name,
	)

	return nil
}

func (s *AppService) ListDeployments(appID string) ([]domain.Deployment, error) {
	_, err := s.appRepo.FindByID(appID)
	if err != nil {
		return nil, err
	}

	deployments, err := s.deploymentRepo.FindByAppID(appID, 50)
	if err != nil {
		return nil, err
	}
	if deployments == nil {
		deployments = []domain.Deployment{}
	}
	return deployments, nil
}

func (s *AppService) TriggerDeploy(appID string, commitSHA string) (*domain.Deployment, error) {
	app, err := s.appRepo.FindByID(appID)
	if err != nil {
		return nil, err
	}

	pending, err := s.deploymentRepo.FindPendingByAppID(appID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if pending != nil {
		return nil, domain.ErrDeployInProgress
	}

	if commitSHA == "" {
		commitSHA = "HEAD"
	}

	input := domain.CreateDeploymentInput{
		AppID:         app.ID,
		CommitSHA:     commitSHA,
		CommitMessage: "Manual deploy triggered",
	}

	return s.deploymentRepo.Create(input)
}

func (s *AppService) TriggerRollback(appID string) (*domain.Deployment, error) {
	_, err := s.appRepo.FindByID(appID)
	if err != nil {
		return nil, err
	}

	latestSuccess, err := s.deploymentRepo.FindLatestByAppID(appID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNoDeployAvailable
		}
		return nil, err
	}

	input := domain.CreateDeploymentInput{
		AppID:         appID,
		CommitSHA:     latestSuccess.CommitSHA,
		CommitMessage: "Rollback to " + latestSuccess.CommitSHA[:7],
	}

	return s.deploymentRepo.Create(input)
}

func (s *AppService) validateCreateInput(input domain.CreateAppInput) error {
	if input.Name == "" {
		return domain.ErrInvalidInput
	}

	if len(input.Name) < 2 || len(input.Name) > 63 {
		return domain.ErrInvalidInput
	}

	if input.RepositoryURL == "" {
		return domain.ErrInvalidInput
	}

	if !strings.HasPrefix(input.RepositoryURL, "https://github.com/") &&
		!strings.HasPrefix(input.RepositoryURL, "git@github.com:") {
		return domain.ErrInvalidInput
	}

	return nil
}

func (s *AppService) SetupWebhook(ctx context.Context, appID string) (*webhook.SetupResult, error) {
	if s.webhookManager == nil {
		return nil, domain.ErrWebhookNotConfigured
	}

	app, err := s.appRepo.FindByID(appID)
	if err != nil {
		return nil, err
	}

	result, err := s.webhookManager.Setup(ctx, webhook.SetupInput{
		RepositoryURL: app.RepositoryURL,
	})
	if err != nil {
		return nil, mapWebhookSetupError(err)
	}

	if result != nil {
		_, err = s.appRepo.Update(app.ID, domain.UpdateAppInput{
			WebhookID: &result.WebhookID,
		})
		if err != nil {
			s.logger.Error("failed to update app with webhook_id",
				"app_id", app.ID,
				"webhook_id", result.WebhookID,
				"error", err,
			)
		}
	}

	return result, nil
}

func (s *AppService) RemoveWebhook(ctx context.Context, appID string) error {
	if s.webhookManager == nil {
		return domain.ErrWebhookNotConfigured
	}

	app, err := s.appRepo.FindByID(appID)
	if err != nil {
		return err
	}

	if app.WebhookID == nil {
		return nil
	}

	err = s.webhookManager.Remove(ctx, webhook.RemoveInput{
		RepositoryURL: app.RepositoryURL,
		WebhookID:     *app.WebhookID,
	})
	if err != nil {
		return err
	}

	var nilWebhookID *int64
	_, err = s.appRepo.Update(app.ID, domain.UpdateAppInput{
		WebhookID: nilWebhookID,
	})
	return err
}

func (s *AppService) GetWebhookStatus(ctx context.Context, appID string) (*webhook.Status, error) {
	if s.webhookManager == nil {
		return &webhook.Status{
			Exists: false,
			Error:  "webhook management not configured",
		}, nil
	}

	app, err := s.appRepo.FindByID(appID)
	if err != nil {
		return nil, err
	}

	if app.WebhookID == nil {
		st := &webhook.Status{Exists: false}
		st.ConfiguredURL = s.webhookManager.WebhookURL()
		return st, nil
	}

	st, err := s.webhookManager.Status(ctx, app.RepositoryURL, *app.WebhookID)
	if err != nil {
		return nil, err
	}
	st.ConfiguredURL = s.webhookManager.WebhookURL()
	return st, nil
}

func mapWebhookSetupError(err error) error {
	if err == nil {
		return nil
	}
	errStr := err.Error()
	if strings.Contains(errStr, "404") || strings.Contains(strings.ToLower(errStr), "not found") {
		return fmt.Errorf("%w: repository not found or not accessible", domain.ErrNotFound)
	}
	if strings.Contains(errStr, "403") || strings.Contains(strings.ToLower(errStr), "forbidden") {
		return fmt.Errorf("%w: insufficient permissions (ensure token has admin:repo_hook scope)", domain.ErrForbidden)
	}
	if strings.Contains(errStr, "422") || strings.Contains(strings.ToLower(errStr), "validation failed") {
		ghDetail := extractGitHubValidationError(errStr)
		base := "webhook validation failed - GIT_HUB_WEBHOOK_URL must be publicly reachable. Verify: curl -I <url> returns 200. Check firewall, SSL, Traefik routing"
		if ghDetail != "" {
			return fmt.Errorf("%w: %s. GitHub: %s", domain.ErrInvalidInput, base, ghDetail)
		}
		return fmt.Errorf("%w: %s", domain.ErrInvalidInput, base)
	}
	return err
}

func extractGitHubValidationError(errStr string) string {
	idx := strings.Index(errStr, "{")
	if idx < 0 {
		return ""
	}
	jsonPart := errStr[idx:]
	var resp struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal([]byte(jsonPart), &resp); err != nil || len(resp.Errors) == 0 {
		return ""
	}
	return resp.Errors[0].Message
}

func (s *AppService) ListCommits(ctx context.Context, appID string, limit int) ([]ghclient.CommitInfo, error) {
	if s.webhookManager == nil {
		return nil, domain.ErrWebhookNotConfigured
	}

	app, err := s.appRepo.FindByID(appID)
	if err != nil {
		return nil, err
	}

	commits, err := s.webhookManager.ListCommits(ctx, app.RepositoryURL, app.Branch, limit)
	if err != nil {
		return nil, err
	}

	if commits == nil {
		commits = []ghclient.CommitInfo{}
	}

	return commits, nil
}
