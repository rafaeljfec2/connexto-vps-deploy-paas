package toolkit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/backend"
	"github.com/paasdeploy/mcp/internal/metrics"
)

type ToolHandler[Input any] func(ctx context.Context, req *mcp.CallToolRequest, in Input) (any, error)

type Deps struct {
	Logger  *slog.Logger
	Backend *backend.Client
}

// mutatingTools is populated at registration time and consulted by the HTTP
// transport to classify requests into the read or mutate rate-limiter bucket.
// We keep it package-level (not on Deps) because the registry must outlive any
// individual server instance and is read-only after startup completes.
var mutatingTools sync.Map

// IsMutatingTool reports whether the named tool was registered as a write or
// destructive operation. Reads (and unknown tools, treated as reads
// conservatively because the SDK validates names) return false.
func IsMutatingTool(name string) bool {
	_, ok := mutatingTools.Load(name)
	return ok
}

// MarkMutatingForTest seeds the mutating-tools registry for unit tests that
// need to drive bucket classification logic without registering full *mcp.Tool
// instances. It is exported because the registry is package-private and the
// transport tests live in a sibling package; production code MUST NOT call it.
func MarkMutatingForTest(name string) {
	mutatingTools.Store(name, struct{}{})
}

func RegisterReadOnly[Input any](
	srv *mcp.Server,
	deps Deps,
	tool *mcp.Tool,
	handler ToolHandler[Input],
) {
	registerTool(srv, deps, tool, handler, "read")
}

func RegisterWrite[Input any](
	srv *mcp.Server,
	deps Deps,
	tool *mcp.Tool,
	handler ToolHandler[Input],
) {
	registerTool(srv, deps, tool, handler, "write")
}

func registerTool[Input any](
	srv *mcp.Server,
	deps Deps,
	tool *mcp.Tool,
	handler ToolHandler[Input],
	mode string,
) {
	if mode == "write" || mode == "destructive" {
		mutatingTools.Store(tool.Name, struct{}{})
	}
	wrapped := func(ctx context.Context, req *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		start := time.Now()
		out, err := handler(ctx, req, in)
		latency := time.Since(start)
		metrics.ObserveToolCall(tool.Name, mode, err == nil, latency.Seconds())
		if err != nil {
			deps.Logger.Warn("tool call failed",
				"tool", tool.Name,
				"mode", mode,
				"latency_ms", latency.Milliseconds(),
				"error", err,
			)
			return nil, nil, formatToolError(tool.Name, err)
		}
		deps.Logger.Info("tool call",
			"tool", tool.Name,
			"mode", mode,
			"latency_ms", latency.Milliseconds(),
		)
		return nil, out, nil
	}
	mcp.AddTool(srv, tool, wrapped)
}

func DecodeBackend(raw backend.Raw) (any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("decode backend payload: %w", err)
	}
	return map[string]any{"data": v}, nil
}

func formatToolError(toolName string, err error) error {
	var apiErr *backend.APIError
	if errors.As(err, &apiErr) {
		return fmt.Errorf("%s: backend %d %s: %s", toolName, apiErr.Status, apiErr.Code, apiErr.Message)
	}
	return fmt.Errorf("%s: %w", toolName, err)
}

func MarshalRaw(value any) (backend.Raw, error) {
	if value == nil {
		return backend.Raw("null"), nil
	}
	return json.Marshal(value)
}

// BuildQuery converts a parameter map into url.Values applying omitempty semantics:
// nil, empty strings, zero numbers and false booleans are OMITTED from the resulting
// query string. This keeps URLs short and matches how most MCP tools want to behave
// (callers usually skip optional filters by leaving them at their zero value).
//
// IMPORTANT: if a tool needs to send a literal "false" or "0" to the backend (i.e.
// the zero value carries semantic meaning, like tail=0 = "no log lines"), pass a
// pointer (*bool, *int, *int32, *int64, *string) instead. Pointers are NEVER
// omitted: nil means "absent", non-nil means "send literally, including zero
// values".
//
// Examples:
//
//	BuildQuery(map[string]any{"force": false})   // omitted
//	BuildQuery(map[string]any{"force": ptrBool(false)}) // -> ?force=false
//	BuildQuery(map[string]any{"tail": 0})        // omitted
//	BuildQuery(map[string]any{"tail": ptrInt(0)}) // -> ?tail=0
func BuildQuery(params map[string]any) url.Values {
	q := url.Values{}
	for k, v := range params {
		if v == nil {
			continue
		}
		switch x := v.(type) {
		case string:
			if x == "" {
				continue
			}
			q.Set(k, x)
		case *string:
			if x == nil {
				continue
			}
			q.Set(k, *x)
		case int:
			if x == 0 {
				continue
			}
			q.Set(k, fmt.Sprintf("%d", x))
		case *int:
			if x == nil {
				continue
			}
			q.Set(k, fmt.Sprintf("%d", *x))
		case int32:
			if x == 0 {
				continue
			}
			q.Set(k, fmt.Sprintf("%d", x))
		case *int32:
			if x == nil {
				continue
			}
			q.Set(k, fmt.Sprintf("%d", *x))
		case int64:
			if x == 0 {
				continue
			}
			q.Set(k, fmt.Sprintf("%d", x))
		case *int64:
			if x == nil {
				continue
			}
			q.Set(k, fmt.Sprintf("%d", *x))
		case bool:
			if x {
				q.Set(k, "true")
			}
		case *bool:
			if x == nil {
				continue
			}
			if *x {
				q.Set(k, "true")
			} else {
				q.Set(k, "false")
			}
		default:
			b, err := json.Marshal(x)
			if err == nil {
				q.Set(k, string(b))
			}
		}
	}
	return q
}

// PtrBool/PtrInt/PtrInt64/PtrString return pointers to literal values so callers
// can use BuildQuery's "explicit" semantics for zero values without spreading
// helper variables across the codebase.
func PtrBool(v bool) *bool       { return &v }
func PtrInt(v int) *int          { return &v }
func PtrInt64(v int64) *int64    { return &v }
func PtrString(v string) *string { return &v }
