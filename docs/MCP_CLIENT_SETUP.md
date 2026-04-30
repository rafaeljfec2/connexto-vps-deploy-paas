# FlowDeploy MCP — client setup

The FlowDeploy MCP server (`apps/mcp/cmd/flowdeploy-mcp`) exposes the FlowDeploy control plane to AI agents over the [Model Context Protocol](https://modelcontextprotocol.io/).

The server speaks two transports:

- **stdio** (default) — single-PAT, single-process. Ideal for local agents (Cursor, Claude Desktop, Zed).
- **HTTP + SSE** (`flowdeploy-mcp serve`) — multi-tenant, PAT forwarded per request. Ideal for remote agents (CI runners, hosted assistants, custom agent platforms). TLS termination is delegated to Traefik.

Phases 1–3 implemented all tools, resources, prompts, and the dry-run + reason contract for destructive operations. Phase 4 added the HTTP transport, per-PAT rate limiting, structured slog, Prometheus metrics, and Traefik wiring.

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

The CLI uses subcommands. Run `flowdeploy-mcp <subcommand> --help` to discover the flags supported by each.

```
flowdeploy-mcp [stdio|serve] [flags]
```

When no subcommand is given, `stdio` is assumed (backward compatible).

### Common flags (both subcommands)

| Flag | Env var | Default | Description |
| --- | --- | --- | --- |
| `--backend-url` | `FLOWDEPLOY_BACKEND_URL` | _required_ | Base URL of the FlowDeploy backend (without `/paas-deploy/v1`). |
| `--token` | `FLOWDEPLOY_TOKEN` | _required_ in `stdio`, optional in `serve` | Personal Access Token. In `serve` mode the agent's PAT is forwarded per request. |
| `--log-level` | `FLOWDEPLOY_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error`. |
| `--client` | `FLOWDEPLOY_MCP_CLIENT` | `custom:flowdeploy-mcp` | Default `X-MCP-Client` value (only used in `stdio` mode; `serve` honours the agent's header). |
| `--request-timeout` | — | `15s` | Per-request HTTP timeout against the backend. |

### `serve` flags

| Flag | Env var | Default | Description |
| --- | --- | --- | --- |
| `--addr` | `FLOWDEPLOY_MCP_ADDR` | `:3001` | TCP listener address. TLS is **terminated by Traefik**; expose this address only on a private network. |
| `--read-rpm` | — | `120` | Per-PAT quota for read tools (rolling minute). |
| `--mutate-rpm` | — | `20` | Per-PAT quota for mutating tools (rolling minute). |
| `--session-max-age` | — | `30m` | Idle timeout for streamable MCP sessions. |
| `--allowed-clients` | — | `cursor,claude-desktop,custom:*,ci:*` | Comma-separated allowlist of `X-MCP-Client` values. Trailing `*` enables prefix matching (e.g. `ci:github-actions`). |
| `--stateless-session` | — | `false` | Run the streamable transport in stateless mode (each request is a fresh session). |

The token is sent on every backend call as `Authorization: Bearer pdp_live_...`. Logs only record the SHA-256 prefix (16 hex chars), never the raw token.

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

## Remote (HTTP + SSE) deployment

The `serve` subcommand opens the MCP server to remote agents. TLS is terminated by Traefik and the binary listens on plain HTTP behind the proxy. The default deploy in [`deploy/docker-compose.yml`](../deploy/docker-compose.yml) ships the `mcp` service ready to go; configure these env vars before `docker compose up -d`:

| Env var | Purpose |
| --- | --- |
| `FLOWDEPLOY_BACKEND_URL` | URL of the FlowDeploy backend reachable from the MCP container (defaults to `http://backend:8080`). |
| `MCP_HOST` | Public hostname for Traefik (e.g. `mcp.flowdeploy.example.com`). |
| `FLOWDEPLOY_MCP_ALLOWED_CLIENTS` | Override the default allowlist if you need to restrict who can connect. |
| `FLOWDEPLOY_MCP_READ_RPM` / `FLOWDEPLOY_MCP_MUTATE_RPM` | Override per-PAT quotas. |
| `ACME_EMAIL` | Used by Traefik to issue the Let's Encrypt certificate. |

### Direct invocation

```bash
flowdeploy-mcp serve \
  --addr=:3001 \
  --backend-url=http://backend:8080 \
  --read-rpm=120 \
  --mutate-rpm=20 \
  --allowed-clients=cursor,claude-desktop,custom:*,ci:*
```

### Endpoints

- `POST /mcp` — Streamable HTTP MCP transport (also returns SSE for server-initiated messages).
- `GET /mcp` — Standalone SSE channel for server-initiated notifications when the client opts in.
- `GET /healthz` — liveness probe (always 200 once the listener is up).
- `GET /readyz` — readiness probe (200 when downstream checks pass).
- `GET /metrics` — Prometheus exposition format with the following collectors:
  - `mcp_tool_calls_total{tool,mode,status}`
  - `mcp_tool_duration_seconds{tool,mode}`
  - `mcp_http_requests_total{method,path,status}`
  - `mcp_http_request_duration_seconds{method,path}`
  - `mcp_ratelimit_drops_total{bucket}`
  - `mcp_auth_failures_total{reason}`
  - `mcp_bucket_classify_failures_total{reason}` — bodies that could not be parsed and were degraded conservatively to mutate (`overflow`, `parse_error`, `read_error`).
  - `mcp_in_flight_requests` — gauge tracking concurrent HTTP requests.

### Required headers

Each agent request MUST carry:

- `Authorization: Bearer pdp_live_...` — the agent's own PAT (forwarded to the backend).
- `X-MCP-Client: <id>` — must match the allowlist (`cursor`, `claude-desktop`, or a wildcard suffix like `ci:github-actions`).
- `X-MCP-Bucket: mutate` — optional override that ALWAYS bills the request against the mutation quota (`destructive` is accepted as a synonym). The transport already classifies `tools/call` payloads server-side by inspecting the JSON-RPC body, so this header is only needed when proxies hide the body or when the client wants to pre-throttle a batch of writes. The header can only UPGRADE a request to `mutate`; it is silently ignored when sent as `X-MCP-Bucket: read` for a destructive call.

### Connecting from Cursor (remote)

```json
{
  "mcpServers": {
    "flowdeploy-remote": {
      "url": "https://mcp.flowdeploy.example.com/mcp",
      "headers": {
        "Authorization": "Bearer pdp_live_...",
        "X-MCP-Client": "cursor"
      }
    }
  }
}
```

### Connecting from a CI runner (curl smoke)

```bash
curl -sS https://mcp.flowdeploy.example.com/mcp \
  -H 'Authorization: Bearer pdp_live_...' \
  -H 'X-MCP-Client: ci:github-actions' \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"ci","version":"1.0"}}}'
```

A successful response is a JSON-RPC envelope containing `serverInfo.name = "flowdeploy"`. Subsequent calls reuse the `Mcp-Session-Id` header returned by the server.

## Deploying via GitHub Actions

The repository ships [`/.github/workflows/deploy-mcp.yml`](../.github/workflows/deploy-mcp.yml) which builds the MCP image, pushes to GHCR and deploys it on the production VPS via the self-hosted GitHub Actions runner (label `flowdeploy-control-plane`). It mirrors `deploy-backend.yml` but only manages the `flowdeploy-mcp` container.

### One-time setup (per environment)

1. **DNS** — create an `A` record pointing the public hostname (e.g. `mcp-deploy.connexto.com.br`) to the same IP as the backend; Traefik already accepts wildcard SANs there.
2. **GitHub → Settings → Environments → Prod → Variables** — add:
   - `MCP_HOST=mcp-deploy.connexto.com.br`
   - `FLOWDEPLOY_BACKEND_URL=http://flowdeploy-backend:8080` (internal docker DNS; the MCP runs on the same `paasdeploy` network as the backend)
   - `FLOWDEPLOY_MCP_ALLOWED_CLIENTS=cursor,claude-desktop,cline`
   - `FLOWDEPLOY_MCP_READ_RPM=120` *(optional; default applied if omitted)*
   - `FLOWDEPLOY_MCP_MUTATE_RPM=20` *(optional; default applied if omitted)*
   - `FLOWDEPLOY_MCP_LOG_LEVEL=info` *(optional)*
   - `FLOWDEPLOY_MCP_SESSION_MAX_AGE=30m` *(optional)*

   The shared `GHCR_USER` (var) and `GHCR_PAT` (secret) come from the backend deploy and do not need to be duplicated. No `SERVER_*`/`SERVER_PASSWORD` is required: the MCP deploy runs on the same self-hosted runner as the backend, talking to the local Docker daemon directly.

### Trigger

The workflow runs automatically on `push` to `main` whenever any of the following changes:

- `apps/mcp/**`
- `deploy/docker-compose.yml`
- `.github/workflows/deploy-mcp.yml`
- `.github/scripts/deploy-mcp.sh`

It can also be dispatched manually from the GitHub Actions UI.

### Smoke test after deploy

```bash
# Public endpoint must reject without a PAT
curl -i https://mcp-deploy.connexto.com.br/mcp
# → HTTP/2 401 (auth_required)

# Internal endpoints stay private (only reachable via docker exec on the VPS)
ssh user@vps 'docker exec flowdeploy-mcp wget -qO- http://127.0.0.1:3001/healthz'
# → {"status":"ok"}

# After issuing a PAT in the dashboard, run an initialize round-trip:
curl -sS https://mcp-deploy.connexto.com.br/mcp \
  -H "Authorization: Bearer $PDP_TOKEN" \
  -H 'X-MCP-Client: ci:smoke' \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"smoke","version":"1.0"}}}'
```

### Rollback

To roll back to a previous image tag without re-running the build:

```bash
ssh user@vps
docker pull ghcr.io/<org>/<repo>-mcp:<short-sha>
docker stop flowdeploy-mcp && docker rm flowdeploy-mcp
# re-run docker run with the desired tag (or just re-trigger the workflow on the previous commit)
```

The MCP is stateless across restarts (rate-limit windows reset, no persistent storage), so a rollback is safe and instantaneous.

## Troubleshooting

| Symptom | Likely cause |
| --- | --- |
| `token must start with pdp_live_` | The token in the env/flag is missing the prefix or was truncated. |
| `backend 401 UNAUTHORIZED` | The PAT was revoked, expired or has no `read` scope. |
| `backend 403 FORBIDDEN` | The PAT user no longer has access to the resource. |
| `dial tcp ... connection refused` | The backend URL is wrong or the backend is not running. |
| Cursor does not show the server | Ensure `.cursor/mcp.json` is at the workspace root and the JSON is valid. |
| `401 missing_token` (HTTP mode) | The agent did not send `Authorization: Bearer pdp_live_...`. |
| `400 missing_client_header` (HTTP mode) | The agent did not send `X-MCP-Client`. |
| `403 client_not_allowed` (HTTP mode) | The `X-MCP-Client` value is not in `--allowed-clients`. |
| `429 rate_limited` (HTTP mode) | The PAT exceeded `--read-rpm` / `--mutate-rpm` in the last rolling minute. |
