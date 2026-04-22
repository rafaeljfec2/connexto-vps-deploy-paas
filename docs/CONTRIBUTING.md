# Contributing to FlowDeploy

Thank you for your interest in contributing to FlowDeploy. This guide explains
how to get the project running locally, the workflow we expect from every
change, and where to look when something does not behave as documented.

> Read these first:
>
> - [`AGENTS.md`](../AGENTS.md) — high-level mental model and hard rules
> - [`docs/ARCHITECTURE.md`](./ARCHITECTURE.md) — runtime architecture
> - [`docs/SECURITY.md`](./SECURITY.md) — security model and hardening checklist
> - `.cursor/rules/flowdeploy-*.mdc` — per-area conventions enforced in review

## 1. Prerequisites

| Tool             | Version       | Notes                                                          |
| ---------------- | ------------- | -------------------------------------------------------------- |
| Go               | 1.24+         | `go.mod` declares `go 1.24.0`, toolchain `go1.24.13`           |
| Node.js          | 20+           | LTS line                                                       |
| pnpm             | 9+            | `packageManager` field pins `pnpm@9.15.0`                      |
| Docker Engine    | 24+           | Required for the local stack and deploys                       |
| Docker Compose   | v2            | Used by the platform itself and by deployed apps               |
| PostgreSQL       | 16+           | Provided by the local compose stack                            |
| `golangci-lint`  | latest stable | Backend / agent / shared linting                               |
| `buf`            | 1.38+         | Protobuf generation and breaking-change checks                 |

Optional but useful: `air` (Go live reload, used by `pnpm backend:dev`),
`gh` (GitHub CLI), and `make`.

## 2. Repository layout

```
apps/
├── backend/      Go HTTP + gRPC + deploy engine
├── agent/        Go gRPC agent installed on remote VPSs
├── frontend/     React 18 + Vite 6 SPA
├── shared/       Zero-dependency Go primitives
└── proto/        buf workspace and proto definitions
deploy/           docker-compose.yml + Traefik config
docs/             Architectural and operational docs
.cursor/rules/    Workspace rules for AI agents and humans
AGENT_VERSION     Single source of truth for the agent binary version
```

## 3. Local setup

```bash
git clone <repo-url> flowdeploy
cd flowdeploy
pnpm install
cp .env.example .env       # adjust values, never commit
pnpm docker:up             # PostgreSQL + Traefik
pnpm backend:dev           # backend with live reload (air)
pnpm dev:web               # frontend on http://localhost:5173
```

The backend automatically applies the embedded SQL migrations on startup.

To exercise the agent locally, run it against the dev backend:

```bash
cd apps/agent
go run cmd/agent/main.go \
  --server-addr=localhost:50051 \
  --server-id=<id> \
  --agent-port=50052
```

## 4. Daily commands

| Command                       | What it does                                          |
| ----------------------------- | ----------------------------------------------------- |
| `pnpm dev`                    | Backend + frontend in development                     |
| `pnpm backend:dev`            | Backend with `air` live reload                        |
| `pnpm dev:web`                | Frontend only                                         |
| `pnpm build`                  | Turborepo build for everything                        |
| `pnpm typecheck`              | TypeScript type-checking across the workspace         |
| `pnpm lint`                   | All linters (Go + TypeScript)                         |
| `pnpm lint:frontend`          | ESLint for the SPA                                    |
| `pnpm lint:go` / `:go-quick`  | `golangci-lint` for backend, agent and shared         |
| `pnpm backend:test`           | `go test ./...` inside `apps/backend`                 |
| `pnpm precommit:local`        | Mirrors the husky pre-commit hook                     |
| `pnpm docker:up` / `:down`    | Bring the local infra stack up/down                   |

## 5. Workflow for every change

1. **Pull and branch**: `git pull --rebase`, then create a focused branch.
2. **Plan** before coding (Planner Mode). Check the relevant
   `.cursor/rules/*.mdc`. If the task touches more than one area
   (backend + proto + frontend), update **all** of them in lockstep.
3. **Implement** following the conventions of the touched area.
4. **Validate locally** (mandatory before push):
   - `pnpm lint`
   - `pnpm typecheck`
   - `pnpm backend:test`
   - `pnpm build` if the change affects build output
5. **Update docs** when behavior, contracts or operational steps change:
   - `README.md` for top-level features
   - `docs/` for architecture, integrations, tutorials
   - `CHANGELOG.md` under the **Unreleased** section
6. **Commit** with a meaningful message. The pre-commit hook bumps patch
   versions of the affected workspaces and runs `lint-staged` automatically.
7. **Open a PR** describing what changed, why, the impact, and how it was
   tested. Link related issues.

## 6. Coding standards (high level)

Full rules live in `.cursor/rules/flowdeploy-*.mdc`. Quick reference:

### Go (backend / agent / shared)

- `gofmt`, `goimports`, `golangci-lint` all clean.
- Layered: `handler → service → repository → domain`.
- DI through `google/wire` in the backend; struct injection elsewhere.
- All shell commands via `shared/pkg/executor` with explicit arg vectors.
- Logging via `log/slog`; no `fmt.Println`.
- Errors wrapped with `%w`; never swallow with blank identifier.
- No business logic inside `apps/shared`.

### TypeScript / React (frontend)

- Strict TypeScript; **never** use `any`.
- Use `??` for nullish defaults, never `||`.
- Mark React component props as `readonly` (sonarqube `typescript:S6759`).
- Server state in TanStack Query; UI state in components.
- Mobile-first Tailwind utility classes; reuse `shadcn/ui` primitives.
- Prefer `Promise.all` only for independent async work, `Promise.allSettled`
  when partial failures are acceptable.

### Protobuf

- Edit `.proto` files in `apps/proto`, regenerate with `buf generate`.
- Run `buf lint` and `buf breaking` before pushing.
- Update both backend and agent implementations in the same commit.

### General

- Code, identifiers and comments in English (US).
- Conversation, planning and PR descriptions in PT-BR (project preference).
- No comments that just narrate the code.
- Keep files under ~800–1000 lines; split when they grow beyond that.

## 7. Testing expectations

- **Unit tests** for business logic in services and any non-trivial helper.
- **Repository tests** for SQL queries, using a disposable PostgreSQL.
- **Frontend tests** for hooks and components with non-trivial logic.
- Test descriptions in **English**, deterministic, independent of execution
  order. Use table-driven tests in Go where it improves clarity.

## 8. Database migrations

- Add a new pair `NNN_description.up.sql` / `NNN_description.down.sql` in
  `apps/backend/migrations/`.
- Numbering is monotonically increasing — never reuse a number.
- The backend applies migrations on startup. There is no separate CLI step.
- Avoid destructive operations (`DROP TABLE`, `DROP COLUMN`) without a clear
  rollback story; prefer additive migrations.

## 9. Protobuf changes

```bash
cd apps/proto
buf lint
buf breaking --against '.git#branch=main'
buf generate
```

Generated code lives in `apps/backend/gen/go` and `apps/agent/gen/go`. After
generating, update both server and client implementations and rerun the linters
and tests.

## 10. Security checklist

Before opening a PR, confirm that you did **not**:

- Add a secret to the repo (use `.env`, encrypted columns, or a secret store).
- Build SQL with string interpolation.
- Run shell commands with `sh -c` or untrusted interpolation.
- Disable mTLS, CORS allow-listing, or webhook signature verification.
- Bypass the `RequireAdminForLocal` guard.
- Log sensitive payloads (tokens, env vars, request bodies with secrets).

See `.cursor/rules/flowdeploy-security.mdc` for the full security playbook.

## 11. Release and versioning

- `package.json` drives the **platform** version.
- `apps/frontend/package.json` and `apps/backend/package.json` track their own
  versions for changelog clarity.
- `AGENT_VERSION` (root file) is the canonical agent version. The backend
  embeds it via `-ldflags`; the agent reports it on registration.
- The pre-commit hook bumps the patch version of every workspace touched by
  the staged files. Bump minor/major manually when needed.
- Update `CHANGELOG.md` (`## [Unreleased]` section) for every user-facing
  change. Keep entries short and grouped (Added / Changed / Fixed / Security).

## 12. Reporting issues

When opening an issue, include:

- Component (`backend`, `agent`, `frontend`, `proto`, `infra`).
- Affected version (`package.json`, `AGENT_VERSION`, commit SHA).
- Steps to reproduce and expected vs. actual behavior.
- Relevant logs (with secrets redacted) and, when possible, a minimal
  `paasdeploy.json` reproducing the issue.

## 13. Where to ask

- Architecture or design questions → start in [`docs/ARCHITECTURE.md`](./ARCHITECTURE.md).
- Contract questions (gRPC) → look at `apps/proto/flowdeploy/v1/*.proto`.
- Conventions enforced in review → `.cursor/rules/flowdeploy-*.mdc`.
- Onboarding for AI agents and new humans → [`AGENTS.md`](../AGENTS.md).
