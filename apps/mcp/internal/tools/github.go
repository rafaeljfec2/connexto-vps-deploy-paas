package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/toolkit"
)

type githubReposInput struct {
	InstallationID string `json:"installation_id,omitempty" jsonschema:"optional GitHub App installation ID"`
}

type githubRepoInput struct {
	Owner string `json:"owner" jsonschema:"GitHub owner (user or organization)"`
	Repo  string `json:"repo" jsonschema:"GitHub repository name"`
}

func RegisterGitHub(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "github_installations",
			Description: "List GitHub App installations the user has connected. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (any, error) {
			return getJSON(ctx, deps.Backend, "/github/installations", nil)
		})

	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "github_repos",
			Description: "List repositories accessible through the GitHub App installation. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in githubReposInput) (any, error) {
			return getJSON(ctx, deps.Backend, "/github/repos", map[string]any{"installationId": in.InstallationID})
		})

	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "github_repo",
			Description: "Get a single GitHub repository's metadata. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in githubRepoInput) (any, error) {
			if in.Owner == "" || in.Repo == "" {
				return nil, errInvalidArg("owner and repo are required")
			}
			return getJSON(ctx, deps.Backend, "/github/repos/"+pathSeg(in.Owner)+"/"+pathSeg(in.Repo), nil)
		})
}
