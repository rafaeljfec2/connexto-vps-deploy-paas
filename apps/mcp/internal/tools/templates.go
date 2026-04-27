package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/toolkit"
)

type templateIDInput struct {
	ID string `json:"id" jsonschema:"the template identifier (e.g. 'postgres', 'minio')"`
}

type templatePortMapping struct {
	HostPort      int    `json:"hostPort"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol,omitempty"`
}

type templateDeployInput struct {
	ID            string                `json:"id" jsonschema:"the template identifier"`
	Name          string                `json:"name" jsonschema:"name to give to the deployed instance"`
	ServerID      string                `json:"server_id,omitempty" jsonschema:"optional remote server ID; defaults to host"`
	Env           map[string]string     `json:"env,omitempty" jsonschema:"environment variables (key/value pairs of strings)"`
	Ports         []templatePortMapping `json:"ports,omitempty" jsonschema:"port mappings (hostPort, containerPort, protocol)"`
	Network       string                `json:"network,omitempty" jsonschema:"Docker network to attach the container to"`
	RestartPolicy string                `json:"restart_policy,omitempty" jsonschema:"Docker restart policy (e.g. 'always', 'unless-stopped')"`
	Command       []string              `json:"command,omitempty" jsonschema:"override the container command"`
}

func RegisterTemplates(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "templates_list",
			Description: "List available service templates (Postgres, Redis, MinIO, ...). Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (any, error) {
			return getJSON(ctx, deps.Backend, "/templates", nil)
		})

	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "templates_get",
			Description: "Get a single template definition with its variables. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in templateIDInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			return getJSON(ctx, deps.Backend, "/templates/"+pathSeg(in.ID), nil)
		})

	toolkit.RegisterWrite(srv, deps,
		&mcp.Tool{
			Name:        "templates_deploy",
			Description: "Deploy a template (e.g. Postgres, Redis, MinIO) to the host or a remote server. Requires scope 'deploy'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in templateDeployInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			if in.Name == "" {
				return nil, errInvalidArg("name is required")
			}
			body := map[string]any{"name": in.Name}
			if len(in.Env) > 0 {
				body["env"] = in.Env
			}
			if len(in.Ports) > 0 {
				body["ports"] = in.Ports
			}
			if in.Network != "" {
				body["network"] = in.Network
			}
			if in.RestartPolicy != "" {
				body["restartPolicy"] = in.RestartPolicy
			}
			if len(in.Command) > 0 {
				body["command"] = in.Command
			}
			return postJSON(ctx, deps.Backend, "/templates/"+pathSeg(in.ID)+"/deploy", body, map[string]any{
				"serverId": in.ServerID,
			})
		})
}
