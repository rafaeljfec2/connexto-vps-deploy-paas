# GitHub Integration

This document describes how FlowDeploy talks to GitHub: user sign-in (OAuth),
private repository access (GitHub App), and automatic deployments triggered by
push events (webhooks). It is the source of truth for anyone touching
`apps/backend/internal/github`, `apps/backend/internal/ghclient`,
`apps/backend/internal/webhook` or the related handlers.

## 1. Goals

1. **Sign in with GitHub** — let users authenticate without managing a password.
2. **Access private repositories** — clone repos at deploy time using
   short-lived installation tokens, not user OAuth tokens.
3. **Auto-deploy on push** — receive verified webhooks and enqueue a deploy.
4. **Stay decoupled** — webhook setup is a separate concern from the deploy
   pipeline; deploys must work with or without an automatic webhook.

## 2. Components

```
apps/backend/internal/
├── github/                 # OAuth + GitHub App orchestration (sign-in, install)
├── ghclient/               # Low-level HTTP client (REST + GraphQL + signatures)
│   ├── client.go           # Personal Access Token client
│   ├── app_client.go       # GitHub App client (JWT + installation tokens)
│   ├── oauth.go            # OAuth code exchange
│   ├── jwt.go              # GitHub App JWT minting
│   ├── webhook.go          # Webhook CRUD on GitHub side
│   ├── signature.go        # HMAC-SHA256 signature verification (X-Hub-Signature-256)
│   ├── pat_provider.go     # Adapts a PAT into a Provider
│   └── provider.go         # Provider abstraction used by the webhook manager
├── webhook/
│   ├── manager.go          # Manager interface (Setup/Remove/Status/List)
│   ├── github_manager.go   # GitHub implementation
│   └── noop_manager.go     # No-op for environments without GitHub credentials
└── handler/
    ├── auth_handler.go         # OAuth sign-in routes
    ├── github_handler.go       # GitHub App install + repository listing
    └── (webhook ingest)        # Receives push events and enqueues deploys
```

The pieces collaborate as follows:

```
       ┌────────────┐  OAuth code flow   ┌────────────┐
       │  Browser   │ ─────────────────► │  github/   │ ──► users + sessions
       └────────────┘                    └────────────┘

       ┌────────────┐  install URL       ┌────────────┐
       │  Browser   │ ─────────────────► │  GitHub    │ ──► installation_id
       └────────────┘                    └────────────┘
                                              │
                                              ▼ webhook (installation event)
                                         ┌────────────────┐
                                         │ github_handler │
                                         │ (verifies sig) │
                                         └────────────────┘

       ┌────────────┐  push event        ┌────────────────┐  enqueue
       │  GitHub    │ ─────────────────► │ webhook ingest │ ─────────► engine.queue
       └────────────┘                    └────────────────┘
```

## 3. Authentication: OAuth sign-in

Used for **sign-in only**. We do not store the user's OAuth token long-term.

1. Frontend redirects to `/api/auth/github/login`.
2. The handler builds the GitHub authorization URL with `state` (CSRF) and
   `scope` reduced to the minimum (`read:user`, `user:email`).
3. GitHub redirects back to `/api/auth/github/callback?code=...&state=...`.
4. `ghclient/oauth.go` exchanges the code for an access token.
5. The handler fetches the user profile, upserts a row in `users`, and creates
   a session (HttpOnly cookie). The OAuth token is discarded.

Email/password sign-in is also available via `internal/password` (bcrypt) for
environments without GitHub.

## 4. Authorization: GitHub App for repository access

OAuth tokens are user-scoped and rate-limited; for cloning private repos at
deploy time we use a **GitHub App**.

1. User clicks **Connect GitHub** in the dashboard.
2. Backend redirects to the App's install URL
   (`/api/github/install` → `appInstallURL`).
3. After installation, GitHub posts an `installation` webhook to
   `/paas-deploy/v1/webhooks/github/app`.
4. `GitHubHandler.HandleInstallationWebhook` verifies the
   `X-Hub-Signature-256` header (HMAC-SHA256 with the App's webhook secret),
   parses the payload and persists the installation in the
   `installations` table (linked to the user).
5. At deploy time, `engine/git_token_provider.go` mints a short-lived
   **installation access token** through `ghclient/app_client.go`:
   - Sign a JWT with the App private key (RS256, 10-minute TTL).
   - Call `POST /app/installations/{id}/access_tokens`.
   - Use the returned token (≤ 1 hour) as a Bearer credential for `git clone`.

This isolates deploy traffic from end-user credentials and respects GitHub
rate limits.

### Required environment variables

| Variable                     | Purpose                                                 |
| ---------------------------- | ------------------------------------------------------- |
| `GITHUB_CLIENT_ID`           | OAuth App / GitHub App client ID                        |
| `GITHUB_CLIENT_SECRET`       | OAuth App / GitHub App client secret                    |
| `GITHUB_APP_ID`              | Numeric GitHub App ID                                   |
| `GITHUB_PRIVATE_KEY_PATH`    | Path to the GitHub App private key (PEM)                |
| `GITHUB_WEBHOOK_SECRET`      | Shared secret used to sign webhook payloads             |
| `GITHUB_APP_INSTALL_URL`     | Public install URL of the GitHub App                    |

## 5. Webhook subsystem

Webhooks are managed by an explicit **interface** so the platform can run with
or without automatic webhook provisioning.

```go
type Manager interface {
    Setup(ctx context.Context, input SetupInput) (*SetupResult, error)
    Remove(ctx context.Context, input RemoveInput) error
    Status(ctx context.Context, repoURL string, webhookID int64) (*Status, error)
    ListCommits(ctx context.Context, repoURL, branch string, perPage int) ([]ghclient.CommitInfo, error)
    WebhookURL() string
}
```

Implementations:

- `webhook/github_manager.go` — talks to GitHub's REST API to create, remove
  and list repo webhooks. Idempotent: if a hook already exists for the same
  URL, it is reused instead of returning an error.
- `webhook/noop_manager.go` — used when no GitHub credentials are configured.
  Returns successful no-ops; deploys still work, they just need to be
  triggered manually or via a webhook configured outside the platform.

### Setup flow

When an app is created with a GitHub repository URL:

1. `service/app_service.go` calls `Manager.Setup`.
2. The manager parses `owner/repo`, builds a `WebhookConfig` (target URL,
   shared secret, content-type JSON, no insecure SSL).
3. The webhook is created (or reused if already present).
4. The resulting `webhook_id` is stored on the app row so it can be removed
   later.

### Removal flow

App deletion calls `Manager.Remove`, which deletes the hook on GitHub. Failure
to delete is logged but does not block the local deletion (apps can be
re-created and the webhook will be reused).

## 6. Inbound webhook ingestion

Pushed events arrive at the public webhook URL (typically
`https://<deploy-host>/paas-deploy/v1/webhooks/github`). Processing:

1. **Read raw body** before any JSON decoding (signature must run on bytes).
2. **Verify signature** — `ghclient.VerifySignature` compares
   `X-Hub-Signature-256` against `HMAC-SHA256(secret, body)` using
   `hmac.Equal` (constant time). Reject with `401` on mismatch.
3. **Persist payload** in `webhook_payloads` (audit + replay).
4. **Filter event type** — only `push` events to the configured branch are
   actionable; `installation`, `ping` and others have dedicated paths.
5. **Resolve app** by `repository.full_name` and `ref`.
6. **Enqueue deploy** with `engine.Queue.Enqueue(...)`. The deploy row is
   inserted with `status = pending` and the SHA from the webhook.
7. **Respond `204`** quickly. The actual deploy happens asynchronously via the
   dispatcher → worker pipeline.

### Deduplication

Migration `000028_deploy_dedup` enforces uniqueness so a duplicate webhook
delivery does not enqueue the same SHA twice for the same app.

## 7. Idempotency and retries

- Hook creation is idempotent (existing-hook detection by normalized URL).
- Hook removal is idempotent (404 on delete is treated as success).
- Webhook ingestion deduplicates by `(app_id, sha)` to handle GitHub's
  at-least-once delivery semantics.
- Installation token minting is on-demand and short-lived; tokens are not
  cached across deploys.

## 8. Security checklist

- ✅ All webhook payloads verified with HMAC-SHA256, constant-time comparison.
- ✅ App private key stored on the host filesystem with `0600` permissions
  (never in the database, never logged).
- ✅ OAuth scopes limited to the bare minimum for sign-in.
- ✅ Installation tokens are never persisted; they live in memory for one
  deploy.
- ✅ Webhook secret rotated by updating both `GITHUB_WEBHOOK_SECRET` and the
  hook configuration via `Manager.Setup`.
- ✅ All HTTP calls to GitHub use TLS with default Go CA roots.

## 9. Local development

The platform works without GitHub credentials: `webhook.NoopManager` is wired
in DI when `GITHUB_*` variables are absent. To exercise the full integration:

1. Create a GitHub App (callback URL `http://localhost:8080/api/auth/github/callback`,
   webhook URL `http://<your-tunnel>/paas-deploy/v1/webhooks/github`,
   permissions: `Contents: Read`, `Metadata: Read`, events: `Push`,
   `Installation`).
2. Download the private key into `~/.flowdeploy/github-app.pem`.
3. Fill the `GITHUB_*` env vars in `.env`.
4. Use `gh webhook forward` (recommended) or a tunnel (cloudflared, ngrok)
   to receive webhooks locally.

## 10. Files to read when changing this area

| Concern                         | File                                                            |
| ------------------------------- | --------------------------------------------------------------- |
| OAuth sign-in                   | `apps/backend/internal/handler/auth_handler.go`                 |
| OAuth code exchange             | `apps/backend/internal/ghclient/oauth.go`                       |
| GitHub App install + listing    | `apps/backend/internal/handler/github_handler.go`               |
| App JWT + installation token    | `apps/backend/internal/ghclient/app_client.go`, `jwt.go`        |
| Webhook signature verification  | `apps/backend/internal/ghclient/signature.go`                   |
| Webhook CRUD on GitHub          | `apps/backend/internal/ghclient/webhook.go`                     |
| Webhook abstraction             | `apps/backend/internal/webhook/manager.go`                      |
| GitHub webhook manager          | `apps/backend/internal/webhook/github_manager.go`               |
| Token provider for deploys      | `apps/backend/internal/engine/git_token_provider.go`            |
