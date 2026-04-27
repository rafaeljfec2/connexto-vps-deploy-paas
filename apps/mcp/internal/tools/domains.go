package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/toolkit"
)

type domainListInput struct {
	AppID string `json:"app_id" jsonschema:"the app UUID"`
}

type domainAddInput struct {
	AppID      string `json:"app_id" jsonschema:"the app UUID"`
	Domain     string `json:"domain" jsonschema:"FQDN to attach to the app, e.g. api.example.com"`
	PathPrefix string `json:"path_prefix,omitempty" jsonschema:"optional path prefix; defaults to /"`
}

func RegisterDomains(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "domains_list",
			Description: "List custom domains configured for an app. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in domainListInput) (any, error) {
			if in.AppID == "" {
				return nil, errInvalidArg("app_id is required")
			}
			return getJSON(ctx, deps.Backend, "/apps/"+pathSeg(in.AppID)+"/domains", nil)
		})

	toolkit.RegisterWrite(srv, deps,
		&mcp.Tool{
			Name:        "domains_add",
			Description: "Attach a custom domain to an app. Requires scope 'config:write'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in domainAddInput) (any, error) {
			if in.AppID == "" {
				return nil, errInvalidArg("app_id is required")
			}
			if in.Domain == "" {
				return nil, errInvalidArg("domain is required")
			}
			body := map[string]any{
				"domain": in.Domain,
			}
			if in.PathPrefix != "" {
				body["pathPrefix"] = in.PathPrefix
			}
			return postJSON(ctx, deps.Backend, "/apps/"+pathSeg(in.AppID)+"/domains", body, nil)
		})
}
