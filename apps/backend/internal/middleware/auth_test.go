package middleware

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/crypto"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/service"
)

const (
	testUserID     = "user-1"
	testTokenID    = "tok-1"
	testSessionTok = "sess_plaintext_value"
	cookieName     = "paasdeploy_session"
)

type fakePATRepo struct {
	token *domain.PersonalAccessToken
	err   error
}

func (f *fakePATRepo) Create(_ context.Context, _ domain.CreatePersonalAccessTokenInput) (*domain.PersonalAccessToken, error) {
	return nil, nil
}

func (f *fakePATRepo) FindByTokenHash(_ context.Context, _ string) (*domain.PersonalAccessToken, error) {
	return f.token, f.err
}

func (f *fakePATRepo) FindByID(_ context.Context, _ string) (*domain.PersonalAccessToken, error) {
	return nil, domain.ErrNotFound
}

func (f *fakePATRepo) ListByUserID(_ context.Context, _ string) ([]domain.PersonalAccessToken, error) {
	return nil, nil
}

func (f *fakePATRepo) Revoke(_ context.Context, _, _ string) error {
	return nil
}

func (f *fakePATRepo) TouchLastUsed(_ context.Context, _ string) error {
	return nil
}

type fakeUserRepo struct {
	user *domain.User
	err  error
}

func (f *fakeUserRepo) FindByID(_ context.Context, _ string) (*domain.User, error) {
	return f.user, f.err
}

func (f *fakeUserRepo) FindByGitHubID(_ context.Context, _ int64) (*domain.User, error) {
	return nil, domain.ErrNotFound
}

func (f *fakeUserRepo) FindByEmail(_ context.Context, _ string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}

func (f *fakeUserRepo) Create(_ context.Context, _ domain.CreateUserInput) (*domain.User, error) {
	return nil, nil
}

func (f *fakeUserRepo) CreateEmailUser(_ context.Context, _ domain.CreateEmailUserInput) (*domain.User, error) {
	return nil, nil
}

func (f *fakeUserRepo) LinkGitHub(_ context.Context, _ string, _ domain.LinkGitHubInput) (*domain.User, error) {
	return nil, nil
}

func (f *fakeUserRepo) SetPassword(_ context.Context, _ string, _ string) (*domain.User, error) {
	return nil, nil
}

func (f *fakeUserRepo) Update(_ context.Context, _ string, _ domain.UpdateUserInput) (*domain.User, error) {
	return nil, nil
}

func (f *fakeUserRepo) Delete(_ context.Context, _ string) error {
	return nil
}

type fakeSessionRepo struct {
	session *domain.Session
	err     error
}

func (f *fakeSessionRepo) Create(_ context.Context, _ domain.CreateSessionInput) (*domain.Session, error) {
	return nil, nil
}

func (f *fakeSessionRepo) FindByTokenHash(_ context.Context, _ string) (*domain.Session, error) {
	return f.session, f.err
}

func (f *fakeSessionRepo) FindByUserID(_ context.Context, _ string) ([]domain.Session, error) {
	return nil, nil
}

func (f *fakeSessionRepo) Delete(_ context.Context, _ string) error {
	return nil
}

func (f *fakeSessionRepo) DeleteByUserID(_ context.Context, _ string) error {
	return nil
}

func (f *fakeSessionRepo) DeleteExpired(_ context.Context) (int64, error) {
	return 0, nil
}

func newActivePAT() *domain.PersonalAccessToken {
	future := time.Now().Add(24 * time.Hour)
	return &domain.PersonalAccessToken{
		ID:        testTokenID,
		UserID:    testUserID,
		Scopes:    []string{domain.ScopeRead},
		ExpiresAt: &future,
	}
}

func newRevokedPAT() *domain.PersonalAccessToken {
	now := time.Now()
	return &domain.PersonalAccessToken{
		ID:        testTokenID,
		UserID:    testUserID,
		Scopes:    []string{domain.ScopeRead},
		RevokedAt: &now,
	}
}

func newExpiredPAT() *domain.PersonalAccessToken {
	past := time.Now().Add(-1 * time.Hour)
	return &domain.PersonalAccessToken{
		ID:        testTokenID,
		UserID:    testUserID,
		Scopes:    []string{domain.ScopeRead},
		ExpiresAt: &past,
	}
}

func buildTestApp(t *testing.T, cfg AuthMiddlewareConfig, handler fiber.Handler) *fiber.App {
	t.Helper()
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if cfg.SessionCookieName == "" {
		cfg.SessionCookieName = cookieName
	}
	mw := NewAuthMiddleware(cfg)

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/protected", mw.Require(), handler)
	return app
}

func okHandler(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusOK)
}

func TestRequireRejectsMissingCredentials(t *testing.T) {
	app := buildTestApp(t, AuthMiddlewareConfig{
		SessionRepo: &fakeSessionRepo{err: domain.ErrNotFound},
		UserRepo:    &fakeUserRepo{err: domain.ErrNotFound},
		PATService:  service.NewPersonalAccessTokenService(&fakePATRepo{err: domain.ErrNotFound}),
	}, okHandler)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestRequireAcceptsValidSessionCookie(t *testing.T) {
	sessionHash := crypto.HashSessionToken(testSessionTok)
	app := buildTestApp(t, AuthMiddlewareConfig{
		SessionRepo: &fakeSessionRepo{session: &domain.Session{
			ID:        "sess-1",
			UserID:    testUserID,
			TokenHash: sessionHash,
		}},
		UserRepo:   &fakeUserRepo{user: &domain.User{ID: testUserID}},
		PATService: service.NewPersonalAccessTokenService(&fakePATRepo{err: domain.ErrNotFound}),
	}, okHandler)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: testSessionTok})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRequireAcceptsValidBearerToken(t *testing.T) {
	app := buildTestApp(t, AuthMiddlewareConfig{
		SessionRepo: &fakeSessionRepo{err: domain.ErrNotFound},
		UserRepo:    &fakeUserRepo{user: &domain.User{ID: testUserID}},
		PATService:  service.NewPersonalAccessTokenService(&fakePATRepo{token: newActivePAT()}),
	}, okHandler)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+service.TokenPrefix+"abc123")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRequireRejectsMalformedBearer(t *testing.T) {
	app := buildTestApp(t, AuthMiddlewareConfig{
		SessionRepo: &fakeSessionRepo{err: domain.ErrNotFound},
		UserRepo:    &fakeUserRepo{user: &domain.User{ID: testUserID}},
		PATService:  service.NewPersonalAccessTokenService(&fakePATRepo{}),
	}, okHandler)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer gh_not_ours")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for malformed prefix, got %d", resp.StatusCode)
	}
}

func TestRequireRejectsExpiredBearer(t *testing.T) {
	app := buildTestApp(t, AuthMiddlewareConfig{
		SessionRepo: &fakeSessionRepo{err: domain.ErrNotFound},
		UserRepo:    &fakeUserRepo{user: &domain.User{ID: testUserID}},
		PATService:  service.NewPersonalAccessTokenService(&fakePATRepo{token: newExpiredPAT()}),
	}, okHandler)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+service.TokenPrefix+"abc123")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", resp.StatusCode)
	}
}

func TestRequireRejectsRevokedBearer(t *testing.T) {
	app := buildTestApp(t, AuthMiddlewareConfig{
		SessionRepo: &fakeSessionRepo{err: domain.ErrNotFound},
		UserRepo:    &fakeUserRepo{user: &domain.User{ID: testUserID}},
		PATService:  service.NewPersonalAccessTokenService(&fakePATRepo{token: newRevokedPAT()}),
	}, okHandler)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+service.TokenPrefix+"abc123")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for revoked token, got %d", resp.StatusCode)
	}
}

func TestRequireRejectsUnknownBearer(t *testing.T) {
	app := buildTestApp(t, AuthMiddlewareConfig{
		SessionRepo: &fakeSessionRepo{err: domain.ErrNotFound},
		UserRepo:    &fakeUserRepo{user: &domain.User{ID: testUserID}},
		PATService:  service.NewPersonalAccessTokenService(&fakePATRepo{err: domain.ErrNotFound}),
	}, okHandler)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+service.TokenPrefix+"abc123")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unknown token, got %d", resp.StatusCode)
	}
}

func TestRequireBearerPrecedesCookie(t *testing.T) {
	app := buildTestApp(t, AuthMiddlewareConfig{
		SessionRepo: &fakeSessionRepo{err: domain.ErrNotFound},
		UserRepo:    &fakeUserRepo{user: &domain.User{ID: testUserID}},
		PATService:  service.NewPersonalAccessTokenService(&fakePATRepo{token: newActivePAT()}),
	}, okHandler)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+service.TokenPrefix+"abc123")
	req.AddCookie(&http.Cookie{Name: cookieName, Value: "irrelevant-session"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected bearer to succeed even with cookie present, got %d", resp.StatusCode)
	}
}

func TestRequireScopeAllowsWhenTokenHasScope(t *testing.T) {
	token := newActivePAT()
	token.Scopes = []string{domain.ScopeContainersWrite}

	cfg := AuthMiddlewareConfig{
		SessionRepo: &fakeSessionRepo{err: domain.ErrNotFound},
		UserRepo:    &fakeUserRepo{user: &domain.User{ID: testUserID}},
		PATService:  service.NewPersonalAccessTokenService(&fakePATRepo{token: token}),
	}
	mw := NewAuthMiddleware(AuthMiddlewareConfig{
		SessionRepo:       cfg.SessionRepo,
		UserRepo:          cfg.UserRepo,
		PATService:        cfg.PATService,
		Logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		SessionCookieName: cookieName,
	})

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/scoped", mw.Require(), RequireScope(domain.ScopeContainersWrite), okHandler)

	req := httptest.NewRequest(http.MethodGet, "/scoped", nil)
	req.Header.Set("Authorization", "Bearer "+service.TokenPrefix+"abc123")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRequireScopeRejectsWhenTokenMissingScope(t *testing.T) {
	token := newActivePAT()
	token.Scopes = []string{domain.ScopeRead}

	mw := NewAuthMiddleware(AuthMiddlewareConfig{
		SessionRepo:       &fakeSessionRepo{err: domain.ErrNotFound},
		UserRepo:          &fakeUserRepo{user: &domain.User{ID: testUserID}},
		PATService:        service.NewPersonalAccessTokenService(&fakePATRepo{token: token}),
		Logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		SessionCookieName: cookieName,
	})

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/scoped", mw.Require(), RequireScope(domain.ScopeDestructive), okHandler)

	req := httptest.NewRequest(http.MethodGet, "/scoped", nil)
	req.Header.Set("Authorization", "Bearer "+service.TokenPrefix+"abc123")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestRequireScopeAllowsSessionUserToBypassScopeCheck(t *testing.T) {
	sessionHash := crypto.HashSessionToken(testSessionTok)
	mw := NewAuthMiddleware(AuthMiddlewareConfig{
		SessionRepo: &fakeSessionRepo{session: &domain.Session{
			ID:        "sess-1",
			UserID:    testUserID,
			TokenHash: sessionHash,
		}},
		UserRepo:          &fakeUserRepo{user: &domain.User{ID: testUserID}},
		PATService:        service.NewPersonalAccessTokenService(&fakePATRepo{err: domain.ErrNotFound}),
		Logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		SessionCookieName: cookieName,
	})

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/scoped", mw.Require(), RequireScope(domain.ScopeDestructive), okHandler)

	req := httptest.NewRequest(http.MethodGet, "/scoped", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: testSessionTok})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for session user, got %d", resp.StatusCode)
	}
}

func TestDenyPATBlocksBearerTokens(t *testing.T) {
	token := newActivePAT()
	token.Scopes = []string{domain.ScopeAdmin}

	mw := NewAuthMiddleware(AuthMiddlewareConfig{
		SessionRepo:       &fakeSessionRepo{err: domain.ErrNotFound},
		UserRepo:          &fakeUserRepo{user: &domain.User{ID: testUserID}},
		PATService:        service.NewPersonalAccessTokenService(&fakePATRepo{token: token}),
		Logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		SessionCookieName: cookieName,
	})

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/sensitive", mw.Require(), DenyPAT("not allowed"), okHandler)

	req := httptest.NewRequest(http.MethodGet, "/sensitive", nil)
	req.Header.Set("Authorization", "Bearer "+service.TokenPrefix+"abc123")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for PAT bearer, got %d", resp.StatusCode)
	}
}

func TestDenyPATAllowsSessionUser(t *testing.T) {
	sessionHash := crypto.HashSessionToken(testSessionTok)
	mw := NewAuthMiddleware(AuthMiddlewareConfig{
		SessionRepo: &fakeSessionRepo{session: &domain.Session{
			ID:        "sess-1",
			UserID:    testUserID,
			TokenHash: sessionHash,
		}},
		UserRepo:          &fakeUserRepo{user: &domain.User{ID: testUserID}},
		PATService:        service.NewPersonalAccessTokenService(&fakePATRepo{err: domain.ErrNotFound}),
		Logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		SessionCookieName: cookieName,
	})

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/sensitive", mw.Require(), DenyPAT("not allowed"), okHandler)

	req := httptest.NewRequest(http.MethodGet, "/sensitive", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: testSessionTok})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for session user, got %d", resp.StatusCode)
	}
}
