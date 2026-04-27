package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

type dryRunSnapshot struct {
	dryRun bool
	reason string
}

func captureDryRun(t *testing.T, method, target string, headers map[string]string) dryRunSnapshot {
	t.Helper()
	app := fiber.New()
	var snap dryRunSnapshot
	app.Add(method, "/probe", func(c *fiber.Ctx) error {
		snap.dryRun = IsDryRun(c)
		snap.reason = GetActionReason(c)
		return c.SendStatus(http.StatusOK)
	})

	req := httptest.NewRequest(method, target, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	return snap
}

func TestIsDryRunHeader(t *testing.T) {
	cases := map[string]bool{
		"true":  true,
		"True":  true,
		"YES":   true,
		"1":     true,
		"on":    true,
		"":      false,
		"false": false,
		"0":     false,
	}
	for value, want := range cases {
		snap := captureDryRun(t, http.MethodPost, "/probe", map[string]string{HeaderDryRun: value})
		if snap.dryRun != want {
			t.Errorf("IsDryRun(header=%q)=%v want %v", value, snap.dryRun, want)
		}
	}
}

func TestIsDryRunQuery(t *testing.T) {
	snap := captureDryRun(t, http.MethodPost, "/probe?dry_run=true", nil)
	if !snap.dryRun {
		t.Errorf("expected dry_run=true via query")
	}

	snap2 := captureDryRun(t, http.MethodPost, "/probe?dry_run=false", nil)
	if snap2.dryRun {
		t.Errorf("dry_run=false must not enable dry-run")
	}
}

func TestGetActionReasonHeaderTakesPrecedence(t *testing.T) {
	snap := captureDryRun(t, http.MethodPost, "/probe?reason=ignore-me",
		map[string]string{HeaderActionReason: "rotate creds"})
	if snap.reason != "rotate creds" {
		t.Errorf("expected header value, got %q", snap.reason)
	}
}

func TestGetActionReasonFallsBackToQuery(t *testing.T) {
	snap := captureDryRun(t, http.MethodPost, "/probe?reason=manual%20cleanup", nil)
	if snap.reason != "manual cleanup" {
		t.Errorf("expected query reason, got %q", snap.reason)
	}
}

func TestIsValidReasonBoundaries(t *testing.T) {
	if IsValidReason("") {
		t.Errorf("empty reason should not be valid")
	}
	if IsValidReason("short") {
		t.Errorf("reason shorter than 8 chars must be invalid")
	}
	if !IsValidReason("rotating credentials") {
		t.Errorf("normal reason must be valid")
	}
	if IsValidReason(strings.Repeat("a", 501)) {
		t.Errorf("reason longer than 500 chars must be invalid")
	}
}

func TestIsValidReasonRuneCount(t *testing.T) {
	if IsValidReason("açãoção") {
		t.Errorf("7-rune reason must be rejected even though byte length >= 8")
	}
	if !IsValidReason("açãoçãot") {
		t.Errorf("8-rune reason must be accepted")
	}
}

func TestIsDryRunFailSafeOnGarbage(t *testing.T) {
	cases := []string{"yep", "maybe", "y", "definitely"}
	for _, value := range cases {
		snap := captureDryRun(t, http.MethodPost, "/probe", map[string]string{HeaderDryRun: value})
		if !snap.dryRun {
			t.Errorf("garbage X-Dry-Run=%q must default to dry-run (fail-safe)", value)
		}
	}
}

func TestIsDryRunHeaderAndQueryCombined(t *testing.T) {
	snap := captureDryRun(t, http.MethodPost, "/probe?dry_run=true",
		map[string]string{HeaderDryRun: "false"})
	if snap.dryRun {
		t.Errorf("explicit header=false must override truthy query (header has precedence)")
	}

	snap2 := captureDryRun(t, http.MethodPost, "/probe?dry_run=false",
		map[string]string{HeaderDryRun: "true"})
	if !snap2.dryRun {
		t.Errorf("explicit header=true must take precedence over query")
	}

	snap3 := captureDryRun(t, http.MethodPost, "/probe?dry_run=true", nil)
	if !snap3.dryRun {
		t.Errorf("query=true alone must enable dry-run")
	}
}
