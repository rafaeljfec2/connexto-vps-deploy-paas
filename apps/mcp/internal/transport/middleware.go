package transport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/paasdeploy/mcp/internal/backend"
	"github.com/paasdeploy/mcp/internal/metrics"
)

const (
	tokenPrefix      = "pdp_live_"
	headerXMCPClient = "X-MCP-Client"
	headerAuth       = "Authorization"
	headerXTraceID   = "X-Trace-Id"
)

// principal carries information extracted from authenticated HTTP requests.
type principal struct {
	Token     string
	TokenHash string
	ClientID  string
	TraceID   string
}

type ctxKey int

const (
	ctxKeyPrincipal ctxKey = iota
)

// withPrincipal stores the authenticated principal in the context so downstream
// handlers (and the backend client) can read it without re-parsing headers.
func withPrincipal(ctx context.Context, p principal) context.Context {
	return context.WithValue(ctx, ctxKeyPrincipal, p)
}

func principalFromContext(ctx context.Context) (principal, bool) {
	p, ok := ctx.Value(ctxKeyPrincipal).(principal)
	return p, ok
}

// authMiddleware validates the bearer PAT, the X-MCP-Client header, and stores
// the resulting principal in the request context. It does NOT contact the
// backend (validation happens once the agent issues a tool call); the goal here
// is to fail fast on malformed credentials and tag requests with an audit hash.
func authMiddleware(allowed []string, logger *slog.Logger) func(http.Handler) http.Handler {
	matcher := compileClientMatcher(allowed)
	deny := func(w http.ResponseWriter, r *http.Request, status int, code, message string) {
		metrics.ObserveAuthFailure(code)
		logger.Warn("mcp http auth rejected",
			"method", r.Method,
			"path", r.URL.Path,
			"reason", code,
			"remote", r.RemoteAddr,
		)
		writeAuthError(w, status, code, message)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := extractBearer(r)
			if err != nil {
				deny(w, r, http.StatusUnauthorized, "missing_token", err.Error())
				return
			}
			if !strings.HasPrefix(token, tokenPrefix) {
				deny(w, r, http.StatusUnauthorized, "invalid_token_prefix", "token must start with pdp_live_")
				return
			}
			clientID := strings.TrimSpace(r.Header.Get(headerXMCPClient))
			if clientID == "" {
				deny(w, r, http.StatusBadRequest, "missing_client_header", "X-MCP-Client header is required")
				return
			}
			if !matcher(clientID) {
				deny(w, r, http.StatusForbidden, "client_not_allowed", "X-MCP-Client value is not in the allowlist")
				return
			}

			p := principal{
				Token:     token,
				TokenHash: hashToken(token),
				ClientID:  clientID,
				TraceID:   strings.TrimSpace(r.Header.Get(headerXTraceID)),
			}

			ctx := withPrincipal(r.Context(), p)
			ctx = backend.WithToken(ctx, p.Token)
			ctx = backend.WithClientID(ctx, p.ClientID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// rateLimitMiddleware enforces per-PAT quotas. It bills the bucket BEFORE the
// inner handler runs so misbehaving agents are throttled even when the
// downstream returns errors.
//
// Bucket classification is delegated to classifyRequest, which peeks the
// JSON-RPC body and matches the requested tool name against the registry of
// mutating tools maintained by the toolkit package. The X-MCP-Bucket header
// can only UPGRADE the bucket to "mutate"; it can never bypass mutate quotas
// by declaring a destructive call as "read".
func rateLimitMiddleware(limiter *RateLimiter, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := principalFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			bucket, outcome, err := classifyRequest(r)
			if outcome != ClassifyOK {
				metrics.ObserveClassifyFailure(string(outcome))
			}
			if err != nil {
				logger.Warn("mcp http bucket classify failed",
					"method", r.Method,
					"path", r.URL.Path,
					"outcome", string(outcome),
					"error", err.Error(),
				)
			}
			if !limiter.Allow(p.TokenHash, bucket) {
				metrics.ObserveRateLimitDrop(string(bucket))
				logger.Warn("mcp http rate limited",
					"method", r.Method,
					"path", r.URL.Path,
					"bucket", string(bucket),
					"token_hash", p.TokenHash,
					"client", p.ClientID,
				)
				writeAuthError(w, http.StatusTooManyRequests, "rate_limited", "per-token quota exceeded; retry later")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// recoverMiddleware traps panics to avoid leaking goroutines and stack traces.
//
// It wraps the response writer with a writeTracker so that, if the panic
// happens AFTER the inner handler has already started flushing bytes (typical
// of SSE streams used by /mcp), we do NOT attempt to overwrite the status
// line — that would either be silently dropped by the http package or result
// in a corrupted Transfer-Encoding: chunked stream. In that case we just log
// the failure and let the connection close.
func recoverMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tracker := &writeTracker{ResponseWriter: w}
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				logger.Error("mcp http panic",
					"path", r.URL.Path,
					"method", r.Method,
					"panic", rec,
					"headers_sent", tracker.headerSent,
					"stack", string(debug.Stack()),
				)
				if tracker.headerSent {
					return
				}
				writeAuthError(tracker, http.StatusInternalServerError, "internal_error", "internal error")
			}()
			next.ServeHTTP(tracker, r)
		})
	}
}

// writeTracker is a minimal http.ResponseWriter wrapper that records whether
// the status line / first byte has been emitted. It is intentionally simpler
// than statusRecorder (no byte counting, no status caching) because its only
// consumer is recoverMiddleware.
type writeTracker struct {
	http.ResponseWriter
	headerSent bool
}

func (t *writeTracker) WriteHeader(code int) {
	t.headerSent = true
	t.ResponseWriter.WriteHeader(code)
}

func (t *writeTracker) Write(p []byte) (int, error) {
	t.headerSent = true
	return t.ResponseWriter.Write(p)
}

func (t *writeTracker) Flush() {
	t.headerSent = true
	if f, ok := t.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// loggingMiddleware emits one structured slog entry per request with the
// principal's token hash (not the raw token) and observes Prometheus metrics.
//
// The path label observed by Prometheus is normalized to a closed set
// (see normalizeRoute) to keep cardinality bounded. Structured logs preserve
// the raw URL for forensic queries.
func loggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			latency := time.Since(start)

			p, _ := principalFromContext(r.Context())
			route := normalizeRoute(r.URL.Path)
			logger.Info("mcp http",
				"method", r.Method,
				"path", r.URL.Path,
				"route", route,
				"status", rec.status,
				"bytes", rec.bytes,
				"latency_ms", latency.Milliseconds(),
				"client", p.ClientID,
				"token_hash", p.TokenHash,
				"trace_id", p.TraceID,
			)
			metrics.ObserveHTTPRequest(r.Method, route, rec.status, latency.Seconds())
		})
	}
}

// normalizeRoute collapses incoming URL paths into a closed set of labels
// suitable for Prometheus. Anything outside the well-known MCP endpoints is
// reported as "other" so a malicious caller cannot inflate the time series
// dictionary by hammering /<random>.
func normalizeRoute(path string) string {
	switch path {
	case "/mcp":
		return "/mcp"
	case "/metrics":
		return "/metrics"
	case "/healthz":
		return "/healthz"
	case "/readyz":
		return "/readyz"
	}
	if strings.HasPrefix(path, "/mcp/") {
		return "/mcp"
	}
	return "other"
}

func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

func extractBearer(r *http.Request) (string, error) {
	raw := r.Header.Get(headerAuth)
	if raw == "" {
		return "", errors.New("Authorization header is required")
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(raw, prefix) {
		return "", errors.New("Authorization must use 'Bearer <token>'")
	}
	token := strings.TrimSpace(raw[len(prefix):])
	if token == "" {
		return "", errors.New("Authorization carried an empty bearer")
	}
	return token, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])[:16]
}

// compileClientMatcher returns a predicate that matches X-MCP-Client values
// against the allowlist. Trailing wildcards (e.g. "ci:*") match any non-empty
// suffix.
func compileClientMatcher(allowed []string) func(string) bool {
	exact := map[string]struct{}{}
	prefixes := []string{}
	for _, raw := range allowed {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		if strings.HasSuffix(entry, ":*") {
			prefixes = append(prefixes, strings.TrimSuffix(entry, "*"))
			continue
		}
		exact[entry] = struct{}{}
	}
	return func(value string) bool {
		if _, ok := exact[value]; ok {
			return true
		}
		for _, p := range prefixes {
			if strings.HasPrefix(value, p) && len(value) > len(p) {
				return true
			}
		}
		return false
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.wroteHeader {
		return
	}
	s.wroteHeader = true
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(p []byte) (int, error) {
	if !s.wroteHeader {
		s.wroteHeader = true
	}
	n, err := s.ResponseWriter.Write(p)
	s.bytes += n
	return n, err
}

// HeaderSent reports whether the response status line has already been emitted.
// It lets recoverMiddleware avoid corrupting in-flight SSE streams when a
// panic happens after the first chunk was already flushed to the client.
func (s *statusRecorder) HeaderSent() bool { return s.wroteHeader }

// Flush implements http.Flusher when the underlying writer supports it; the
// streamable transport relies on chunked SSE responses.
func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
