package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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

func (f *fakePATHandlerRepo) FindByID(_ context.Context, _ string) (*domain.PersonalAccessToken, error) {
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

func newPATTestApp(t *testing.T, repo domain.PersonalAccessTokenRepository) *fiber.App {
	t.Helper()
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(func(c *fiber.Ctx) error {
		requestctx.SetUserInContext(c, &domain.User{ID: "user-1", Role: domain.RoleAdmin})
		return c.Next()
	})
	h := NewPATHandler(service.NewPersonalAccessTokenService(repo))
	h.Register(app.Group(APIPrefix))
	return app
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
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/x", func(c *fiber.Ctx) error {
		return mapPATError(c, service.ErrInvalidTokenName)
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
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/x", func(c *fiber.Ctx) error {
		return mapPATError(c, service.ErrNoScopesProvided)
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
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/x", func(c *fiber.Ctx) error {
		return mapPATError(c, service.ErrInvalidScope)
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
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/x", func(c *fiber.Ctx) error {
		return mapPATError(c, service.ErrExpiryOutOfRange)
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
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/x", func(c *fiber.Ctx) error {
		return mapPATError(c, errors.New("boom"))
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
	repo := &fakePATHandlerRepo{}
	app := newPATTestApp(t, repo)

	resp := doJSON(t, app, http.MethodPost, APIPrefix+"/tokens", map[string]any{
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
	if len(repo.tokens) != 1 {
		t.Fatalf("expected 1 token persisted, got %d", len(repo.tokens))
	}
}

func TestPATHandlerCreateRejectsInvalidScope(t *testing.T) {
	app := newPATTestApp(t, &fakePATHandlerRepo{})

	resp := doJSON(t, app, http.MethodPost, APIPrefix+"/tokens", map[string]any{
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
	app := newPATTestApp(t, repo)

	resp := doJSON(t, app, http.MethodGet, APIPrefix+"/tokens", nil)

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
	repo := &fakePATHandlerRepo{}
	app := newPATTestApp(t, repo)

	resp := doJSON(t, app, http.MethodDelete, APIPrefix+"/tokens/tok-42", nil)

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if len(repo.revokedIDs) != 1 || repo.revokedIDs[0] != "tok-42" {
		t.Fatalf("expected revoke called with tok-42, got %v", repo.revokedIDs)
	}
}

func TestPATHandlerRevokeReturnsNotFoundWhenRepoSaysSo(t *testing.T) {
	app := newPATTestApp(t, &fakePATHandlerRepo{revokeErr: domain.ErrNotFound})

	resp := doJSON(t, app, http.MethodDelete, APIPrefix+"/tokens/missing", nil)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
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
