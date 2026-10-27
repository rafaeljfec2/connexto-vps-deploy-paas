package health

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCheckWithBackoff_TimeoutIncludesLastErrorAndDoesNotSleepPastDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	checker := NewChecker(time.Hour, 10, time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	checker.SetTimeout(150 * time.Millisecond)
	checker.SetInterval(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	started := time.Now()
	err := checker.CheckWithBackoff(ctx, server.URL)
	elapsed := time.Since(started)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "unhealthy status code: 503") {
		t.Fatalf("timeout hid the last probe error: %v", err)
	}
	if elapsed > time.Second {
		t.Fatalf("slept past the health deadline: %s", elapsed)
	}
}

func TestCheckUntil_ReturnsBeforeAttemptErrorWithoutProbing(t *testing.T) {
	probes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		probes++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	checker := NewChecker(time.Minute, 3, time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	err := checker.CheckUntil(context.Background(), time.Now().Add(time.Minute), CheckHooks{
		BeforeAttempt: func(context.Context) error {
			return context.Canceled
		},
		ResolveURL: func(context.Context) (string, error) {
			return server.URL, nil
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected before-attempt error, got %v", err)
	}
	if probes != 0 {
		t.Fatalf("probed %d times after a fatal container state", probes)
	}
}

func TestCheckUntil_SucceedsOnFirstHealthyResponse(t *testing.T) {
	checker := NewChecker(time.Minute, 3, time.Hour, slog.New(slog.NewTextHandler(io.Discard, nil)))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	started := time.Now()
	err := checker.CheckUntil(context.Background(), time.Now().Add(time.Minute), CheckHooks{
		ResolveURL: func(context.Context) (string, error) {
			return server.URL, nil
		},
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("healthy response waited for the backoff interval")
	}
}
