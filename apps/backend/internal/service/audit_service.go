package service

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/domain"
)

type AuditService struct {
	repo   domain.AuditLogRepository
	logger *slog.Logger
}

func NewAuditService(repo domain.AuditLogRepository, logger *slog.Logger) *AuditService {
	return &AuditService{
		repo:   repo,
		logger: logger,
	}
}

type AuditContext struct {
	UserID    *string
	UserName  *string
	IPAddress *string
	UserAgent *string
}

func (s *AuditService) ExtractContext(c *fiber.Ctx) AuditContext {
	ctx := AuditContext{}

	if userID := c.Locals("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			ctx.UserID = &id
		}
	}

	if userName := c.Locals("user_name"); userName != nil {
		if name, ok := userName.(string); ok {
			ctx.UserName = &name
		}
	}

	ip := c.IP()
	ctx.IPAddress = &ip

	ua := c.Get("User-Agent")
	if ua != "" {
		ctx.UserAgent = &ua
	}

	return ctx
}

func (s *AuditService) Log(ctx context.Context, auditCtx AuditContext, eventType domain.EventType, resourceType domain.ResourceType, resourceID, resourceName *string, details map[string]interface{}) {
	input := domain.CreateAuditLogInput{
		EventType:    eventType,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		UserID:       auditCtx.UserID,
		UserName:     auditCtx.UserName,
		Details:      details,
		IPAddress:    auditCtx.IPAddress,
		UserAgent:    auditCtx.UserAgent,
	}

	_, err := s.repo.Create(input)
	if err != nil {
		s.logger.Error("failed to create audit log",
			"event_type", eventType,
			"resource_type", resourceType,
			"resource_id", resourceID,
			"error", err,
		)
	}
}

func (s *AuditService) LogAppCreated(ctx context.Context, auditCtx AuditContext, appID, appName, repoURL string) {
	s.Log(ctx, auditCtx, domain.EventAppCreated, domain.ResourceApp, &appID, &appName, map[string]interface{}{
		"repository_url": repoURL,
	})
}

func (s *AuditService) LogAppDeleted(ctx context.Context, auditCtx AuditContext, appID, appName string) {
	s.Log(ctx, auditCtx, domain.EventAppDeleted, domain.ResourceApp, &appID, &appName, nil)
}

func (s *AuditService) LogAppPurged(ctx context.Context, auditCtx AuditContext, appID, appName string) {
	s.Log(ctx, auditCtx, domain.EventAppPurged, domain.ResourceApp, &appID, &appName, nil)
}

func (s *AuditService) LogDeployStarted(ctx context.Context, auditCtx AuditContext, deployID, appID, appName, commitSHA string) {
	s.Log(ctx, auditCtx, domain.EventDeployStarted, domain.ResourceDeployment, &deployID, &appName, map[string]interface{}{
		"app_id":     appID,
		"commit_sha": commitSHA,
	})
}

func (s *AuditService) LogDeploySuccess(ctx context.Context, auditCtx AuditContext, deployID, appID, appName string) {
	s.Log(ctx, auditCtx, domain.EventDeploySuccess, domain.ResourceDeployment, &deployID, &appName, map[string]interface{}{
		"app_id": appID,
	})
}

func (s *AuditService) LogDeployFailed(ctx context.Context, auditCtx AuditContext, deployID, appID, appName, errorMsg string) {
	s.Log(ctx, auditCtx, domain.EventDeployFailed, domain.ResourceDeployment, &deployID, &appName, map[string]interface{}{
		"app_id": appID,
		"error":  errorMsg,
	})
}

func (s *AuditService) LogEnvCreated(ctx context.Context, auditCtx AuditContext, envID, appID, key string, isSecret bool) {
	s.Log(ctx, auditCtx, domain.EventEnvCreated, domain.ResourceEnvVar, &envID, &key, map[string]interface{}{
		"app_id":    appID,
		"is_secret": isSecret,
	})
}

func (s *AuditService) LogEnvUpdated(ctx context.Context, auditCtx AuditContext, envID, appID, key string) {
	s.Log(ctx, auditCtx, domain.EventEnvUpdated, domain.ResourceEnvVar, &envID, &key, map[string]interface{}{
		"app_id": appID,
	})
}

func (s *AuditService) LogEnvDeleted(ctx context.Context, auditCtx AuditContext, envID, appID, key string) {
	s.Log(ctx, auditCtx, domain.EventEnvDeleted, domain.ResourceEnvVar, &envID, &key, map[string]interface{}{
		"app_id": appID,
	})
}

func (s *AuditService) LogDomainAdded(ctx context.Context, auditCtx AuditContext, domainID, appID, domainName, pathPrefix string) {
	s.Log(ctx, auditCtx, domain.EventDomainAdded, domain.ResourceDomain, &domainID, &domainName, map[string]interface{}{
		"app_id":      appID,
		"path_prefix": pathPrefix,
	})
}

func (s *AuditService) LogDomainRemoved(ctx context.Context, auditCtx AuditContext, domainID, appID, domainName string) {
	s.Log(ctx, auditCtx, domain.EventDomainRemoved, domain.ResourceDomain, &domainID, &domainName, map[string]interface{}{
		"app_id": appID,
	})
}

func (s *AuditService) LogContainerStarted(ctx context.Context, auditCtx AuditContext, containerID, containerName string) {
	s.Log(ctx, auditCtx, domain.EventContainerStarted, domain.ResourceContainer, &containerID, &containerName, nil)
}

func (s *AuditService) LogContainerStopped(ctx context.Context, auditCtx AuditContext, containerID, containerName string) {
	s.Log(ctx, auditCtx, domain.EventContainerStopped, domain.ResourceContainer, &containerID, &containerName, nil)
}

func (s *AuditService) LogContainerRemoved(ctx context.Context, auditCtx AuditContext, containerID, containerName string) {
	s.Log(ctx, auditCtx, domain.EventContainerRemoved, domain.ResourceContainer, &containerID, &containerName, nil)
}

func (s *AuditService) LogContainerCreated(ctx context.Context, auditCtx AuditContext, containerID, containerName, image string) {
	s.Log(ctx, auditCtx, domain.EventContainerCreated, domain.ResourceContainer, &containerID, &containerName, map[string]interface{}{
		"image": image,
	})
}

func (s *AuditService) LogUserLoggedIn(ctx context.Context, auditCtx AuditContext, userID, userName string) {
	s.Log(ctx, auditCtx, domain.EventUserLoggedIn, domain.ResourceUser, &userID, &userName, nil)
}

func (s *AuditService) LogUserLoggedOut(ctx context.Context, auditCtx AuditContext, userID, userName string) {
	s.Log(ctx, auditCtx, domain.EventUserLoggedOut, domain.ResourceUser, &userID, &userName, nil)
}

func (s *AuditService) LogImageRemoved(ctx context.Context, auditCtx AuditContext, imageID string) {
	s.Log(ctx, auditCtx, domain.EventImageRemoved, domain.ResourceImage, &imageID, nil, nil)
}

func (s *AuditService) LogImagesPruned(ctx context.Context, auditCtx AuditContext, count int, spaceReclaimed int64) {
	s.Log(ctx, auditCtx, domain.EventImagesPruned, domain.ResourceImage, nil, nil, map[string]interface{}{
		"images_deleted":  count,
		"space_reclaimed": spaceReclaimed,
	})
}

func (s *AuditService) Query(filter domain.AuditLogFilter) ([]domain.AuditLog, int, error) {
	return s.repo.FindAll(filter)
}

func (s *AuditService) Cleanup(retentionDays int) (int64, error) {
	return s.repo.DeleteOlderThan(retentionDays)
}
