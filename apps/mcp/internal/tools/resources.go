package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/toolkit"
)

type resourceListInput struct {
	ServerID string `json:"server_id,omitempty" jsonschema:"optional remote server ID"`
}

type networkCreateInput struct {
	Name     string `json:"name" jsonschema:"Docker network name"`
	ServerID string `json:"server_id,omitempty" jsonschema:"optional remote server ID"`
}

type networkConnectInput struct {
	ContainerID string `json:"container_id" jsonschema:"the container ID or name"`
	Network     string `json:"network" jsonschema:"target Docker network name"`
	ServerID    string `json:"server_id,omitempty" jsonschema:"optional remote server ID"`
}

type networkDisconnectInput struct {
	ContainerID string `json:"container_id" jsonschema:"the container ID or name"`
	Network     string `json:"network" jsonschema:"target Docker network name"`
	ServerID    string `json:"server_id,omitempty" jsonschema:"optional remote server ID"`
}

type volumeCreateInput struct {
	Name     string `json:"name" jsonschema:"Docker volume name"`
	ServerID string `json:"server_id,omitempty" jsonschema:"optional remote server ID"`
}

func RegisterResources(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "networks_list",
			Description: "List Docker networks managed by FlowDeploy. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in resourceListInput) (any, error) {
			return getJSON(ctx, deps.Backend, "/networks", map[string]any{"serverId": in.ServerID})
		})

	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "volumes_list",
			Description: "List Docker volumes managed by FlowDeploy. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in resourceListInput) (any, error) {
			return getJSON(ctx, deps.Backend, "/volumes", map[string]any{"serverId": in.ServerID})
		})

	toolkit.RegisterWrite(srv, deps,
		&mcp.Tool{
			Name:        "networks_create",
			Description: "Create a new Docker bridge network managed by FlowDeploy. Requires scope 'resources:write'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in networkCreateInput) (any, error) {
			if in.Name == "" {
				return nil, errInvalidArg("name is required")
			}
			body := map[string]any{"name": in.Name}
			return postJSON(ctx, deps.Backend, "/networks", body, map[string]any{"serverId": in.ServerID})
		})

	toolkit.RegisterWrite(srv, deps,
		&mcp.Tool{
			Name:        "networks_connect",
			Description: "Attach a container to a Docker network. Requires scope 'resources:write'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in networkConnectInput) (any, error) {
			if in.ContainerID == "" {
				return nil, errInvalidArg("container_id is required")
			}
			if in.Network == "" {
				return nil, errInvalidArg("network is required")
			}
			body := map[string]any{"network": in.Network}
			return postJSON(ctx, deps.Backend, "/containers/"+pathSeg(in.ContainerID)+"/networks", body, map[string]any{"serverId": in.ServerID})
		})

	toolkit.RegisterWrite(srv, deps,
		&mcp.Tool{
			Name:        "networks_disconnect",
			Description: "Detach a container from a Docker network. Requires scope 'resources:write'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in networkDisconnectInput) (any, error) {
			if in.ContainerID == "" {
				return nil, errInvalidArg("container_id is required")
			}
			if in.Network == "" {
				return nil, errInvalidArg("network is required")
			}
			return deleteJSON(ctx, deps.Backend, "/containers/"+pathSeg(in.ContainerID)+"/networks/"+pathSeg(in.Network), map[string]any{
				"serverId": in.ServerID,
			})
		})

	toolkit.RegisterWrite(srv, deps,
		&mcp.Tool{
			Name:        "volumes_create",
			Description: "Create a new Docker volume managed by FlowDeploy. Requires scope 'resources:write'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in volumeCreateInput) (any, error) {
			if in.Name == "" {
				return nil, errInvalidArg("name is required")
			}
			body := map[string]any{"name": in.Name}
			return postJSON(ctx, deps.Backend, "/volumes", body, map[string]any{"serverId": in.ServerID})
		})
}
