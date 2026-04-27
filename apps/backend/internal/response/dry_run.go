package response

import "github.com/gofiber/fiber/v2"

// DryRunReport describes what a destructive action would do, without executing it.
// Returned with HTTP 200 and the standard envelope when the request comes with
// X-Dry-Run: true (or ?dry_run=true).
type DryRunReport struct {
	Action      string         `json:"action"`
	Resource    string         `json:"resource"`
	ResourceID  string         `json:"resourceId,omitempty"`
	Description string         `json:"description"`
	Effects     []string       `json:"effects"`
	Reversible  bool           `json:"reversible"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// DryRun returns 200 with a DryRunReport in the envelope's `data` and a top-level
// `meta.warnings` entry making it explicit no mutation was performed.
func DryRun(c *fiber.Ctx, report DryRunReport) error {
	meta := Meta{
		TraceID:  getTraceID(c),
		Warnings: []string{"dry-run: no mutation performed"},
	}
	return sendWithMeta(c, fiber.StatusOK, report, nil, meta)
}
