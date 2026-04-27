package tools

import (
	"net/http"
	"strings"
	"testing"
)

func TestDeployTriggerSendsCommitSHA(t *testing.T) {
	fake := &fakeBackend{body: `{"success":true,"data":{"deploymentId":"d1"},"error":null,"meta":{}}`}
	cs := setupServer(t, fake, RegisterDeploys)
	res := callTool(t, cs, "deploy_trigger", map[string]any{
		"id":         "app-1",
		"commit_sha": "abc123",
	})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if got := fake.requests[0].method; got != http.MethodPost {
		t.Errorf("expected POST, got %s", got)
	}
	if got := fake.requests[0].path; got != "/paas-deploy/v1/apps/app-1/redeploy" {
		t.Errorf("unexpected path: %s", got)
	}
	if !strings.Contains(fake.requests[0].body, `"commitSha":"abc123"`) {
		t.Errorf("missing commitSha in body: %s", fake.requests[0].body)
	}
}

func TestDeployRollbackRequiresID(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterDeploys)
	res := callTool(t, cs, "deploy_rollback", map[string]any{})
	if !res.IsError {
		t.Fatal("expected error for missing id")
	}
}

func TestDeployRollbackPostsToRollbackEndpoint(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterDeploys)
	res := callTool(t, cs, "deploy_rollback", map[string]any{"id": "app-1"})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	req := fake.requests[0]
	if req.method != http.MethodPost {
		t.Errorf("expected POST, got %s", req.method)
	}
	if req.path != "/paas-deploy/v1/apps/app-1/rollback" {
		t.Errorf("unexpected path: %s", req.path)
	}
}

func TestContainersStartCallsCorrectEndpoint(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterContainers)
	res := callTool(t, cs, "containers_start", map[string]any{"id": "c1", "server_id": "srv-9"})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	req := fake.requests[0]
	if req.method != http.MethodPost {
		t.Errorf("expected POST, got %s", req.method)
	}
	if req.path != "/paas-deploy/v1/containers/c1/start" {
		t.Errorf("unexpected path: %s", req.path)
	}
	if !strings.Contains(req.query, "serverId=srv-9") {
		t.Errorf("missing serverId in query: %s", req.query)
	}
}

func TestContainersHealthcheckPostsToHealthcheck(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterContainers)
	res := callTool(t, cs, "containers_healthcheck", map[string]any{"id": "c1"})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if got := fake.requests[0].path; got != "/paas-deploy/v1/containers/c1/healthcheck" {
		t.Errorf("unexpected path: %s", got)
	}
}

func TestEnvUpsertSendsBody(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterEnvVars)
	res := callTool(t, cs, "env_upsert", map[string]any{
		"app_id": "app-1",
		"key":    "DB_URL",
		"value":  "postgres://x",
		"secret": true,
	})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	body := fake.requests[0].body
	if !strings.Contains(body, `"key":"DB_URL"`) ||
		!strings.Contains(body, `"value":"postgres://x"`) ||
		!strings.Contains(body, `"isSecret":true`) {
		t.Errorf("unexpected env body: %s", body)
	}
}

func TestEnvBulkRequiresEntries(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterEnvVars)
	res := callTool(t, cs, "env_bulk", map[string]any{
		"app_id":  "app-1",
		"entries": []any{},
	})
	if !res.IsError {
		t.Fatal("expected error for empty entries")
	}
}

func TestEnvBulkSendsVarsPayload(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterEnvVars)
	res := callTool(t, cs, "env_bulk", map[string]any{
		"app_id": "app-1",
		"entries": []map[string]any{
			{"key": "A", "value": "1"},
			{"key": "B", "value": "2", "isSecret": true},
		},
	})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(fake.requests))
	}
	req := fake.requests[0]
	if req.method != "PUT" || req.path != "/paas-deploy/v1/apps/app-1/env/bulk" {
		t.Errorf("unexpected method/path: %s %s", req.method, req.path)
	}
	if !strings.Contains(req.body, `"vars":[`) {
		t.Errorf("expected backend payload to use 'vars' key, got: %s", req.body)
	}
	if strings.Contains(req.body, `"truncate"`) {
		t.Errorf("did not expect 'truncate' field in payload, got: %s", req.body)
	}
	if !strings.Contains(req.body, `"key":"A"`) || !strings.Contains(req.body, `"isSecret":true`) {
		t.Errorf("unexpected vars content: %s", req.body)
	}
}

func TestDomainsAddRequiresDomain(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterDomains)
	res := callTool(t, cs, "domains_add", map[string]any{"app_id": "app-1"})
	if !res.IsError {
		t.Fatal("expected error for missing domain")
	}
}

func TestDomainsAddSendsBody(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterDomains)
	res := callTool(t, cs, "domains_add", map[string]any{
		"app_id":      "app-1",
		"domain":      "api.example.com",
		"path_prefix": "/api",
	})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	req := fake.requests[0]
	if req.method != "POST" || req.path != "/paas-deploy/v1/apps/app-1/domains" {
		t.Errorf("unexpected method/path: %s %s", req.method, req.path)
	}
	if !strings.Contains(req.body, `"domain":"api.example.com"`) ||
		!strings.Contains(req.body, `"pathPrefix":"/api"`) {
		t.Errorf("unexpected body: %s", req.body)
	}
	if strings.Contains(req.body, `"recordType"`) {
		t.Errorf("recordType must not be sent (backend ignores it): %s", req.body)
	}
}

func TestDatabaseTLSConfigureSendsBackendShape(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterSSL)
	res := callTool(t, cs, "database_tls_configure", map[string]any{
		"container_id":  "c1",
		"server_id":     "srv-1",
		"database_user": "postgres",
		"database_name": "appdb",
	})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	req := fake.requests[0]
	if req.method != http.MethodPost || req.path != "/paas-deploy/v1/containers/c1/ssl" {
		t.Errorf("unexpected method/path: %s %s", req.method, req.path)
	}
	if !strings.Contains(req.body, `"serverId":"srv-1"`) ||
		!strings.Contains(req.body, `"databaseUser":"postgres"`) ||
		!strings.Contains(req.body, `"databaseName":"appdb"`) ||
		!strings.Contains(req.body, `"databaseType":"postgresql"`) {
		t.Errorf("body missing expected fields: %s", req.body)
	}
	if strings.Contains(req.body, `"domain"`) || strings.Contains(req.body, `"email"`) {
		t.Errorf("legacy SSL fields should not be present: %s", req.body)
	}
}

func TestDatabaseTLSConfigureRequiresFields(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterSSL)
	res := callTool(t, cs, "database_tls_configure", map[string]any{
		"container_id": "c1",
	})
	if !res.IsError {
		t.Fatal("expected error when required fields are missing")
	}
}

func TestDatabaseTLSStatusSendsBackendShape(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterSSL)
	res := callTool(t, cs, "database_tls_status", map[string]any{
		"container_id":  "c1",
		"server_id":     "srv-1",
		"database_user": "postgres",
		"database_name": "appdb",
	})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	req := fake.requests[0]
	if req.method != "GET" {
		t.Errorf("unexpected method: %s", req.method)
	}
	if !strings.Contains(req.query, "serverId=srv-1") ||
		!strings.Contains(req.query, "databaseUser=postgres") ||
		!strings.Contains(req.query, "databaseName=appdb") ||
		!strings.Contains(req.query, "databaseType=postgresql") {
		t.Errorf("query string missing expected params: %s", req.query)
	}
}

func TestNetworksConnectSendsBody(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterResources)
	res := callTool(t, cs, "networks_connect", map[string]any{
		"container_id": "c1",
		"network":      "paasdeploy",
	})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	body := fake.requests[0].body
	if !strings.Contains(body, `"network":"paasdeploy"`) {
		t.Errorf("missing network in body: %s", body)
	}
	if strings.Contains(body, `"aliases"`) {
		t.Errorf("aliases must not be sent (backend ignores it): %s", body)
	}
}

func TestNetworksDisconnectUsesDelete(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterResources)
	res := callTool(t, cs, "networks_disconnect", map[string]any{
		"container_id": "c1",
		"network":      "paasdeploy",
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
	if req.path != "/paas-deploy/v1/containers/c1/networks/paasdeploy" {
		t.Errorf("unexpected path: %s", req.path)
	}
	if strings.Contains(req.query, "force=") {
		t.Errorf("force query must not be sent (backend ignores it): %s", req.query)
	}
}

func TestNetworksCreateOmitsDriver(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterResources)
	res := callTool(t, cs, "networks_create", map[string]any{
		"name": "paasdeploy",
	})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if strings.Contains(fake.requests[0].body, `"driver"`) {
		t.Errorf("driver must not be sent (backend ignores it): %s", fake.requests[0].body)
	}
}

func TestServersProvisionPostsToProvisionEndpoint(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterServers)
	res := callTool(t, cs, "servers_provision", map[string]any{"id": "srv-1"})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if got := fake.requests[0].path; got != "/paas-deploy/v1/servers/srv-1/provision" {
		t.Errorf("unexpected path: %s", got)
	}
}

func TestServersManageRequiresAction(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterServers)
	res := callTool(t, cs, "servers_manage", map[string]any{"id": "srv-1"})
	if !res.IsError {
		t.Fatal("expected error for missing action")
	}
}

func TestServersManageRejectsUnknownAction(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterServers)
	res := callTool(t, cs, "servers_manage", map[string]any{
		"id":     "srv-1",
		"action": "reboot",
	})
	if !res.IsError {
		t.Fatal("expected error for action not in whitelist")
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.requests) != 0 {
		t.Errorf("invalid action must not reach the backend, got %d requests", len(fake.requests))
	}
}

func TestServersManageAcceptsRestartAgent(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterServers)
	res := callTool(t, cs, "servers_manage", map[string]any{
		"id":     "srv-1",
		"action": "restart_agent",
	})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	req := fake.requests[0]
	if req.method != "POST" || req.path != "/paas-deploy/v1/servers/srv-1/manage" {
		t.Errorf("unexpected method/path: %s %s", req.method, req.path)
	}
	if !strings.Contains(req.body, `"action":"restart_agent"`) {
		t.Errorf("missing action in body: %s", req.body)
	}
}

func TestTemplatesDeploySendsBackendShape(t *testing.T) {
	fake := &fakeBackend{}
	cs := setupServer(t, fake, RegisterTemplates)
	res := callTool(t, cs, "templates_deploy", map[string]any{
		"id":   "postgres",
		"name": "db-1",
		"env": map[string]string{
			"POSTGRES_PASSWORD": "secret",
		},
		"ports": []map[string]any{
			{"hostPort": 5432, "containerPort": 5432, "protocol": "tcp"},
		},
		"network":        "paasdeploy",
		"restart_policy": "unless-stopped",
		"command":        []string{"postgres", "-c", "max_connections=200"},
	})
	if res.IsError {
		t.Fatalf("expected success, got %s", extractText(t, res))
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	req := fake.requests[0]
	if req.path != "/paas-deploy/v1/templates/postgres/deploy" {
		t.Errorf("unexpected path: %s", req.path)
	}
	if !strings.Contains(req.body, `"name":"db-1"`) ||
		!strings.Contains(req.body, `"env":{"POSTGRES_PASSWORD":"secret"}`) ||
		!strings.Contains(req.body, `"ports":[`) ||
		!strings.Contains(req.body, `"hostPort":5432`) ||
		!strings.Contains(req.body, `"network":"paasdeploy"`) ||
		!strings.Contains(req.body, `"restartPolicy":"unless-stopped"`) ||
		!strings.Contains(req.body, `"command":["postgres","-c","max_connections=200"]`) {
		t.Errorf("body missing expected fields: %s", req.body)
	}
	if strings.Contains(req.body, `"variables"`) {
		t.Errorf("legacy 'variables' key should not appear in payload: %s", req.body)
	}
}
