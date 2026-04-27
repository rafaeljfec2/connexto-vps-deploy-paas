package toolkit

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	minReasonRunes = 8
	maxReasonRunes = 500
)

// DryRunOptions captures the safety levers exposed by every destructive tool.
// The zero value (Commit=false, Reason="") is intentionally SAFE: it triggers a
// dry-run preview without mutating anything. Callers must set Commit=true AND
// supply a non-trivial Reason to actually execute.
//
// Tools should embed this struct literally on their input type so the MCP SDK
// generates the corresponding schema entries.
type DryRunOptions struct {
	Commit bool   `json:"commit,omitempty" jsonschema:"set to true to actually execute the destructive action; defaults to false (dry-run preview)"`
	Reason string `json:"reason,omitempty" jsonschema:"mandatory free-text justification (8-500 chars) when commit=true; captured in the audit log"`
}

const (
	HeaderDryRun       = "X-Dry-Run"
	HeaderActionReason = "X-Action-Reason"
)

// DestructiveHeaders builds the HTTP headers that must accompany every
// destructive backend call. When opts.Commit is false the backend returns a
// DryRunReport instead of executing. When Commit is true, opts.Reason is
// forwarded (trimmed) so the audit trail records why the operation was
// performed.
func DestructiveHeaders(opts DryRunOptions) map[string]string {
	headers := map[string]string{}
	if !opts.Commit {
		headers[HeaderDryRun] = "true"
	}
	if reason := strings.TrimSpace(opts.Reason); reason != "" {
		headers[HeaderActionReason] = reason
	}
	return headers
}

// EnsureDestructiveCommit validates that a commit (Commit=true) carries a
// well-formed reason. Tools call this BEFORE issuing the backend request so the
// caller gets a clear local error instead of a remote 400. Whitespace is
// trimmed before validation and length is measured in runes (so that PT-BR
// justifications with accents satisfy the same character budget).
func EnsureDestructiveCommit(opts DryRunOptions) error {
	if !opts.Commit {
		return nil
	}
	reason := strings.TrimSpace(opts.Reason)
	length := utf8.RuneCountInString(reason)
	if length < minReasonRunes || length > maxReasonRunes {
		return fmt.Errorf("reason must be between %d and %d characters when commit=true", minReasonRunes, maxReasonRunes)
	}
	return nil
}

// RegisterDestructive wires a destructive tool with the same logging surface as
// RegisterWrite but tagged with mode "destructive" so observability can filter
// them apart easily.
func RegisterDestructive[Input any](
	srv *mcp.Server,
	deps Deps,
	tool *mcp.Tool,
	handler ToolHandler[Input],
) {
	registerTool(srv, deps, tool, func(ctx context.Context, req *mcp.CallToolRequest, in Input) (any, error) {
		return handler(ctx, req, in)
	}, "destructive")
}
