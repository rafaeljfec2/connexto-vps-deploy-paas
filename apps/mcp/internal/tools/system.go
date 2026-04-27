package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/toolkit"
)

func RegisterSystem(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "system_stats",
			Description: "Get aggregated system stats (CPU, memory, disk, container counts) for the FlowDeploy host. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (any, error) {
			return getJSON(ctx, deps.Backend, "/system/stats", nil)
		})
}
