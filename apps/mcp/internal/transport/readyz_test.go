package transport

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadinessHandlerReportsOKWhenAllProbesPass(t *testing.T) {
	h := readinessHandler([]ReadinessCheck{
		{Name: "backend", Probe: func(context.Context) error { return nil }},
	})
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 when all probes ok, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"status":"ready"`) {
		t.Fatalf("expected ready payload, got %q", body)
	}
}

func TestReadinessHandlerReportsDegradedWhenProbeFails(t *testing.T) {
	h := readinessHandler([]ReadinessCheck{
		{Name: "backend", Probe: func(context.Context) error { return errors.New("dial tcp: refused") }},
		{Name: "docker", Probe: func(context.Context) error { return nil }},
	})
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when a probe fails, got %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "degraded") {
		t.Fatalf("expected degraded payload, got %q", string(body))
	}
	if !strings.Contains(string(body), "backend=") {
		t.Fatalf("expected probe name in failures payload, got %q", string(body))
	}
}

func TestReadinessHandlerSkipsNilProbes(t *testing.T) {
	h := readinessHandler([]ReadinessCheck{
		{Name: "noop", Probe: nil},
	})
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 when only nil probes are provided, got %d", rec.Code)
	}
}

func TestReadinessHandlerRejectsNonGETMethods(t *testing.T) {
	h := readinessHandler(nil)
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(method, "/readyz", nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405 for %s /readyz, got %d", method, rec.Code)
		}
	}
}

func TestHealthzHandlerOnlyAllowsGETHEAD(t *testing.T) {
	h := healthzHandler()
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET /healthz, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("expected ok payload, got %q", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodHead, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for HEAD /healthz, got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD must not return a body, got %q", rec.Body.String())
	}

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions} {
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(method, "/healthz", nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405 for %s /healthz, got %d", method, rec.Code)
		}
		if rec.Header().Get("Allow") != "GET, HEAD" {
			t.Fatalf("expected Allow header on 405, got %q", rec.Header().Get("Allow"))
		}
	}
}
