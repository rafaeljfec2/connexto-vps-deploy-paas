package backend

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestApplyHeadersUsesContextToken(t *testing.T) {
	captured := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured <- r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"data":{}}`))
	}))
	defer srv.Close()

	c, err := New(Options{
		BaseURL:                srv.URL,
		ClientID:               "ci:test",
		Timeout:                time.Second,
		Logger:                 slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRetries:             1,
		AcceptTokenFromContext: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := WithToken(context.Background(), "pdp_live_from_ctx")
	if _, err := Do[map[string]any](ctx, c, RequestOptions{Method: http.MethodGet, Path: "/system/status"}); err != nil {
		t.Fatalf("Do: %v", err)
	}

	got := <-captured
	if got != "Bearer pdp_live_from_ctx" {
		t.Fatalf("expected token from context to win, got %q", got)
	}
}

func TestApplyHeadersFailsWhenTokenMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, err := New(Options{
		BaseURL:                srv.URL,
		Timeout:                time.Second,
		Logger:                 slog.New(slog.NewTextHandler(io.Discard, nil)),
		AcceptTokenFromContext: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := Do[map[string]any](context.Background(), c, RequestOptions{Method: http.MethodGet, Path: "/system/status"}); err == nil {
		t.Fatalf("expected error when no token is configured nor injected via context")
	}
}

func TestApplyHeadersClientIDFromContextWins(t *testing.T) {
	captured := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured <- r.Header.Get("X-MCP-Client")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"data":{}}`))
	}))
	defer srv.Close()

	c, err := New(Options{
		BaseURL:    srv.URL,
		Token:      "pdp_live_static",
		ClientID:   "static:client",
		Timeout:    time.Second,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := WithClientID(context.Background(), "cursor")
	if _, err := Do[map[string]any](ctx, c, RequestOptions{Method: http.MethodGet, Path: "/system/status"}); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if got := <-captured; got != "cursor" {
		t.Fatalf("expected client id from context to win, got %q", got)
	}
}

func TestNewRequiresTokenOrAcceptFlag(t *testing.T) {
	if _, err := New(Options{BaseURL: "https://api.test"}); err == nil {
		t.Fatalf("expected error when token missing AND AcceptTokenFromContext=false")
	}
	if _, err := New(Options{BaseURL: "https://api.test", AcceptTokenFromContext: true}); err != nil {
		t.Fatalf("expected New to accept empty token in multi-tenant mode, got %v", err)
	}
}

// Regression: in multi-tenant mode the static c.token MUST NEVER be used as a
// fallback. If a misconfigured operator sets both AcceptTokenFromContext=true
// AND Token="pdp_live_static", a forgotten WithToken in the request path used
// to silently send the operator's PAT to the backend, leaking the credential
// across tenants. The fix returns an explicit error instead.
func TestApplyHeadersDoesNotFallBackToStaticTokenInMultiTenantMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatalf("backend MUST NOT be hit when token is missing in multi-tenant mode")
	}))
	defer srv.Close()

	c, err := New(Options{
		BaseURL:                srv.URL,
		Token:                  "pdp_live_operator_static",
		Timeout:                time.Second,
		Logger:                 slog.New(slog.NewTextHandler(io.Discard, nil)),
		AcceptTokenFromContext: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = Do[map[string]any](context.Background(), c, RequestOptions{Method: http.MethodGet, Path: "/system/status"})
	if err == nil {
		t.Fatalf("expected error: multi-tenant client must refuse to fall back to static token")
	}
}

// Regression: non-idempotent mutations (POST/PUT/PATCH without explicit
// Idempotent=true) MUST NOT be retried on transport errors or 5xx, otherwise
// the backend may apply the side effect twice. GET and Idempotent=true are
// allowed to retry.
func TestDoDoesNotRetryNonIdempotentMutationsOn5xx(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"unavailable","message":"503"}}`))
	}))
	defer srv.Close()

	c, err := New(Options{
		BaseURL:    srv.URL,
		Token:      "pdp_live_test",
		Timeout:    time.Second,
		MaxRetries: 5,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = Do[map[string]any](context.Background(), c, RequestOptions{
		Method: http.MethodPost,
		Path:   "/apps/x/deploy",
	})
	if err == nil {
		t.Fatalf("expected error from 503")
	}
	if attempts != 1 {
		t.Fatalf("non-idempotent POST must NOT retry on 5xx; got %d attempts", attempts)
	}
}

func TestDoRetriesIdempotentMutationsOn5xx(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"data":{"ok":true}}`))
	}))
	defer srv.Close()

	c, err := New(Options{
		BaseURL:    srv.URL,
		Token:      "pdp_live_test",
		Timeout:    time.Second,
		MaxRetries: 3,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := Do[map[string]any](context.Background(), c, RequestOptions{
		Method:     http.MethodDelete,
		Path:       "/apps/x",
		Idempotent: true,
	}); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("idempotent DELETE must retry once after 503; got %d attempts", attempts)
	}
}

func TestDoRetriesGetByDefault(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"data":{}}`))
	}))
	defer srv.Close()

	c, err := New(Options{
		BaseURL:    srv.URL,
		Token:      "pdp_live_test",
		Timeout:    time.Second,
		MaxRetries: 3,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := Do[map[string]any](context.Background(), c, RequestOptions{Method: http.MethodGet, Path: "/system/status"}); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("GET must retry on 503 by default; got %d attempts", attempts)
	}
}

func TestPingReturnsNilWhenBackendReachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c, err := New(Options{
		BaseURL: srv.URL,
		Token:   "pdp_live_test",
		Timeout: time.Second,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := c.Ping(context.Background()); err != nil {
		t.Fatalf("Ping must succeed regardless of HTTP status when reachable; got %v", err)
	}
}

func TestPingReturnsErrorWhenUnreachable(t *testing.T) {
	c, err := New(Options{
		BaseURL: "http://127.0.0.1:1",
		Token:   "pdp_live_test",
		Timeout: 200 * time.Millisecond,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	if err := c.Ping(ctx); err == nil {
		t.Fatalf("Ping must fail on transport error")
	}
}
