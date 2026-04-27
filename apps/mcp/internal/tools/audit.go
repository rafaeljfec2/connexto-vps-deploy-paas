package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/toolkit"
)

type auditLogsInput struct {
	ActorType string `json:"actor_type,omitempty" jsonschema:"filter by actor type: user, pat, system"`
	ActorID   string `json:"actor_id,omitempty" jsonschema:"filter by actor ID"`
	Action    string `json:"action,omitempty" jsonschema:"filter by action name"`
	From      string `json:"from,omitempty" jsonschema:"start timestamp (RFC3339)"`
	To        string `json:"to,omitempty" jsonschema:"end timestamp (RFC3339)"`
	Limit     int    `json:"limit,omitempty" jsonschema:"max results (default 50, max 500)"`
}

func RegisterAudit(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "audit_logs",
			Description: "Query audit logs. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in auditLogsInput) (any, error) {
			return getJSON(ctx, deps.Backend, "/audit/logs", map[string]any{
				"actorType": in.ActorType,
				"actorId":   in.ActorID,
				"action":    in.Action,
				"from":      in.From,
				"to":        in.To,
				"limit":     in.Limit,
			})
		})

	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "audit_webhook_payloads",
			Description: "List recent inbound webhook payloads (GitHub, etc). Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in auditLogsInput) (any, error) {
			return getJSON(ctx, deps.Backend, "/audit/webhook-payloads", map[string]any{"limit": in.Limit})
		})
}
