package config

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

const (
	envBackendURL = "FLOWDEPLOY_BACKEND_URL"
	envToken      = "FLOWDEPLOY_TOKEN"
	envLogLevel   = "FLOWDEPLOY_LOG_LEVEL"
	envClient     = "FLOWDEPLOY_MCP_CLIENT"
	tokenPrefix   = "pdp_live_"
)

type Config struct {
	BackendURL     string
	Token          string
	LogLevel       slog.Level
	ClientID       string
	RequestTimeout time.Duration
	Stdio          bool
}

type Flags struct {
	Stdio          bool
	BackendURL     string
	Token          string
	LogLevel       string
	ClientID       string
	RequestTimeout time.Duration
}

func (f *Flags) Register(fs *flag.FlagSet) {
	fs.BoolVar(&f.Stdio, "stdio", true, "use stdio transport (default; required for Phase 1)")
	fs.StringVar(&f.BackendURL, "backend-url", "", "FlowDeploy backend base URL (e.g. https://api.flowdeploy.example.com)")
	fs.StringVar(&f.Token, "token", "", "Personal access token with prefix pdp_live_; can also be set via FLOWDEPLOY_TOKEN")
	fs.StringVar(&f.LogLevel, "log-level", "info", "log level: debug, info, warn, error")
	fs.StringVar(&f.ClientID, "client", "custom:flowdeploy-mcp", "X-MCP-Client header value")
	fs.DurationVar(&f.RequestTimeout, "request-timeout", 15*time.Second, "per-request HTTP timeout against the backend")
}

func Resolve(f Flags) (Config, error) {
	cfg := Config{
		BackendURL:     firstNonEmpty(f.BackendURL, os.Getenv(envBackendURL)),
		Token:          firstNonEmpty(f.Token, os.Getenv(envToken)),
		ClientID:       firstNonEmpty(f.ClientID, os.Getenv(envClient), "custom:flowdeploy-mcp"),
		RequestTimeout: f.RequestTimeout,
		Stdio:          f.Stdio,
	}

	logLevel := firstNonEmpty(f.LogLevel, os.Getenv(envLogLevel), "info")
	level, err := parseLogLevel(logLevel)
	if err != nil {
		return Config{}, err
	}
	cfg.LogLevel = level

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.BackendURL == "" {
		return errors.New("backend-url is required (flag --backend-url or env FLOWDEPLOY_BACKEND_URL)")
	}
	if !strings.HasPrefix(c.BackendURL, "http://") && !strings.HasPrefix(c.BackendURL, "https://") {
		return fmt.Errorf("backend-url must include http:// or https:// scheme: got %q", c.BackendURL)
	}
	if c.Token == "" {
		return errors.New("token is required (flag --token or env FLOWDEPLOY_TOKEN)")
	}
	if !strings.HasPrefix(c.Token, tokenPrefix) {
		return fmt.Errorf("token must start with %s", tokenPrefix)
	}
	if c.RequestTimeout <= 0 {
		return errors.New("request-timeout must be positive")
	}
	return nil
}

func parseLogLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log level %q (expected debug|info|warn|error)", s)
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func (c Config) RedactedToken() string {
	if len(c.Token) <= len(tokenPrefix)+4 {
		return tokenPrefix + "****"
	}
	return c.Token[:len(tokenPrefix)+4] + "****"
}
