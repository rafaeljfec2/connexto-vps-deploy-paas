package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paasdeploy/mcp/internal/toolkit"
)

const (
	defaultDeployTimeoutSeconds = 300
	maxDeployTimeoutSeconds     = 1800
	defaultDeployPollSeconds    = 5
	minDeployPollSeconds        = 2
)

var deployTerminalStates = map[string]bool{
	"success":   true,
	"succeeded": true,
	"failed":    true,
	"failure":   true,
	"cancelled": true,
	"canceled":  true,
	"error":     true,
}

func waitForDeployment(ctx context.Context, req *mcp.CallToolRequest, deps toolkit.Deps, in deployStatusInput) (any, error) {
	timeout := time.Duration(clampInt(in.TimeoutSeconds, 1, maxDeployTimeoutSeconds, defaultDeployTimeoutSeconds)) * time.Second
	interval := time.Duration(clampInt(in.PollIntervalSec, minDeployPollSeconds, 60, defaultDeployPollSeconds)) * time.Second

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	token := req.Params.GetProgressToken()
	var lastStatus string
	progress := float64(0)

	for {
		current, status, err := fetchLatestDeployment(ctx, deps, in.ID)
		if err != nil {
			return nil, err
		}
		normalized := strings.ToLower(strings.TrimSpace(status))
		if status != lastStatus {
			progress++
			notifyDeployProgress(ctx, req, token, status, progress)
			lastStatus = status
		}
		if deployTerminalStates[normalized] {
			return current, nil
		}
		if time.Now().After(deadline) {
			return current, fmt.Errorf("timeout waiting for deployment of app %s (last status %q)", in.ID, status)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func fetchLatestDeployment(ctx context.Context, deps toolkit.Deps, appID string) (any, string, error) {
	payload, err := getJSON(ctx, deps.Backend, "/apps/"+pathSeg(appID)+"/deployments", nil)
	if err != nil {
		return nil, "", err
	}
	envelope, ok := payload.(map[string]any)
	if !ok {
		return payload, "", nil
	}
	data := envelope["data"]
	switch v := data.(type) {
	case []any:
		if len(v) == 0 {
			return payload, "", nil
		}
		first, _ := v[0].(map[string]any)
		return payload, statusFromDeployment(first), nil
	case map[string]any:
		return payload, statusFromDeployment(v), nil
	default:
		return payload, "", nil
	}
}

func statusFromDeployment(deployment map[string]any) string {
	if deployment == nil {
		return ""
	}
	for _, key := range []string{"status", "state", "phase"} {
		if raw, ok := deployment[key]; ok {
			if s, ok := raw.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

func notifyDeployProgress(ctx context.Context, req *mcp.CallToolRequest, token any, status string, progress float64) {
	if token == nil || req == nil || req.Session == nil {
		return
	}
	_ = req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
		ProgressToken: token,
		Message:       deployProgressMessage(status),
		Progress:      progress,
	})
}

func deployProgressMessage(status string) string {
	if status == "" {
		return "deployment in progress"
	}
	return "deployment status: " + status
}

func clampInt(value, min, max, fallback int) int {
	if value <= 0 {
		return fallback
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
