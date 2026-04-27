package tools

import (
	"context"
	"encoding/json"
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
	"github.com/paasdeploy/mcp/internal/toolkit"
)

type capturedRequest struct {
	method string
	path   string
	query  string
	body   string
}

type fakeBackend struct {
	mu       sync.Mutex
	requests []capturedRequest
	body     string
	status   int
}

func (f *fakeBackend) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		f.mu.Lock()
		f.requests = append(f.requests, capturedRequest{
			method: r.Method,
			path:   r.URL.Path,
			query:  r.URL.RawQuery,
			body:   string(raw),
		})
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		status := f.status
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		body := f.body
		if body == "" {
			body = `{"success":true,"data":[],"error":null,"meta":{}}`
		}
		_, _ = w.Write([]byte(body))
	}
}

func setupServer(t *testing.T, fake *fakeBackend, register func(*mcp.Server, toolkit.Deps)) *mcp.ClientSession {
	t.Helper()
	httpSrv := httptest.NewServer(fake.handler())
	t.Cleanup(httpSrv.Close)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bk, err := backend.New(backend.Options{
		BaseURL:    httpSrv.URL,
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	if _, err := srv.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.0"}, nil)
	cs, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func callTool(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	return res
}

func extractText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("expected at least one content")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	return tc.Text
}

func TestAppsListCallsCorrectPath(t *testing.T) {
	fake := &fakeBackend{body: `{"success":true,"data":[{"id":"a","name":"x"}],"error":null,"meta":{}}`}
	cs := setupServer(t, fake, RegisterApps)
	res := callTool(t, cs, "apps_list", map[string]any{})
	if res.IsError {
		t.Fatal("expected success")
	}
	got := extractText(t, res)
	if !strings.Contains(got, `"id":"a"`) {
		t.Errorf("missing data: %s", got)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(fake.requests))
	}
	if fake.requests[0].path != "/paas-deploy/v1/apps" {
		t.Errorf("unexpected path: %s", fake.requests[0].path)
	}
}

func TestAppsGetRequiresID(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterApps)
	res := callTool(t, cs, "apps_get", map[string]any{})
	if !res.IsError {
		t.Fatal("expected error result")
	}
	got := strings.ToLower(extractText(t, res))
	if !strings.Contains(got, "id") {
		t.Errorf("expected error mentioning 'id', got %s", got)
	}
}

func TestContainersLogsForwardsTailAndSince(t *testing.T) {
	fake := &fakeBackend{body: `{"success":true,"data":"log","error":null,"meta":{}}`}
	cs := setupServer(t, fake, RegisterContainers)
	res := callTool(t, cs, "containers_logs", map[string]any{
		"id":    "abc",
		"tail":  100,
		"since": "1h",
	})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if got := fake.requests[0].path; got != "/paas-deploy/v1/containers/abc/logs" {
		t.Errorf("unexpected path: %s", got)
	}
	if !strings.Contains(fake.requests[0].query, "tail=100") {
		t.Errorf("missing tail in query: %s", fake.requests[0].query)
	}
	if !strings.Contains(fake.requests[0].query, "since=1h") {
		t.Errorf("missing since in query: %s", fake.requests[0].query)
	}
}

func TestServersListReturnsBackendData(t *testing.T) {
	fake := &fakeBackend{body: `{"success":true,"data":[{"id":"s1"}],"error":null,"meta":{}}`}
	cs := setupServer(t, fake, RegisterServers)
	res := callTool(t, cs, "servers_list", map[string]any{})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	if got := fake.requests[0].path; got != "/paas-deploy/v1/servers" {
		t.Errorf("unexpected path: %s", got)
	}
}

func TestSystemStatsCallsExpectedPath(t *testing.T) {
	fake := &fakeBackend{body: `{"success":true,"data":{"cpu":1},"error":null,"meta":{}}`}
	cs := setupServer(t, fake, RegisterSystem)
	res := callTool(t, cs, "system_stats", map[string]any{})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	if got := fake.requests[0].path; got != "/paas-deploy/v1/system/stats" {
		t.Errorf("unexpected path: %s", got)
	}
}

func TestAuditLogsForwardsFilters(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterAudit)
	res := callTool(t, cs, "audit_logs", map[string]any{
		"actor_type": "pat",
		"limit":      25,
	})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	if got := fake.requests[0].query; !strings.Contains(got, "actorType=pat") || !strings.Contains(got, "limit=25") {
		t.Errorf("missing filters in query: %s", got)
	}
}

func TestImagesListSendsServerID(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterImages)
	res := callTool(t, cs, "images_list", map[string]any{"server_id": "srv-1"})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	if got := fake.requests[0].query; !strings.Contains(got, "serverId=srv-1") {
		t.Errorf("missing serverId in query: %s", got)
	}
}

func TestTemplatesGetRequiresID(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterTemplates)
	res := callTool(t, cs, "templates_get", map[string]any{})
	if !res.IsError {
		t.Fatal("expected error result for empty id")
	}
}

func TestGitHubRepoRequiresOwnerAndRepo(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterGitHub)
	res := callTool(t, cs, "github_repo", map[string]any{"owner": "octocat"})
	if !res.IsError {
		t.Fatal("expected error result")
	}
	got := strings.ToLower(extractText(t, res))
	if !strings.Contains(got, "repo") {
		t.Errorf("expected error mentioning 'repo', got %s", got)
	}
}

func TestGitHubReposReturnsBackendPayload(t *testing.T) {
	fake := &fakeBackend{body: `{"success":true,"data":[{"name":"r"}],"error":null,"meta":{}}`}
	cs := setupServer(t, fake, RegisterGitHub)
	res := callTool(t, cs, "github_repos", map[string]any{})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	var payload struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(extractText(t, res)), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !strings.Contains(string(payload.Data), `"name":"r"`) {
		t.Errorf("unexpected payload: %s", payload.Data)
	}
}
