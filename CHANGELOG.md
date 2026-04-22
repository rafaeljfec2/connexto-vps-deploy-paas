# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> Versions track the platform `package.json`. The agent has its own version
> in `AGENT_VERSION` and is mentioned in each entry when bumped.

## [Unreleased]

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
