package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/requestctx"
	"github.com/paasdeploy/backend/internal/service"
)

type fakePATHandlerRepo struct {
	tokens     []domain.PersonalAccessToken
	createErr  error
	listErr    error
	revokeErr  error
	revokedIDs []string
}

func (f *fakePATHandlerRepo) Create(_ context.Context, input domain.CreatePersonalAccessTokenInput) (*domain.PersonalAccessToken, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	tok := domain.PersonalAccessToken{
		ID:          "tok-new",
		UserID:      input.UserID,
		Name:        input.Name,
		TokenHash:   input.TokenHash,
		TokenPrefix: input.TokenPrefix,
		Scopes:      input.Scopes,
		ExpiresAt:   input.ExpiresAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	f.tokens = append(f.tokens, tok)
	return &tok, nil
}

func (f *fakePATHandlerRepo) FindByTokenHash(_ context.Context, _ string) (*domain.PersonalAccessToken, error) {
	return nil, domain.ErrNotFound
}

func (f *fakePATHandlerRepo) FindByID(_ context.Context, id string) (*domain.PersonalAccessToken, error) {
	for i := range f.tokens {
		if f.tokens[i].ID == id {
			t := f.tokens[i]
			return &t, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakePATHandlerRepo) ListByUserID(_ context.Context, _ string) ([]domain.PersonalAccessToken, error) {
	return f.tokens, f.listErr
}

func (f *fakePATHandlerRepo) Revoke(_ context.Context, id, _ string) error {
	if f.revokeErr != nil {
		return f.revokeErr
	}
	f.revokedIDs = append(f.revokedIDs, id)
	return nil
}

func (f *fakePATHandlerRepo) TouchLastUsed(_ context.Context, _ string) error { return nil }

type fakeAuditRepo struct {
	entries []domain.CreateAuditLogInput
}

func (f *fakeAuditRepo) Create(input domain.CreateAuditLogInput) (*domain.AuditLog, error) {
	f.entries = append(f.entries, input)
	return &domain.AuditLog{
		ID:        "audit-new",
		EventType: input.EventType,
	}, nil
}

func (f *fakeAuditRepo) FindByID(_ string) (*domain.AuditLog, error) {
	return nil, domain.ErrNotFound
}

func (f *fakeAuditRepo) FindAll(_ domain.AuditLogFilter) ([]domain.AuditLog, int, error) {
	return nil, 0, nil
}

func (f *fakeAuditRepo) DeleteOlderThan(_ int) (int64, error) { return 0, nil }

type patTestDeps struct {
	app       *fiber.App
	repo      *fakePATHandlerRepo
	auditRepo *fakeAuditRepo
}

func newPATTestApp(t *testing.T, repo domain.PersonalAccessTokenRepository) patTestDeps {
	t.Helper()
	auditRepo := &fakeAuditRepo{}
	auditSvc := service.NewAuditService(auditRepo, slog.Default())
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(func(c *fiber.Ctx) error {
		requestctx.SetUserInContext(c, &domain.User{ID: "user-1", Role: domain.RoleAdmin})
		return c.Next()
	})
	h := NewPATHandler(service.NewPersonalAccessTokenService(repo), auditSvc, slog.Default())
	h.Register(app.Group(APIPrefix))

	concrete, _ := repo.(*fakePATHandlerRepo)
	return patTestDeps{app: app, repo: concrete, auditRepo: auditRepo}
}

func newHandlerForMapErrorTest(t *testing.T) *PATHandler {
	t.Helper()
	repo := &fakePATHandlerRepo{}
	return NewPATHandler(service.NewPersonalAccessTokenService(repo), nil, slog.Default())
}

func doJSON(t *testing.T, app *fiber.App, method, path string, body any) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	return resp
}

func patTestRequest(path string) *http.Request {
	return httptest.NewRequest(http.MethodGet, path, nil)
}

func TestMapPATErrorMapsInvalidName(t *testing.T) {
	h := newHandlerForMapErrorTest(t)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/x", func(c *fiber.Ctx) error {
		return h.mapPATError(c, service.ErrInvalidTokenName)
	})

	resp, err := app.Test(patTestRequest("/x"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestMapPATErrorMapsNoScopes(t *testing.T) {
	h := newHandlerForMapErrorTest(t)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/x", func(c *fiber.Ctx) error {
		return h.mapPATError(c, service.ErrNoScopesProvided)
	})

	resp, err := app.Test(patTestRequest("/x"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestMapPATErrorMapsInvalidScope(t *testing.T) {
	h := newHandlerForMapErrorTest(t)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/x", func(c *fiber.Ctx) error {
		return h.mapPATError(c, service.ErrInvalidScope)
	})

	resp, err := app.Test(patTestRequest("/x"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestMapPATErrorMapsExpiryRange(t *testing.T) {
	h := newHandlerForMapErrorTest(t)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/x", func(c *fiber.Ctx) error {
		return h.mapPATError(c, service.ErrExpiryOutOfRange)
	})

	resp, err := app.Test(patTestRequest("/x"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestMapPATErrorMapsUnknownErrorAsInternal(t *testing.T) {
	h := newHandlerForMapErrorTest(t)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/x", func(c *fiber.Ctx) error {
		return h.mapPATError(c, errors.New("boom"))
	})

	resp, err := app.Test(patTestRequest("/x"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

func TestPATHandlerCreateReturnsCreatedWithPlaintext(t *testing.T) {
	deps := newPATTestApp(t, &fakePATHandlerRepo{})

	resp := doJSON(t, deps.app, http.MethodPost, APIPrefix+"/tokens", map[string]any{
		"name":   "ci-bot",
		"scopes": []string{domain.ScopeRead, domain.ScopeDeploy},
	})

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var body struct {
		Data struct {
			Token struct {
				ID          string   `json:"id"`
				Name        string   `json:"name"`
				TokenPrefix string   `json:"tokenPrefix"`
				Scopes      []string `json:"scopes"`
			} `json:"token"`
			PlaintextToken string `json:"plaintextToken"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Data.Token.Name != "ci-bot" {
		t.Fatalf("expected name ci-bot, got %q", body.Data.Token.Name)
	}
	if body.Data.PlaintextToken == "" {
		t.Fatal("plaintextToken must be present in the create response")
	}
	if len(body.Data.PlaintextToken) < len(service.TokenPrefix)+10 {
		t.Fatalf("plaintextToken seems truncated: %q", body.Data.PlaintextToken)
	}
	if len(deps.repo.tokens) != 1 {
		t.Fatalf("expected 1 token persisted, got %d", len(deps.repo.tokens))
	}
}

func TestPATHandlerCreateEmitsAuditOnSuccess(t *testing.T) {
	deps := newPATTestApp(t, &fakePATHandlerRepo{})

	resp := doJSON(t, deps.app, http.MethodPost, APIPrefix+"/tokens", map[string]any{
		"name":   "ci-bot",
		"scopes": []string{domain.ScopeRead, domain.ScopeDeploy},
	})
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	if len(deps.auditRepo.entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(deps.auditRepo.entries))
	}
	entry := deps.auditRepo.entries[0]
	if entry.EventType != domain.EventTokenCreated {
		t.Fatalf("expected event token.created, got %q", entry.EventType)
	}
	if entry.ResourceType != domain.ResourceToken {
		t.Fatalf("expected resource token, got %q", entry.ResourceType)
	}
	if entry.ResourceID == nil || *entry.ResourceID != "tok-new" {
		t.Fatalf("expected resource id tok-new, got %v", entry.ResourceID)
	}
	if entry.ResourceName == nil || *entry.ResourceName != "ci-bot" {
		t.Fatalf("expected resource name ci-bot, got %v", entry.ResourceName)
	}
	if entry.ActorType != domain.ActorUser {
		t.Fatalf("expected actor user, got %q", entry.ActorType)
	}
	if entry.UserID == nil || *entry.UserID != "user-1" {
		t.Fatalf("expected user_id user-1, got %v", entry.UserID)
	}
	scopes, ok := entry.Details["scopes"].([]string)
	if !ok || len(scopes) != 2 {
		t.Fatalf("expected scopes details to have 2 entries, got %#v", entry.Details["scopes"])
	}
}

func TestPATHandlerCreateDoesNotEmitAuditOnFailure(t *testing.T) {
	deps := newPATTestApp(t, &fakePATHandlerRepo{createErr: errors.New("db down")})

	resp := doJSON(t, deps.app, http.MethodPost, APIPrefix+"/tokens", map[string]any{
		"name":   "ci-bot",
		"scopes": []string{domain.ScopeRead},
	})
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}

	if len(deps.auditRepo.entries) != 0 {
		t.Fatalf("expected 0 audit entries on failure, got %d", len(deps.auditRepo.entries))
	}
}

func TestPATHandlerCreateRejectsInvalidScope(t *testing.T) {
	deps := newPATTestApp(t, &fakePATHandlerRepo{})

	resp := doJSON(t, deps.app, http.MethodPost, APIPrefix+"/tokens", map[string]any{
		"name":   "broken",
		"scopes": []string{"not-a-real-scope"},
	})

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 for invalid scope, got %d", resp.StatusCode)
	}
}

func TestPATHandlerListReturnsPersistedTokens(t *testing.T) {
	now := time.Now()
	repo := &fakePATHandlerRepo{
		tokens: []domain.PersonalAccessToken{
			{ID: "tok-1", Name: "first", TokenPrefix: "pdp_live_", Scopes: []string{domain.ScopeRead}, CreatedAt: now},
			{ID: "tok-2", Name: "second", TokenPrefix: "pdp_live_", Scopes: []string{domain.ScopeDeploy}, CreatedAt: now},
		},
	}
	deps := newPATTestApp(t, repo)

	resp := doJSON(t, deps.app, http.MethodGet, APIPrefix+"/tokens", nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body struct {
		Data struct {
			Tokens []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"tokens"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data.Tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(body.Data.Tokens))
	}
}

func TestPATHandlerRevokeReturnsNoContentAndCallsRepo(t *testing.T) {
	repo := &fakePATHandlerRepo{
		tokens: []domain.PersonalAccessToken{
			{ID: "tok-42", UserID: "user-1", Name: "ci-bot", TokenPrefix: "pdp_live_", Scopes: []string{domain.ScopeRead}},
		},
	}
	deps := newPATTestApp(t, repo)

	resp := doJSON(t, deps.app, http.MethodDelete, APIPrefix+"/tokens/tok-42", nil)

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if len(deps.repo.revokedIDs) != 1 || deps.repo.revokedIDs[0] != "tok-42" {
		t.Fatalf("expected revoke called with tok-42, got %v", deps.repo.revokedIDs)
	}
	if len(deps.auditRepo.entries) != 1 {
		t.Fatalf("expected 1 audit entry for revoke, got %d", len(deps.auditRepo.entries))
	}
	entry := deps.auditRepo.entries[0]
	if entry.EventType != domain.EventTokenRevoked {
		t.Fatalf("expected event token.revoked, got %q", entry.EventType)
	}
	if entry.ResourceID == nil || *entry.ResourceID != "tok-42" {
		t.Fatalf("expected resource id tok-42, got %v", entry.ResourceID)
	}
	if entry.ResourceName == nil || *entry.ResourceName != "ci-bot" {
		t.Fatalf("expected resource name ci-bot, got %v", entry.ResourceName)
	}
	if entry.ActorType != domain.ActorUser {
		t.Fatalf("expected actor user, got %q", entry.ActorType)
	}
	if entry.UserID == nil || *entry.UserID != "user-1" {
		t.Fatalf("expected user_id user-1, got %v", entry.UserID)
	}
}

func TestPATHandlerRevokeReturnsNotFoundWhenRepoSaysSo(t *testing.T) {
	deps := newPATTestApp(t, &fakePATHandlerRepo{revokeErr: domain.ErrNotFound})

	resp := doJSON(t, deps.app, http.MethodDelete, APIPrefix+"/tokens/missing", nil)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	if len(deps.auditRepo.entries) != 0 {
		t.Fatalf("expected 0 audit entries on revoke not-found, got %d", len(deps.auditRepo.entries))
	}
}

func TestToTokenResponseCopiesAllFields(t *testing.T) {
	token := &domain.PersonalAccessToken{
		ID:          "tok-1",
		Name:        "demo",
		TokenPrefix: "pdp_live_abc",
		Scopes:      []string{"read", "deploy"},
	}

	resp := toTokenResponse(token)

	if resp.ID != "tok-1" {
		t.Fatalf("unexpected ID: %s", resp.ID)
	}
	if resp.Name != "demo" {
		t.Fatalf("unexpected Name: %s", resp.Name)
	}
	if len(resp.Scopes) != 2 {
		t.Fatalf("expected 2 scopes, got %d", len(resp.Scopes))
	}
}
