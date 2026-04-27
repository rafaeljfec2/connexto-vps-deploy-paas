package transport

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newAuthHandler(t *testing.T, allowed []string) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mw := authMiddleware(allowed, logger)
	return mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := principalFromContext(r.Context())
		if !ok {
			t.Fatalf("expected principal to be present")
		}
		w.Header().Set("X-Principal-Client", p.ClientID)
		w.Header().Set("X-Principal-Hash", p.TokenHash)
		w.WriteHeader(http.StatusNoContent)
	}))
}

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	h := newAuthHandler(t, []string{"cursor"})
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set(headerXMCPClient, "cursor")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddlewareRejectsInvalidPrefix(t *testing.T) {
	h := newAuthHandler(t, []string{"cursor"})
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set(headerAuth, "Bearer not_a_pat")
	req.Header.Set(headerXMCPClient, "cursor")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invalid_token_prefix") {
		t.Fatalf("expected invalid_token_prefix code, got %s", rec.Body.String())
	}
}

func TestAuthMiddlewareRequiresXMCPClient(t *testing.T) {
	h := newAuthHandler(t, []string{"cursor"})
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set(headerAuth, "Bearer pdp_live_abc")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "missing_client_header") {
		t.Fatalf("expected missing_client_header code, got %s", rec.Body.String())
	}
}

func TestAuthMiddlewareRejectsClientNotAllowed(t *testing.T) {
	h := newAuthHandler(t, []string{"cursor"})
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set(headerAuth, "Bearer pdp_live_abc")
	req.Header.Set(headerXMCPClient, "evil-bot")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestAuthMiddlewareAcceptsExactAndWildcardClients(t *testing.T) {
	cases := []string{"cursor", "claude-desktop", "ci:github-actions", "custom:flowdeploy-mcp"}
	h := newAuthHandler(t, []string{"cursor", "claude-desktop", "ci:*", "custom:*"})
	for _, client := range cases {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.Header.Set(headerAuth, "Bearer pdp_live_abc")
		req.Header.Set(headerXMCPClient, client)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204 for client %q, got %d (%s)", client, rec.Code, rec.Body.String())
		}
	}
}

func TestAuthMiddlewareRejectsBareWildcardPrefix(t *testing.T) {
	h := newAuthHandler(t, []string{"ci:*"})
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set(headerAuth, "Bearer pdp_live_abc")
	req.Header.Set(headerXMCPClient, "ci:")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for empty wildcard suffix, got %d", rec.Code)
	}
}

func TestRateLimitMiddlewareDeniesAfterQuota(t *testing.T) {
	limiter := NewRateLimiter(2, 1)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rl := rateLimitMiddleware(limiter, logger)
	auth := authMiddleware([]string{"cursor"}, logger)

	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := chain(final, auth, rl)

	doRequest := func(t *testing.T) int {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.Header.Set(headerAuth, "Bearer pdp_live_token-1")
		req.Header.Set(headerXMCPClient, "cursor")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	if got := doRequest(t); got != http.StatusOK {
		t.Fatalf("expected 200 first call, got %d", got)
	}
	if got := doRequest(t); got != http.StatusOK {
		t.Fatalf("expected 200 second call, got %d", got)
	}
	if got := doRequest(t); got != http.StatusTooManyRequests {
		t.Fatalf("expected 429 third call, got %d", got)
	}
}

func TestRateLimitMiddlewareIsolatesPATHashes(t *testing.T) {
	limiter := NewRateLimiter(1, 1)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := chain(final,
		authMiddleware([]string{"cursor"}, logger),
		rateLimitMiddleware(limiter, logger),
	)
	mk := func(token string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.Header.Set(headerAuth, "Bearer "+token)
		req.Header.Set(headerXMCPClient, "cursor")
		return req
	}

	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, mk("pdp_live_alpha"))
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, mk("pdp_live_beta"))
	if rec1.Code != http.StatusOK || rec2.Code != http.StatusOK {
		t.Fatalf("limiter must isolate distinct PAT hashes; got %d/%d", rec1.Code, rec2.Code)
	}
}

func TestRateLimitMiddlewareUsesMutateBucketWhenSignaled(t *testing.T) {
	limiter := NewRateLimiter(100, 1)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := chain(final,
		authMiddleware([]string{"cursor"}, logger),
		rateLimitMiddleware(limiter, logger),
	)
	build := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.Header.Set(headerAuth, "Bearer pdp_live_token")
		req.Header.Set(headerXMCPClient, "cursor")
		req.Header.Set("X-MCP-Bucket", "mutate")
		return req
	}

	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, build())
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, build())
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200 first mutate call, got %d", rec1.Code)
	}
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 second mutate call (mutate quota=1), got %d", rec2.Code)
	}
}

func TestExtractBearerErrors(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		if _, err := extractBearer(req); err == nil {
			t.Fatalf("expected error for missing header")
		}
	})
	t.Run("wrong scheme", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.Header.Set(headerAuth, "Basic abc")
		if _, err := extractBearer(req); err == nil {
			t.Fatalf("expected error for non-bearer scheme")
		}
	})
	t.Run("empty bearer", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.Header.Set(headerAuth, "Bearer    ")
		if _, err := extractBearer(req); err == nil {
			t.Fatalf("expected error for empty bearer")
		}
	})
}

func TestHashTokenIsDeterministicAndShort(t *testing.T) {
	a := hashToken("pdp_live_abc")
	b := hashToken("pdp_live_abc")
	c := hashToken("pdp_live_xyz")
	if a != b {
		t.Fatalf("hash must be deterministic")
	}
	if a == c {
		t.Fatalf("hash collisions are not expected for distinct inputs")
	}
	if len(a) != 16 {
		t.Fatalf("expected 16-char prefix, got %d", len(a))
	}
}
