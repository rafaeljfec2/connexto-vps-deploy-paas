package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/toolkit"
)

type envListInput struct {
	AppID string `json:"app_id" jsonschema:"the app UUID"`
}

type envUpsertInput struct {
	AppID  string `json:"app_id" jsonschema:"the app UUID"`
	Key    string `json:"key" jsonschema:"environment variable key"`
	Value  string `json:"value" jsonschema:"environment variable value"`
	Secret bool   `json:"secret,omitempty" jsonschema:"whether the value should be stored as a secret"`
}

type envBulkEntry struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"isSecret,omitempty"`
}

type envBulkInput struct {
	AppID   string         `json:"app_id" jsonschema:"the app UUID"`
	Entries []envBulkEntry `json:"entries" jsonschema:"list of env var entries to upsert (each entry: key, value, isSecret)"`
}

func RegisterEnvVars(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "env_list",
			Description: "List environment variables configured for an app. Secret values are masked. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in envListInput) (any, error) {
			if in.AppID == "" {
				return nil, errInvalidArg("app_id is required")
			}
			return getJSON(ctx, deps.Backend, "/apps/"+pathSeg(in.AppID)+"/env", nil)
		})

	toolkit.RegisterWrite(srv, deps,
		&mcp.Tool{
			Name:        "env_upsert",
			Description: "Create or update a single environment variable for an app. Requires scope 'config:write'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in envUpsertInput) (any, error) {
			if in.AppID == "" {
				return nil, errInvalidArg("app_id is required")
			}
			if in.Key == "" {
				return nil, errInvalidArg("key is required")
			}
			body := map[string]any{
				"key":      in.Key,
				"value":    in.Value,
				"isSecret": in.Secret,
			}
			return postJSON(ctx, deps.Backend, "/apps/"+pathSeg(in.AppID)+"/env", body, nil)
		})

	toolkit.RegisterWrite(srv, deps,
		&mcp.Tool{
			Name:        "env_bulk",
			Description: "Bulk upsert environment variables for an app. Requires scope 'config:write'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in envBulkInput) (any, error) {
			if in.AppID == "" {
				return nil, errInvalidArg("app_id is required")
			}
			if len(in.Entries) == 0 {
				return nil, errInvalidArg("entries must not be empty")
			}
			body := map[string]any{
				"vars": in.Entries,
			}
			return putJSON(ctx, deps.Backend, "/apps/"+pathSeg(in.AppID)+"/env/bulk", body, nil)
		})
}
