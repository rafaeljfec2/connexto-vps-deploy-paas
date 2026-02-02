package handler

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/backend/internal/service"
)

type AuditHandler struct {
	auditService *service.AuditService
}

func NewAuditHandler(auditService *service.AuditService) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

func (h *AuditHandler) Register(router fiber.Router) {
	audit := router.Group("/audit")
	audit.Get("/logs", h.ListLogs)
	audit.Post("/cleanup", h.Cleanup)
}

type AuditLogResponse struct {
	ID           string          `json:"id"`
	EventType    string          `json:"eventType"`
	ResourceType string          `json:"resourceType"`
	ResourceID   *string         `json:"resourceId,omitempty"`
	ResourceName *string         `json:"resourceName,omitempty"`
	UserID       *string         `json:"userId,omitempty"`
	UserName     *string         `json:"userName,omitempty"`
	Details      interface{}     `json:"details,omitempty"`
	IPAddress    *string         `json:"ipAddress,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
}

type AuditLogsResponse struct {
	Logs   []AuditLogResponse `json:"logs"`
	Total  int                `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}

func (h *AuditHandler) ListLogs(c *fiber.Ctx) error {
	filter := buildAuditFilter(c)

	logs, total, err := h.auditService.Query(filter)
	if err != nil {
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to fetch audit logs")
	}

	result := toAuditLogResponses(logs)

	return response.OK(c, AuditLogsResponse{
		Logs:   result,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	})
}

func buildAuditFilter(c *fiber.Ctx) domain.AuditLogFilter {
	filter := domain.AuditLogFilter{
		Limit:  c.QueryInt("limit", 50),
		Offset: c.QueryInt("offset", 0),
	}

	if eventType := c.Query("eventType"); eventType != "" {
		et := domain.EventType(eventType)
		filter.EventType = &et
	}

	if resourceType := c.Query("resourceType"); resourceType != "" {
		rt := domain.ResourceType(resourceType)
		filter.ResourceType = &rt
	}

	if resourceID := c.Query("resourceId"); resourceID != "" {
		filter.ResourceID = &resourceID
	}

	if userID := c.Query("userId"); userID != "" {
		filter.UserID = &userID
	}

	if startDate := parseQueryTime(c, "startDate"); startDate != nil {
		filter.StartDate = startDate
	}

	if endDate := parseQueryTime(c, "endDate"); endDate != nil {
		filter.EndDate = endDate
	}

	return filter
}

func parseQueryTime(c *fiber.Ctx, key string) *time.Time {
	value := c.Query(key)
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return &parsed
}

func toAuditLogResponses(logs []domain.AuditLog) []AuditLogResponse {
	result := make([]AuditLogResponse, len(logs))
	for i, log := range logs {
		result[i] = AuditLogResponse{
			ID:           log.ID,
			EventType:    string(log.EventType),
			ResourceType: string(log.ResourceType),
			ResourceID:   log.ResourceID,
			ResourceName: log.ResourceName,
			UserID:       log.UserID,
			UserName:     log.UserName,
			IPAddress:    log.IPAddress,
			CreatedAt:    log.CreatedAt,
		}

		if len(log.Details) > 0 {
			var details interface{}
			if err := json.Unmarshal(log.Details, &details); err == nil {
				result[i].Details = details
			}
		}
	}
	return result
}

type CleanupRequest struct {
	RetentionDays int `json:"retentionDays"`
}

type CleanupResponse struct {
	DeletedCount int64 `json:"deletedCount"`
}

func (h *AuditHandler) Cleanup(c *fiber.Ctx) error {
	var req CleanupRequest
	if err := c.BodyParser(&req); err != nil {
		req.RetentionDays = 30
	}

	if req.RetentionDays <= 0 {
		req.RetentionDays = 30
	}

	deleted, err := h.auditService.Cleanup(req.RetentionDays)
	if err != nil {
		return response.ServerError(c, fiber.StatusInternalServerError, "Failed to cleanup audit logs")
	}

	return response.OK(c, CleanupResponse{DeletedCount: deleted})
}
