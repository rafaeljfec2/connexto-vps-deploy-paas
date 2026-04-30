package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/toolkit"
)

// auditLogsInput mirrors the query parameters honored by the backend's
// /audit/logs handler (apps/backend/internal/handler/audit_handler.go,
// buildAuditFilter). Field names use snake_case in JSON (MCP convention) and
// are mapped to the backend's camelCase query keys when forwarding the
// request. Filters not honoured by the backend (actor_type, actor_id, action,
// user_id) were removed to stop declaring filters that silently no-op. Note:
// the backend always overrides filter.UserID to the authenticated user's id
// (audit_handler.go:62-63), so user_id from the client is never honored
// regardless of role.
type auditLogsInput struct {
	EventType    string `json:"event_type,omitempty" jsonschema:"filter by event type (e.g. token.created, user.logged_in)"`
	ResourceType string `json:"resource_type,omitempty" jsonschema:"filter by resource type (e.g. token, user, app, server)"`
	ResourceID   string `json:"resource_id,omitempty" jsonschema:"filter by resource ID (UUID)"`
	StartDate    string `json:"start_date,omitempty" jsonschema:"start timestamp (RFC3339)"`
	EndDate      string `json:"end_date,omitempty" jsonschema:"end timestamp (RFC3339)"`
	Limit        int    `json:"limit,omitempty" jsonschema:"max results (default 50, max 500)"`
	Offset       int    `json:"offset,omitempty" jsonschema:"pagination offset (default 0)"`
}

type auditWebhookPayloadsInput struct {
	Limit  int `json:"limit,omitempty" jsonschema:"max results (default 25)"`
	Offset int `json:"offset,omitempty" jsonschema:"pagination offset (default 0)"`
}

func RegisterAudit(srv *mcp.Server, deps toolkit.Deps) {
	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "audit_logs",
			Description: "Query audit logs. Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in auditLogsInput) (any, error) {
			return getJSON(ctx, deps.Backend, "/audit/logs", map[string]any{
				"eventType":    in.EventType,
				"resourceType": in.ResourceType,
				"resourceId":   in.ResourceID,
				"startDate":    in.StartDate,
				"endDate":      in.EndDate,
				"limit":        in.Limit,
				"offset":       in.Offset,
			})
		})

	toolkit.RegisterReadOnly(srv, deps,
		&mcp.Tool{
			Name:        "audit_webhook_payloads",
			Description: "List recent inbound webhook payloads (GitHub, etc). Requires scope 'read'.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, in auditWebhookPayloadsInput) (any, error) {
			return getJSON(ctx, deps.Backend, "/audit/webhook-payloads", map[string]any{
				"limit":  in.Limit,
				"offset": in.Offset,
			})
		})
}
