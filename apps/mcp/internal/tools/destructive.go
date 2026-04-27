package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/toolkit"
)

// RegisterDestructive registers every tool that mutates persistent state in a
// non-recoverable way. All of them share the same dry-run-by-default contract:
// dry_run=true (default) returns a DryRunReport from the backend; dry_run=false
// requires a non-empty reason (8-500 chars) and produces an audit log entry.
func RegisterDestructive(srv *mcp.Server, deps toolkit.Deps) {
	registerAppsDelete(srv, deps)
	registerAppsWebhookRemove(srv, deps)
	registerContainersRemove(srv, deps)
	registerImagesRemove(srv, deps)
	registerImagesPrune(srv, deps)
	registerCleanupContainers(srv, deps)
	registerCleanupVolumes(srv, deps)
	registerNetworksRemove(srv, deps)
	registerVolumesRemove(srv, deps)
	registerDomainsRemove(srv, deps)
	registerEnvDelete(srv, deps)
	registerServersDelete(srv, deps)
}

type appsDeleteInput struct {
	toolkit.DryRunOptions
	ID    string `json:"id"`
	Purge bool   `json:"purge,omitempty"`
}

func registerAppsDelete(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterDestructive(srv, deps,
		&mcp.Tool{
			Name:        "apps_delete",
			Description: "Delete an application. Soft-delete by default; pass purge=true to remove containers/images/files. DESTRUCTIVE: defaults to dry-run preview; set commit=true and supply a reason (8-500 chars) to execute. Requires scope 'destructive'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in appsDeleteInput) (any, error) {
			if strings.TrimSpace(in.ID) == "" {
				return nil, errInvalidArg("id is required")
			}
			if err := toolkit.EnsureDestructiveCommit(in.DryRunOptions); err != nil {
				return nil, err
			}
			return deleteJSONWithHeaders(ctx, deps.Backend,
				fmt.Sprintf("/apps/%s", pathSeg(in.ID)),
				map[string]any{"purge": in.Purge},
				toolkit.DestructiveHeaders(in.DryRunOptions),
			)
		},
	)
}

type appsWebhookRemoveInput struct {
	toolkit.DryRunOptions
	AppID string `json:"app_id"`
}

func registerAppsWebhookRemove(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterDestructive(srv, deps,
		&mcp.Tool{
			Name:        "apps_webhook_remove",
			Description: "Remove the GitHub webhook of an app. DESTRUCTIVE: defaults to dry-run preview; set commit=true and supply a reason (8-500 chars) to execute. Requires scope 'destructive'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in appsWebhookRemoveInput) (any, error) {
			if strings.TrimSpace(in.AppID) == "" {
				return nil, errInvalidArg("app_id is required")
			}
			if err := toolkit.EnsureDestructiveCommit(in.DryRunOptions); err != nil {
				return nil, err
			}
			return deleteJSONWithHeaders(ctx, deps.Backend,
				fmt.Sprintf("/apps/%s/webhook", pathSeg(in.AppID)),
				nil,
				toolkit.DestructiveHeaders(in.DryRunOptions),
			)
		},
	)
}

type containersRemoveInput struct {
	toolkit.DryRunOptions
	ID       string `json:"id"`
	Force    bool   `json:"force,omitempty"`
	ServerID string `json:"server_id,omitempty"`
}

func registerContainersRemove(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterDestructive(srv, deps,
		&mcp.Tool{
			Name:        "containers_remove",
			Description: "Remove a Docker container. DESTRUCTIVE: defaults to dry-run preview; set commit=true and supply a reason (8-500 chars) to execute. Requires scope 'destructive'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in containersRemoveInput) (any, error) {
			if strings.TrimSpace(in.ID) == "" {
				return nil, errInvalidArg("id is required")
			}
			if err := toolkit.EnsureDestructiveCommit(in.DryRunOptions); err != nil {
				return nil, err
			}
			return deleteJSONWithHeaders(ctx, deps.Backend,
				fmt.Sprintf("/containers/%s", pathSeg(in.ID)),
				map[string]any{"force": in.Force, "serverId": in.ServerID},
				toolkit.DestructiveHeaders(in.DryRunOptions),
			)
		},
	)
}

type imagesRemoveInput struct {
	toolkit.DryRunOptions
	ID       string `json:"id"`
	Ref      string `json:"ref,omitempty"`
	Force    bool   `json:"force,omitempty"`
	ServerID string `json:"server_id,omitempty"`
}

func registerImagesRemove(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterDestructive(srv, deps,
		&mcp.Tool{
			Name:        "images_remove",
			Description: "Remove a Docker image by id or ref. DESTRUCTIVE: defaults to dry-run preview; set commit=true and supply a reason (8-500 chars) to execute. Requires scope 'destructive'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in imagesRemoveInput) (any, error) {
			if strings.TrimSpace(in.ID) == "" {
				return nil, errInvalidArg("id is required")
			}
			if err := toolkit.EnsureDestructiveCommit(in.DryRunOptions); err != nil {
				return nil, err
			}
			return deleteJSONWithHeaders(ctx, deps.Backend,
				fmt.Sprintf("/images/%s", pathSeg(in.ID)),
				map[string]any{"ref": in.Ref, "force": in.Force, "serverId": in.ServerID},
				toolkit.DestructiveHeaders(in.DryRunOptions),
			)
		},
	)
}

type imagesPruneInput struct {
	toolkit.DryRunOptions
	ServerID string `json:"server_id,omitempty"`
}

func registerImagesPrune(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterDestructive(srv, deps,
		&mcp.Tool{
			Name:        "images_prune",
			Description: "Remove dangling Docker images. DESTRUCTIVE: defaults to dry-run preview; set commit=true and supply a reason (8-500 chars) to execute. Requires scope 'destructive'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in imagesPruneInput) (any, error) {
			if err := toolkit.EnsureDestructiveCommit(in.DryRunOptions); err != nil {
				return nil, err
			}
			return postJSONWithHeaders(ctx, deps.Backend,
				"/images/prune",
				nil,
				map[string]any{"serverId": in.ServerID},
				toolkit.DestructiveHeaders(in.DryRunOptions),
			)
		},
	)
}

type cleanupInput struct {
	toolkit.DryRunOptions
	ServerID string `json:"server_id"`
}

func registerCleanupContainers(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterDestructive(srv, deps,
		&mcp.Tool{
			Name:        "cleanup_containers",
			Description: "Prune all stopped containers on a server. DESTRUCTIVE: defaults to dry-run preview; set commit=true and supply a reason (8-500 chars) to execute. Requires scope 'destructive'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in cleanupInput) (any, error) {
			if strings.TrimSpace(in.ServerID) == "" {
				return nil, errInvalidArg("server_id is required")
			}
			if err := toolkit.EnsureDestructiveCommit(in.DryRunOptions); err != nil {
				return nil, err
			}
			return postJSONWithHeaders(ctx, deps.Backend,
				fmt.Sprintf("/servers/%s/cleanup/containers", pathSeg(in.ServerID)),
				nil,
				nil,
				toolkit.DestructiveHeaders(in.DryRunOptions),
			)
		},
	)
}

func registerCleanupVolumes(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterDestructive(srv, deps,
		&mcp.Tool{
			Name:        "cleanup_volumes",
			Description: "Prune dangling Docker volumes on a server. DESTRUCTIVE: defaults to dry-run preview; set commit=true and supply a reason (8-500 chars) to execute. Requires scope 'destructive'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in cleanupInput) (any, error) {
			if strings.TrimSpace(in.ServerID) == "" {
				return nil, errInvalidArg("server_id is required")
			}
			if err := toolkit.EnsureDestructiveCommit(in.DryRunOptions); err != nil {
				return nil, err
			}
			return postJSONWithHeaders(ctx, deps.Backend,
				fmt.Sprintf("/servers/%s/cleanup/volumes", pathSeg(in.ServerID)),
				nil,
				nil,
				toolkit.DestructiveHeaders(in.DryRunOptions),
			)
		},
	)
}

type networksRemoveInput struct {
	toolkit.DryRunOptions
	Name     string `json:"name"`
	ServerID string `json:"server_id,omitempty"`
}

func registerNetworksRemove(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterDestructive(srv, deps,
		&mcp.Tool{
			Name:        "networks_remove",
			Description: "Remove a Docker network by name. DESTRUCTIVE: defaults to dry-run preview; set commit=true and supply a reason (8-500 chars) to execute. Requires scope 'destructive'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in networksRemoveInput) (any, error) {
			if strings.TrimSpace(in.Name) == "" {
				return nil, errInvalidArg("name is required")
			}
			if err := toolkit.EnsureDestructiveCommit(in.DryRunOptions); err != nil {
				return nil, err
			}
			return deleteJSONWithHeaders(ctx, deps.Backend,
				fmt.Sprintf("/networks/%s", pathSeg(in.Name)),
				map[string]any{"serverId": in.ServerID},
				toolkit.DestructiveHeaders(in.DryRunOptions),
			)
		},
	)
}

type volumesRemoveInput struct {
	toolkit.DryRunOptions
	Name     string `json:"name"`
	ServerID string `json:"server_id,omitempty"`
}

func registerVolumesRemove(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterDestructive(srv, deps,
		&mcp.Tool{
			Name:        "volumes_remove",
			Description: "Remove a Docker volume by name. DESTRUCTIVE: defaults to dry-run preview; set commit=true and supply a reason (8-500 chars) to execute. Requires scope 'destructive'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in volumesRemoveInput) (any, error) {
			if strings.TrimSpace(in.Name) == "" {
				return nil, errInvalidArg("name is required")
			}
			if err := toolkit.EnsureDestructiveCommit(in.DryRunOptions); err != nil {
				return nil, err
			}
			return deleteJSONWithHeaders(ctx, deps.Backend,
				fmt.Sprintf("/volumes/%s", pathSeg(in.Name)),
				map[string]any{"serverId": in.ServerID},
				toolkit.DestructiveHeaders(in.DryRunOptions),
			)
		},
	)
}

type domainsRemoveInput struct {
	toolkit.DryRunOptions
	AppID    string `json:"app_id"`
	DomainID string `json:"domain_id"`
}

func registerDomainsRemove(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterDestructive(srv, deps,
		&mcp.Tool{
			Name:        "domains_remove",
			Description: "Detach a custom domain from an app. DESTRUCTIVE: defaults to dry-run preview; set commit=true and supply a reason (8-500 chars) to execute. Requires scope 'destructive'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in domainsRemoveInput) (any, error) {
			if strings.TrimSpace(in.AppID) == "" {
				return nil, errInvalidArg("app_id is required")
			}
			if strings.TrimSpace(in.DomainID) == "" {
				return nil, errInvalidArg("domain_id is required")
			}
			if err := toolkit.EnsureDestructiveCommit(in.DryRunOptions); err != nil {
				return nil, err
			}
			return deleteJSONWithHeaders(ctx, deps.Backend,
				fmt.Sprintf("/apps/%s/domains/%s", pathSeg(in.AppID), pathSeg(in.DomainID)),
				nil,
				toolkit.DestructiveHeaders(in.DryRunOptions),
			)
		},
	)
}

type envDeleteInput struct {
	toolkit.DryRunOptions
	AppID string `json:"app_id"`
	VarID string `json:"var_id"`
}

func registerEnvDelete(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterDestructive(srv, deps,
		&mcp.Tool{
			Name:        "env_delete",
			Description: "Delete an environment variable from an app. DESTRUCTIVE: defaults to dry-run preview; set commit=true and supply a reason (8-500 chars) to execute. Requires scope 'destructive'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in envDeleteInput) (any, error) {
			if strings.TrimSpace(in.AppID) == "" {
				return nil, errInvalidArg("app_id is required")
			}
			if strings.TrimSpace(in.VarID) == "" {
				return nil, errInvalidArg("var_id is required")
			}
			if err := toolkit.EnsureDestructiveCommit(in.DryRunOptions); err != nil {
				return nil, err
			}
			return deleteJSONWithHeaders(ctx, deps.Backend,
				fmt.Sprintf("/apps/%s/env/%s", pathSeg(in.AppID), pathSeg(in.VarID)),
				nil,
				toolkit.DestructiveHeaders(in.DryRunOptions),
			)
		},
	)
}

type serversDeleteInput struct {
	toolkit.DryRunOptions
	ID string `json:"id"`
}

func registerServersDelete(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterDestructive(srv, deps,
		&mcp.Tool{
			Name:        "servers_delete",
			Description: "Deprovision and delete a server. ORPHANS apps assigned to it. DESTRUCTIVE: defaults to dry-run preview; set commit=true and supply a reason (8-500 chars) to execute. Requires scope 'destructive'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in serversDeleteInput) (any, error) {
			if strings.TrimSpace(in.ID) == "" {
				return nil, errInvalidArg("id is required")
			}
			if err := toolkit.EnsureDestructiveCommit(in.DryRunOptions); err != nil {
				return nil, err
			}
			return deleteJSONWithHeaders(ctx, deps.Backend,
				fmt.Sprintf("/servers/%s", pathSeg(in.ID)),
				nil,
				toolkit.DestructiveHeaders(in.DryRunOptions),
			)
		},
	)
}
