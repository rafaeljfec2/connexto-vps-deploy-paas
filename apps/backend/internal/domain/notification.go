package domain

import (
	"encoding/json"
	"time"
)

type NotificationChannelType string

const (
	NotificationChannelSlack   NotificationChannelType = "slack"
	NotificationChannelDiscord NotificationChannelType = "discord"
	NotificationChannelEmail   NotificationChannelType = "email"
)

type NotificationChannel struct {
	ID        string                   `json:"id"`
	Type      NotificationChannelType  `json:"type"`
	Name      string                  `json:"name"`
	Config    json.RawMessage         `json:"config"`
	AppID     *string                 `json:"appId,omitempty"`
	CreatedAt time.Time               `json:"createdAt"`
	UpdatedAt time.Time               `json:"updatedAt"`
}

type CreateNotificationChannelInput struct {
	Type   NotificationChannelType `json:"type"`
	Name   string                  `json:"name"`
	Config json.RawMessage         `json:"config"`
	AppID  *string                 `json:"appId,omitempty"`
}

type UpdateNotificationChannelInput struct {
	Name   *string          `json:"name,omitempty"`
	Config *json.RawMessage `json:"config,omitempty"`
	AppID  *string          `json:"appId,omitempty"`
}

type NotificationRule struct {
	ID        string    `json:"id"`
	EventType string    `json:"eventType"`
	ChannelID string    `json:"channelId"`
	AppID     *string   `json:"appId,omitempty"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateNotificationRuleInput struct {
	EventType string  `json:"eventType"`
	ChannelID string  `json:"channelId"`
	AppID     *string `json:"appId,omitempty"`
	Enabled   bool    `json:"enabled"`
}

type UpdateNotificationRuleInput struct {
	EventType *string `json:"eventType,omitempty"`
	AppID     *string `json:"appId,omitempty"`
	Enabled   *bool   `json:"enabled,omitempty"`
}

const (
	EventTypeDeployRunning   = "deploy_running"
	EventTypeDeploySuccess   = "deploy_success"
	EventTypeDeployFailed    = "deploy_failed"
	EventTypeContainerDown   = "container_down"
	EventTypeHealthUnhealthy = "health_unhealthy"
)

type NotificationChannelRepository interface {
	FindAll() ([]NotificationChannel, error)
	FindByID(id string) (*NotificationChannel, error)
	FindByAppID(appID string) ([]NotificationChannel, error)
	FindGlobal() ([]NotificationChannel, error)
	Create(input CreateNotificationChannelInput) (*NotificationChannel, error)
	Update(id string, input UpdateNotificationChannelInput) (*NotificationChannel, error)
	Delete(id string) error
}

type NotificationRuleRepository interface {
	FindAll() ([]NotificationRule, error)
	FindByID(id string) (*NotificationRule, error)
	FindByChannelID(channelID string) ([]NotificationRule, error)
	FindByEventType(eventType string) ([]NotificationRule, error)
	FindActiveByEventType(eventType string, appID *string) ([]NotificationRule, error)
	Create(input CreateNotificationRuleInput) (*NotificationRule, error)
	Update(id string, input UpdateNotificationRuleInput) (*NotificationRule, error)
	Delete(id string) error
}
