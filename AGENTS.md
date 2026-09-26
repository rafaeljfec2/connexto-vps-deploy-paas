# AGENTS.md — FlowDeploy

This file is the entry point for coding agents. It is a map, not a spec.
If anything here conflicts with `.cursor/rules/flowdeploy-development-protocol.mdc`,
the rule wins.

## 1. What this is

Self-hosted PaaS: GitHub webhook → deploy queue → Docker on the control-plane
host or a remote VPS agent (gRPC + mTLS). Dashboard is a React SPA (SSE logs).

Make the smallest correct change. Do not change business behavior unless asked.

## 2. Repository map

```
apps/backend    Go control plane (Fiber HTTP + deploy engine + gRPC)
apps/agent      Go gRPC agent on remote VPS hosts
apps/mcp        Go MCP server (PAT-scoped tools over the HTTP API)
apps/proto      buf protos → generated into apps/backend/gen/go
apps/shared     Go primitives only (docker, executor, git, lock, paths)
apps/frontend   React 18 + Vite + TanStack Query + shadcn/ui
deploy/         Compose + Traefik for the control-plane host
docs/           Specs — do not edit unless asked
```

Go modules: `github.com/paasdeploy/{backend,agent,shared,mcp}`.
`backend` and `agent` `replace` shared (and agent replaces backend for generated proto).
Do not remove those `replace` directives.

## 3. Fixed stack

Do not add a framework, ORM or major dependency without an explicit ask.

| Layer    | Choice                                     |
| -------- | ------------------------------------------ |
| API      | Fiber v2, pgx via database/sql, wire, slog |
| RPC      | proto3 + buf, grpc-go                      |
| Agent    | gRPC, mTLS, stdlib-first                   |
| Frontend | React 18, Vite, Tailwind, shadcn, TanStack |
| Data     | PostgreSQL, golang-migrate                 |
| Proxy    | Traefik 3.x                                |

## 4. How to work

- Inspect consumers (`Grep`) before editing a shared symbol.
- Prefer an existing helper over a new abstraction.
- Do not guess when the repo can answer.
- Keep the change scoped to the request. No drive-by refactors.
- After edits: read the diff, run the narrowest test/lint, then widen if the change crosses a boundary.
- English (US) in code/comments/commits. PT-BR in conversation with the user.

## 5. Forbidden

- Weaken mTLS / set `InsecureSkipVerify` (emergency switch must default false).
- Edit `apps/backend/gen/go/**` — change `.proto` and run `make proto`.
- Edit an applied migration. Add the next sequential pair.
- Edit `.env`. Change `.env.example` and ask.
- Log or return secrets (tokens, env values, PEM, webhook payloads).
- `sh -c` or string-interpolated shell. Use `shared/pkg/executor` with `name, args...`.
- JWT for sessions. Opaque cookie + hashed token in DB is intentional.
- `any` in TypeScript; `||` for fallback (`??` / explicit zero-check).
- Force-push `main` or `--no-verify` without explicit approval.

## 6. Stop and re-read the security rule before changing

Auth, PKI/mTLS, webhook HMAC, migrations, deploy engine/queue, agent execution,
Docker socket, public proto/API, production CI/CD.

## 7. Canonical commands

```bash
make proto                          # buf lint + generate
pnpm run lint:go-quick
cd apps/backend && go test ./internal/<pkg>/...
cd apps/mcp && go test ./internal/tools/
cd apps/agent && go test ./...
pnpm --filter @paasdeploy/frontend run typecheck
```

Versions: platform in `package.json`; agent binary in `AGENT_VERSION`
(`make bump-agent-version v=X.Y.Z` when agent behavior changes).

## 8. Load before editing

| Touching                                            | Load                                                                 |
| --------------------------------------------------- | -------------------------------------------------------------------- |
| `apps/{backend,agent,shared,proto,mcp}/**`, `*.go`  | `.cursor/skills/golang-engineering-expert/SKILL.md`                  |
| `apps/frontend/**`                                  | `.cursor/skills/frontend-engineering-expert/SKILL.md`                |
| Any behavior-changing edit                          | `.cursor/skills/business-rule-guardian/SKILL.md` as a **fresh readonly** review |

Layer conventions: `.cursor/rules/flowdeploy-{backend-go,agent-go,frontend-react,proto-grpc,shared-go,security,ops}.mdc`.

## 9. Deeper docs (do not copy here)

- `docs/ARCHITECTURE.md` — runtime
- `docs/SECURITY.md` — threat model
- `docs/GITHUB_INTEGRATION.md` — OAuth / webhooks
- `docs/REMOTE_SSH_DEPLOY.md` — agent provision
- `docs/MCP_CLIENT_SETUP.md` — MCP tools and client config
