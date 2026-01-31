package domain

import (
	"context"
	"time"
)

type Installation struct {
	ID                  string
	InstallationID      int64
	AccountType         string // "User" or "Organization"
	AccountID           int64
	AccountLogin        string
	RepositorySelection string // "all" or "selected"
	Permissions         map[string]string
	SuspendedAt         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type CreateInstallationInput struct {
	InstallationID      int64
	AccountType         string
	AccountID           int64
	AccountLogin        string
	RepositorySelection string
	Permissions         map[string]string
}

type UpdateInstallationInput struct {
	RepositorySelection *string
	Permissions         map[string]string
	SuspendedAt         *time.Time
}

type UserInstallation struct {
	ID             string
	UserID         string
	InstallationID string
	IsDefault      bool
	CreatedAt      time.Time
}

type InstallationRepository interface {
	FindByID(ctx context.Context, id string) (*Installation, error)
	FindByInstallationID(ctx context.Context, installationID int64) (*Installation, error)
	FindByAccountLogin(ctx context.Context, accountLogin string) (*Installation, error)
	Create(ctx context.Context, input CreateInstallationInput) (*Installation, error)
	Update(ctx context.Context, id string, input UpdateInstallationInput) (*Installation, error)
	Delete(ctx context.Context, id string) error

	// User-Installation relationships
	LinkUserToInstallation(ctx context.Context, userID, installationID string, isDefault bool) error
	UnlinkUserFromInstallation(ctx context.Context, userID, installationID string) error
	FindUserInstallations(ctx context.Context, userID string) ([]Installation, error)
	FindDefaultInstallation(ctx context.Context, userID string) (*Installation, error)
	SetDefaultInstallation(ctx context.Context, userID, installationID string) error
}
