package tools

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/backend"
	"github.com/paasdeploy/mcp/internal/toolkit"
)

type stagedHandler struct {
	mu       sync.Mutex
	current  int
	stages   []string
	requests int32
}

func (s *stagedHandler) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&s.requests, 1)
		s.mu.Lock()
		idx := s.current
		if idx >= len(s.stages) {
			idx = len(s.stages) - 1
		}
		body := s.stages[idx]
		s.current++
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}
}

func setupClientWithProgress(t *testing.T, register func(*mcp.Server, toolkit.Deps), backendURL string, capture func(*mcp.ProgressNotificationClientRequest)) *mcp.ClientSession {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bk, err := backend.New(backend.Options{
		BaseURL:    backendURL,
		Token:      "pdp_live_test",
		ClientID:   "test",
		Timeout:    2 * time.Second,
		Logger:     logger,
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("backend.New: %v", err)
	}
	srv := mcp.NewServer(&mcp.Implementation{Name: "flowdeploy-test", Version: "0.0.0"}, nil)
	register(srv, toolkit.Deps{Logger: logger, Backend: bk})

	t1, t2 := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	if _, err := srv.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.0"}, &mcp.ClientOptions{
		ProgressNotificationHandler: func(_ context.Context, req *mcp.ProgressNotificationClientRequest) {
			capture(req)
		},
	})
	cs, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func TestDeployStatusWaitEmitsProgressUntilTerminal(t *testing.T) {
	staged := &stagedHandler{
		stages: []string{
			`{"success":true,"data":[{"status":"pending"}],"error":null,"meta":{}}`,
			`{"success":true,"data":[{"status":"running"}],"error":null,"meta":{}}`,
			`{"success":true,"data":[{"status":"success"}],"error":null,"meta":{}}`,
		},
	}
	httpSrv := httptest.NewServer(staged.handler())
	t.Cleanup(httpSrv.Close)

	var mu sync.Mutex
	var notifications []string
	cs := setupClientWithProgress(t, RegisterDeploys, httpSrv.URL, func(req *mcp.ProgressNotificationClientRequest) {
		mu.Lock()
		defer mu.Unlock()
		notifications = append(notifications, req.Params.Message)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "deploy_status",
		Arguments: map[string]any{
			"id":                    "app-1",
			"wait":                  true,
			"timeout_seconds":       5,
			"poll_interval_seconds": 1,
		},
		Meta: mcp.Meta{"progressToken": "tok-1"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got %s", asText(t, res))
	}

	mu.Lock()
	defer mu.Unlock()
	got := strings.Join(notifications, "|")
	if !strings.Contains(got, "pending") || !strings.Contains(got, "running") || !strings.Contains(got, "success") {
		t.Errorf("expected progress through pending/running/success, got %s", got)
	}
}

func TestContainersLogsFollowEmitsChunks(t *testing.T) {
	staged := &stagedHandler{
		stages: []string{
			`{"success":true,"data":"line-1\n","error":null,"meta":{}}`,
			`{"success":true,"data":"line-2\n","error":null,"meta":{}}`,
			`{"success":true,"data":"line-3\n","error":null,"meta":{}}`,
		},
	}
	httpSrv := httptest.NewServer(staged.handler())
	t.Cleanup(httpSrv.Close)

	var mu sync.Mutex
	var chunks []string
	cs := setupClientWithProgress(t, RegisterContainers, httpSrv.URL, func(req *mcp.ProgressNotificationClientRequest) {
		mu.Lock()
		defer mu.Unlock()
		chunks = append(chunks, req.Params.Message)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "containers_logs",
		Arguments: map[string]any{
			"id":               "c1",
			"follow":           true,
			"timeout_seconds":  3,
			"interval_seconds": 1,
		},
		Meta: mcp.Meta{"progressToken": "tok-2"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got %s", asText(t, res))
	}

	mu.Lock()
	defer mu.Unlock()
	if len(chunks) == 0 {
		t.Fatalf("expected at least one log chunk")
	}
	joined := strings.Join(chunks, "")
	if !strings.Contains(joined, "line-1") {
		t.Errorf("expected line-1 in chunks, got %v", chunks)
	}
}

func asText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		return ""
	}
	if tc, ok := res.Content[0].(*mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}
