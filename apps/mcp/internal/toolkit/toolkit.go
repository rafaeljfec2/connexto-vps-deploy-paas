package toolkit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/backend"
)

type ToolHandler[Input any] func(ctx context.Context, req *mcp.CallToolRequest, in Input) (any, error)

type Deps struct {
	Logger  *slog.Logger
	Backend *backend.Client
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
	wrapped := func(ctx context.Context, req *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		start := time.Now()
		out, err := handler(ctx, req, in)
		latency := time.Since(start)
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
		case int:
			if x == 0 {
				continue
			}
			q.Set(k, fmt.Sprintf("%d", x))
		case int32:
			if x == 0 {
				continue
			}
			q.Set(k, fmt.Sprintf("%d", x))
		case int64:
			if x == 0 {
				continue
			}
			q.Set(k, fmt.Sprintf("%d", x))
		case bool:
			if x {
				q.Set(k, "true")
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
