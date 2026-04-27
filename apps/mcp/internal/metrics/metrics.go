// Package metrics centralizes Prometheus collectors used across the MCP server.
//
// Counters and histograms are registered against a private registry so the same
// process can expose `/metrics` via [Handler] without leaking unrelated
// collectors from third-party libraries. The defaults are SAFE in stdio mode
// (no HTTP exposure) and become observable when the HTTP transport is enabled
// through `flowdeploy-mcp serve`.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "mcp"

var (
	registry *prometheus.Registry

	toolCalls        *prometheus.CounterVec
	toolDuration     *prometheus.HistogramVec
	httpRequests     *prometheus.CounterVec
	httpDuration     *prometheus.HistogramVec
	rateLimitDrops   *prometheus.CounterVec
	authFailures     *prometheus.CounterVec
	classifyFailures *prometheus.CounterVec
	inFlightRequests prometheus.Gauge
)

func init() {
	registry = prometheus.NewRegistry()

	toolCalls = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "tool_calls_total",
		Help:      "Total number of MCP tool invocations grouped by tool, mode (read/write/destructive) and status (ok/error).",
	}, []string{"tool", "mode", "status"})

	toolDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "tool_duration_seconds",
		Help:      "Wall-clock latency of MCP tool invocations.",
		Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
	}, []string{"tool", "mode"})

	httpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "http_requests_total",
		Help:      "Total number of HTTP requests handled by the MCP transport.",
	}, []string{"method", "path", "status"})

	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "http_request_duration_seconds",
		Help:      "Latency of HTTP requests handled by the MCP transport.",
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	}, []string{"method", "path"})

	rateLimitDrops = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "ratelimit_drops_total",
		Help:      "HTTP requests rejected by per-token rate limiting, grouped by quota bucket (read/mutate).",
	}, []string{"bucket"})

	authFailures = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "auth_failures_total",
		Help:      "HTTP requests rejected during authentication or client validation.",
	}, []string{"reason"})

	classifyFailures = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "bucket_classify_failures_total",
		Help:      "HTTP requests for which classifyRequest could not parse the body and degraded conservatively to mutate.",
	}, []string{"reason"})

	inFlightRequests = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "in_flight_requests",
		Help:      "Number of HTTP requests currently being processed by the MCP transport.",
	})

	registry.MustRegister(toolCalls, toolDuration, httpRequests, httpDuration, rateLimitDrops, authFailures, classifyFailures, inFlightRequests)
}

// ObserveToolCall records a tool invocation outcome and its latency.
func ObserveToolCall(tool, mode string, ok bool, durationSeconds float64) {
	status := "ok"
	if !ok {
		status = "error"
	}
	toolCalls.WithLabelValues(tool, mode, status).Inc()
	toolDuration.WithLabelValues(tool, mode).Observe(durationSeconds)
}

// ObserveHTTPRequest records an HTTP request outcome and its latency.
//
// IMPORTANT: callers MUST pass a normalized path label drawn from a closed
// set (e.g. "/mcp", "/metrics", "/healthz", "/readyz", "other"). Passing the
// raw r.URL.Path would create unbounded cardinality if the server is exposed
// to arbitrary URLs. The transport layer normalizes paths before calling in.
func ObserveHTTPRequest(method, path string, status int, durationSeconds float64) {
	statusLabel := httpStatusLabel(status)
	httpRequests.WithLabelValues(method, path, statusLabel).Inc()
	httpDuration.WithLabelValues(method, path).Observe(durationSeconds)
}

// ObserveRateLimitDrop records a request rejected by the rate limiter.
func ObserveRateLimitDrop(bucket string) {
	rateLimitDrops.WithLabelValues(bucket).Inc()
}

// ObserveAuthFailure records a request rejected by auth/X-MCP-Client validation.
func ObserveAuthFailure(reason string) {
	authFailures.WithLabelValues(reason).Inc()
}

// ObserveClassifyFailure records a request whose body could not be parsed by
// the bucket classifier. The reason label MUST come from a closed set
// ("read_error", "parse_error", "overflow") so cardinality stays bounded.
func ObserveClassifyFailure(reason string) {
	classifyFailures.WithLabelValues(reason).Inc()
}

// IncInFlightRequests / DecInFlightRequests track concurrent HTTP requests.
// They are paired by the sessionInstrumentation middleware for every request
// served through the streamable handler.
func IncInFlightRequests() { inFlightRequests.Inc() }
func DecInFlightRequests() { inFlightRequests.Dec() }

// Handler returns the http.Handler that serves the metrics in the standard
// Prometheus exposition format. Uses our own registry to avoid leaking
// unrelated collectors registered via prometheus.DefaultRegisterer.
func Handler() http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{Registry: registry})
}

// Registry exposes the underlying Prometheus registry for advanced wiring or
// testing. Production code should prefer [Handler].
func Registry() *prometheus.Registry {
	return registry
}

func httpStatusLabel(status int) string {
	switch {
	case status >= 500:
		return "5xx"
	case status >= 400:
		return "4xx"
	case status >= 300:
		return "3xx"
	case status >= 200:
		return "2xx"
	default:
		return "1xx"
	}
}
