package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrTokenExpired     = errors.New("personal access token expired")
	ErrTokenRevoked     = errors.New("personal access token revoked")
	ErrTokenScopeDenied = errors.New("personal access token missing required scope")
)

const (
	ScopeRead              = "read"
	ScopeDeploy            = "deploy"
	ScopeContainersWrite   = "containers:write"
	ScopeConfigWrite       = "config:write"
	ScopeResourcesWrite    = "resources:write"
	ScopeServersWrite      = "servers:write"
	ScopeDestructive       = "destructive"
	ScopeAdmin             = "admin"
)

var AllScopes = []string{
	ScopeRead,
	ScopeDeploy,
	ScopeContainersWrite,
	ScopeConfigWrite,
	ScopeResourcesWrite,
	ScopeServersWrite,
	ScopeDestructive,
	ScopeAdmin,
}

func IsValidScope(scope string) bool {
	for _, s := range AllScopes {
		if s == scope {
			return true
		}
	}
	return false
}

var memberAllowedScopes = map[string]struct{}{
	ScopeRead:            {},
	ScopeDeploy:          {},
	ScopeContainersWrite: {},
	ScopeConfigWrite:     {},
}

func RoleAllowsScope(role string, scope string) bool {
	if role == RoleAdmin {
		return true
	}
	_, ok := memberAllowedScopes[scope]
	return ok
}

type PersonalAccessToken struct {
	ID          string
	UserID      string
	Name        string
	TokenHash   string
	TokenPrefix string
	Scopes      []string
	LastUsedAt  *time.Time
	ExpiresAt   *time.Time
	RevokedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (t *PersonalAccessToken) IsActive(now time.Time) error {
	if t.RevokedAt != nil {
		return ErrTokenRevoked
	}
	if t.ExpiresAt != nil && !t.ExpiresAt.After(now) {
		return ErrTokenExpired
	}
	return nil
}

func (t *PersonalAccessToken) HasScope(scope string) bool {
	for _, s := range t.Scopes {
		if s == ScopeAdmin || s == scope {
			return true
		}
	}
	return false
}

type CreatePersonalAccessTokenInput struct {
	UserID      string
	Name        string
	TokenHash   string
	TokenPrefix string
	Scopes      []string
	ExpiresAt   *time.Time
}

type PersonalAccessTokenRepository interface {
	Create(ctx context.Context, input CreatePersonalAccessTokenInput) (*PersonalAccessToken, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (*PersonalAccessToken, error)
	FindByID(ctx context.Context, id string) (*PersonalAccessToken, error)
	ListByUserID(ctx context.Context, userID string) ([]PersonalAccessToken, error)
	Revoke(ctx context.Context, id string, userID string) error
	TouchLastUsed(ctx context.Context, id string) error
}
