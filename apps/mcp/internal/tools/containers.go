package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/toolkit"
)

// normaliseSince accepts either an RFC3339 absolute timestamp or a Go duration
// shorthand (e.g. "1h", "30m", "15m30s") and returns the canonical RFC3339
// string the backend expects. Empty input yields empty output (no filter).
// "now-relative" shorthand is computed against the provided clock to keep
// tests deterministic.
func normaliseSince(raw string, now func() time.Time) (string, error) {
	if raw == "" {
		return "", nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC().Format(time.RFC3339), nil
	}
	if d, err := time.ParseDuration(raw); err == nil {
		if d <= 0 {
			return "", fmt.Errorf("since duration must be positive, got %q", raw)
		}
		return now().Add(-d).UTC().Format(time.RFC3339), nil
	}
	return "", fmt.Errorf("since must be RFC3339 (e.g. 2026-04-30T19:00:00Z) or a duration shorthand (e.g. 1h, 30m), got %q", raw)
}

type containerListInput struct {
	ServerID string `json:"server_id,omitempty" jsonschema:"optional remote server ID; defaults to host"`
}

type containerIDInput struct {
	ID       string `json:"id" jsonschema:"the container ID or name"`
	ServerID string `json:"server_id,omitempty" jsonschema:"optional remote server ID; defaults to host"`
}

type containerLogsInput struct {
	ID             string `json:"id" jsonschema:"the container ID or name"`
	Tail           int    `json:"tail,omitempty" jsonschema:"max number of log lines to return (default 200, max 5000)"`
	Since          string `json:"since,omitempty" jsonschema:"return logs since timestamp (RFC3339 or duration like 1h)"`
	ServerID       string `json:"server_id,omitempty" jsonschema:"optional remote server ID; defaults to host"`
	Follow         bool   `json:"follow,omitempty" jsonschema:"when true, polls for new logs and emits MCP progress notifications until timeout"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty" jsonschema:"max time to follow when follow=true (default 60, max 600)"`
	IntervalSec    int    `json:"interval_seconds,omitempty" jsonschema:"interval between polls when follow=true (default 3)"`
}

func RegisterContainers(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "containers_list",
			Description: "List Docker containers managed by FlowDeploy on the host or a remote server. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in containerListInput) (any, error) {
			return getJSON(ctx, deps.Backend, "/containers", map[string]any{"serverId": in.ServerID})
		})

	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "containers_get",
			Description: "Inspect a single container including state, mounts and networks. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in containerIDInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			return getJSON(ctx, deps.Backend, "/containers/"+pathSeg(in.ID), map[string]any{"serverId": in.ServerID})
		})

	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "containers_logs",
			Description: "Fetch container logs (paginated). When follow=true, streams new log chunks via MCP progress notifications. Requires scope 'read'.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, in containerLogsInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			normalisedSince, err := normaliseSince(in.Since, time.Now)
			if err != nil {
				return nil, errInvalidArg(err.Error())
			}
			in.Since = normalisedSince
			if !in.Follow {
				return getJSON(ctx, deps.Backend, "/containers/"+pathSeg(in.ID)+"/logs", map[string]any{
					"tail":     in.Tail,
					"since":    in.Since,
					"serverId": in.ServerID,
				})
			}
			return followContainerLogs(ctx, req, deps, in)
		})

	registerContainerLifecycleAction(srv, deps, "containers_start", "start", "Start a stopped container by ID. Requires scope 'containers:write'.")
	registerContainerLifecycleAction(srv, deps, "containers_stop", "stop", "Stop a running container by ID. Requires scope 'containers:write'.")
	registerContainerLifecycleAction(srv, deps, "containers_restart", "restart", "Restart a container by ID. Requires scope 'containers:write'.")
	registerContainerLifecycleAction(srv, deps, "containers_healthcheck", "healthcheck", "Run the container's healthcheck on demand and return the result. Requires scope 'containers:write'.")
}

func registerContainerLifecycleAction(srv *mcp.Server, deps toolkit.Deps, toolName, action, description string) {
	toolkit.RegisterWrite(srv, deps,
		&mcp.Tool{
			Name:        toolName,
			Description: description,
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in containerIDInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			return postJSON(ctx, deps.Backend, "/containers/"+pathSeg(in.ID)+"/"+action, map[string]any{}, map[string]any{
				"serverId": in.ServerID,
			})
		})
}
