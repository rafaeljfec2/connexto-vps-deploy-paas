package domain

import (
	"context"
	"time"
)

type Session struct {
	ID        string
	UserID    string
	TokenHash string
	IPAddress string
	UserAgent string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type CreateSessionInput struct {
	UserID    string
	TokenHash string
	IPAddress string
	UserAgent string
	ExpiresAt time.Time
}

type SessionRepository interface {
	Create(ctx context.Context, input CreateSessionInput) (*Session, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	FindByUserID(ctx context.Context, userID string) ([]Session, error)
	Delete(ctx context.Context, id string) error
	DeleteByUserID(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) (int64, error)
}
