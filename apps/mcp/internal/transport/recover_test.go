package transport

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoverMiddlewareWritesErrorWhenHeadersNotSent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mw := recoverMiddleware(logger)
	h := mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(errors.New("boom"))
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/mcp", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when panic happens before headers, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "internal_error") {
		t.Fatalf("expected internal_error code in body, got %s", rec.Body.String())
	}
}

// Regression: when recoverMiddleware is nested INSIDE loggingMiddleware, a
// panic from a downstream handler must still surface as an HTTP 500 in the
// statusRecorder so Prometheus observes a 5xx and operators get an error
// log entry with latency. Before the fix, recover was outermost and the
// logging middleware never executed for panicked requests.
func TestRecoverMiddlewareNestedInLoggingObserves500(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	logging := loggingMiddleware(logger)
	recover := recoverMiddleware(logger)

	final := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(errors.New("boom"))
	})
	h := chain(final, logging, recover)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/mcp", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("logging->recover order must surface panic as 500, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "internal_error") {
		t.Fatalf("expected internal_error in body, got %s", rec.Body.String())
	}
}

func TestRecoverMiddlewareSkipsWriteHeaderForInflightStream(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mw := recoverMiddleware(logger)

	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("event: ping\ndata: 1\n\n"))
		panic(errors.New("late panic mid-SSE"))
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/mcp", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status to remain 200 after late panic, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: ping") {
		t.Fatalf("expected SSE prefix to be preserved, got %q", body)
	}
	if strings.Contains(body, "internal_error") {
		t.Fatalf("recover must NOT inject error JSON into already-sent SSE body, got %q", body)
	}
}
