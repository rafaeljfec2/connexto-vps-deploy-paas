package domain

import (
	"time"
)

type DeployStatus string

const (
	DeployStatusPending   DeployStatus = "pending"
	DeployStatusRunning   DeployStatus = "running"
	DeployStatusSuccess   DeployStatus = "success"
	DeployStatusFailed    DeployStatus = "failed"
	DeployStatusCancelled DeployStatus = "cancelled"
)

type Deployment struct {
	ID               string       `json:"id"`
	AppID            string       `json:"appId"`
	CommitSHA        string       `json:"commitSha"`
	CommitMessage    string       `json:"commitMessage,omitempty"`
	Status           DeployStatus `json:"status"`
	StartedAt        *time.Time   `json:"startedAt,omitempty"`
	FinishedAt       *time.Time   `json:"finishedAt,omitempty"`
	ErrorMessage     string       `json:"errorMessage,omitempty"`
	Logs             string       `json:"logs,omitempty"`
	PreviousImageTag string       `json:"previousImageTag,omitempty"`
	CurrentImageTag  string       `json:"currentImageTag,omitempty"`
	CreatedAt        time.Time    `json:"createdAt"`
}

type CreateDeploymentInput struct {
	AppID         string `json:"appId"`
	CommitSHA     string `json:"commitSha"`
	CommitMessage string `json:"commitMessage,omitempty"`
}

type UpdateDeploymentInput struct {
	Status           *DeployStatus `json:"status,omitempty"`
	StartedAt        *time.Time    `json:"startedAt,omitempty"`
	FinishedAt       *time.Time    `json:"finishedAt,omitempty"`
	ErrorMessage     *string       `json:"errorMessage,omitempty"`
	Logs             *string       `json:"logs,omitempty"`
	PreviousImageTag *string       `json:"previousImageTag,omitempty"`
	CurrentImageTag  *string       `json:"currentImageTag,omitempty"`
}

type DeploymentRepository interface {
	FindByID(id string) (*Deployment, error)
	FindByAppID(appID string, limit int) ([]Deployment, error)
	FindPendingByAppID(appID string) (*Deployment, error)
	FindLatestByAppID(appID string) (*Deployment, error)
	Create(input CreateDeploymentInput) (*Deployment, error)
	Update(id string, input UpdateDeploymentInput) (*Deployment, error)
	AppendLogs(id string, logs string) error
	GetNextPending() (*Deployment, error)
	MarkAsRunning(id string) error
	MarkAsSuccess(id string, imageTag string) error
	MarkAsFailed(id string, errorMessage string) error
	DeleteByAppID(appID string) error
}
