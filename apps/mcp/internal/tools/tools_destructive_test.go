package tools

import (
	"net/http"
	"strings"
	"testing"
)

func TestDestructiveDryRunIsDefault(t *testing.T) {
	fake := &fakeBackend{body: `{"success":true,"data":{"action":"apps.delete","resource":"app"},"error":null,"meta":{"warnings":["dry-run: no mutation performed"]}}`}
	cs := setupServer(t, fake, RegisterDestructive)

	res := callTool(t, cs, "apps_delete", map[string]any{"id": "app-1"})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(fake.requests))
	}
	req := fake.requests[0]
	if req.method != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", req.method)
	}
	if req.path != "/paas-deploy/v1/apps/app-1" {
		t.Errorf("unexpected path: %s", req.path)
	}
	if got := req.headers["X-Dry-Run"]; got != "true" {
		t.Errorf("expected X-Dry-Run=true, got %q", got)
	}
	if reason, ok := req.headers["X-Action-Reason"]; ok && reason != "" {
		t.Errorf("dry-run must not send X-Action-Reason, got %q", reason)
	}
}

func TestDestructiveCommitWithoutReasonRejectsLocally(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterDestructive)

	res := callTool(t, cs, "apps_delete", map[string]any{
		"id":     "app-1",
		"commit": true,
	})
	if !res.IsError {
		t.Fatalf("expected error, got success: %s", extractText(t, res))
	}
	if got := extractText(t, res); !strings.Contains(got, "reason must be between") {
		t.Errorf("expected reason validation error, got %q", got)
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.requests) != 0 {
		t.Errorf("backend must not be called when commit lacks reason, got %d requests", len(fake.requests))
	}
}

func TestDestructiveCommitWithReasonSendsHeader(t *testing.T) {
	fake := &fakeBackend{body: `{"success":true,"data":null,"error":null,"meta":{}}`}
	cs := setupServer(t, fake, RegisterDestructive)

	res := callTool(t, cs, "apps_delete", map[string]any{
		"id":     "app-1",
		"commit": true,
		"reason": "manual cleanup before maintenance window",
	})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()
	req := fake.requests[0]
	if got := req.headers["X-Action-Reason"]; got != "manual cleanup before maintenance window" {
		t.Errorf("expected reason header forwarded, got %q", got)
	}
	if got := req.headers["X-Dry-Run"]; got == "true" {
		t.Errorf("commit must not send X-Dry-Run=true, got %q", got)
	}
}

func TestContainersRemoveDryRunUsesQueryAndHeaders(t *testing.T) {
	fake := &fakeBackend{body: `{"success":true,"data":{},"error":null,"meta":{}}`}
	cs := setupServer(t, fake, RegisterDestructive)

	res := callTool(t, cs, "containers_remove", map[string]any{
		"id":        "c1",
		"force":     true,
		"server_id": "srv-9",
	})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()
	req := fake.requests[0]
	if !strings.Contains(req.query, "force=true") {
		t.Errorf("expected force=true in query, got %s", req.query)
	}
	if !strings.Contains(req.query, "serverId=srv-9") {
		t.Errorf("expected serverId in query, got %s", req.query)
	}
	if got := req.headers["X-Dry-Run"]; got != "true" {
		t.Errorf("expected X-Dry-Run=true, got %q", got)
	}
}

func TestImagesPrunePostsToPruneEndpoint(t *testing.T) {
	fake := &fakeBackend{body: `{"success":true,"data":{},"error":null,"meta":{}}`}
	cs := setupServer(t, fake, RegisterDestructive)

	res := callTool(t, cs, "images_prune", map[string]any{
		"server_id": "srv-1",
		"commit":    true,
		"reason":    "freeing 2GB before next deploy",
	})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()
	req := fake.requests[0]
	if req.method != http.MethodPost {
		t.Errorf("expected POST, got %s", req.method)
	}
	if req.path != "/paas-deploy/v1/images/prune" {
		t.Errorf("unexpected path: %s", req.path)
	}
	if !strings.Contains(req.query, "serverId=srv-1") {
		t.Errorf("expected serverId in query, got %s", req.query)
	}
	if got := req.headers["X-Action-Reason"]; got != "freeing 2GB before next deploy" {
		t.Errorf("expected reason header, got %q", got)
	}
}

func TestServersDeleteRequiresID(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterDestructive)

	res := callTool(t, cs, "servers_delete", map[string]any{})
	if !res.IsError {
		t.Fatal("expected error for missing id")
	}
}

func TestEnvDeleteRequiresVarID(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterDestructive)

	res := callTool(t, cs, "env_delete", map[string]any{
		"app_id": "app-1",
	})
	if !res.IsError {
		t.Fatal("expected error for missing var_id")
	}
}

func TestDomainsRemoveCallsDeleteEndpoint(t *testing.T) {
	fake := &fakeBackend{body: `{"success":true,"data":null,"error":null,"meta":{}}`}
	cs := setupServer(t, fake, RegisterDestructive)

	res := callTool(t, cs, "domains_remove", map[string]any{
		"app_id":    "app-1",
		"domain_id": "dom-9",
	})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()
	req := fake.requests[0]
	if req.method != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", req.method)
	}
	if req.path != "/paas-deploy/v1/apps/app-1/domains/dom-9" {
		t.Errorf("unexpected path: %s", req.path)
	}
	if got := req.headers["X-Dry-Run"]; got != "true" {
		t.Errorf("expected X-Dry-Run=true (default), got %q", got)
	}
}

func TestNetworksRemoveAndVolumesRemove(t *testing.T) {
	fake := &fakeBackend{body: `{"success":true,"data":null,"error":null,"meta":{}}`}
	cs := setupServer(t, fake, RegisterDestructive)

	res := callTool(t, cs, "networks_remove", map[string]any{"name": "paasdeploy"})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	res = callTool(t, cs, "volumes_remove", map[string]any{"name": "v1"})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(fake.requests))
	}
	if fake.requests[0].path != "/paas-deploy/v1/networks/paasdeploy" {
		t.Errorf("unexpected network path: %s", fake.requests[0].path)
	}
	if fake.requests[1].path != "/paas-deploy/v1/volumes/v1" {
		t.Errorf("unexpected volume path: %s", fake.requests[1].path)
	}
}
