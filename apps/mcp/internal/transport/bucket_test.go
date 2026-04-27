package transport

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/paasdeploy/mcp/internal/toolkit"
)

var registerOnce sync.Once

func primeRegistry(t *testing.T) {
	t.Helper()
	registerOnce.Do(func() {
		toolkit.MarkMutatingForTest("apps_delete")
		toolkit.MarkMutatingForTest("env_set")
	})
}

func mustClassify(t *testing.T, req *http.Request) (Bucket, ClassifyOutcome) {
	t.Helper()
	bucket, outcome, err := classifyRequest(req)
	if err != nil {
		t.Fatalf("unexpected classifyRequest error: %v", err)
	}
	return bucket, outcome
}

func TestClassifyRequestGetIsRead(t *testing.T) {
	primeRegistry(t)
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	got, outcome := mustClassify(t, req)
	if got != BucketRead {
		t.Fatalf("GET must always be read bucket, got %s", got)
	}
	if outcome != ClassifyOK {
		t.Fatalf("GET must classify cleanly, got outcome %q", outcome)
	}
}

func TestClassifyRequestEmptyBodyDefaultsToRead(t *testing.T) {
	primeRegistry(t)
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	got, _ := mustClassify(t, req)
	if got != BucketRead {
		t.Fatalf("empty POST defaults to read, got %s", got)
	}
}

func TestClassifyRequestToolsCallReadIsRead(t *testing.T) {
	primeRegistry(t)
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"apps_list","arguments":{}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	got, _ := mustClassify(t, req)
	if got != BucketRead {
		t.Fatalf("read tool must classify as read bucket, got %s", got)
	}
	preserved, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read rewound body: %v", err)
	}
	if string(preserved) != body {
		t.Fatalf("body must be rewound for downstream handlers, got %q", string(preserved))
	}
}

func TestClassifyRequestToolsCallMutatingIsMutate(t *testing.T) {
	primeRegistry(t)
	body := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"apps_delete","arguments":{"id":"x"}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	got, _ := mustClassify(t, req)
	if got != BucketMutate {
		t.Fatalf("destructive tools/call must classify as mutate bucket, got %s", got)
	}
}

func TestClassifyRequestHeaderCanUpgradeToMutate(t *testing.T) {
	primeRegistry(t)
	body := `{"jsonrpc":"2.0","id":3,"method":"tools/list","params":{}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("X-MCP-Bucket", "mutate")
	got, _ := mustClassify(t, req)
	if got != BucketMutate {
		t.Fatalf("X-MCP-Bucket header must upgrade non-tools call, got %s", got)
	}
}

func TestClassifyRequestHeaderCannotDowngradeMutate(t *testing.T) {
	primeRegistry(t)
	body := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"apps_delete","arguments":{"id":"x"}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("X-MCP-Bucket", "read")
	got, _ := mustClassify(t, req)
	if got != BucketMutate {
		t.Fatalf("destructive call must remain mutate even with X-MCP-Bucket=read, got %s", got)
	}
}

// Regression: a JSON-RPC batch carrying a destructive call MUST be billed
// against the mutate bucket. Before the fix, json.Unmarshal on a struct
// silently failed for arrays and the request fell through to BucketRead,
// allowing per-PAT mutate quotas to be bypassed by batching.
func TestClassifyRequestBatchWithMutatingCallIsMutate(t *testing.T) {
	primeRegistry(t)
	body := `[{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}},` +
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"apps_delete","arguments":{"id":"prod"}}}]`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	got, outcome := mustClassify(t, req)
	if got != BucketMutate {
		t.Fatalf("batch carrying destructive call MUST classify as mutate bucket; defense-in-depth bypass otherwise. got %s", got)
	}
	if outcome != ClassifyOK {
		t.Fatalf("well-formed batch must classify cleanly; got outcome %q", outcome)
	}
	preserved, _ := io.ReadAll(req.Body)
	if string(preserved) != body {
		t.Fatalf("body must be rewound after batch classification")
	}
}

func TestClassifyRequestBatchOfReadsStaysRead(t *testing.T) {
	primeRegistry(t)
	body := `[{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}},` +
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"apps_list","arguments":{}}}]`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	got, _ := mustClassify(t, req)
	if got != BucketRead {
		t.Fatalf("batch of pure reads must stay in read bucket, got %s", got)
	}
}

func TestClassifyRequestMalformedBodyDegradesToMutate(t *testing.T) {
	primeRegistry(t)
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"method":"tools/call","params":`))
	got, outcome := mustClassify(t, req)
	if got != BucketMutate {
		t.Fatalf("malformed JSON must degrade to mutate (defense-in-depth), got %s", got)
	}
	if outcome != ClassifyParseError {
		t.Fatalf("malformed JSON must report parse_error outcome, got %q", outcome)
	}
}

// Regression: payloads larger than maxBucketPeekBytes used to be silently
// truncated for downstream handlers. The fix concatenates the peeked head
// with the untouched tail so the StreamableHTTPHandler still sees the full
// JSON-RPC envelope, while classification escalates conservatively to mutate.
func TestClassifyRequestOverflowPreservesFullBodyAndUpgrades(t *testing.T) {
	primeRegistry(t)
	tail := strings.Repeat("x", 8*1024)
	full := strings.Repeat("a", maxBucketPeekBytes+1) + tail
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(full))
	got, outcome := mustClassify(t, req)
	if got != BucketMutate {
		t.Fatalf("oversize body must classify as mutate, got %s", got)
	}
	if outcome != ClassifyOverflow {
		t.Fatalf("oversize body must report overflow outcome, got %q", outcome)
	}
	preserved, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read rebuilt body: %v", err)
	}
	if len(preserved) != len(full) {
		t.Fatalf("body length must be preserved across peek; got %d want %d", len(preserved), len(full))
	}
	if !bytes.Equal(preserved, []byte(full)) {
		t.Fatalf("body bytes must be preserved verbatim across peek")
	}
}

// Failing readers should surface as ClassifyReadError so the operator can
// tell legitimate flaky transports apart from clients sending oversize
// payloads (overflow) or malformed JSON (parse_error).
func TestClassifyRequestReaderErrorReportsReadError(t *testing.T) {
	primeRegistry(t)
	req := httptest.NewRequest(http.MethodPost, "/mcp", &errReader{})
	bucket, outcome, err := classifyRequest(req)
	if err == nil {
		t.Fatalf("expected error from broken reader")
	}
	if bucket != BucketMutate {
		t.Fatalf("broken reader must degrade to mutate, got %s", bucket)
	}
	if outcome != ClassifyReadError {
		t.Fatalf("broken reader must report read_error outcome, got %q", outcome)
	}
}

func TestNormalizeRouteCollapsesUnknownPaths(t *testing.T) {
	cases := map[string]string{
		"/mcp":                         "/mcp",
		"/mcp/":                        "/mcp",
		"/mcp/anything-here":           "/mcp",
		"/metrics":                     "/metrics",
		"/healthz":                     "/healthz",
		"/readyz":                      "/readyz",
		"/random":                      "other",
		"/../../etc/passwd":            "other",
		"/" + strings.Repeat("a", 256): "other",
	}
	for input, want := range cases {
		if got := normalizeRoute(input); got != want {
			t.Fatalf("normalizeRoute(%q) = %q, want %q", input, got, want)
		}
	}
}

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) { return 0, errors.New("boom") }
func (errReader) Close() error               { return nil }
