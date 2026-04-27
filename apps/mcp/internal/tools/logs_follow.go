package tools

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/toolkit"
)

const (
	defaultLogsTimeoutSeconds = 60
	maxLogsTimeoutSeconds     = 600
	defaultLogsIntervalSec    = 3
	minLogsIntervalSec        = 1
)

func followContainerLogs(ctx context.Context, req *mcp.CallToolRequest, deps toolkit.Deps, in containerLogsInput) (any, error) {
	timeout := time.Duration(clampInt(in.TimeoutSeconds, 1, maxLogsTimeoutSeconds, defaultLogsTimeoutSeconds)) * time.Second
	interval := time.Duration(clampInt(in.IntervalSec, minLogsIntervalSec, 60, defaultLogsIntervalSec)) * time.Second

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	token := req.Params.GetProgressToken()
	since := in.Since
	progress := float64(0)
	collected := []any{}

	for {
		payload, err := getJSON(ctx, deps.Backend, "/containers/"+pathSeg(in.ID)+"/logs", map[string]any{
			"tail":     in.Tail,
			"since":    since,
			"serverId": in.ServerID,
		})
		if err != nil {
			return nil, err
		}

		text := extractLogsText(payload)
		if text != "" {
			progress++
			collected = append(collected, text)
			notifyLogChunk(ctx, req, token, text, progress)
		}

		since = time.Now().UTC().Format(time.RFC3339)

		if time.Now().After(deadline) {
			return map[string]any{"data": collected, "tail_last_since": since}, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func extractLogsText(payload any) string {
	envelope, ok := payload.(map[string]any)
	if !ok {
		return ""
	}
	data := envelope["data"]
	switch v := data.(type) {
	case string:
		return v
	case []any:
		var out string
		for _, item := range v {
			if s, ok := item.(string); ok {
				out += s
			}
		}
		return out
	default:
		return ""
	}
}

func notifyLogChunk(ctx context.Context, req *mcp.CallToolRequest, token any, chunk string, progress float64) {
	if token == nil || req == nil || req.Session == nil {
		return
	}
	_ = req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
		ProgressToken: token,
		Message:       chunk,
		Progress:      progress,
	})
}
