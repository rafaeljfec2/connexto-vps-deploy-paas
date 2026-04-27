package transport

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
	"github.com/paasdeploy/mcp/internal/mcpserver"
	"github.com/paasdeploy/mcp/internal/toolkit"
)

// fakeBackend captures the bearer token forwarded by the MCP server, so we
// can assert that the per-request PAT (not the static config) reaches the
// downstream API.
type fakeBackend struct {
	mu     sync.Mutex
	tokens []string
	server *httptest.Server
}

func newFakeBackend() *fakeBackend {
	fb := &fakeBackend{}
	fb.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fb.mu.Lock()
		fb.tokens = append(fb.tokens, r.Header.Get("Authorization"))
		fb.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"data":{"items":[],"total":0}}`))
	}))
	return fb
}

func (f *fakeBackend) URL() string { return f.server.URL }

func (f *fakeBackend) Close() { f.server.Close() }

func (f *fakeBackend) Tokens() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.tokens))
	copy(out, f.tokens)
	return out
}

func newTestTransportServer(t *testing.T, fb *fakeBackend) (*Server, *httptest.Server) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	bk, err := backend.New(backend.Options{
		BaseURL:                fb.URL(),
		Logger:                 logger,
		Timeout:                2 * time.Second,
		AcceptTokenFromContext: true,
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
	mcpserver.RegisterAllDestructive(srv, deps)

	transportSrv, err := NewServer(ServerOptions{
		Addr:           ":0",
		AllowedClients: []string{"cursor", "ci:*"},
		ReadRPM:        100,
		MutateRPM:      10,
		Logger:         logger,
		MCPServer:      srv,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	return transportSrv, httptest.NewServer(transportSrv.http.Handler)
}

func TestHTTPTransportRejectsUnauthenticatedMCP(t *testing.T) {
	fb := newFakeBackend()
	defer fb.Close()
	_, ts := newTestTransportServer(t, fb)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/mcp", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestHTTPTransportHealthAndMetricsEndpoints(t *testing.T) {
	fb := newFakeBackend()
	defer fb.Close()
	_, ts := newTestTransportServer(t, fb)
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/healthz expected 200, got %d", resp.StatusCode)
	}

	resp2, err := ts.Client().Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("/metrics expected 200, got %d", resp2.StatusCode)
	}
	body, _ := io.ReadAll(resp2.Body)
	if !strings.Contains(string(body), "mcp_") {
		t.Fatalf("expected mcp_* counters in /metrics body, got: %s", string(body))
	}
}

func TestHTTPTransportForwardsPATToBackend(t *testing.T) {
	fb := newFakeBackend()
	defer fb.Close()
	transportSrv, ts := newTestTransportServer(t, fb)
	defer ts.Close()
	_ = transportSrv

	httpClient := &http.Client{Transport: &headerInjectingTransport{
		base:    http.DefaultTransport,
		headers: map[string]string{"Authorization": "Bearer pdp_live_alpha", "X-MCP-Client": "cursor"},
	}}
	clientTransport := &mcp.StreamableClientTransport{
		Endpoint:   ts.URL + "/mcp",
		HTTPClient: httpClient,
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	args, err := json.Marshal(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "apps_list", Arguments: json.RawMessage(args)})
	if err != nil {
		t.Fatalf("CallTool apps_list: %v", err)
	}
	if res != nil && res.IsError {
		for i, c := range res.Content {
			if tc, ok := c.(*mcp.TextContent); ok {
				t.Logf("tool error content[%d]: %s", i, tc.Text)
			} else {
				t.Logf("tool error content[%d]: %+v", i, c)
			}
		}
	}

	tokens := fb.Tokens()
	if len(tokens) == 0 {
		t.Fatalf("expected backend to be called at least once")
	}
	for _, h := range tokens {
		if h != "Bearer pdp_live_alpha" {
			t.Fatalf("expected backend Authorization to forward agent PAT, got %q", h)
		}
	}
}

// headerInjectingTransport injects authentication headers on every outbound
// request, simulating a real MCP client that always carries a PAT.
type headerInjectingTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (h *headerInjectingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range h.headers {
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}
	return h.base.RoundTrip(req)
}
