package mcpserver_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/backend"
	"github.com/paasdeploy/mcp/internal/mcpserver"
	"github.com/paasdeploy/mcp/internal/toolkit"
)

type recordedRequest struct {
	method string
	path   string
}

type fakeFlowBackend struct {
	mu             sync.Mutex
	requests       []recordedRequest
	deployStages   []string
	deployIndex    int
	restartCalls   int
	containerCalls int
}

func newFakeFlowBackend() *fakeFlowBackend {
	return &fakeFlowBackend{
		deployStages: []string{
			`{"success":true,"data":[{"id":"d-1","status":"running"}],"error":null,"meta":{}}`,
			`{"success":true,"data":[{"id":"d-1","status":"success"}],"error":null,"meta":{}}`,
		},
	}
}

func (f *fakeFlowBackend) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.requests = append(f.requests, recordedRequest{method: r.Method, path: r.URL.Path})
		f.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/redeploy"):
			_, _ = w.Write([]byte(`{"success":true,"data":{"deploymentId":"d-1"},"error":null,"meta":{}}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/deployments"):
			f.mu.Lock()
			idx := f.deployIndex
			if idx >= len(f.deployStages) {
				idx = len(f.deployStages) - 1
			}
			body := f.deployStages[idx]
			f.deployIndex++
			f.mu.Unlock()
			_, _ = w.Write([]byte(body))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/restart"):
			f.mu.Lock()
			f.restartCalls++
			f.mu.Unlock()
			_, _ = w.Write([]byte(`{"success":true,"data":{"ok":true},"error":null,"meta":{}}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/containers/"):
			f.mu.Lock()
			f.containerCalls++
			f.mu.Unlock()
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":"c1","state":"running"},"error":null,"meta":{}}`))
		default:
			_, _ = w.Write([]byte(`{"success":true,"data":null,"error":null,"meta":{}}`))
		}
	}
}

func TestIntegrationDeployAndContainerRestart(t *testing.T) {
	fake := newFakeFlowBackend()
	httpSrv := httptest.NewServer(fake.handler())
	t.Cleanup(httpSrv.Close)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bk, err := backend.New(backend.Options{
		BaseURL:    httpSrv.URL,
		Token:      "pdp_live_test",
		ClientID:   "test",
		Timeout:    3 * time.Second,
		Logger:     logger,
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("backend.New: %v", err)
	}
	srv, err := mcpserver.New(mcpserver.Deps{Logger: logger, Backend: bk})
	if err != nil {
		t.Fatalf("mcpserver.New: %v", err)
	}
	deps := toolkit.Deps{Logger: logger, Backend: bk}
	mcpserver.RegisterAllReadOnly(srv, deps)
	mcpserver.RegisterAllWrites(srv, deps)

	t1, t2 := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	if _, err := srv.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("server.Connect: %v", err)
	}

	var progressMu sync.Mutex
	var progressMessages []string
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, &mcp.ClientOptions{
		ProgressNotificationHandler: func(_ context.Context, req *mcp.ProgressNotificationClientRequest) {
			progressMu.Lock()
			defer progressMu.Unlock()
			progressMessages = append(progressMessages, req.Params.Message)
		},
	})
	cs, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })

	triggerRes, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "deploy_trigger",
		Arguments: map[string]any{
			"id":         "app-1",
			"commit_sha": "abc",
		},
	})
	if err != nil {
		t.Fatalf("deploy_trigger: %v", err)
	}
	if triggerRes.IsError {
		t.Fatalf("deploy_trigger error: %+v", triggerRes)
	}

	statusRes, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "deploy_status",
		Arguments: map[string]any{
			"id":                    "app-1",
			"wait":                  true,
			"timeout_seconds":       6,
			"poll_interval_seconds": 1,
		},
		Meta: mcp.Meta{"progressToken": "deploy-1"},
	})
	if err != nil {
		t.Fatalf("deploy_status: %v", err)
	}
	if statusRes.IsError {
		t.Fatalf("deploy_status error: %+v", statusRes)
	}

	restartRes, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "containers_restart",
		Arguments: map[string]any{"id": "c1"},
	})
	if err != nil {
		t.Fatalf("containers_restart: %v", err)
	}
	if restartRes.IsError {
		t.Fatalf("containers_restart error: %+v", restartRes)
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.restartCalls != 1 {
		t.Errorf("expected 1 restart call, got %d", fake.restartCalls)
	}

	var sawRedeploy, sawDeployments, sawRestart bool
	for _, req := range fake.requests {
		switch {
		case req.method == http.MethodPost && strings.HasSuffix(req.path, "/redeploy"):
			sawRedeploy = true
		case req.method == http.MethodGet && strings.HasSuffix(req.path, "/deployments"):
			sawDeployments = true
		case req.method == http.MethodPost && strings.HasSuffix(req.path, "/containers/c1/restart"):
			sawRestart = true
		}
	}
	if !sawRedeploy {
		t.Error("expected redeploy POST to be recorded")
	}
	if !sawDeployments {
		t.Error("expected deployments GET to be recorded")
	}
	if !sawRestart {
		t.Error("expected container restart POST to be recorded")
	}

	progressMu.Lock()
	defer progressMu.Unlock()
	joined := strings.Join(progressMessages, "|")
	if !strings.Contains(joined, "running") {
		t.Errorf("expected running progress, got %s", joined)
	}
	if !strings.Contains(joined, "success") {
		t.Errorf("expected success progress, got %s", joined)
	}
}
