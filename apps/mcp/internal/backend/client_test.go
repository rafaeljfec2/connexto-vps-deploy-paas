package backend

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c, err := New(Options{
		BaseURL:    srv.URL,
		Token:      "pdp_live_test_token",
		ClientID:   "test",
		Timeout:    2 * time.Second,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func writeEnvelope(t *testing.T, w http.ResponseWriter, status int, data any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	body := map[string]any{"success": status < 400, "data": data, "error": nil, "meta": map[string]any{}}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Fatalf("encode: %v", err)
	}
}

func TestClientInjectsAuthHeader(t *testing.T) {
	var gotAuth, gotClient string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotClient = r.Header.Get("X-MCP-Client")
		writeEnvelope(t, w, http.StatusOK, map[string]string{"hello": "world"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	data, err := Do[json.RawMessage](context.Background(), c, RequestOptions{
		Method: http.MethodGet,
		Path:   "/apps",
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if !strings.Contains(string(data), "world") {
		t.Errorf("unexpected data: %s", data)
	}
	if gotAuth != "Bearer pdp_live_test_token" {
		t.Errorf("missing/incorrect Authorization: %q", gotAuth)
	}
	if gotClient != "test" {
		t.Errorf("missing/incorrect X-MCP-Client: %q", gotClient)
	}
}

func TestClientResolvesBasePath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeEnvelope(t, w, http.StatusOK, nil)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if _, err := Do[json.RawMessage](context.Background(), c, RequestOptions{
		Method: http.MethodGet,
		Path:   "/apps/123",
	}); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if gotPath != "/paas-deploy/v1/apps/123" {
		t.Errorf("expected base path prepended, got %q", gotPath)
	}
}

func TestClientRetriesOn5xx(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 2 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		writeEnvelope(t, w, http.StatusOK, "ok")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if _, err := Do[json.RawMessage](context.Background(), c, RequestOptions{
		Method: http.MethodGet,
		Path:   "/system/stats",
	}); err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if got := attempts.Load(); got != 2 {
		t.Errorf("expected 2 attempts, got %d", got)
	}
}

func TestClientDoesNotRetryOn4xx(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"FORBIDDEN","message":"nope"}}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := Do[json.RawMessage](context.Background(), c, RequestOptions{
		Method: http.MethodGet,
		Path:   "/apps",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Status != http.StatusForbidden || apiErr.Code != "FORBIDDEN" {
		t.Errorf("unexpected error: %+v", apiErr)
	}
	if got := attempts.Load(); got != 1 {
		t.Errorf("expected single attempt, got %d", got)
	}
}

func TestClientHandlesNoContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if _, err := Do[json.RawMessage](context.Background(), c, RequestOptions{
		Method: http.MethodDelete,
		Path:   "/apps/abc",
	}); err != nil {
		t.Fatalf("expected nil error on 204, got %v", err)
	}
}

func TestClientForwardsQueryString(t *testing.T) {
	var gotRaw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRaw = r.URL.RawQuery
		writeEnvelope(t, w, http.StatusOK, nil)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	q := newQuery("limit", "5", "actor", "pat")
	if _, err := Do[json.RawMessage](context.Background(), c, RequestOptions{
		Method: http.MethodGet,
		Path:   "/audit/logs",
		Query:  q,
	}); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if !strings.Contains(gotRaw, "limit=5") || !strings.Contains(gotRaw, "actor=pat") {
		t.Errorf("unexpected query: %s", gotRaw)
	}
}
