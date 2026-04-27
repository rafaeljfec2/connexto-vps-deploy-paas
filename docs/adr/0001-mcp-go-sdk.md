# ADR 0001 — MCP Go SDK selection

- Status: accepted
- Date: 2026-04-27
- Deciders: FlowDeploy core team

## Context

Phase 1 of the [MCP Server for FlowDeploy plan](../../.cursor/plans/mcp_server_for_flowdeploy_7df85276.plan.md) introduces a new Go binary (`apps/mcp/cmd/flowdeploy-mcp`) that speaks the Model Context Protocol over stdio (Phase 1) and HTTP+SSE (Phase 4). The plan calls for an explicit ADR before fixing the SDK version.

Two viable Go SDK options were evaluated:

1. `github.com/modelcontextprotocol/go-sdk` — the official SDK maintained by the MCP project in collaboration with Google.
2. `github.com/mark3labs/mcp-go` — a community SDK that predates the official one.

## Decision

We adopt the **official `github.com/modelcontextprotocol/go-sdk`** at version **v1.5.0** (released 2026-03-31).

Reasons:

- Officially maintained by the MCP project and Google, aligning with the protocol roadmap.
- Implements the full MCP spec (latest `2025-11-25`) with backward compatibility down to `2024-11-05`.
- Stable API surface since v1.x — no major breaking changes expected.
- First-class support for stdio, command, HTTP+SSE transports (matches the Phase 4 plan).
- Stabilised client-side OAuth in v1.5.0, which we will leverage in Phase 4 for the Device Authorization Grant.
- Active release cadence (multiple releases in March/April 2026) with backports.

The community SDK (`mark3labs/mcp-go`) is a credible fallback but introduces a different idiomatic style (functional options vs. struct-based) and would need to be revisited if the project evolves on a slower cadence than the official one.

## Version pinning

```
go 1.25.0

require github.com/modelcontextprotocol/go-sdk v1.5.0
```

`go 1.25.0` is the minimum Go toolchain required by the SDK. All other modules in the monorepo (`apps/backend`, `apps/agent`, `apps/shared`) remain on `go 1.24.x` because they do not depend on this SDK. Developers will need Go 1.24+ installed; the `toolchain` directive will auto-download Go 1.25 when building this module.

## Consequences

- Bumping the SDK requires a follow-up ADR amendment.
- Bumping the SDK is gated by the [`flowdeploy-development-protocol.mdc`](../../.cursor/rules/flowdeploy-development-protocol.mdc) review process: business-rule-guardian + senior-code-reviewer must approve any breaking changes before merging.
- Phase 4 (HTTP+SSE + Device Authorization Grant) directly relies on the OAuth primitives shipped with v1.5.0; downgrading would force re-implementation.

## Fallback plan

If the official SDK becomes unmaintained or introduces a hard regression, we will:

1. Migrate to `github.com/mark3labs/mcp-go` with a new ADR.
2. Provide an adapter layer in `apps/mcp/internal/mcpserver/` so tool handlers do not need to change.
