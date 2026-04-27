package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/paasdeploy/backend/internal/crypto"
	"github.com/paasdeploy/backend/internal/domain"
)

const (
	TokenPrefix      = "pdp_live_"
	TokenSecretBytes = 32
	// TokenDisplayPrefixLen stores ONLY the well-known prefix in the token_prefix
	// column (no characters from the secret). This avoids leaking ~42 bits of
	// entropy gratuitously. The UI identifies a token by name + last-used; if a
	// short visual disambiguator is needed in the future, prefer "pdp_live_…XXXX"
	// where XXXX is the LAST 4 characters (lower correlation risk than the first).
	TokenDisplayPrefixLen = len(TokenPrefix)
	MaxTokenNameLength    = 120
	MinTokenNameLength    = 3
	DefaultExpiryDays     = 90
	MaxExpiryDays         = 365
)

var (
	ErrInvalidTokenName     = errors.New("invalid token name")
	ErrNoScopesProvided     = errors.New("at least one scope is required")
	ErrInvalidScope         = errors.New("invalid scope")
	ErrScopeNotAllowed      = errors.New("scope not allowed for user role")
	ErrExpiryOutOfRange     = errors.New("expiry must be between 1 and 365 days in the future")
	ErrMalformedTokenString = errors.New("malformed token")
)

type PersonalAccessTokenService struct {
	repo domain.PersonalAccessTokenRepository
}

func NewPersonalAccessTokenService(repo domain.PersonalAccessTokenRepository) *PersonalAccessTokenService {
	return &PersonalAccessTokenService{repo: repo}
}

type CreateTokenInput struct {
	UserID    string
	UserRole  string
	Name      string
	Scopes    []string
	ExpiresAt *time.Time
}

type CreateTokenResult struct {
	Token          *domain.PersonalAccessToken
	PlaintextToken string
}

func (s *PersonalAccessTokenService) Create(ctx context.Context, input CreateTokenInput) (*CreateTokenResult, error) {
	name := strings.TrimSpace(input.Name)
	if len(name) < MinTokenNameLength || len(name) > MaxTokenNameLength {
		return nil, ErrInvalidTokenName
	}

	scopes, err := validateScopes(input.Scopes, input.UserRole)
	if err != nil {
		return nil, err
	}

	expiresAt, err := validateExpiry(input.ExpiresAt)
	if err != nil {
		return nil, err
	}

	plaintext, err := generatePlaintextToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	hash := crypto.HashSessionToken(plaintext)
	displayPrefix := plaintext
	if len(displayPrefix) > TokenDisplayPrefixLen {
		displayPrefix = displayPrefix[:TokenDisplayPrefixLen]
	}

	token, err := s.repo.Create(ctx, domain.CreatePersonalAccessTokenInput{
		UserID:      input.UserID,
		Name:        name,
		TokenHash:   hash,
		TokenPrefix: displayPrefix,
		Scopes:      scopes,
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("persist token: %w", err)
	}

	return &CreateTokenResult{
		Token:          token,
		PlaintextToken: plaintext,
	}, nil
}

func (s *PersonalAccessTokenService) List(ctx context.Context, userID string) ([]domain.PersonalAccessToken, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *PersonalAccessTokenService) Revoke(ctx context.Context, tokenID, userID string) error {
	return s.repo.Revoke(ctx, tokenID, userID)
}

func (s *PersonalAccessTokenService) Authenticate(ctx context.Context, plaintext string) (*domain.PersonalAccessToken, error) {
	if !strings.HasPrefix(plaintext, TokenPrefix) {
		return nil, ErrMalformedTokenString
	}

	hash := crypto.HashSessionToken(plaintext)
	token, err := s.repo.FindByTokenHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	if err := token.IsActive(time.Now()); err != nil {
		return nil, err
	}

	return token, nil
}

func (s *PersonalAccessTokenService) TouchLastUsed(ctx context.Context, id string) {
	_ = s.repo.TouchLastUsed(ctx, id)
}

func generatePlaintextToken() (string, error) {
	bytes := make([]byte, TokenSecretBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	secret := base64.RawURLEncoding.EncodeToString(bytes)
	return TokenPrefix + secret, nil
}

func validateScopes(scopes []string, role string) ([]string, error) {
	if len(scopes) == 0 {
		return nil, ErrNoScopesProvided
	}
	seen := make(map[string]struct{}, len(scopes))
	cleaned := make([]string, 0, len(scopes))
	for _, s := range scopes {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if !domain.IsValidScope(s) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidScope, s)
		}
		if !domain.RoleAllowsScope(role, s) {
			return nil, fmt.Errorf("%w: %s", ErrScopeNotAllowed, s)
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		cleaned = append(cleaned, s)
	}
	if len(cleaned) == 0 {
		return nil, ErrNoScopesProvided
	}
	return cleaned, nil
}

func validateExpiry(expiresAt *time.Time) (*time.Time, error) {
	if expiresAt == nil {
		fallback := time.Now().Add(time.Duration(DefaultExpiryDays) * 24 * time.Hour)
		return &fallback, nil
	}
	now := time.Now()
	minAllowed := now.Add(1 * time.Hour)
	maxAllowed := now.Add(time.Duration(MaxExpiryDays) * 24 * time.Hour)
	if expiresAt.Before(minAllowed) || expiresAt.After(maxAllowed) {
		return nil, ErrExpiryOutOfRange
	}
	return expiresAt, nil
}
