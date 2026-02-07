package domain

import (
	"context"
	"time"
)

type User struct {
	ID                    string
	GitHubID              *int64
	GitHubLogin           string
	Name                  string
	Email                 string
	AvatarURL             string
	AccessTokenEncrypted  string
	RefreshTokenEncrypted string
	TokenExpiresAt        *time.Time
	PasswordHash          string
	AuthProvider          string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type CreateUserInput struct {
	GitHubID              int64
	GitHubLogin           string
	Name                  string
	Email                 string
	AvatarURL             string
	AccessTokenEncrypted  string
	RefreshTokenEncrypted string
	TokenExpiresAt        *time.Time
}

type CreateEmailUserInput struct {
	Email        string
	Name         string
	PasswordHash string
}

type LinkGitHubInput struct {
	GitHubID              int64
	GitHubLogin           string
	Name                  string
	Email                 string
	AvatarURL             string
	AccessTokenEncrypted  string
	RefreshTokenEncrypted string
	TokenExpiresAt        *time.Time
}

type UpdateUserInput struct {
	GitHubLogin           *string
	Name                  *string
	Email                 *string
	AvatarURL             *string
	AccessTokenEncrypted  *string
	RefreshTokenEncrypted *string
	TokenExpiresAt        *time.Time
}

type UserRepository interface {
	FindByID(ctx context.Context, id string) (*User, error)
	FindByGitHubID(ctx context.Context, githubID int64) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, input CreateUserInput) (*User, error)
	CreateEmailUser(ctx context.Context, input CreateEmailUserInput) (*User, error)
	LinkGitHub(ctx context.Context, userID string, input LinkGitHubInput) (*User, error)
	SetPassword(ctx context.Context, userID string, passwordHash string) (*User, error)
	Update(ctx context.Context, id string, input UpdateUserInput) (*User, error)
	Delete(ctx context.Context, id string) error
}
