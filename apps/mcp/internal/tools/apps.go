package tools

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/backend"
	"github.com/paasdeploy/mcp/internal/toolkit"
)

type appsListInput struct{}

type appIDInput struct {
	ID string `json:"id" jsonschema:"the app UUID"`
}

type deployTriggerInput struct {
	ID        string `json:"id" jsonschema:"the app UUID"`
	CommitSHA string `json:"commit_sha,omitempty" jsonschema:"optional commit SHA to deploy (defaults to latest)"`
}

type deployStatusInput struct {
	ID                string `json:"id" jsonschema:"the app UUID"`
	Wait              bool   `json:"wait,omitempty" jsonschema:"when true, poll until deployment reaches a terminal state, emitting progress notifications"`
	TimeoutSeconds    int    `json:"timeout_seconds,omitempty" jsonschema:"max time to wait when wait=true (default 300, max 1800)"`
	PollIntervalSec   int    `json:"poll_interval_seconds,omitempty" jsonschema:"interval between polls in seconds when wait=true (default 5)"`
}

func RegisterApps(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "apps_list",
			Description: "List all FlowDeploy apps the authenticated user can access. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ appsListInput) (any, error) {
			return getJSON(ctx, deps.Backend, "/apps", nil)
		})

	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "apps_get",
			Description: "Get details for a single app by ID. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in appIDInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			return getJSON(ctx, deps.Backend, "/apps/"+pathSeg(in.ID), nil)
		})

	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "apps_deployments",
			Description: "List deployment history for an app. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in appIDInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			return getJSON(ctx, deps.Backend, "/apps/"+pathSeg(in.ID)+"/deployments", nil)
		})

	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "apps_commits",
			Description: "List recent commits for the app's connected repository. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in appIDInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			return getJSON(ctx, deps.Backend, "/apps/"+pathSeg(in.ID)+"/commits", nil)
		})

	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "apps_health",
			Description: "Get container health summary for an app. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in appIDInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			return getJSON(ctx, deps.Backend, "/apps/"+pathSeg(in.ID)+"/health", nil)
		})
}

func RegisterDeploys(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterWrite(srv, deps,
		&mcp.Tool{
			Name:        "deploy_trigger",
			Description: "Trigger a new deploy for an app. Optionally targets a specific commit SHA. Requires scope 'deploy'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in deployTriggerInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			body := map[string]any{}
			if in.CommitSHA != "" {
				body["commitSha"] = in.CommitSHA
			}
			return postJSON(ctx, deps.Backend, "/apps/"+pathSeg(in.ID)+"/redeploy", body, nil)
		})

	toolkit.RegisterWrite(srv, deps,
		&mcp.Tool{
			Name:        "deploy_rollback",
			Description: "Rollback the app to its previous successful deployment. Requires scope 'deploy'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in appIDInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			return postJSON(ctx, deps.Backend, "/apps/"+pathSeg(in.ID)+"/rollback", map[string]any{}, nil)
		})

	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "deploy_status",
			Description: "Return the latest deployments and current status for an app. When wait=true, polls until terminal state (success/failed) is reached, emitting MCP progress notifications. Requires scope 'read'.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, in deployStatusInput) (any, error) {
			if in.ID == "" {
				return nil, errInvalidArg("id is required")
			}
			if !in.Wait {
				return getJSON(ctx, deps.Backend, "/apps/"+pathSeg(in.ID)+"/deployments", nil)
			}
			return waitForDeployment(ctx, req, deps, in)
		})
}

func getJSON(ctx context.Context, c *backend.Client, path string, query map[string]any) (any, error) {
	raw, err := backend.Do[backend.Raw](ctx, c, backend.RequestOptions{
		Method: http.MethodGet,
		Path:   path,
		Query:  toolkit.BuildQuery(query),
	})
	if err != nil {
		return nil, err
	}
	return toolkit.DecodeBackend(raw)
}
