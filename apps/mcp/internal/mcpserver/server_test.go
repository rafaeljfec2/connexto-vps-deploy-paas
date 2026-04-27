package mcpserver_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/backend"
	"github.com/paasdeploy/mcp/internal/mcpserver"
	"github.com/paasdeploy/mcp/internal/toolkit"
)

func TestStdioIntegrationListsToolsAndCallsAppsList(t *testing.T) {
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"data":[{"id":"app-1","name":"my-app"}],"error":null,"meta":{}}`))
	}))
	defer httpSrv.Close()

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
	srv, err := mcpserver.New(mcpserver.Deps{Logger: logger, Backend: bk})
	if err != nil {
		t.Fatalf("mcpserver.New: %v", err)
	}
	mcpserver.RegisterAllReadOnly(srv, toolkit.Deps{Logger: logger, Backend: bk})

	t1, t2 := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := srv.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("server.Connect: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	cs, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer cs.Close()

	toolNames := map[string]bool{}
	for tool, err := range cs.Tools(ctx, nil) {
		if err != nil {
			t.Fatalf("Tools: %v", err)
		}
		toolNames[tool.Name] = true
	}
	expected := []string{"apps_list", "containers_list", "servers_list", "system_stats", "audit_logs", "templates_list"}
	for _, name := range expected {
		if !toolNames[name] {
			t.Errorf("expected tool %s to be registered", name)
		}
	}

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "apps_list",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected non-error tool result, got %+v", res)
	}
	if len(res.Content) == 0 {
		t.Fatal("expected non-empty Content")
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	var payload struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(text.Text), &payload); err != nil {
		t.Fatalf("unmarshal text content: %v", err)
	}
	if len(payload.Data) != 1 || payload.Data[0]["id"] != "app-1" {
		t.Errorf("unexpected payload: %s", text.Text)
	}
}

func TestStdioIntegrationListsResourcesAndPrompts(t *testing.T) {
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"data":[],"error":null,"meta":{}}`))
	}))
	defer httpSrv.Close()

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
	srv, err := mcpserver.New(mcpserver.Deps{Logger: logger, Backend: bk})
	if err != nil {
		t.Fatalf("mcpserver.New: %v", err)
	}
	mcpserver.RegisterAllReadOnly(srv, toolkit.Deps{Logger: logger, Backend: bk})

	t1, t2 := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := srv.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	cs, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer cs.Close()

	resourceURIs := map[string]bool{}
	for r, err := range cs.Resources(ctx, nil) {
		if err != nil {
			t.Fatalf("Resources: %v", err)
		}
		resourceURIs[r.URI] = true
	}
	if !resourceURIs["flowdeploy://apps"] {
		t.Errorf("expected flowdeploy://apps resource")
	}
	if !resourceURIs["flowdeploy://system"] {
		t.Errorf("expected flowdeploy://system resource")
	}

	promptNames := map[string]bool{}
	for p, err := range cs.Prompts(ctx, nil) {
		if err != nil {
			t.Fatalf("Prompts: %v", err)
		}
		promptNames[p.Name] = true
	}
	if !promptNames["diagnose_app"] {
		t.Errorf("expected diagnose_app prompt")
	}
	if !promptNames["audit_recent_changes"] {
		t.Errorf("expected audit_recent_changes prompt")
	}
}
