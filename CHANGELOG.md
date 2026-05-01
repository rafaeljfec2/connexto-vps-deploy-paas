# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> Versions track the platform `package.json`. The agent has its own version
> in `AGENT_VERSION` and is mentioned in each entry when bumped.

## [Unreleased]

### Added

- **Container logs `since` parameter is honoured end-to-end.** Previously the
  proto field `flowdeploy.v1.ContainerLogsRequest.since` (declared at
  `apps/proto/flowdeploy/v1/server.proto:181`) was wired only at the proto
  layer: the agent gRPC handler, shared docker wrapper, backend agent client
  and both backend HTTP handlers (`container_logs_handler.go`,
  `container_health_handler.go`) silently dropped the value. The MCP tool
  declared `since` in its input schema and forwarded it to the backend, but
  the backend's query parser did not read `?since=`. Net effect: the MCP
  `containers_logs` tool with `since=...` returned the full tail window
  every time, and `logs_follow.go`'s polling loop re-pulled the same lines
  on every tick. Now: `apps/shared/pkg/docker/client.go.{ContainerLogs,
  StreamContainerLogs}` accept `since *time.Time` and emit
  `--since=<RFC3339-UTC>` on the `docker logs` CLI; the agent
  `GetContainerLogs` handler reads `req.Since.AsTime()`; the backend agent
  client builds the proto request with `timestamppb.New(since.UTC())`; both
  HTTP handlers parse `?since=` as RFC3339 and reject malformed input with
  `400 Bad Request` (`ParseSinceQuery` helper); the MCP tool accepts both
  RFC3339 absolute (`2026-04-30T19:00:00Z`) and Go duration shorthand (`1h`,
  `30m`, `1h30m`), normalises to RFC3339-UTC before forwarding, rejects
  `0`/negative durations and unparseable strings with `errInvalidArg`, and
  returns 0 backend calls on validation failure.   Backwards-compatible:
  callers passing `nil` (or omitting the query param) get the old behavior
  unchanged. One incidental normalisation in
  `ContainerHealthHandler.getRemoteContainerLogs`: the response payload
  now joins log entries with `\n` (matching the sibling
  `ContainerHandler.getRemoteContainerLogs`) instead of appending
  `entry.Message + "\n"` per line; the previous trailing newline is gone.
  No client in this repo depends on it; UI rendering is unaffected.
  Existing tests across the chain remain green; new tests cover every
  layer (`buildLogsArgs`, `extractSinceFromRequest`,
  `buildContainerLogsRequest`, `ParseSinceQuery`, `normaliseSince`).
  **AGENT_VERSION bumped 0.22.2 → 0.22.3** because remote agents must
  upgrade to honour `req.Since`; older agents continue to serve the full
  tail (graceful degradation, no error).
- **`make install-lint` target.** Builds `apps/backend/bin/golangci-lint`
  with `GOTOOLCHAIN=go1.24.13`, the version the project targets. A
  `golangci-lint` binary built with Go < 1.24 silently exits non-zero on
  the first lint run and the pre-commit hook quietly skips Go lint —
  removing developer protection. The `lint-go.sh` "not found" error
  message and the pre-commit hook's "older Go version" warning now both
  point to `make install-lint`. Variables `GOLANGCI_LINT_VERSION` (default
  `v1.64.8` — earlier v1.61.x silently fails to load Go 1.24 export-data
  even when rebuilt with the right toolchain, because the bundled
  `golang.org/x/tools` is too old) and `GO_TOOLCHAIN` (default
  `go1.24.13`) can be overridden from the command line. Goes a touch
  beyond the original 🟢 tech-debt scope ("rebuild bin/golangci-lint on
  the CI runner") because the same root cause hits every developer's
  machine — fixing it once at the Makefile level is cheaper than telling
  each contributor to remember the install incantation. CI is unaffected
  by this binary either way: the existing GitHub Actions workflows
  (`.github/workflows/deploy-{backend,mcp}.yml`) **do not run a Go lint
  job today** — that's a separate gap tracked as future tech-debt; this
  PR only restores the local pre-commit safety net.
- **Personal Access Tokens (PAT)**: users can mint, list and revoke
  long-lived API tokens (`pdp_live_…`) from the dashboard. Tokens are
  hashed at rest, scoped per user and rejected from token-management
  endpoints by the `DenyPAT` middleware. Backend service, repository,
  HTTP handler and frontend feature (`features/tokens/**`) included.
- **PAT audit trail**: `PATHandler.Create` and `PATHandler.Revoke` now
  emit `token.created` / `token.revoked` audit log entries with
  `resource_type=token`, `actor_type=user`, the token id and name, and
  `{scopes, expires_at}` details on creation. Failures (validation /
  repository errors) do NOT emit audit, matching the behavior of every
  other write path in the backend.
- **Dry-run middleware for destructive operations**: write-mode endpoints
  honour an `X-FlowDeploy-Dry-Run: true` header to short-circuit the
  effect and return a structured preview envelope. Wired across app,
  container, image, env, server, template and webhook handlers.
- **MCP server (`apps/mcp`)**: new self-contained Go service exposing the
  FlowDeploy public API as Model Context Protocol tools. Ships both
  `stdio` (single-PAT, single-process) and `serve` (HTTP, multi-tenant)
  transports, with per-PAT rate limits, allowed-clients allowlist and a
  Traefik route at `https://${MCP_HOST}/mcp`. `/metrics`, `/healthz` and
  `/readyz` stay internal to the `paasdeploy` network.
- `audit_logs.actor_type` + `actor_logs.actor_id` columns (already in
  migrations `000029`/`000030` on `main`) are now consumed end-to-end:
  every audit entry records whether the action came from a `user`, `pat`,
  `system` or `webhook` actor.

### Fixed

- **MCP `audit_logs` filters were silently ignored**: the tool declared
  `actor_type`, `actor_id`, `action`, `from`, `to` in its input schema and
  forwarded them to the backend as `actorType`, `actorId`, `action`, `from`,
  `to` — but `apps/backend/internal/handler/audit_handler.go` only honours
  `eventType`, `resourceType`, `resourceId`, `startDate`, `endDate`. Result:
  every filter except `limit` was a no-op (the tool returned the unfiltered
  page regardless of arguments). Inputs renamed to `event_type`,
  `resource_type`, `resource_id`, `start_date`, `end_date` (snake_case stays
  the MCP convention) and the downstream map now uses the camelCase keys the
  backend actually reads. Added `offset` for pagination. `user_id` was
  intentionally NOT exposed because the backend always overrides
  `filter.UserID` to the authenticated caller (audit_handler.go:62-63),
  so accepting it client-side would just silently no-op like the legacy
  fields did. Existing test (`TestAuditLogsForwardsFilters`) was
  green-but-wrong (only validated MCP→HTTP forwarding for parameters the
  backend ignored); replaced by
  `TestAuditLogsForwardsFiltersToBackendCamelCase`,
  `TestAuditLogsOmitsEmptyFiltersFromQuery`,
  `TestAuditLogsRejectsUnknownFilters` (asserts the schema rejects the
  removed `user_id`) and
  `TestAuditWebhookPayloadsForwardsLimitAndOffset`.
- **MCP `audit_recent_changes` prompt referenced removed `from` field**:
  the prompt template (`apps/mcp/internal/mcpserver/prompts.go`)
  instructed the model to call `audit_logs(limit=N, from=24h-ago)` —
  but `from` was never read by the backend and is no longer part of the
  tool schema (which now uses `additionalProperties:false`). Fix: the
  prompt now computes the actual RFC3339 timestamp 24 hours ago and
  passes `start_date=<timestamp>`, and groups by `actorType` /
  `eventType` (the field names actually returned in the response).
- **`deploy.sh` and `deploy-mcp.sh` had no retry against `ghcr.io`**: 3
  consecutive backend deploys (commits `a17888c`, `5af7abd`, `1e4e83e`)
  failed with `dial tcp 4.228.31.152:443: i/o timeout` on `docker login` /
  `docker pull`, leaving production stuck on `sha-203514d` (without the
  PAT audit-trail and Guardian fixes). Both deploy scripts (backend and
  MCP) gained a `retry()` bash helper with fixed backoff (5s before
  attempt 2, 15s before attempt 3) and now wrap the two network-bound
  steps (`docker login`, `docker pull`) in 3 attempts. The
  `backoffs=(5 15 30)` array carries a 30s slot for any future bump of
  `max`. Deterministic steps (`docker run`, healthcheck, rollback) are
  NOT retried because masking their failures would hide real bugs (bad
  image, bad env, container unhealthy). The helper is duplicated rather
  than extracted to a shared `lib/`; if it grows, refactor to a
  source-able file.
- **PAT creation returning 500 in production**: the `scopes` column of
  `personal_access_tokens` existed in the production database as
  `text[]` (the table was created out-of-band before the `000029`
  migration file landed), while the Go repository marshals scopes as
  JSON before the INSERT. Migration `000031_fix_personal_access_tokens_scopes_type`
  converts the column to JSONB in place (idempotent: no-op on fresh
  installs where 000029 already provisioned it as JSONB), dropping the
  default first because Postgres cannot auto-cast a `'{}'::text[]`
  default to jsonb (SQLSTATE 42804). `PATHandler.{List,Create,Revoke}`
  now log the underlying error before returning `response.InternalError`,
  closing a previously silent observability gap that turned PG errors
  into opaque 500s.

### Changed

- `PATHandler` now receives `*slog.Logger` and `*service.AuditService`
  via the wire-managed constructor (was using `slog.Default()` with no
  audit emission). Logs flow through the same JSON handler used by the
  rest of the backend.
- `PostgresPersonalAccessTokenRepository` now has unit tests via
  `go-sqlmock` (test-only dependency, v1.5.2) covering the JSONB INSERT
  round-trip that regressed in production, plus `FindByTokenHash`,
  `ListByUserID`, `Revoke` and `TouchLastUsed`.
- **Wire DI cleanup**: `apps/backend/internal/di/wire_gen.go` is now fully
  regenerable from source. `EngineSet` gained
  `wire.Struct(new(engine.Params), "*")` so wire builds the `engine.New`
  parameter struct from existing providers, and
  `handler.NewContainerHealthHandler` is wrapped by a thin
  `ProvideContainerHealthHandler` (in `providers_handlers.go`) that
  extracts `cfg.GRPC.AgentPort` — the same pattern used by every other
  handler with an agent port. The manual edit in `wire_gen.go` (added
  during the PAT audit-trail turn because the generator could not resolve
  `engine.Params` and the bare `int` for `agentPort`) is removed; running
  `wire ./internal/di/...` now produces clean output.
- **CI/CD: backend deploy moved to a self-hosted GitHub Actions runner on
  the control-plane VPS.** The `deploy` job now runs locally on the VPS
  (label `flowdeploy-control-plane`), eliminating inbound SSH from
  GitHub-hosted runners (was timing out due to upstream network filtering
  at the provider). The `build-and-push` job continues on
  `ubuntu-latest`. Image tag is now pinned to the build's short SHA
  (instead of `:latest`) for build→deploy traceability. Container health
  is now actively waited for (`HEALTHCHECK` or HTTP `/health`) before the
  deploy is considered successful.
- **CI/CD: MCP deploy aligned with the same self-hosted-runner pattern.**
  `.github/workflows/deploy-mcp.yml` no longer SSHes into the VPS via
  `expect`. The `deploy` job now runs on `[self-hosted,
  flowdeploy-control-plane]` and invokes `.github/scripts/deploy-mcp.sh`
  locally, with SHA-pinned image tag, active healthcheck on `/healthz`
  and automatic rollback to the previous image on failure. The public
  smoke test against `https://${MCP_HOST}/mcp` (asserting HTTP 401 for
  unauth requests) is preserved.
- The host port `9005` is now bound to `127.0.0.1` only (was bound to all
  interfaces). External traffic to the backend goes exclusively through
  Traefik (HTTPS + Let's Encrypt + middlewares).
- `deploy/docker-compose.yml`: standardized the MCP container name from
  `paasdeploy-mcp` to `flowdeploy-mcp` to match the production deploy
  pipeline.

### Removed

- Deleted `.github/scripts/setup-traefik.sh` and the SSH-based deploy
  (`webfactory/ssh-agent`, `expect` scripts). Traefik is provisioned
  out-of-band on the VPS; reprovisioning is a runbook task, not a CI step.
- Deleted `.github/scripts/deploy-mcp.exp` and the `SERVER_PASSWORD`
  dependency from the MCP pipeline (replaced by the local
  `deploy-mcp.sh` running on the self-hosted runner).

### Documentation

- Rewrote the entire `docs/` folder to reflect the current architecture:
  added `docs/README.md` index, replaced the obsolete `REMOTE_DEPLOY_USAGE.md`
  and `ZERO_POLLING_SSE_MIGRATION.md`, refreshed `ARCHITECTURE.md`,
  `CONTRIBUTING.md`, `GITHUB_INTEGRATION.md`, `FEATURES_ROADMAP.md`,
  `REMOTE_SSH_DEPLOY.md`, `REMOTE_SERVER_TUTORIAL.md`,
  `AUTO_DNS_CONFIGURATION.md`, `K3S_DEPLOY_ROADMAP.md`, and added
  `docs/SECURITY.md` consolidating the security model.
- Updated `README.md` with the current package layout and a Documentation
  section linking the new docs.
- Replaced the obsolete `stflow-*.mdc` workspace rules with the
  `flowdeploy-*.mdc` family covering protocol, backend, agent, frontend,
  proto, shared, security and ops.
- Added `AGENTS.md` at the repository root as the onboarding entry point
  for AI agents and new humans.

## [0.2.5] - 2026-04-20

### Added

- **Container customization**: new `command` and `entrypoint` fields in the
  `paasdeploy.json` schema and the template deploy request, propagated
  end-to-end through generators, frontend forms and tests.

### Changed

- Adjusted dialog header padding in the frontend for better layout density.

## [0.2.4] - 2026-04-04

### Added

- **Host port mapping**: the schema now binds `hostPort` to `localhost` by
  default, with new generator tests covering the binding logic.

### Changed

- Bumped Go toolchain to `1.24.13` across all modules
  (`backend`, `agent`, `shared`).
- Updated `paasdeploy.schema.json` description for `hostPort`.

## [0.2.3] - 2026-03-25

### Fixed

- **SSE log integrity**: prevented refetch storms from overwriting in-flight
  deploy logs by routing real-time updates exclusively through the SSE hub.

### Changed

- Enhanced SSE event handling and monitoring in the backend.
- Refactored `deployStatusFromEvent` to use the strongly typed `DeployStatus`.

## [0.2.2] - 2026-03-25

### Added

- **Agent v0.20.1**: enhanced health check configuration with merged TLS
  settings from `paasdeploy.json` and improved monorepo deploy semantics
  (re-deploy all apps that share modified files).

### Fixed

- Removed unnecessary `Close()` calls on output writers in Docker build and
  deploy methods.
- Improved panic recovery and structured logging in the worker loop.

## [0.2.0] - 2026-03-24

This release consolidates the work done during the 0.1.x cycle into the
first stable 0.2.x line. The platform graduated from a single-host pipeline
into a full multi-server PaaS with remote agents, integrated identity, DNS
automation and notifications.

### Added

- **Multi-server management**:
  - Lightweight Go agent (`apps/agent`) installed on each remote VPS,
    communicating with the backend over gRPC.
  - SSH provisioner (`internal/provisioner`) that installs Docker, Traefik
    and the agent idempotently.
  - Backend gRPC server for agent registration and heartbeat.
  - Server CRUD UI with status, metrics and provisioning logs.
  - ACME email per server for automatic Let's Encrypt issuance.
- **Mutual TLS (mTLS)**:
  - Internal certificate authority (`internal/pki`, migration
    `000014_pki_ca`).
  - Per-agent client certificates signed by the CA.
  - Backend pins the CA and the server identity on every gRPC call.
- **Agent auto-update**:
  - `PushUpdate` streaming RPC pushes new agent binaries to remote hosts.
  - `agentdownload` package serves agent binaries to the auto-update flow.
  - Per-server update mode (`auto` / `manual`) via migration `000025`.
  - Agent versioning tracked through the root `AGENT_VERSION` file
    (reached `0.20.0` in this release with SSL configuration features).
- **GitHub integration**:
  - GitHub OAuth sign-in.
  - GitHub App for cloning private repositories with short-lived
    installation tokens (`internal/ghclient/app_client.go`,
    `internal/engine/git_token_provider.go`).
  - Webhook subsystem (`internal/webhook`) with `Manager` interface,
    GitHub implementation and a no-op fallback.
  - HMAC-SHA256 webhook signature verification with constant-time compare.
  - Webhook payload persistence (`webhook_payloads`, migration `000016`)
    for audit and replay.
  - Deploy deduplication by `(app_id, sha)` (migration `000028`).
- **Cloudflare DNS automation**:
  - OAuth flow to connect a Cloudflare account
    (`/api/auth/cloudflare/*`).
  - API token alternative for environments without OAuth.
  - Automatic CNAME creation on the user's zone for custom domains
    (migration `000007_cloudflare_domains`).
- **Email/password authentication**:
  - bcrypt-based credentials (`internal/password`).
  - Migration `000017_email_auth` with `000018_fix_github_login_nullable`
    follow-up.
- **Notifications**:
  - Slack, Discord and SMTP Email senders (`internal/notification`).
  - Notification channels and rules persisted in the database
    (migrations `000011`, `000012`, `000022_add_notification_user_id`).
  - Notification handler with full CRUD and per-channel rules.
- **Authorization model**:
  - User roles (`admin`, `user`) via migration `000024_add_user_role`.
  - Per-app ownership enforced in repositories
    (migration `000019_enforce_app_ownership`).
  - `RequireAdminForLocal` guard preventing non-admin deploys on the
    backend host.
- **Audit logging**:
  - `audit_service` and migration `000010_audit_logs` track platform
    events with payload history.
- **Real-time observability**:
  - Hardened SSE hub at `/events/deploys` multiplexing `deploy`, `health`,
    `stats`, `systemStats`, `serverStats`, `provision`, `agent_update`
    and `resource` events.
  - `system_stats_monitor` and `server_stats_monitor` collect platform
    and per-VPS metrics via gRPC.
  - Frontend consumes SSE through `services/sse.ts` and updates TanStack
    Query caches without polling.
- **Container management surface**:
  - Interactive `docker exec` over a bidirectional gRPC stream with PTY
    support (`creack/pty`).
  - Container start/stop/restart/remove on local and remote servers.
  - Container logs and stats streamed through gRPC.
  - Image, network and volume management with prune operations.
  - Scheduled cleanup with history (`cleanup_logs`, migration `000027`).
- **Templates**:
  - Catalog of pre-configured applications (PostgreSQL, MySQL, Redis,
    MongoDB, Nginx, RabbitMQ, Grafana, etc.) deployable on local and
    remote servers.
- **TLS for individual containers**:
  - `ConfigureContainerSSL` and `GetContainerSSLStatus` agent RPCs.
  - Backend handlers and frontend UI to enable HTTPS per container.
- **Frontend foundations**:
  - Feature-sliced layout under `apps/frontend/src/features/`.
  - shadcn/ui primitives + Tailwind 3.4 with mobile-first conventions.
  - Light/dark/system theme toggle with persistence.

### Changed

- Backend gRPC client/server connection management refactored for
  resilience (reconnect, deadline propagation, error classification).
- Worker constructor uses a `WorkerDeps` struct (8 → 3 parameters).
- Deploy worker propagates runtime metadata from agent to backend on
  remote deploys.
- Build pipeline embeds `AGENT_VERSION` via `-ldflags` so the backend
  always knows which agent binary it serves.

### Security

- All shell commands routed through `shared/pkg/executor` with explicit
  argument vectors (no `sh -c`, no string interpolation).
- Encrypted columns for sensitive payloads (env vars, OAuth tokens,
  refresh tokens, Cloudflare tokens).
- CORS allow-list enforced (no wildcard fallback in production).
- HttpOnly + Secure + SameSite=Lax session cookies.

## [0.1.0] - 2026-01-30

### Added

- Core deployment pipeline with worker pool.
- PostgreSQL-backed deployment queue with `SELECT ... FOR UPDATE SKIP LOCKED`.
- Real-time deployment logs via Server-Sent Events (SSE).
- Rollback support with automatic health check verification.
- Docker-based application containerization.
- React dashboard with React Query state management.
- Traefik reverse proxy integration.
- Health check system with configurable retries and timeouts.
- Application CRUD operations via REST API.
- Structured logging with `slog`.
- Monorepo support via the `workdir` field in application configuration.
- Automatic schema management using `golang-migrate` on backend startup.
- Reusable frontend components (`PageHeader`, `IconText`, `FormField`,
  `LoadingGrid`, `ErrorMessage`).
- Developer experience: VSCode launch configurations, recommended
  extensions, ESLint flat config and Prettier with import sorting.
