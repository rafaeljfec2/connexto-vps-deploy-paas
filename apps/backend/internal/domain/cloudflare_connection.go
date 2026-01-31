package domain

import (
	"context"
	"time"
)

type CloudflareConnection struct {
	ID                    string
	UserID                string
	CloudflareAccountID   string
	CloudflareEmail       string
	AccessTokenEncrypted  string
	RefreshTokenEncrypted string
	TokenExpiresAt        *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type CreateCloudflareConnectionInput struct {
	UserID                string
	CloudflareAccountID   string
	CloudflareEmail       string
	AccessTokenEncrypted  string
	RefreshTokenEncrypted string
	TokenExpiresAt        *time.Time
}

type CloudflareConnectionRepository interface {
	Create(ctx context.Context, input CreateCloudflareConnectionInput) (*CloudflareConnection, error)
	FindByUserID(ctx context.Context, userID string) (*CloudflareConnection, error)
	Update(ctx context.Context, connection *CloudflareConnection) error
	Upsert(ctx context.Context, input CreateCloudflareConnectionInput) (*CloudflareConnection, error)
	DeleteByUserID(ctx context.Context, userID string) error
}
