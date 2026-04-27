package tools

import (
	"context"
	"net/http"
	"net/url"

	"github.com/paasdeploy/mcp/internal/backend"
	"github.com/paasdeploy/mcp/internal/toolkit"
)

func pathSeg(value string) string {
	return url.PathEscape(value)
}

func postJSON(ctx context.Context, c *backend.Client, path string, body any, query map[string]any) (any, error) {
	return doJSON(ctx, c, http.MethodPost, path, body, query)
}

func putJSON(ctx context.Context, c *backend.Client, path string, body any, query map[string]any) (any, error) {
	return doJSON(ctx, c, http.MethodPut, path, body, query)
}

func deleteJSON(ctx context.Context, c *backend.Client, path string, query map[string]any) (any, error) {
	return doJSON(ctx, c, http.MethodDelete, path, nil, query)
}

func doJSON(ctx context.Context, c *backend.Client, method, path string, body any, query map[string]any) (any, error) {
	raw, err := backend.Do[backend.Raw](ctx, c, backend.RequestOptions{
		Method: method,
		Path:   path,
		Query:  toolkit.BuildQuery(query),
		Body:   body,
	})
	if err != nil {
		return nil, err
	}
	return toolkit.DecodeBackend(raw)
}
