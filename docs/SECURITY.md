# FlowDeploy Security Model

This document is the **single, public-facing source of truth** for the
FlowDeploy security posture: what we protect, how we protect it, and how to
report a vulnerability. It complements (and never overrides) the operational
rules enforced in
[`.cursor/rules/flowdeploy-security.mdc`](../.cursor/rules/flowdeploy-security.mdc),
which is normative for engineers and AI agents working in the codebase.

> **Threat surface in plain English.** FlowDeploy is a self-hosted PaaS that
> controls Docker on every connected VPS, ingests GitHub webhooks, exposes a
> public dashboard, and stores user-supplied secrets (env vars, OAuth tokens,
> Cloudflare tokens). A compromise of the control plane is effectively
> remote code execution on every connected server. Treat every change here
> with that lens.

## 1. Threat model

### Assets

| Asset                                  | Why it matters                                                |
| -------------------------------------- | ------------------------------------------------------------- |
| Backend host (control plane)           | Issues commands to every VPS; mounts Docker socket            |
| Internal CA private key                | Compromise = ability to forge valid agent certificates        |
| Per-agent TLS keypairs                 | Compromise = impersonate one agent (CN-bound to `server_id`)  |
| GitHub App private key                 | Compromise = clone any installed private repository           |
| GitHub OAuth secrets                   | Compromise = impersonate the FlowDeploy OAuth app             |
| Webhook shared secret                  | Compromise = inject fake `push` events to trigger deploys     |
| Token encryption key (AES)             | Decrypts every persisted secret (env vars, OAuth tokens)      |
| User session tokens                    | Compromise = full account takeover                            |
| Database (PostgreSQL)                  | Holds the above + audit log + queue + PKI material            |
| Tenant container env vars              | May contain customer-owned secrets                            |

### Adversaries we plan for

1. **Untrusted internet**: anyone reaching the public dashboard, the public
   webhook URL, or the gRPC TCP port.
2. **Malicious tenant**: a logged-in user trying to escalate to another
   user's apps, servers or secrets.
3. **Compromised dependency**: a transitive package shipping malicious code.
4. **Compromised remote VPS**: an attacker with root on one VPS trying to
   pivot back to the backend or to other VPSs.
5. **Insider mistake**: a developer commit that accidentally exposes a
   secret, weakens TLS, or bypasses an auth check.

### Adversaries explicitly **out of scope** for v1

- Adversaries with physical access to the backend host.
- Side-channel attacks on the host CPU.
- Pre-image attacks against bcrypt or AES-GCM.
- Targeted DDoS at L3/L4 (mitigated at the cloud provider level).

## 2. Trust boundaries

```
   Internet
      │  HTTPS (TLS, Let's Encrypt via Traefik)
      ▼
   Dashboard / API / Webhook  ──►  Backend process
                                       │ mTLS (gRPC, internal CA)
                                       ▼
                                  Remote agents (one per VPS)
                                       │ local socket
                                       ▼
                                  Docker engine on the VPS
```

Each arrow is a **trust boundary** with its own authentication mechanism.
Crossing one without the documented credential is a security bug.

## 3. Backend ↔ Agent (mTLS)

- Every gRPC call between the backend and any agent uses **mutual TLS**.
- The internal CA lives in `apps/backend/internal/pki/` and is persisted via
  `domain.CertificateAuthorityRepository` (migration `000014_pki_ca`). It is
  the **only** trust anchor for agent traffic.
- Per-agent client certificates are issued at provisioning time. The CN/SAN
  encode the `server_id` so a compromised cert cannot impersonate another
  server.
- The agent's gRPC server requires the backend's certificate signed by the
  same CA (`tls.RequireAndVerifyClientCert`).
- The emergency-only flag `GRPCConfig.AgentTLSInsecureSkipVerify` exists for
  CA-loss recovery scenarios. **It must default to `false`** and is never
  acceptable in normal operation.
- Cert rotation: re-issue via the PKI service; the agent picks up the new
  cert on next boot. Old certs are invalidated through the CA's revocation
  list (planned for v0.3.x).

## 4. User authentication

Two paths, both terminating in a session cookie:

| Path                  | Implementation                                  | Secret material                          |
| --------------------- | ----------------------------------------------- | ---------------------------------------- |
| GitHub OAuth          | `apps/backend/internal/github/`                 | `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET` |
| Email + password      | `apps/backend/internal/password/` (bcrypt)      | bcrypt hash stored in `users.password_hash` |

Session rules (`internal/handler/cookie_helpers.go`,
`internal/crypto/token.go`):

- Cookie name: `flowdeploy_session` (configurable).
- Flags: `HttpOnly`, `SameSite=Lax` (`Strict` for the dashboard origin in
  production), `Secure` in production (`AuthConfig.SecureCookie`).
- Token format: opaque random 32-byte token. The cookie carries the raw
  token; the database stores **only** `crypto.HashSessionToken(rawToken)`.
- TTL: configurable via `SessionMaxAge` (default 7 days).
- Logout: hard-delete the session row, clear the cookie.
- We **do not** use JWTs. Opaque tokens give us instant revocation.

## 5. Authorization

- Roles: `admin` and `user` (migration `000024_add_user_role`).
- App ownership: every `domain.App` carries a `user_id`. Repositories
  enforce ownership through `FindAllByUserID` and `FindByIDAndUserID`. Do
  not bypass with `FindByID` in user-facing handlers.
- The `RequireAdminForLocal(c, serverID)` guard blocks any mutation that
  could affect the control-plane host itself (templates on local server,
  container ops on local server, system-wide cleanup).
- Notifications, servers and secrets are scoped to the owning user
  (migrations `000021_add_server_user_id`, `000022_add_notification_user_id`).

## 6. GitHub integration

| Secret                              | Storage                                  | Notes                                          |
| ----------------------------------- | ---------------------------------------- | ---------------------------------------------- |
| `GITHUB_CLIENT_ID` / `_SECRET`      | Environment only                         | Never logged, never echoed                     |
| GitHub App private key (PEM)        | Filesystem `0600` or env                 | Loaded once into memory, never logged          |
| `GITHUB_WEBHOOK_SECRET`             | Environment only                         | Used for HMAC-SHA256 webhook signature verify  |
| Per-user OAuth tokens               | `users` table, AES-encrypted             | Encryption key from `AuthConfig.TokenEncryptionKey` |
| Installation tokens (deploy time)   | In-memory only                           | Lifetime ≤ 1 hour, re-minted on demand          |

Webhook ingestion (see [`docs/GITHUB_INTEGRATION.md`](./GITHUB_INTEGRATION.md)):

1. Read the raw body before any JSON decoding.
2. Verify `X-Hub-Signature-256` with constant-time HMAC-SHA256
   (`internal/ghclient/signature.go`).
3. **Reject with 401 before any routing decision** if the signature is
   invalid or missing.
4. Persist the validated payload to `webhook_payloads` (migration `000016`)
   for audit and replay. Redact known-secret fields before persisting.
5. Enqueue the deploy in the queue — **never** execute inline in the
   webhook handler.

## 7. Secrets at rest

- Token encryption uses **AES-GCM** with a 32-byte key supplied via
  `AuthConfig.TokenEncryptionKey`. The key **must** be set in production.
- Encrypted columns:
  - `users.github_access_token_encrypted`
  - `cloudflare_connections.access_token_encrypted` /
    `refresh_token_encrypted`
  - `servers.ssh_password_encrypted` (migration `000015`)
  - `env_vars.value_encrypted` (migration `000004`)
  - `pki_ca.private_key_encrypted` (migration `000014`)
- Per-app env vars (`domain.EnvVar`) are pushed to the agent via mTLS
  gRPC, written to `DEPLOY_DATA_DIR/<app>/.env` with `0600` perms, and
  mounted into the container by docker-compose. The frontend never receives
  raw values back; reveal endpoints are explicit and audited.

## 8. Network exposure

- **Public**: HTTPS (Traefik + Let's Encrypt) for the dashboard, the API
  and the webhook ingest URL.
- **Public (TCP route SNI)**: the backend's gRPC port, exposed for agents
  to reach the control plane.
- **Internal**: PostgreSQL only listens on the Docker network of the
  compose stack; no host port exposure in production.
- **Outbound from the backend**: Docker socket (control plane), GitHub API,
  Cloudflare API, SMTP for email notifications, Slack/Discord webhook URLs.

CORS:

- `internal/server/...` configures Fiber's CORS using
  `ServerConfig.CorsOrigins` (comma-separated allow-list).
- **No wildcard fallback.** Production refuses to start without an explicit
  list. `Access-Control-Allow-Credentials: true` is required because the
  frontend uses cookie-based auth.

## 9. SSH provisioning of remote agents

- Connections use **host-key verification** (`server.SSHHostKey`,
  migration `000023`). Unknown hosts are rejected after the initial
  trust-on-first-use during provisioning.
- The install script runs with the necessary `sudo` only — no full-root
  shell is requested, no persistent backdoor is installed
  (`internal/provisioner/privilege.go`).
- The agent's TLS keypair is generated **on the backend** and pushed to
  the remote over SCP/SFTP. We never ask the remote host to generate its
  own private key.
- Files arrive with `0600` perms. The provisioning password (if any) is
  stored encrypted (migration `000015_add_server_password`) and used once.

## 10. Input validation

Backend handlers validate every user-supplied input before calling
services. Common patterns live in `internal/handler/helpers.go`:

- App names: must match a DNS label (lowercase, digits, hyphens, ≤ 63
  chars). They become container/network names.
- Repository URLs: HTTPS only (or a known SSH form for GitHub App use).
  Other schemes are rejected.
- Branch names: anything containing `..`, leading `/`, or control
  characters is rejected.
- File paths from API: passed through `shared/pkg/safepath.IsSafeFileName`
  for any user-supplied path component.
- Query params: typed accessors (`c.QueryInt`, `c.Query`) with explicit
  range guards.

## 11. Shell command safety

- All shell calls (Docker CLI, Git, OpenSSH tools) go through
  `shared/pkg/executor` with explicit `name, args...` vectors.
- **Never** `sh -c "..."`. **Never** string interpolation into a single
  argument.
- For `docker exec`, the user-controlled command is always `[]string`,
  passed verbatim — never a single string.
- For Git, use the helpers in `shared/pkg/git`, which inject credentials
  through environment variables (no `https://token@...` URLs in argv).

## 12. Database

- Parameterized queries only. The repository pattern uses positional
  placeholders. **Never** `fmt.Sprintf` user input into a query.
- Soft deletes (`deleted_at` / `status = 'deleted'`) for user-visible
  aggregates. `HardDelete` exists on `AppRepository` for cleanup paths
  only and is gated behind admin checks.
- Any migration that adds a column carrying secrets must pair with
  encryption-at-rest in the repository layer; plaintext is forbidden.

## 13. Logging hygiene

We **never** log:

- Passwords, raw session tokens, env var values.
- GitHub access tokens, GitHub App private key bytes.
- Webhook secrets, the CA private key, agent TLS keys, encryption keys.

We **do** log:

- Correlation IDs, user IDs, server IDs, deploy IDs, app IDs.
- Timing, status codes, error categories.

When an error may carry sensitive data (for example
`json.Unmarshal` failures on a request body), redact before logging.

## 14. Container and Docker socket

- The backend container mounts `/var/run/docker.sock` (read-write for local
  deploys). This is effectively root on the host. The backend is treated as
  a privileged service:
  - `restart: unless-stopped` is configured.
  - Failing security checks **abort the deploy**; we do not retry on
    security errors (no infinite restart loops).
  - Image hardening: Alpine base, non-root user where the engine does not
    need socket access, drop unneeded capabilities.
- Tenant images run as their own `Dockerfile` declares. We do not force
  non-root for tenant images, but we **do** isolate them in dedicated
  Docker networks per app.

## 15. Dependencies

- **Frontend**: `pnpm audit` is expected to be clean. Dependabot / Renovate
  updates are reviewed for breaking transitive changes.
- **Backend / Agent**: `govulncheck ./...` is run before each release as a
  manual release gate. Adding a new Go dependency requires confirming that
  it is well-maintained and audited.
- The standard library is the default. Acceptable third-party modules
  include `pgx`, `fiber`, `grpc`, `jwt`, `swag`, `creack/pty`,
  `golang-migrate`, `golang.org/x/crypto`.

## 16. Hardening checklist for production

Before exposing FlowDeploy to the public internet, confirm:

- [ ] `AuthConfig.TokenEncryptionKey` set to a 32-byte random value.
- [ ] `AuthConfig.SecureCookie = true`.
- [ ] `CORS_ORIGINS` set to an explicit allow-list.
- [ ] HTTPS enabled at Traefik with Let's Encrypt and a valid ACME email.
- [ ] gRPC TCP route SNI configured for the agent port.
- [ ] PostgreSQL not exposed on the host network.
- [ ] `GITHUB_WEBHOOK_SECRET`, `GITHUB_CLIENT_*` and the GitHub App
      private key configured.
- [ ] CA bootstrap (migration `000014`) executed and the CA key stored
      encrypted.
- [ ] At least one `admin` user exists.
- [ ] `RequireAdminForLocal` covers every local-host mutation handler.
- [ ] Backups configured for the PostgreSQL volume.
- [ ] `govulncheck` clean on the latest backend and agent build.
- [ ] `pnpm audit` clean on the frontend build.
- [ ] Logs forwarded to a destination that supports audit retention.

## 17. Reporting a vulnerability

If you find a vulnerability:

1. **Do not** open a public issue or PR with details.
2. Email the maintainers (private channel) with:
   - A clear description of the issue.
   - Reproduction steps, including affected versions
     (`package.json`, `AGENT_VERSION`, commit SHA).
   - Impact assessment: what data or systems are at risk.
   - Suggested mitigation, if you have one.
3. Expect an acknowledgement within 48 business hours.
4. We will coordinate a fix and a coordinated disclosure date with you.

## 18. References

- [`.cursor/rules/flowdeploy-security.mdc`](../.cursor/rules/flowdeploy-security.mdc) — normative rules for engineers and AI agents
- [`docs/ARCHITECTURE.md`](./ARCHITECTURE.md) — runtime architecture
- [`docs/GITHUB_INTEGRATION.md`](./GITHUB_INTEGRATION.md) — OAuth, GitHub App, webhook signature verification
- [`docs/REMOTE_SSH_DEPLOY.md`](./REMOTE_SSH_DEPLOY.md) — agent provisioning and mTLS lifecycle
- [`docs/AUTO_DNS_CONFIGURATION.md`](./AUTO_DNS_CONFIGURATION.md) — Cloudflare token handling
