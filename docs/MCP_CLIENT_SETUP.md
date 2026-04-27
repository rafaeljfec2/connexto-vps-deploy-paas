# FlowDeploy MCP — client setup

The FlowDeploy MCP server (`apps/mcp/cmd/flowdeploy-mcp`) exposes the FlowDeploy control plane to AI agents over the [Model Context Protocol](https://modelcontextprotocol.io/).

Phase 1 ships **read-only** tools, resources and prompts over the **stdio** transport. Phase 4 will add HTTP+SSE for remote agents.

## Prerequisites

1. Go 1.25+ installed locally (the SDK requires Go 1.25; the rest of the monorepo still uses Go 1.24).
2. A FlowDeploy Personal Access Token (PAT) with at least the `read` scope.
   - Create one at `Settings → Personal Access Tokens` (or `POST /paas-deploy/v1/tokens`).
   - Tokens start with `pdp_live_` and are shown **once** at creation time.
3. The FlowDeploy backend reachable from the machine that runs the MCP client. The `--backend-url`/`FLOWDEPLOY_BACKEND_URL` value must include the scheme and **must NOT** include the `/paas-deploy/v1` suffix — the server appends it automatically.

## Build

```bash
cd apps/mcp
go build ./cmd/flowdeploy-mcp
# binary: ./flowdeploy-mcp
```

## Configuration

| Flag | Env var | Default | Description |
| --- | --- | --- | --- |
| `--stdio` | — | `true` | Use stdio transport. Required for Phase 1. |
| `--backend-url` | `FLOWDEPLOY_BACKEND_URL` | _required_ | Base URL of the FlowDeploy backend. |
| `--token` | `FLOWDEPLOY_TOKEN` | _required_ | Personal Access Token. |
| `--log-level` | `FLOWDEPLOY_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error`. |
| `--client` | `FLOWDEPLOY_MCP_CLIENT` | `custom:flowdeploy-mcp` | Identifies the MCP client in audit logs. |
| `--request-timeout` | — | `15s` | Per-request HTTP timeout. |

The token is sent on every backend call as `Authorization: Bearer pdp_live_...`. Logs only record the redacted prefix.

## Cursor

The repository ships [`/.cursor/mcp.json`](../.cursor/mcp.json) at the workspace root. Edit it once to set your token (or define `FLOWDEPLOY_TOKEN` in your shell):

```json
{
  "mcpServers": {
    "flowdeploy": {
      "command": "go",
      "args": ["run", "./apps/mcp/cmd/flowdeploy-mcp", "--stdio"],
      "env": {
        "FLOWDEPLOY_BACKEND_URL": "http://localhost:8080",
        "FLOWDEPLOY_TOKEN": "pdp_live_...",
        "FLOWDEPLOY_LOG_LEVEL": "info",
        "FLOWDEPLOY_MCP_CLIENT": "cursor"
      }
    }
  }
}
```

After editing, reload Cursor. The `flowdeploy` server should appear under `Settings → MCP`. The Composer agent will be able to call `apps_list`, `containers_list`, etc.

## Claude Desktop

Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "flowdeploy": {
      "command": "/absolute/path/to/flowdeploy-mcp",
      "args": ["--stdio"],
      "env": {
        "FLOWDEPLOY_BACKEND_URL": "https://api.flowdeploy.example.com",
        "FLOWDEPLOY_TOKEN": "pdp_live_...",
        "FLOWDEPLOY_MCP_CLIENT": "claude-desktop"
      }
    }
  }
}
```

Restart Claude Desktop afterwards.

## Zed

Edit `~/.config/zed/settings.json`:

```json
{
  "context_servers": {
    "flowdeploy": {
      "command": {
        "path": "/absolute/path/to/flowdeploy-mcp",
        "args": ["--stdio"],
        "env": {
          "FLOWDEPLOY_BACKEND_URL": "https://api.flowdeploy.example.com",
          "FLOWDEPLOY_TOKEN": "pdp_live_...",
          "FLOWDEPLOY_MCP_CLIENT": "zed"
        }
      }
    }
  }
}
```

## Available tools (Phase 1)

All read-only, all require scope `read`:

- **Apps**: `apps_list`, `apps_get`, `apps_deployments`, `apps_commits`, `apps_health`.
- **Containers**: `containers_list`, `containers_get`, `containers_logs` (`tail`, `since`, `server_id`).
- **Servers**: `servers_list`, `servers_get`, `servers_stats`, `servers_health`, `servers_apps`.
- **Images**: `images_list`, `images_dangling`.
- **Resources**: `networks_list`, `volumes_list`.
- **Templates**: `templates_list`, `templates_get`.
- **GitHub**: `github_installations`, `github_repos`, `github_repo`.
- **Audit**: `audit_logs`, `audit_webhook_payloads`.
- **System**: `system_stats`.

## MCP resources

- `flowdeploy://apps` — list of apps.
- `flowdeploy://apps/{id}` — single app.
- `flowdeploy://servers/{id}` — single server.
- `flowdeploy://system` — system stats.

## MCP prompts

- `diagnose_app(app_id)` — orchestrates `apps_get` + `apps_deployments` + `apps_health` + `containers_logs` to produce a root-cause hypothesis.
- `audit_recent_changes(limit?)` — summarises the last 24h of audit logs grouped by actor.

## Smoke test

Without an MCP client, you can exercise the server using `mcp-inspector` or any MCP-aware client. Quick connectivity check from a shell:

```bash
export FLOWDEPLOY_BACKEND_URL=http://localhost:8080
export FLOWDEPLOY_TOKEN=pdp_live_...
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"smoke","version":"1.0.0"}}}' | go run ./apps/mcp/cmd/flowdeploy-mcp --stdio
```

You should see a JSON-RPC response containing `serverInfo.name = "flowdeploy"`.

## Troubleshooting

| Symptom | Likely cause |
| --- | --- |
| `token must start with pdp_live_` | The token in the env/flag is missing the prefix or was truncated. |
| `backend 401 UNAUTHORIZED` | The PAT was revoked, expired or has no `read` scope. |
| `backend 403 FORBIDDEN` | The PAT user no longer has access to the resource. |
| `dial tcp ... connection refused` | The backend URL is wrong or the backend is not running. |
| Cursor does not show the server | Ensure `.cursor/mcp.json` is at the workspace root and the JSON is valid. |
