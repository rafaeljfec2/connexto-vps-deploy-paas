# AGENTS.md — FlowDeploy

> **Audience**: AI coding agents (Cursor, Claude, Codex CLI, GitHub Copilot Workspace) and onboarding humans.
>
> **Purpose**: a single, authoritative starting point describing how to think, build, test, and ship safely in this repository.

If anything here conflicts with `.cursor/rules/flowdeploy-development-protocol.mdc`, the rule wins. This file is a _guide_; the rules are _contract_.

---

## 1. What This Project Is

**FlowDeploy** (`connexto-vps-deploy-paas`) is a self-hosted PaaS for automatic deployments from GitHub repositories with multi-server management via remote agents. Think of it as a lightweight Vercel/Railway you run yourself, where:

- Pushing to a connected branch → automatic deploy via webhook.
- Apps run as Docker containers behind Traefik with Let's Encrypt TLS.
- Remote VPS hosts run a tiny Go agent that the control plane talks to over gRPC + mTLS.
- The dashboard is a React SPA with real-time deploy logs over SSE.

Current platform version: **0.2.5** (root `package.json`). Current agent binary version: see `AGENT_VERSION`.

---

## 2. Repository Map

```
connexto-vps-deploy-paas/
├── apps/
│   ├── backend/      Go 1.24 — Fiber HTTP API, deploy engine, control-plane gRPC
│   ├── agent/        Go 1.24 — gRPC server installed on remote VPS hosts
│   ├── frontend/     React 18 + Vite + TypeScript + shadcn/ui + Tailwind + TanStack Query
│   ├── proto/        Protocol Buffers (buf) — generated into apps/backend/gen/go
│   └── shared/       Reusable Go packages (docker, executor, git, health, lock, paths, …)
├── deploy/           docker-compose.yml + Traefik static config + start/stop scripts
├── docs/             Architecture, roadmaps, integration guides (do NOT edit without asking)
├── scripts/          bump-versions.sh
├── .github/          GitHub Actions CI/CD
├── .husky/           Git hooks (pre-commit only)
├── .cursor/          Cursor rules + skills
├── AGENT_VERSION     Single-line agent binary version
├── Makefile          proto/build/build-agent/bump-agent-version targets
├── package.json      pnpm workspace root + scripts (lint, test, dev, docker:*)
├── pnpm-workspace.yaml
└── turbo.json
```

Detailed conventions live under `.cursor/rules/`:

- `flowdeploy-orchestrator.mdc` — **always applies**; Architect → Approval Gate → Developer (Backend/Frontend/Proto/Agent/Shared/Infra/Docs) → Senior Reviewer, with mandatory visual banners
- `flowdeploy-development-protocol.mdc` — root, always applies
- `flowdeploy-backend-go.mdc` — `apps/backend/**`
- `flowdeploy-agent-go.mdc` — `apps/agent/**`
- `flowdeploy-frontend-react.mdc` — `apps/frontend/**`
- `flowdeploy-proto-grpc.mdc` — `apps/proto/**` and `apps/backend/gen/go/**`
- `flowdeploy-shared-go.mdc` — `apps/shared/**`
- `flowdeploy-security.mdc` — always applies
- `flowdeploy-ops.mdc` — `deploy/**`, `.github/**`, `scripts/**`, `Dockerfile*`, `AGENT_VERSION`, `Makefile`

Read the relevant rule before editing files in that area.

**Mandatory skills** (enforced by `flowdeploy-development-protocol.mdc`):

| Trigger | Skill | Invocation | When |
| ------- | ----- | ---------- | ---- |
| Files touched: `apps/backend/**`, `apps/agent/**`, `apps/shared/**`, `apps/proto/**`, `**/*.go`, `**/*.proto`, `go.mod`, `migrations/**` | `.cursor/skills/golang-engineering-expert/SKILL.md` | inline | **before** editing |
| Files touched: `apps/frontend/**`, `**/*.tsx`, `**/*.ts` (frontend scope), Tailwind/shadcn/ui changes | `.cursor/skills/frontend-engineering-expert/SKILL.md` | inline | **before** editing |
| **ANY coding turn that modified files** (code, proto, migration, config, rule, skill, behavioral docs) | `.cursor/skills/business-rule-guardian/SKILL.md` | fresh readonly subagent (`Task`, `subagent_type: generalPurpose`, `readonly: true`) | **after** coding, as the final gate before closing the turn |

Rules own **conventions** (layout, layering, naming). Engineering-expert skills own **craft** (idioms, error design, concurrency, tests, performance). Load the matching engineering skill **before** editing. The business-rule guardian runs **as the final gate of every coding turn** — never inline (inline inherits developer bias and invalidates the review). Exceptions in `flowdeploy-development-protocol.mdc` §3.1.

---

## 3. Tech Stack at a Glance

| Layer          | Choice                                                                    |
| -------------- | ------------------------------------------------------------------------- |
| Backend        | Go 1.24, Fiber v2, pgx/v5 (database/sql), golang-migrate, slog + tint     |
| Agent          | Go 1.24, gRPC, mTLS, creack/pty                                           |
| RPC            | Protocol Buffers (proto3) + buf, grpc-go v1.79+                           |
| Database       | PostgreSQL 16, schema migrations under `apps/backend/migrations/`         |
| DI             | `google/wire` (`apps/backend/internal/di/`)                               |
| Auth           | GitHub OAuth (primary) + email/password, opaque session tokens in cookies |
| Frontend       | React 18 + Vite 6 + TypeScript 5 (strict) + React Router 6                |
| UI             | shadcn/ui (Radix primitives) + Tailwind 3.4 + lucide-react                |
| State (server) | TanStack Query 5                                                          |
| State (client) | React Context + local hooks (no Redux/Zustand)                            |
| Real-time      | SSE (`/events/deploys`) + closed event-name set                           |
| Reverse proxy  | Traefik 3.x (HTTP + gRPC TCP passthrough)                                 |
| Containers     | Docker + Docker Compose v2 (CLI subprocess via `shared/pkg/docker`)       |
| Monorepo       | pnpm 9 + Turborepo                                                        |
| Linters        | ESLint flat config + Prettier (FE), golangci-lint v1.61 (BE/agent/shared) |
| Tests (BE)     | Standard library `testing`                                                |
| CI             | GitHub Actions (`.github/workflows/deploy-backend.yml`)                   |

---

## 4. Setup From Zero

```bash
# 1. Install pnpm 9 and Go 1.24+
node --version          # >= 20
pnpm --version          # >= 9
go version              # >= 1.24

# 2. Install JS dependencies
pnpm install

# 3. Configure environment
cp .env.example .env
# Edit .env — at minimum DATABASE_URL and GITHUB_* if you want OAuth

# 4. Bring up Postgres + Traefik (Docker Compose)
pnpm docker:up

# 5. Run backend (with hot reload via `air`)
pnpm backend:dev

# 6. In another terminal, run frontend
pnpm dev:web

# Dashboard: http://localhost:3000
# API:       http://localhost:8080/api
# SSE:       http://localhost:8080/events/deploys
```

For agent development on a real VPS, see `docs/REMOTE_SERVER_TUTORIAL.md`.

---

## 5. Daily Commands

### Common

```bash
pnpm install                    # install JS deps (root + workspaces)
pnpm dev                        # backend + frontend together
pnpm dev:web                    # frontend only
pnpm dev:api                    # backend only with `air`
pnpm build                      # turbo run build (everything)
pnpm test                       # turbo run test
pnpm lint                       # turbo run lint
pnpm typecheck                  # turbo run typecheck
```

### Backend

```bash
cd apps/backend && go run ./cmd/api    # run from source
pnpm backend:build                     # build with embedded AGENT_VERSION
pnpm backend:test                      # go test ./...
pnpm run lint:go-quick                 # fast Go lint
pnpm run lint:go                       # full Go lint (CI parity)
```

### Agent

```bash
cd apps/agent
go test ./...
go run ./cmd/agent --server-addr=localhost:50051 \
                   --server-id=<id> \
                   --ca-cert=/path/to/ca.pem \
                   --cert=/path/to/agent.crt \
                   --key=/path/to/agent.key \
                   --agent-port=50052
```

### Proto

```bash
make proto                    # buf lint + buf generate
make proto-lint
make proto-go
```

### Versioning

```bash
make bump-agent-version v=0.21.2     # bump agent binary version
# Frontend / backend / root patch versions auto-bump on `main` via the pre-commit hook.
```

### Docker stack

```bash
pnpm docker:up
pnpm docker:down
pnpm docker:logs
pnpm docker:build
```

---

## 6. The Workflow Every Change Must Follow

1. **Read context** — open the file, find consumers (`Grep` for the symbol), open the matching `.cursor/rules/flowdeploy-*.mdc`, **and load the mandatory engineering skill** (`golang-engineering-expert` or `frontend-engineering-expert`) for that area.
2. **Plan briefly** for anything bigger than a one-liner (intent + diff plan).
3. **Implement minimally** — no opportunistic refactors.
4. **Lint locally**:
   ```bash
   pnpm --filter @paasdeploy/frontend run lint
   pnpm --filter @paasdeploy/frontend run typecheck
   pnpm run lint:go-quick
   ```
5. **Run tests** at least for the package(s) you touched:
   ```bash
   cd apps/backend && go test ./internal/<pkg>/...
   cd apps/shared  && go test ./pkg/<pkg>/...
   cd apps/agent   && go test ./internal/<pkg>/...
   ```
6. **Self-review the diff**: naming, error wrapping, missing tests, unused imports, no `any`, no `||` for fallbacks.
7. **Summarize** the change (1–2 paragraphs) and propose follow-ups if any.
8. **Run the business-rule guardian (mandatory final gate)** — spawn a **fresh readonly subagent** (`Task`, `subagent_type: generalPurpose`, `readonly: true`) loading `.cursor/skills/business-rule-guardian/SKILL.md`. Pass three blocks only: original request (verbatim), files under review, optional acceptance criteria. Do **not** attach your own rationale — the guardian's value is judging with zero developer bias.
   - ✅ Approved → close the turn.
   - 🟡 Approved with conditions → fix now or record as explicit follow-ups.
   - 🔴 Rejected → fix blockers, re-run the guardian. Never ship a 🔴.

   Exceptions: pure read/exploration turns, typo/formatting-only doc edits, explicit user opt-out for that turn, or when the current session is itself a guardian subagent. See `flowdeploy-development-protocol.mdc` §3.1 for the full contract.

The pre-commit hook (`.husky/pre-commit`) runs lint-staged + Go quick lint + frontend typecheck, and on `main` auto-bumps patch versions. Don't bypass it.

---

## 7. Architecture Mental Model

```
┌──────────────────────────────────────────────────────────────────┐
│                            Traefik                               │
│              (HTTP + TLS termination + gRPC TCP passthrough)     │
└──────────┬─────────────────────────────────┬─────────────────────┘
           │                                 │
   ┌───────▼────────┐                ┌───────▼────────┐
   │   Frontend     │                │    Backend     │
   │ React + Vite   │                │  Fiber + gRPC  │
   └────────────────┘                └───────┬────────┘
                                             │
                          ┌──────────────────┼──────────────────┐
                          │                  │                  │
                  ┌───────▼────────┐  ┌──────▼──────┐  ┌────────▼────────┐
                  │ Deploy Engine  │  │ gRPC Server │  │   PostgreSQL    │
                  │ (worker pool)  │  │ (agents)    │  │ (queue + data)  │
                  └───────┬────────┘  └──────┬──────┘  └─────────────────┘
                          │                  │
              ┌───────────┤                  │
              │           │           ┌──────▼─────────────────────┐
      ┌───────▼──────┐    │           │   Remote Agents (gRPC)     │
      │ Docker (host)│    │           │  ┌─────────┐ ┌─────────┐   │
      └──────────────┘    │           │  │ Agent 1 │ │ Agent N │   │
                          │           │  │ +Docker │ │ +Docker │   │
                          │           │  └─────────┘ └─────────┘   │
                  ┌───────▼─────┐     └────────────────────────────┘
                  │   GitHub    │
                  │  (webhooks) │
                  └─────────────┘
```

Key flows:

- **Deploy from webhook**: GitHub → backend `webhook` handler → signature verified → enqueue in `engine.Queue` → `engine.Worker` picks up via `SELECT … FOR UPDATE SKIP LOCKED` → executes locally (Docker) or remotely (agent gRPC `ExecuteDeploy`) → streams logs back via `engine.notifier` → SSE to dashboard.
- **Container actions** (start/stop/restart/logs/exec/stats): backend handler → `agentclient` → agent `handlers_containers.go` (or local `shared/pkg/docker` for the control-plane host).
- **Agent provisioning**: `provisioner.SSHProvisioner` connects via SSH, installs the binary, drops mTLS keypair issued by the local PKI, registers a systemd unit.

---

## 8. Hard Rules (PROHIBITED)

These are non-negotiable. Read `flowdeploy-development-protocol.mdc` and `flowdeploy-security.mdc` for the full list:

- **Never** weaken mTLS verification (`InsecureSkipVerify` is for emergency only and must default to false).
- **Never** edit generated proto code under `apps/backend/gen/go/**` — regenerate.
- **Never** edit `.env` (the user's local file). Add new variables to `.env.example` and ask.
- **Never** modify a migration that has been merged. Add a new sequential one.
- **Never** use `any` in TypeScript (including tests). Use `unknown` and narrow.
- **Never** use `||` for fallback assignment. Use `??` (TS) or explicit `if x == "" { x = default }` (Go).
- **Never** shell out with `sh -c`. Use `shared/pkg/executor.Run(ctx, "name", "arg1", "arg2", …)`.
- **Never** log secrets, raw session tokens, env var values, GitHub tokens, private keys, or webhook payloads with sensitive data.
- **Never** add a new top-level dependency without approval — verify it's not already covered by the current stack.
- **Never** import across feature folders in the frontend (`@/features/foo` from `@/features/bar`).
- **Never** narrate code with comments. Comment intent, trade-offs, gotchas — not mechanics.
- **Never** force-push to `main`. Never bypass `--no-verify` without explicit approval.

---

## 9. Coding Style Quick Reference

### Both languages

- English (US) for code, identifiers, comments and commit messages.
- PT-BR for conversation with the user, planning notes, PR descriptions.
- File length: hard cap 1000 lines, soft cap 800. Split by responsibility.
- Function length: prefer ≤ 60 lines; cognitive complexity low (early returns over nested ifs).
- Reuse before creating — search first.

### Go

- Idiomatic Go (`PascalCase` exported, `camelCase` unexported).
- File names `snake_case.go`.
- Imports grouped: stdlib / external / internal, separated by blank lines.
- `context.Context` first parameter in any IO-touching function.
- Wrap errors: `fmt.Errorf("operation X: %w", err)`.
- Map `sql.ErrNoRows` to `domain.ErrNotFound`.
- All response bodies through `internal/response/envelope.go` helpers (`response.OK/Created/BadRequest/...`).

### TypeScript / React

- File names `kebab-case.ts(x)`. Components `PascalCase`. Hooks `useThing`.
- Props: `Readonly<{...}>` or `interface` with all `readonly` fields.
- All API calls through `services/api/client.ts` (`fetchApi`, `fetchApiList`, `fetchApiDelete`).
- Server state through TanStack Query — query keys are arrays starting with the resource name.
- Routes are lazy-loaded in `app/routes.tsx`.
- Mobile-first Tailwind classes; tap targets ≥ 44 × 44 px.

---

## 10. Testing Expectations

The repo has tests today in:

- `apps/backend/internal/{pki,handler,provisioner,ghclient,webhook}/`
- `apps/shared/pkg/{compose,version}/`
- `apps/agent/internal/deploy/`

For new code:

- **Critical packages** (`pki`, `engine`, `webhook`, `provisioner`, `compose`, `executor`, deploy version detection): tests required in the same change.
- **For a reproduced bug** in any tested package: write a failing test first.
- Standard library `testing` only. Test names in English: `TestSubject_Behavior_Condition`.
- Failures via `t.Fatalf`/`t.Errorf` with descriptive messages.
- Table-driven for parsers/validators.
- Frontend tests are not yet pervasive — do not invent a test framework. If/when added, use Vitest + Testing Library.

---

## 11. Real-Time Events (SSE)

Single SSE endpoint `/events/deploys`. Closed set of event names (must stay in lockstep between backend `engine/notifier.go` and frontend `services/sse.ts` `SSE_EVENT_NAMES`):

```
deploy | log | health | stats | system_stats | server_stats | invalidate | provision | agent_update
```

Adding a new event = update both sides in the same change.

---

## 12. Versioning Cheat Sheet

| Where                        | What                  | When to bump                                                         |
| ---------------------------- | --------------------- | -------------------------------------------------------------------- |
| Root `package.json`          | Platform meta version | Auto-bumped by pre-commit when shared/proto/deploy/Dockerfile change |
| `apps/frontend/package.json` | Frontend version      | Auto-bumped when anything under `apps/frontend/**` changes           |
| `apps/backend/package.json`  | Backend version       | Auto-bumped when anything under `apps/backend/**` changes            |
| `AGENT_VERSION`              | Agent binary version  | `make bump-agent-version v=X.Y.Z` whenever agent behavior changes    |

The agent binary embeds `AGENT_VERSION` via `-ldflags -X github.com/paasdeploy/agent/internal/agent.Version=…`. The backend embeds the same value into `internal/handler.LatestAgentVersion` to know what to offer remote agents on update.

---

## 13. Gotchas (Things That Will Bite)

- **Go module replaces**: `apps/backend/go.mod` and `apps/agent/go.mod` both `replace github.com/paasdeploy/shared => ../shared`. The agent additionally `replace`s the backend module to import generated proto. Don't remove these.
- **Generated proto is checked in**: edit `.proto` files, run `make proto`, commit BOTH the `.proto` and the generated Go in the same change.
- **mTLS is mandatory**: agents identify themselves with a per-server cert from the backend's internal CA. Don't add toggles to disable verification.
- **Migrations run on boot**: a broken migration takes down the backend. Test on a scratch DB before merging.
- **SSE event names are a closed set**: producers and consumers must agree.
- **Docker socket access = root on the host**: the backend container has `/var/run/docker.sock` mounted RW. Treat any code path that reaches the docker package with extreme care.
- **`||` in TypeScript can mask `false`/`""`/`0`**: use `??` for fallback assignment. Several existing files were converted; new code must follow.
- **Don't import across features in the frontend**: promote shared bits to `@/components`, `@/hooks`, or `@/lib`.
- **`.env.example` is the contract**: production secrets come from CI / managed secret stores.

---

## 14. When You're Not Sure

Default to asking. Prefer 2–4 well-targeted questions over guessing. Especially for:

- Anything touching `apps/proto/**`, migrations, or `pki/`.
- Anything touching the deploy lifecycle (engine, queue, dispatcher, worker).
- Anything affecting authentication (cookie name, GitHub OAuth flow, mTLS).
- Anything affecting the production CI/CD pipeline.

The cost of a small wrong change in this codebase is high — it can take down the dashboard, lock out users from their servers, or expose every connected VPS host. A 30-second clarification is always cheaper.

---

## 15. Where to Find More

- `README.md` — user-facing project overview and feature list.
- `CHANGELOG.md` — version history.
- `docs/README.md` — index of all documentation files.
- `docs/ARCHITECTURE.md` — deeper architecture notes.
- `docs/CONTRIBUTING.md` — local setup, daily commands, workflow, testing.
- `docs/REMOTE_SSH_DEPLOY.md`, `docs/REMOTE_SERVER_TUTORIAL.md` — agent provisioning and remote deploys.
- `docs/AUTO_DNS_CONFIGURATION.md` — Cloudflare DNS integration.
- `docs/GITHUB_INTEGRATION.md` — webhooks, OAuth, GitHub App.
- `docs/SECURITY.md` — threat model, trust boundaries, hardening checklist, vulnerability reporting.
- `docs/K3S_DEPLOY_ROADMAP.md` — planned K3s runtime (opt-in).
- `docs/FEATURES_ROADMAP.md` — what's implemented today and what's planned next.

Skills in `.cursor/skills/`:

**Engineering expert skills** (mandatory — loaded automatically on matching triggers):

- `golang-engineering-expert` — Go 1.24 + Fiber + pgx + gRPC + wire craft; load for any `apps/backend/**`, `apps/agent/**`, `apps/shared/**`, `apps/proto/**`, or `**/*.go` edit.
- `frontend-engineering-expert` — React 18 + TanStack Query 5 + shadcn/ui + Tailwind mobile-first craft; load for any `apps/frontend/**` edit.

**Review skills** (mandatory final gate on every coding turn):

- `business-rule-guardian` — critical senior code reviewer that reads the original request, extracts business rules, maps rule → code, audits the full engineering quality axis (architecture, patterns, code smells, error handling, performance, security, observability, tests, docs, build), and refuses scope creep and "works locally". **Runs as the mandatory final gate of every coding turn** via a **fresh readonly subagent** (`Task`, `subagent_type: generalPurpose`, `readonly: true`) — never inline in the same session that wrote the code. Inline invocation inherits developer bias and defeats the skill's purpose. See `flowdeploy-development-protocol.mdc` §3.1 for the invocation contract and exception list, and the skill's "MANDATORY Invocation Rule" section for the exact prompt template.

**Workflow skills** (on demand):

- `create-adr` — for architectural decisions worth recording.
- `create-rfc` — for proposals before deciding.
- `create-technical-design-doc` — for implementation plans of large changes.
- `gh-address-comments` — for reacting to PR review feedback.
- `gh-fix-ci` — for failing GitHub Actions checks.
- `mermaid-studio` — for diagrams.

---

**Welcome aboard. Read the rule that matches your edit before you edit.**
