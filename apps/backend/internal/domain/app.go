package domain

import (
	"encoding/json"
	"time"
)

type AppStatus string

const (
	AppStatusActive   AppStatus = "active"
	AppStatusInactive AppStatus = "inactive"
	AppStatusDeleted  AppStatus = "deleted"
)

type App struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	RepositoryURL  string          `json:"repositoryUrl"`
	Branch         string          `json:"branch"`
	Workdir        string          `json:"workdir"`
	Runtime        *string         `json:"runtime,omitempty"`
	Config         json.RawMessage `json:"config"`
	Status         AppStatus       `json:"status"`
	WebhookID      *int64          `json:"webhookId,omitempty"`
	LastDeployedAt *time.Time      `json:"lastDeployedAt,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

type CreateAppInput struct {
	Name          string          `json:"name"`
	RepositoryURL string          `json:"repositoryUrl"`
	Branch        string          `json:"branch"`
	Workdir       string          `json:"workdir"`
	Config        json.RawMessage `json:"config,omitempty"`
}

type UpdateAppInput struct {
	Name          *string          `json:"name,omitempty"`
	RepositoryURL *string          `json:"repositoryUrl,omitempty"`
	Branch        *string          `json:"branch,omitempty"`
	Workdir       *string          `json:"workdir,omitempty"`
	Runtime       *string          `json:"runtime,omitempty"`
	Config        *json.RawMessage `json:"config,omitempty"`
	Status        *AppStatus       `json:"status,omitempty"`
	WebhookID     *int64           `json:"webhookId,omitempty"`
}

type AppRepository interface {
	FindAll() ([]App, error)
	FindByID(id string) (*App, error)
	FindByName(name string) (*App, error)
	FindByRepoURL(repoURL string) (*App, error)
	Create(input CreateAppInput) (*App, error)
	Update(id string, input UpdateAppInput) (*App, error)
	Delete(id string) error
	HardDelete(id string) error
	UpdateLastDeployedAt(id string, deployedAt time.Time) error
}
