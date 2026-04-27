package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/paasdeploy/backend/internal/domain"
)

type fakePATRepo struct {
	createErr       error
	createdInput    domain.CreatePersonalAccessTokenInput
	createdToken    *domain.PersonalAccessToken
	findByHashToken *domain.PersonalAccessToken
	findByHashErr   error
	listResult      []domain.PersonalAccessToken
	listErr         error
	revokeErr       error
	touchErr        error
	touchCalls      int
}

func (f *fakePATRepo) Create(_ context.Context, input domain.CreatePersonalAccessTokenInput) (*domain.PersonalAccessToken, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.createdInput = input
	token := &domain.PersonalAccessToken{
		ID:          "tok_1",
		UserID:      input.UserID,
		Name:        input.Name,
		TokenHash:   input.TokenHash,
		TokenPrefix: input.TokenPrefix,
		Scopes:      input.Scopes,
		ExpiresAt:   input.ExpiresAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	f.createdToken = token
	return token, nil
}

func (f *fakePATRepo) FindByTokenHash(_ context.Context, _ string) (*domain.PersonalAccessToken, error) {
	if f.findByHashErr != nil {
		return nil, f.findByHashErr
	}
	return f.findByHashToken, nil
}

func (f *fakePATRepo) FindByID(_ context.Context, _ string) (*domain.PersonalAccessToken, error) {
	return nil, domain.ErrNotFound
}

func (f *fakePATRepo) ListByUserID(_ context.Context, _ string) ([]domain.PersonalAccessToken, error) {
	return f.listResult, f.listErr
}

func (f *fakePATRepo) Revoke(_ context.Context, _ string, _ string) error {
	return f.revokeErr
}

func (f *fakePATRepo) TouchLastUsed(_ context.Context, _ string) error {
	f.touchCalls++
	return f.touchErr
}

func adminInput(name string, scopes []string) CreateTokenInput {
	return CreateTokenInput{
		UserID:   "user-1",
		UserRole: domain.RoleAdmin,
		Name:     name,
		Scopes:   scopes,
	}
}

func TestCreateTokenRejectsShortName(t *testing.T) {
	svc := NewPersonalAccessTokenService(&fakePATRepo{})

	_, err := svc.Create(context.Background(), adminInput("ab", []string{domain.ScopeRead}))

	if !errors.Is(err, ErrInvalidTokenName) {
		t.Fatalf("expected ErrInvalidTokenName, got %v", err)
	}
}

func TestCreateTokenRejectsLongName(t *testing.T) {
	svc := NewPersonalAccessTokenService(&fakePATRepo{})

	_, err := svc.Create(context.Background(), adminInput(strings.Repeat("a", MaxTokenNameLength+1), []string{domain.ScopeRead}))

	if !errors.Is(err, ErrInvalidTokenName) {
		t.Fatalf("expected ErrInvalidTokenName, got %v", err)
	}
}

func TestCreateTokenRejectsMissingScopes(t *testing.T) {
	svc := NewPersonalAccessTokenService(&fakePATRepo{})

	_, err := svc.Create(context.Background(), adminInput("valid-name", nil))

	if !errors.Is(err, ErrNoScopesProvided) {
		t.Fatalf("expected ErrNoScopesProvided, got %v", err)
	}
}

func TestCreateTokenRejectsInvalidScope(t *testing.T) {
	svc := NewPersonalAccessTokenService(&fakePATRepo{})

	_, err := svc.Create(context.Background(), adminInput("valid-name", []string{"not-a-scope"}))

	if !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("expected ErrInvalidScope, got %v", err)
	}
}

func TestCreateTokenRejectsScopeAboveUserRole(t *testing.T) {
	svc := NewPersonalAccessTokenService(&fakePATRepo{})

	_, err := svc.Create(context.Background(), CreateTokenInput{
		UserID:   "user-1",
		UserRole: domain.RoleMember,
		Name:     "valid-name",
		Scopes:   []string{domain.ScopeAdmin},
	})

	if !errors.Is(err, ErrScopeNotAllowed) {
		t.Fatalf("expected ErrScopeNotAllowed, got %v", err)
	}
}

func TestCreateTokenAllowsMemberSafeScopes(t *testing.T) {
	repo := &fakePATRepo{}
	svc := NewPersonalAccessTokenService(repo)

	_, err := svc.Create(context.Background(), CreateTokenInput{
		UserID:   "user-1",
		UserRole: domain.RoleMember,
		Name:     "valid-name",
		Scopes:   []string{domain.ScopeRead, domain.ScopeDeploy},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.createdInput.Scopes) != 2 {
		t.Fatalf("expected 2 scopes persisted, got %v", repo.createdInput.Scopes)
	}
}

func TestCreateTokenDeduplicatesScopes(t *testing.T) {
	repo := &fakePATRepo{}
	svc := NewPersonalAccessTokenService(repo)

	_, err := svc.Create(context.Background(), adminInput("valid-name", []string{domain.ScopeRead, domain.ScopeRead, domain.ScopeDeploy}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.createdInput.Scopes) != 2 {
		t.Fatalf("expected scopes to be deduplicated, got %v", repo.createdInput.Scopes)
	}
}

func TestCreateTokenRejectsExpiryTooSoon(t *testing.T) {
	svc := NewPersonalAccessTokenService(&fakePATRepo{})
	past := time.Now().Add(-1 * time.Hour)
	in := adminInput("valid-name", []string{domain.ScopeRead})
	in.ExpiresAt = &past

	_, err := svc.Create(context.Background(), in)

	if !errors.Is(err, ErrExpiryOutOfRange) {
		t.Fatalf("expected ErrExpiryOutOfRange, got %v", err)
	}
}

func TestCreateTokenRejectsExpiryTooFar(t *testing.T) {
	svc := NewPersonalAccessTokenService(&fakePATRepo{})
	tooFar := time.Now().Add(time.Duration(MaxExpiryDays+1) * 24 * time.Hour)
	in := adminInput("valid-name", []string{domain.ScopeRead})
	in.ExpiresAt = &tooFar

	_, err := svc.Create(context.Background(), in)

	if !errors.Is(err, ErrExpiryOutOfRange) {
		t.Fatalf("expected ErrExpiryOutOfRange, got %v", err)
	}
}

func TestCreateTokenUsesDefaultExpiryWhenNotProvided(t *testing.T) {
	repo := &fakePATRepo{}
	svc := NewPersonalAccessTokenService(repo)

	_, err := svc.Create(context.Background(), adminInput("valid-name", []string{domain.ScopeRead}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.createdInput.ExpiresAt == nil {
		t.Fatal("expected default expiry to be set")
	}
	diff := time.Until(*repo.createdInput.ExpiresAt)
	expected := time.Duration(DefaultExpiryDays) * 24 * time.Hour
	if diff < expected-2*time.Hour || diff > expected+2*time.Hour {
		t.Fatalf("expected default expiry near %v, got %v", expected, diff)
	}
}

func TestCreateTokenReturnsPlaintextWithCorrectPrefix(t *testing.T) {
	repo := &fakePATRepo{}
	svc := NewPersonalAccessTokenService(repo)

	result, err := svc.Create(context.Background(), adminInput("valid-name", []string{domain.ScopeRead}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(result.PlaintextToken, TokenPrefix) {
		t.Fatalf("expected plaintext to start with %q, got %q", TokenPrefix, result.PlaintextToken)
	}
	if result.Token.TokenHash == result.PlaintextToken {
		t.Fatal("token hash must not equal plaintext")
	}
	if repo.createdInput.TokenPrefix == "" || len(repo.createdInput.TokenPrefix) > TokenDisplayPrefixLen {
		t.Fatalf("invalid display prefix: %q", repo.createdInput.TokenPrefix)
	}
}

func TestAuthenticateRejectsWrongPrefix(t *testing.T) {
	svc := NewPersonalAccessTokenService(&fakePATRepo{})

	_, err := svc.Authenticate(context.Background(), "gh_fake_token")

	if !errors.Is(err, ErrMalformedTokenString) {
		t.Fatalf("expected ErrMalformedTokenString, got %v", err)
	}
}

func TestAuthenticatePropagatesRepoError(t *testing.T) {
	repo := &fakePATRepo{findByHashErr: domain.ErrNotFound}
	svc := NewPersonalAccessTokenService(repo)

	_, err := svc.Authenticate(context.Background(), TokenPrefix+"abc123")

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAuthenticateRejectsRevokedToken(t *testing.T) {
	now := time.Now()
	repo := &fakePATRepo{findByHashToken: &domain.PersonalAccessToken{
		ID:        "tok_1",
		RevokedAt: &now,
	}}
	svc := NewPersonalAccessTokenService(repo)

	_, err := svc.Authenticate(context.Background(), TokenPrefix+"abc123")

	if !errors.Is(err, domain.ErrTokenRevoked) {
		t.Fatalf("expected ErrTokenRevoked, got %v", err)
	}
}

func TestAuthenticateRejectsExpiredToken(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	repo := &fakePATRepo{findByHashToken: &domain.PersonalAccessToken{
		ID:        "tok_1",
		ExpiresAt: &past,
	}}
	svc := NewPersonalAccessTokenService(repo)

	_, err := svc.Authenticate(context.Background(), TokenPrefix+"abc123")

	if !errors.Is(err, domain.ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}

func TestAuthenticateReturnsActiveToken(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	repo := &fakePATRepo{findByHashToken: &domain.PersonalAccessToken{
		ID:        "tok_1",
		ExpiresAt: &future,
		Scopes:    []string{domain.ScopeRead},
	}}
	svc := NewPersonalAccessTokenService(repo)

	token, err := svc.Authenticate(context.Background(), TokenPrefix+"abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token.ID != "tok_1" {
		t.Fatalf("expected ID tok_1, got %q", token.ID)
	}
}

func TestTouchLastUsedDelegatesToRepo(t *testing.T) {
	repo := &fakePATRepo{}
	svc := NewPersonalAccessTokenService(repo)

	svc.TouchLastUsed(context.Background(), "tok_1")

	if repo.touchCalls != 1 {
		t.Fatalf("expected 1 touch call, got %d", repo.touchCalls)
	}
}
