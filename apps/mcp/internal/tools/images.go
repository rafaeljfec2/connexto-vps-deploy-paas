package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/toolkit"
)

type imageListInput struct {
	ServerID string `json:"server_id,omitempty" jsonschema:"optional remote server ID"`
}

func RegisterImages(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "images_list",
			Description: "List Docker images. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in imageListInput) (any, error) {
			return getJSON(ctx, deps.Backend, "/images", map[string]any{"serverId": in.ServerID})
		})

	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "images_dangling",
			Description: "List dangling Docker images that can be safely pruned. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in imageListInput) (any, error) {
			return getJSON(ctx, deps.Backend, "/images/dangling", map[string]any{"serverId": in.ServerID})
		})
}
