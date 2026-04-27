package mcpserver

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/backend"
	"github.com/paasdeploy/mcp/internal/toolkit"
)

const (
	ServerName    = "flowdeploy"
	ServerVersion = "0.1.0"
)

type Deps struct {
	Logger  *slog.Logger
	Backend *backend.Client
}

func New(d Deps) (*mcp.Server, error) {
	if d.Backend == nil {
		return nil, fmt.Errorf("mcpserver.New: Backend is required")
	}
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    ServerName,
		Version: ServerVersion,
		Title:   "FlowDeploy",
	}, &mcp.ServerOptions{
		Instructions: "FlowDeploy MCP server. Use scoped Personal Access Tokens to read and operate the FlowDeploy control plane (apps, containers, servers, images, resources, templates, audit, system). Read-only tools require scope 'read'. Mutating and destructive tools follow the dry-run + reason contract documented in docs/MCP_PLAYBOOKS.md.",
	})
	return srv, nil
}

func Run(ctx context.Context, srv *mcp.Server, transport mcp.Transport) error {
	return srv.Run(ctx, transport)
}

func ToolkitDeps(d Deps) toolkit.Deps {
	return toolkit.Deps{Logger: d.Logger, Backend: d.Backend}
}
