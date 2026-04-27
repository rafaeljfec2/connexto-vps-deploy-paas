package middleware

import (
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/response"
)

const (
	HeaderDryRun       = "X-Dry-Run"
	HeaderActionReason = "X-Action-Reason"
	QueryDryRun        = "dry_run"
	QueryReason        = "reason"
	MinReasonLength    = 8
	MaxReasonLength    = 500
)

// IsDryRun returns true when the caller signals dry-run mode via the X-Dry-Run
// header or the dry_run query string. Truthy values: "1", "true", "yes", "on"
// (case-insensitive). Falsy values: missing, "0", "false", "no", "off".
//
// Fail-safe: if the header/query is present but unrecognized (e.g. "yep" or
// random garbage), the value is treated as DRY-RUN. This protects against
// buggy clients that mistype the flag and expect a preview.
func IsDryRun(c *fiber.Ctx) bool {
	if v := c.Get(HeaderDryRun); v != "" {
		return !isFalsy(v)
	}
	if v := c.Query(QueryDryRun); v != "" {
		return !isFalsy(v)
	}
	return false
}

// GetActionReason returns the reason supplied by the caller (header takes priority
// over the query parameter) trimmed of surrounding whitespace.
func GetActionReason(c *fiber.Ctx) string {
	if reason := strings.TrimSpace(c.Get(HeaderActionReason)); reason != "" {
		return reason
	}
	return strings.TrimSpace(c.Query(QueryReason))
}

// IsValidReason checks the length boundaries (counted in runes, not bytes, so
// that Unicode-rich justifications such as PT-BR sentences with accents satisfy
// the same character budget that English sentences do). Boundaries are
// intentionally tight enough to require a meaningful explanation while
// allowing freeform sentences.
func IsValidReason(reason string) bool {
	length := utf8.RuneCountInString(reason)
	return length >= MinReasonLength && length <= MaxReasonLength
}

// RequireReasonFromPAT enforces that PAT-authenticated callers supply a meaningful
// reason for destructive actions (header X-Action-Reason or query reason). It is
// a no-op for session/user-authenticated callers, who are already accountable via
// their session and audit identity.
func RequireReasonFromPAT(c *fiber.Ctx) error {
	if GetTokenFromContext(c) == nil {
		return nil
	}
	reason := GetActionReason(c)
	if reason == "" {
		return response.BadRequest(c, "X-Action-Reason header (or reason query) is required for destructive actions when using a personal access token")
	}
	if !IsValidReason(reason) {
		return response.BadRequest(c, "reason must be between 8 and 500 characters")
	}
	return nil
}

// EnforceDestructive bundles the two safety checks every destructive handler
// must run: dry-run preview short-circuit AND PAT reason validation. Returning
// (true, nil) means a dry-run response has already been written and the
// handler must abort. Returning (false, err) means the request must be aborted
// with err. Returning (false, nil) means the handler can proceed with the
// actual mutation.
//
// Usage:
//
//	if abort, err := middleware.EnforceDestructive(c, report); abort || err != nil {
//	    return err
//	}
func EnforceDestructive(c *fiber.Ctx, report response.DryRunReport) (bool, error) {
	if IsDryRun(c) {
		return true, response.DryRun(c, report)
	}
	if err := RequireReasonFromPAT(c); err != nil {
		return false, err
	}
	return false, nil
}

func isFalsy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "0", "false", "no", "off":
		return true
	default:
		return false
	}
}
