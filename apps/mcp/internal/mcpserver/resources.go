package mcpserver

import (
	"context"
	"net/http"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/backend"
	"github.com/paasdeploy/mcp/internal/toolkit"
)

func RegisterMCPResources(srv *mcp.Server, deps toolkit.Deps) {
	srv.AddResource(
		&mcp.Resource{
			URI:         "flowdeploy://apps",
			Name:        "All apps",
			Description: "List of all apps managed by FlowDeploy.",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			data, err := backend.Do[backend.Raw](ctx, deps.Backend, backend.RequestOptions{
				Method: http.MethodGet,
				Path:   "/apps",
			})
			if err != nil {
				return nil, err
			}
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{{
					URI:      "flowdeploy://apps",
					MIMEType: "application/json",
					Text:     string(data),
				}},
			}, nil
		},
	)

	srv.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "flowdeploy://apps/{id}",
			Name:        "App",
			Description: "Detail view of a single app by ID.",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			id := extractTemplatePart(req.Params.URI, "flowdeploy://apps/")
			if id == "" {
				return nil, mcp.ResourceNotFoundError(req.Params.URI)
			}
			data, err := backend.Do[backend.Raw](ctx, deps.Backend, backend.RequestOptions{
				Method: http.MethodGet,
				Path:   "/apps/" + url.PathEscape(id),
			})
			if err != nil {
				return nil, err
			}
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(data),
				}},
			}, nil
		},
	)

	srv.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "flowdeploy://servers/{id}",
			Name:        "Server",
			Description: "Detail view of a single server (host or remote VPS).",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			id := extractTemplatePart(req.Params.URI, "flowdeploy://servers/")
			if id == "" {
				return nil, mcp.ResourceNotFoundError(req.Params.URI)
			}
			data, err := backend.Do[backend.Raw](ctx, deps.Backend, backend.RequestOptions{
				Method: http.MethodGet,
				Path:   "/servers/" + url.PathEscape(id),
			})
			if err != nil {
				return nil, err
			}
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(data),
				}},
			}, nil
		},
	)

	srv.AddResource(
		&mcp.Resource{
			URI:         "flowdeploy://system",
			Name:        "System stats",
			Description: "Aggregated system stats for the FlowDeploy host.",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			data, err := backend.Do[backend.Raw](ctx, deps.Backend, backend.RequestOptions{
				Method: http.MethodGet,
				Path:   "/system/stats",
			})
			if err != nil {
				return nil, err
			}
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{{
					URI:      "flowdeploy://system",
					MIMEType: "application/json",
					Text:     string(data),
				}},
			}, nil
		},
	)
}

func extractTemplatePart(uri, prefix string) string {
	if len(uri) <= len(prefix) || uri[:len(prefix)] != prefix {
		return ""
	}
	return uri[len(prefix):]
}
