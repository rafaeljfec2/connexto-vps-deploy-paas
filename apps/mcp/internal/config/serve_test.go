package config

import (
	"strings"
	"testing"
	"time"
)

func TestResolveServeAcceptsEmptyTokenForMultiTenant(t *testing.T) {
	t.Setenv(envBackendURL, "")
	t.Setenv(envToken, "")
	cfg, err := ResolveServe(Flags{
		BackendURL:     "https://api.flowdeploy.test",
		LogLevel:       "info",
		RequestTimeout: time.Second,
		Addr:           ":3001",
		ReadRPM:        defaultReadRPM,
		MutateRPM:      defaultMutateRPM,
		SessionMaxAge:  10 * time.Minute,
		AllowedClients: "cursor,custom:*",
	})
	if err != nil {
		t.Fatalf("ResolveServe rejected empty token: %v", err)
	}
	if cfg.Mode != ModeServe {
		t.Fatalf("expected ModeServe, got %s", cfg.Mode)
	}
	if cfg.Token != "" {
		t.Fatalf("expected empty token, got %q", cfg.Token)
	}
}

func TestResolveServeRejectsBadAllowedClients(t *testing.T) {
	t.Setenv(envBackendURL, "")
	t.Setenv(envToken, "")
	_, err := ResolveServe(Flags{
		BackendURL:     "https://api.flowdeploy.test",
		LogLevel:       "info",
		RequestTimeout: time.Second,
		Addr:           ":3001",
		ReadRPM:        10,
		MutateRPM:      5,
		SessionMaxAge:  time.Minute,
		AllowedClients: "  ,  ,  ",
	})
	if err == nil || !strings.Contains(err.Error(), "allowed-clients") {
		t.Fatalf("expected allowed-clients error, got %v", err)
	}
}

func TestResolveServeRejectsMalformedToken(t *testing.T) {
	t.Setenv(envBackendURL, "")
	t.Setenv(envToken, "")
	_, err := ResolveServe(Flags{
		BackendURL:     "https://api.flowdeploy.test",
		Token:          "wrong_prefix_value",
		LogLevel:       "info",
		RequestTimeout: time.Second,
		Addr:           ":3001",
		ReadRPM:        10,
		MutateRPM:      5,
		SessionMaxAge:  time.Minute,
		AllowedClients: "cursor",
	})
	if err == nil || !strings.Contains(err.Error(), "pdp_live_") {
		t.Fatalf("expected pdp_live_ error, got %v", err)
	}
}

func TestResolveServeFillsDefaultsForZeroQuotas(t *testing.T) {
	t.Setenv(envBackendURL, "")
	t.Setenv(envToken, "")
	cfg, err := ResolveServe(Flags{
		BackendURL:     "https://api.flowdeploy.test",
		LogLevel:       "info",
		RequestTimeout: time.Second,
		Addr:           ":3001",
		ReadRPM:        0,
		MutateRPM:      0,
		SessionMaxAge:  time.Minute,
		AllowedClients: "cursor",
	})
	if err != nil {
		t.Fatalf("ResolveServe: %v", err)
	}
	if cfg.ReadRPM != defaultReadRPM || cfg.MutateRPM != defaultMutateRPM {
		t.Fatalf("expected defaults to be filled in for zero quotas; got read=%d mutate=%d", cfg.ReadRPM, cfg.MutateRPM)
	}
}

func TestResolveServeFillsDefaults(t *testing.T) {
	t.Setenv(envBackendURL, "")
	t.Setenv(envToken, "")
	cfg, err := ResolveServe(Flags{
		BackendURL:     "https://api.flowdeploy.test",
		LogLevel:       "info",
		RequestTimeout: time.Second,
		AllowedClients: "cursor",
	})
	if err != nil {
		t.Fatalf("ResolveServe: %v", err)
	}
	if cfg.Addr != defaultServeAddr {
		t.Fatalf("expected default addr %s, got %s", defaultServeAddr, cfg.Addr)
	}
	if cfg.ReadRPM != defaultReadRPM {
		t.Fatalf("expected default read rpm %d, got %d", defaultReadRPM, cfg.ReadRPM)
	}
	if cfg.MutateRPM != defaultMutateRPM {
		t.Fatalf("expected default mutate rpm %d, got %d", defaultMutateRPM, cfg.MutateRPM)
	}
}
