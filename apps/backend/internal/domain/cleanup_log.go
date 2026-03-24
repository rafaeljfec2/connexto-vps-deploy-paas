package domain

import "time"

type CleanupType string

const (
	CleanupTypeContainers CleanupType = "containers"
	CleanupTypeVolumes    CleanupType = "volumes"
	CleanupTypeImages     CleanupType = "images"
)

type CleanupTrigger string

const (
	CleanupTriggerScheduled CleanupTrigger = "scheduled"
	CleanupTriggerManual    CleanupTrigger = "manual"
)

type CleanupStatus string

const (
	CleanupStatusSuccess CleanupStatus = "success"
	CleanupStatusFailed  CleanupStatus = "failed"
)

type CleanupLog struct {
	ID                  string         `json:"id"`
	ServerID            string         `json:"serverId"`
	CleanupType         CleanupType    `json:"cleanupType"`
	ItemsRemoved        int            `json:"itemsRemoved"`
	SpaceReclaimedBytes int64          `json:"spaceReclaimedBytes"`
	Trigger             CleanupTrigger `json:"trigger"`
	Status              CleanupStatus  `json:"status"`
	ErrorMessage        string         `json:"errorMessage,omitempty"`
	CreatedAt           time.Time      `json:"createdAt"`
}

type CreateCleanupLogInput struct {
	ServerID            string
	CleanupType         CleanupType
	ItemsRemoved        int
	SpaceReclaimedBytes int64
	Trigger             CleanupTrigger
	Status              CleanupStatus
	ErrorMessage        string
}

type CleanupLogRepository interface {
	Create(input CreateCleanupLogInput) (*CleanupLog, error)
	FindByServerID(serverID string, limit int, offset int) ([]CleanupLog, error)
}
