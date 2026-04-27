package config

import (
	"strings"
	"testing"
	"time"
)

func TestResolveSucceedsWithFlags(t *testing.T) {
	cfg, err := Resolve(Flags{
		BackendURL:     "https://api.flowdeploy.test",
		Token:          "pdp_live_abcdef1234567890",
		LogLevel:       "debug",
		ClientID:       "cursor",
		RequestTimeout: 5 * time.Second,
		Stdio:          true,
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if cfg.BackendURL != "https://api.flowdeploy.test" {
		t.Errorf("unexpected backend url: %s", cfg.BackendURL)
	}
	if cfg.ClientID != "cursor" {
		t.Errorf("unexpected client id: %s", cfg.ClientID)
	}
	if cfg.RedactedToken() == cfg.Token {
		t.Errorf("redacted token must not equal raw token")
	}
}

func TestResolveRequiresBackendURL(t *testing.T) {
	t.Setenv(envBackendURL, "")
	t.Setenv(envToken, "")
	_, err := Resolve(Flags{
		Token:          "pdp_live_abcdef1234567890",
		LogLevel:       "info",
		RequestTimeout: time.Second,
		Stdio:          true,
	})
	if err == nil || !strings.Contains(err.Error(), "backend-url") {
		t.Fatalf("expected backend-url error, got %v", err)
	}
}

func TestResolveRequiresTokenPrefix(t *testing.T) {
	t.Setenv(envBackendURL, "")
	t.Setenv(envToken, "")
	_, err := Resolve(Flags{
		BackendURL:     "https://api.flowdeploy.test",
		Token:          "wrong_prefix_value",
		LogLevel:       "info",
		RequestTimeout: time.Second,
		Stdio:          true,
	})
	if err == nil || !strings.Contains(err.Error(), "pdp_live_") {
		t.Fatalf("expected token prefix error, got %v", err)
	}
}

func TestResolveRequiresHTTPScheme(t *testing.T) {
	t.Setenv(envBackendURL, "")
	t.Setenv(envToken, "")
	_, err := Resolve(Flags{
		BackendURL:     "api.flowdeploy.test",
		Token:          "pdp_live_abcdef1234567890",
		LogLevel:       "info",
		RequestTimeout: time.Second,
		Stdio:          true,
	})
	if err == nil || !strings.Contains(err.Error(), "scheme") {
		t.Fatalf("expected scheme error, got %v", err)
	}
}

func TestResolveReadsEnvFallbacks(t *testing.T) {
	t.Setenv(envBackendURL, "https://env.flowdeploy.test")
	t.Setenv(envToken, "pdp_live_envvalue1234567890")
	cfg, err := Resolve(Flags{
		LogLevel:       "info",
		RequestTimeout: time.Second,
		Stdio:          true,
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if cfg.BackendURL != "https://env.flowdeploy.test" {
		t.Errorf("expected env backend URL, got %s", cfg.BackendURL)
	}
}

func TestRedactedTokenHidesSensitivePart(t *testing.T) {
	cfg := Config{Token: "pdp_live_abcdef1234567890"}
	red := cfg.RedactedToken()
	if !strings.HasPrefix(red, "pdp_live_") {
		t.Errorf("expected prefix preserved, got %s", red)
	}
	if strings.Contains(red, "1234567890") {
		t.Errorf("redacted token leaks suffix: %s", red)
	}
}
