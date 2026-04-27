package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/toolkit"
)

type serverIDInput struct {
	ID string `json:"id" jsonschema:"the server UUID"`
}

type serverManageInput struct {
	ID     string `json:"id" jsonschema:"the server UUID"`
	Action string `json:"action" jsonschema:"management action; one of: restart_agent, restart_user_manager, agent_logs, fix_docker_permissions"`
}

var allowedServerManageActions = map[string]struct{}{
	"restart_agent":          {},
	"restart_user_manager":   {},
	"agent_logs":             {},
	"fix_docker_permissions": {},
}

func RegisterServers(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "servers_list",
			Description: "List provisioned remote servers (VPS) and the local host. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (any, error) {
			return getJSON(ctx, deps.Backend, "/servers", nil)
		})

	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "servers_get",
			Description: "Get details for a single server. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in serverIDInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			return getJSON(ctx, deps.Backend, "/servers/"+pathSeg(in.ID), nil)
		})

	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "servers_stats",
			Description: "Get CPU, memory and disk stats for a server. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in serverIDInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			return getJSON(ctx, deps.Backend, "/servers/"+pathSeg(in.ID)+"/stats", nil)
		})

	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "servers_health",
			Description: "Run a health probe against a server's agent. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in serverIDInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			return getJSON(ctx, deps.Backend, "/servers/"+pathSeg(in.ID)+"/health", nil)
		})

	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "servers_apps",
			Description: "List apps deployed on a specific server. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in serverIDInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			return getJSON(ctx, deps.Backend, "/servers/"+pathSeg(in.ID)+"/apps", nil)
		})

	toolkit.RegisterWrite(srv, deps,
		&mcp.Tool{
			Name:        "servers_provision",
			Description: "Run the provisioning playbook on a server (installs Docker, Traefik, agent). Requires scope 'servers:write'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in serverIDInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			return postJSON(ctx, deps.Backend, "/servers/"+pathSeg(in.ID)+"/provision", map[string]any{}, nil)
		})

	toolkit.RegisterWrite(srv, deps,
		&mcp.Tool{
			Name:        "servers_update_agent",
			Description: "Trigger an agent update on a server. Requires scope 'servers:write'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in serverIDInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			return postJSON(ctx, deps.Backend, "/servers/"+pathSeg(in.ID)+"/update-agent", map[string]any{}, nil)
		})

	toolkit.RegisterWrite(srv, deps,
		&mcp.Tool{
			Name:        "servers_manage",
			Description: "Run a SSH-based management action on a server. Allowed actions: 'restart_agent', 'restart_user_manager', 'agent_logs', 'fix_docker_permissions'. Requires scope 'servers:write'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in serverManageInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			if in.Action == "" {
				return nil, errInvalidArg("action is required")
			}
			if _, ok := allowedServerManageActions[in.Action]; !ok {
				return nil, errInvalidArg("invalid action; allowed: restart_agent, restart_user_manager, agent_logs, fix_docker_permissions")
			}
			body := map[string]any{"action": in.Action}
			return postJSON(ctx, deps.Backend, "/servers/"+pathSeg(in.ID)+"/manage", body, nil)
		})
}
