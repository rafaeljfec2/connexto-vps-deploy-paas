package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	BasePath          = "/paas-deploy/v1"
	headerAuth        = "Authorization"
	headerContentType = "Content-Type"
	headerMCPClient   = "X-MCP-Client"
	headerTraceID     = "X-Trace-Id"
	defaultRetries    = 3
	maxBackoff        = 4 * time.Second
)

type Client struct {
	baseURL    *url.URL
	token      string
	clientID   string
	httpClient *http.Client
	logger     *slog.Logger
	maxRetries int
	requireCtx bool
}

type Options struct {
	BaseURL  string
	Token    string
	ClientID string
	Timeout  time.Duration
	Logger   *slog.Logger
	// MaxRetries caps the exponential-backoff retries on 5xx and transport errors.
	MaxRetries int
	// AcceptTokenFromContext switches the client into multi-tenant mode: callers
	// must inject a token via [WithToken] for every request, and the static
	// Token in Options is treated as a fallback (or empty if the transport
	// always supplies one).
	//
	// Used by the HTTP/SSE transport, which forwards each agent's PAT downstream.
	AcceptTokenFromContext bool
}

type ctxKey int

const (
	ctxKeyToken ctxKey = iota
	ctxKeyClientID
)

// WithToken returns a context that carries the given PAT. The HTTP transport
// uses this to forward each agent's bearer token to the backend on a
// per-request basis without mutating the shared Client.
func WithToken(ctx context.Context, token string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxKeyToken, token)
}

// WithClientID returns a context that carries the X-MCP-Client identifier,
// typically the validated value reported by the upstream agent.
func WithClientID(ctx context.Context, id string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxKeyClientID, id)
}

func tokenFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxKeyToken).(string)
	return v
}

func clientIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxKeyClientID).(string)
	return v
}

func New(opts Options) (*Client, error) {
	if opts.BaseURL == "" {
		return nil, errors.New("backend.New: BaseURL is required")
	}
	if opts.Token == "" && !opts.AcceptTokenFromContext {
		return nil, errors.New("backend.New: Token is required (or AcceptTokenFromContext must be true)")
	}
	parsed, err := url.Parse(strings.TrimRight(opts.BaseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("backend.New: invalid base URL: %w", err)
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 15 * time.Second
	}
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = defaultRetries
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.ClientID == "" {
		opts.ClientID = "custom:flowdeploy-mcp"
	}
	return &Client{
		baseURL:    parsed,
		token:      opts.Token,
		clientID:   opts.ClientID,
		httpClient: &http.Client{Timeout: opts.Timeout},
		logger:     opts.Logger,
		maxRetries: opts.MaxRetries,
		requireCtx: opts.AcceptTokenFromContext,
	}, nil
}

// Ping is a connectivity probe used by readiness handlers. It does NOT require
// a valid PAT and does not retry; the goal is to verify that the backend host
// is reachable from the MCP process. Any HTTP response (even 401/404) means
// the backend is alive — only network/transport failures are reported as
// errors.
func (c *Client) Ping(ctx context.Context) error {
	if c == nil {
		return errors.New("backend.Ping: nil client")
	}
	target := c.resolve("/health", nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return fmt.Errorf("backend.Ping: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("backend.Ping: %w", err)
	}
	_ = resp.Body.Close()
	return nil
}

type RequestOptions struct {
	Method  string
	Path    string
	Query   url.Values
	Body    any
	Headers map[string]string
	// Idempotent indicates whether the operation can be safely retried.
	// Read-only calls (GET) and idempotent mutations (PUT, DELETE that
	// match REST semantics) should set this to true. Non-idempotent
	// mutations (POST that creates resources, deploy/start/stop semantics)
	// MUST set this to false to avoid duplicating side effects on
	// transport errors or 5xx responses.
	//
	// Defaults to false when zero; callers must explicitly opt-in.
	Idempotent bool
}

type APIError struct {
	Status  int             `json:"-"`
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Details json.RawMessage `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("backend %d %s: %s", e.Status, e.Code, e.Message)
	}
	return fmt.Sprintf("backend %d: %s", e.Status, e.Message)
}

type envelope[T any] struct {
	Success bool      `json:"success"`
	Data    T         `json:"data"`
	Error   *APIError `json:"error"`
}

func Do[T any](ctx context.Context, c *Client, opts RequestOptions) (T, error) {
	var zero T
	body, err := encodeBody(opts.Body)
	if err != nil {
		return zero, err
	}

	// Idempotency policy: GET is always safe to retry. Other methods only
	// retry when the caller explicitly opts-in via Idempotent=true. This
	// prevents the transport from duplicating non-idempotent side effects
	// (POST /apps/:id/deploy, POST /workers, etc.) on a 503 mid-flight.
	maxAttempts := 1
	if opts.Method == http.MethodGet || opts.Idempotent {
		maxAttempts = c.maxRetries + 1
	}

	target := c.resolve(opts.Path, opts.Query)
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			backoff := backoffDuration(attempt)
			c.logger.Debug("backend retry", "attempt", attempt, "backoff", backoff, "method", opts.Method, "path", opts.Path)
			if err := sleepCtx(ctx, backoff); err != nil {
				return zero, err
			}
		}

		req, err := http.NewRequestWithContext(ctx, opts.Method, target, bytes.NewReader(body))
		if err != nil {
			return zero, fmt.Errorf("backend: build request: %w", err)
		}
		if err := c.applyHeaders(req, opts.Headers, len(body) > 0); err != nil {
			return zero, err
		}

		start := time.Now()
		resp, err := c.httpClient.Do(req)
		latency := time.Since(start)
		if err != nil {
			lastErr = fmt.Errorf("backend: do request: %w", err)
			c.logger.Warn("backend transport error", "method", opts.Method, "path", opts.Path, "attempt", attempt, "error", err)
			if !shouldRetryError(err) {
				return zero, lastErr
			}
			continue
		}

		result, retry, err := handleResponse[T](resp, latency, c, opts)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !retry {
			return zero, err
		}
	}
	if lastErr == nil {
		lastErr = errors.New("backend: exhausted retries")
	}
	return zero, lastErr
}

func handleResponse[T any](resp *http.Response, latency time.Duration, c *Client, opts RequestOptions) (T, bool, error) {
	defer resp.Body.Close()
	var zero T
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, true, fmt.Errorf("backend: read body: %w", err)
	}
	c.logger.Debug("backend call",
		"method", opts.Method,
		"path", opts.Path,
		"status", resp.StatusCode,
		"latency_ms", latency.Milliseconds(),
		"bytes", len(raw),
	)

	if resp.StatusCode == http.StatusNoContent {
		return zero, false, nil
	}

	if resp.StatusCode >= 500 {
		return zero, true, parseError(resp.StatusCode, raw)
	}

	if resp.StatusCode >= 400 {
		return zero, false, parseError(resp.StatusCode, raw)
	}

	if len(raw) == 0 {
		return zero, false, nil
	}

	var env envelope[T]
	if err := json.Unmarshal(raw, &env); err != nil {
		return zero, false, fmt.Errorf("backend: decode envelope: %w (body=%s)", err, truncate(string(raw), 256))
	}
	if env.Error != nil {
		env.Error.Status = resp.StatusCode
		return zero, false, env.Error
	}
	return env.Data, false, nil
}

func parseError(status int, body []byte) error {
	var env envelope[json.RawMessage]
	if err := json.Unmarshal(body, &env); err == nil && env.Error != nil {
		env.Error.Status = status
		return env.Error
	}
	return &APIError{
		Status:  status,
		Code:    fmt.Sprintf("HTTP_%d", status),
		Message: truncate(string(body), 512),
	}
}

func (c *Client) resolve(path string, query url.Values) string {
	full := *c.baseURL
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasPrefix(path, BasePath) {
		path = BasePath + path
	}
	full.Path = strings.TrimRight(full.Path, "/") + path
	if query != nil {
		full.RawQuery = query.Encode()
	}
	return full.String()
}

func (c *Client) applyHeaders(req *http.Request, extra map[string]string, hasBody bool) error {
	token := tokenFromContext(req.Context())
	// In multi-tenant mode (HTTP transport) the static c.token is NEVER used
	// as a fallback: doing so would silently leak operator credentials when a
	// caller forgets to inject the agent's PAT via WithToken. Stdio mode keeps
	// the legacy fallback so operator-driven invocations still work.
	if token == "" && !c.requireCtx {
		token = c.token
	}
	if token == "" {
		if c.requireCtx {
			return errors.New("backend: missing PAT in context (multi-tenant transport requires WithToken)")
		}
		return errors.New("backend: missing PAT (no static Token configured and no token in context)")
	}
	req.Header.Set(headerAuth, "Bearer "+token)

	clientID := clientIDFromContext(req.Context())
	if clientID == "" {
		clientID = c.clientID
	}
	if clientID != "" {
		req.Header.Set(headerMCPClient, clientID)
	}

	req.Header.Set("Accept", "application/json")
	if hasBody {
		req.Header.Set(headerContentType, "application/json")
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	return nil
}

func encodeBody(body any) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	if raw, ok := body.([]byte); ok {
		return raw, nil
	}
	return json.Marshal(body)
}

func backoffDuration(attempt int) time.Duration {
	d := time.Duration(1<<attempt) * 200 * time.Millisecond
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func shouldRetryError(err error) bool {
	return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}
