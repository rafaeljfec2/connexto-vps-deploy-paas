# Follow-up: `containers_logs.since` end-to-end

Status: **OPEN** — needs dedicated PR
Owner: backend + agent
Created: 2026-04-30
Source: senior code review of MCP audit/follow-up cleanup turn

---

## Summary

The `since` parameter on the MCP `containers_logs` tool is silently ignored
end-to-end today. As a consequence, the `logs_follow` polling loop
(`apps/mcp/internal/tools/logs_follow.go`) keeps re-pulling the same
window on every tick instead of incrementally streaming new lines.

The fix is straightforward in shape but touches **proto, agent, backend,
shared and MCP** in a single change, plus a bump of `AGENT_VERSION` —
that is why it was carved out as a dedicated PR rather than bundled with
the MCP filter cleanup.

---

## Current state (what is wired vs. what is broken)

| Layer | File | State |
|---|---|---|
| Proto | `apps/proto/flowdeploy/v1/server.proto:181` | ✅ already declares `optional google.protobuf.Timestamp since = 5` on `ContainerLogsRequest`. No regen needed. |
| Generated Go | `apps/backend/gen/go/flowdeploy/v1/*.pb.go` | ✅ already has `GetSince()` accessor. |
| Agent gRPC handler | `apps/agent/internal/grpcserver/handlers_containers.go:74-98` | ❌ `GetContainerLogs` reads `req.ContainerId`, `req.Tail`, `req.Follow` — never reads `req.Since`. Both `sendStaticLogs` and `streamFollowLogs` have no `since` plumbing. |
| Agent docker wrapper | `apps/shared/pkg/docker/client.go:315` (`ContainerLogs`) and `:338` (`StreamContainerLogs`) | ❌ `func (d *Client) ContainerLogs(ctx, id, tail) (string, error)` does not accept `since`. `docker logs --since` flag is not passed. |
| Backend agent client | `apps/backend/internal/agentclient/agent_client_containers.go:38` | ❌ `func (c *AgentClient) GetContainerLogs(ctx, host, port, containerID, tail, follow, onLog)` — no `since` parameter. |
| Backend HTTP handler (apps) | `apps/backend/internal/handler/container_health_handler.go:160-238` | ❌ `GetContainerLogs` reads `tail` and `follow` only. |
| Backend HTTP handler (containers) | `apps/backend/internal/handler/container_logs_handler.go:22-113` | ❌ Same issue. Local `h.docker.ContainerLogs` and remote `h.agentClient.GetContainerLogs` both miss `since`. |
| MCP tool schema | `apps/mcp/internal/tools/containers.go:23` | ⚠️ `Since string` is already declared and `containers.go:64` already forwards it as `since=<value>` to the backend. Needs RFC3339/duration validation hardened (reject malformed values early via `errInvalidArg` instead of silently shipping garbage to the backend). |
| MCP follow loop | `apps/mcp/internal/tools/logs_follow.go:33-49` | ❌ Already forwards `since`, but backend ignores it — every tick re-pulls the full tail window. Will start working correctly once the agent/backend chain honours the parameter. |

---

## Scope (single PR)

1. **Backend HTTP**: read `c.Query("since")` (RFC3339), parse to
   `*time.Time`, return 400 on invalid. Affects:
   - `container_logs_handler.go` (`GetContainerLogs`, `getRemoteContainerLogs`,
     `streamRemoteContainerLogs`, `streamContainerLogs`).
   - `container_health_handler.go` (`GetContainerLogs`).
2. **Backend agent client**: extend signature
   `GetContainerLogs(ctx, host, port, containerID, tail, follow, since *time.Time, onLog)`.
   Update all call sites in the two handlers above.
3. **Backend local docker wrapper** (`apps/shared/pkg/docker`): extend
   `Client.ContainerLogs` and `Client.StreamContainerLogs` to accept
   `since *time.Time`. Map to `docker logs --since=<RFC3339>` when set.
   Audit all call sites (engine health monitor, stats monitor, deploy
   logger — they pass `tail` only and stay backwards-compatible by
   passing `nil`).
4. **Agent gRPC handler**: read `req.GetSince()` and forward to the
   shared docker wrapper. Both `sendStaticLogs` and `streamFollowLogs`.
5. **MCP tool schema**: the `Since string` field already exists
   (`containers.go:23`) and is already forwarded (`containers.go:64`,
   `logs_follow.go:35`). Add early RFC3339/duration validation in the
   `containers_logs` handler so malformed values are rejected with
   `errInvalidArg("invalid since: must be RFC3339 or duration like 1h")`
   instead of silently propagating to the backend (which today returns
   500/empty).
6. **MCP follow loop**: keep the existing polling logic — it will work
   correctly once the backend respects `since`. Recommended improvement:
   parse the last log entry's timestamp and update `since` from it
   (instead of `time.Now()`), to avoid skipping logs that arrive within
   the same tick.
7. **Tests**:
   - Unit test on `agent_client_containers_test.go` (proto request
     carries `since`).
   - Handler test on `container_logs_handler_test.go` covering invalid
     RFC3339 → 400.
   - Agent gRPC handler test covering `req.Since` propagation.
   - MCP test confirming `since` is forwarded to backend as `since=`.
8. **`AGENT_VERSION` bump**: required, the agent contract behavior
   changes. Run `make bump-agent-version v=X.Y.Z`.

---

## Out of scope (separate work)

- Switching the MCP follow tool to **server-sent events** (currently
  polls). That would replace the loop entirely.
- Replacing the SSE bridge in `streamRemoteContainerLogs` /
  `streamContainerLogs` with a single gRPC follow stream that respects
  client cancellation more aggressively.

---

## Estimated effort

~1.5–2h of focused work plus review. Risk is medium because it touches
the shared docker wrapper (consumed by engine health/stats and the deploy
logger). Backwards compatibility is preserved by making `since`
optional everywhere.

---

## Acceptance criteria

- `curl /paas-deploy/v1/containers/<id>/logs?since=2026-04-30T12:00:00Z`
  returns only entries newer than the cutoff (local + remote paths).
- `--since` is observed on the host via `docker logs --since=...` (CLI
  argument carried verbatim).
- MCP `containers_logs` accepts `since` and forwards it without
  mangling. The `logs_follow` tool consumes only **new** lines per tick
  in steady state.
- All unit tests green; `pnpm run lint:go-quick` and full `lint:go`
  pass; `cd apps/backend && go test ./...`,
  `cd apps/shared && go test ./...`,
  `cd apps/agent && go test ./...` all pass.
- `AGENT_VERSION` bumped; backend embeds the new value via `-ldflags`
  (verified in `internal/handler.LatestAgentVersion`).
