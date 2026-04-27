// Package transport hosts the HTTP/SSE transport for the FlowDeploy MCP
// server. It wraps the SDK's StreamableHTTPHandler with FlowDeploy-specific
// concerns: bearer-PAT authentication, X-MCP-Client allowlist, per-token rate
// limiting, structured logging and Prometheus metrics.
//
// TLS termination is delegated to Traefik. The HTTP server here always speaks
// plaintext on a private interface (typically `:3001`) and Traefik fronts it
// at `mcp.flowdeploy.<domain>` via the dynamic configuration in
// `deploy/traefik/dynamic/mcp.yml`.
package transport

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/metrics"
)

// ServerOptions configure the HTTP transport.
type ServerOptions struct {
	Addr             string
	AllowedClients   []string
	ReadRPM          int
	MutateRPM        int
	SessionMaxAge    time.Duration
	StatelessSession bool
	Logger           *slog.Logger
	// MCPServer is the configured *mcp.Server with all tools/resources/prompts
	// already registered. The transport reuses the same instance across all
	// HTTP requests; authentication state is propagated via context.
	MCPServer *mcp.Server
	// ReadinessChecks are invoked by /healthz to assess downstream health.
	ReadinessChecks []ReadinessCheck
}

// ReadinessCheck is a named probe executed against /readyz.
type ReadinessCheck struct {
	Name  string
	Probe func(context.Context) error
}

// Server bundles the http.Server and supporting collaborators so callers can
// run/stop the transport from a single entry point.
type Server struct {
	http    *http.Server
	logger  *slog.Logger
	limiter *RateLimiter
}

// NewServer wires the streamable handler with all middlewares.
func NewServer(opts ServerOptions) (*Server, error) {
	if opts.MCPServer == nil {
		return nil, errors.New("transport.NewServer: MCPServer is required")
	}
	if opts.Addr == "" {
		return nil, errors.New("transport.NewServer: Addr is required")
	}
	if len(opts.AllowedClients) == 0 {
		return nil, errors.New("transport.NewServer: AllowedClients must contain at least one entry")
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.SessionMaxAge <= 0 {
		opts.SessionMaxAge = 30 * time.Minute
	}

	limiter := NewRateLimiter(opts.ReadRPM, opts.MutateRPM)

	streamable := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return opts.MCPServer
	}, &mcp.StreamableHTTPOptions{
		Stateless:      opts.StatelessSession,
		Logger:         opts.Logger,
		SessionTimeout: opts.SessionMaxAge,
	})

	// Middleware order matters:
	//   logging (outermost)  → captures status/latency for EVERY request, even
	//                          ones that panic in the inner chain
	//   recover              → traps panics; nested inside logging so the
	//                          recovered request still emits a Prometheus
	//                          observation with status 5xx
	//   auth → ratelimit → session instrumentation (innermost) → handler
	authChain := []func(http.Handler) http.Handler{
		loggingMiddleware(opts.Logger),
		recoverMiddleware(opts.Logger),
		authMiddleware(opts.AllowedClients, opts.Logger),
		rateLimitMiddleware(limiter, opts.Logger),
		sessionInstrumentation(),
	}
	mux := http.NewServeMux()
	mux.Handle("/mcp", chain(streamable, authChain...))
	mux.Handle("/mcp/", chain(streamable, authChain...))

	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/healthz", healthzHandler())
	mux.HandleFunc("/readyz", readinessHandler(opts.ReadinessChecks))

	httpSrv := &http.Server{
		Addr:              opts.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       opts.SessionMaxAge,
	}

	return &Server{http: httpSrv, logger: opts.Logger, limiter: limiter}, nil
}

// Run starts the HTTP server and blocks until ctx is cancelled or the server
// returns an error. A graceful shutdown is attempted on cancellation.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("mcp http listening", "addr", s.http.Addr)
		errCh <- s.http.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.logger.Info("mcp http shutting down")
		if err := s.http.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("mcp http shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("mcp http listen: %w", err)
	}
}

// Addr returns the listener address (useful for tests with :0).
func (s *Server) Addr() string { return s.http.Addr }

func sessionInstrumentation() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			metrics.IncInFlightRequests()
			defer metrics.DecInFlightRequests()
			next.ServeHTTP(w, r)
		})
	}
}

// healthzHandler returns the liveness probe. It only accepts GET/HEAD so
// arbitrary verbs (PUT, DELETE, OPTIONS) cannot be used to fingerprint or
// probe the binary. Liveness is intentionally cheap: returning 200 means the
// process is alive; readiness/dependency status lives in /readyz.
func healthzHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}
	}
}

func readinessHandler(checks []ReadinessCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		failures := map[string]string{}
		for _, c := range checks {
			if c.Probe == nil {
				continue
			}
			if err := c.Probe(ctx); err != nil {
				failures[c.Name] = err.Error()
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if len(failures) > 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprintf(w, `{"status":"degraded","failures":%q}`, formatFailures(failures))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	}
}

func formatFailures(failures map[string]string) string {
	if len(failures) == 0 {
		return ""
	}
	out := ""
	for name, msg := range failures {
		if out != "" {
			out += ";"
		}
		out += name + "=" + msg
	}
	return out
}
