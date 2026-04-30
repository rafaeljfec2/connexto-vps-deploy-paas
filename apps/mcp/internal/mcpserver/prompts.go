package mcpserver

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/toolkit"
)

func RegisterMCPPrompts(srv *mcp.Server, _ toolkit.Deps) {
	srv.AddPrompt(
		&mcp.Prompt{
			Name:        "diagnose_app",
			Description: "Diagnose an app by inspecting its details, recent deployments, container health and recent logs.",
			Arguments: []*mcp.PromptArgument{
				{Name: "app_id", Description: "the FlowDeploy app UUID", Required: true},
			},
		},
		func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			appID := req.Params.Arguments["app_id"]
			if appID == "" {
				return nil, mcp.ResourceNotFoundError("app_id is required")
			}
			text := "You are a FlowDeploy operator. Diagnose app '" + appID + "' end-to-end:\n\n" +
				"1. Call apps_get with id='" + appID + "' and report status, branch and last deployment.\n" +
				"2. Call apps_deployments with id='" + appID + "' and summarize the last 3 deployments (status, commit, finished_at).\n" +
				"3. Call apps_health with id='" + appID + "' and surface unhealthy containers.\n" +
				"4. For any unhealthy container call containers_logs (tail=200) and highlight the last error.\n" +
				"5. Conclude with a 3-line root-cause hypothesis and a recommended next action (do NOT mutate state)."
			return &mcp.GetPromptResult{
				Description: "Diagnose a FlowDeploy app",
				Messages: []*mcp.PromptMessage{
					{Role: "user", Content: &mcp.TextContent{Text: text}},
				},
			}, nil
		},
	)

	srv.AddPrompt(
		&mcp.Prompt{
			Name:        "audit_recent_changes",
			Description: "Summarize FlowDeploy audit logs from the last 24h grouped by actor and action.",
			Arguments: []*mcp.PromptArgument{
				{Name: "limit", Description: "max number of audit log entries to inspect (default 200)"},
			},
		},
		func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			limit := req.Params.Arguments["limit"]
			if limit == "" {
				limit = "200"
			}
			startDate := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
			text := "You are a FlowDeploy auditor. Summarize recent activity using the audit log:\n\n" +
				"1. Call audit_logs with limit=" + limit + " and start_date=" + startDate + " (RFC3339; 24h ago).\n" +
				"2. Group the returned entries by actorType (user, pat, system, webhook) and by eventType.\n" +
				"3. Highlight any destructive action (delete, prune, remove) and the reason recorded.\n" +
				"4. Output a markdown table: actor | event | resource | timestamp | reason."
			return &mcp.GetPromptResult{
				Description: "Audit recent changes",
				Messages: []*mcp.PromptMessage{
					{Role: "user", Content: &mcp.TextContent{Text: text}},
				},
			}, nil
		},
	)
}
