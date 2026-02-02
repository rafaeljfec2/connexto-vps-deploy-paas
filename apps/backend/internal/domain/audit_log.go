package domain

import (
	"encoding/json"
	"time"
)

type EventType string

const (
	EventAppCreated       EventType = "app.created"
	EventAppUpdated       EventType = "app.updated"
	EventAppDeleted       EventType = "app.deleted"
	EventAppPurged        EventType = "app.purged"
	EventDeployStarted    EventType = "deploy.started"
	EventDeploySuccess    EventType = "deploy.success"
	EventDeployFailed     EventType = "deploy.failed"
	EventEnvCreated       EventType = "env.created"
	EventEnvUpdated       EventType = "env.updated"
	EventEnvDeleted       EventType = "env.deleted"
	EventEnvBulkUpdated   EventType = "env.bulk_updated"
	EventDomainAdded      EventType = "domain.added"
	EventDomainRemoved    EventType = "domain.removed"
	EventContainerStarted EventType = "container.started"
	EventContainerStopped EventType = "container.stopped"
	EventContainerRemoved EventType = "container.removed"
	EventContainerCreated EventType = "container.created"
	EventUserLoggedIn     EventType = "user.logged_in"
	EventUserLoggedOut    EventType = "user.logged_out"
	EventWebhookCreated   EventType = "webhook.created"
	EventWebhookRemoved   EventType = "webhook.removed"
	EventImageRemoved     EventType = "image.removed"
	EventImagesPruned     EventType = "images.pruned"
)

type ResourceType string

const (
	ResourceApp        ResourceType = "app"
	ResourceDeployment ResourceType = "deployment"
	ResourceEnvVar     ResourceType = "env_var"
	ResourceDomain     ResourceType = "domain"
	ResourceContainer  ResourceType = "container"
	ResourceUser       ResourceType = "user"
	ResourceWebhook    ResourceType = "webhook"
	ResourceImage      ResourceType = "image"
)

type AuditLog struct {
	ID           string          `json:"id"`
	EventType    EventType       `json:"eventType"`
	ResourceType ResourceType    `json:"resourceType"`
	ResourceID   *string         `json:"resourceId,omitempty"`
	ResourceName *string         `json:"resourceName,omitempty"`
	UserID       *string         `json:"userId,omitempty"`
	UserName     *string         `json:"userName,omitempty"`
	Details      json.RawMessage `json:"details,omitempty"`
	IPAddress    *string         `json:"ipAddress,omitempty"`
	UserAgent    *string         `json:"userAgent,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
}

type CreateAuditLogInput struct {
	EventType    EventType
	ResourceType ResourceType
	ResourceID   *string
	ResourceName *string
	UserID       *string
	UserName     *string
	Details      map[string]interface{}
	IPAddress    *string
	UserAgent    *string
}

type AuditLogFilter struct {
	EventType    *EventType
	ResourceType *ResourceType
	ResourceID   *string
	UserID       *string
	StartDate    *time.Time
	EndDate      *time.Time
	Limit        int
	Offset       int
}

type AuditLogRepository interface {
	Create(input CreateAuditLogInput) (*AuditLog, error)
	FindByID(id string) (*AuditLog, error)
	FindAll(filter AuditLogFilter) ([]AuditLog, int, error)
	DeleteOlderThan(days int) (int64, error)
}
