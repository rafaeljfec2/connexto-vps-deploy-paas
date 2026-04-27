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
	envServeAddr  = "FLOWDEPLOY_MCP_ADDR"

	tokenPrefix = "pdp_live_"

	defaultServeAddr     = ":3001"
	defaultReadRPM       = 120
	defaultMutateRPM     = 20
	defaultSessionMaxAge = 30 * time.Minute
)

type Mode string

const (
	ModeStdio Mode = "stdio"
	ModeServe Mode = "serve"
)

type Config struct {
	Mode           Mode
	BackendURL     string
	Token          string
	LogLevel       slog.Level
	ClientID       string
	RequestTimeout time.Duration
	Stdio          bool

	Addr             string
	ReadRPM          int
	MutateRPM        int
	SessionMaxAge    time.Duration
	AllowedClients   []string
	StatelessSession bool
}

type Flags struct {
	Stdio          bool
	BackendURL     string
	Token          string
	LogLevel       string
	ClientID       string
	RequestTimeout time.Duration

	Addr             string
	ReadRPM          int
	MutateRPM        int
	SessionMaxAge    time.Duration
	AllowedClients   string
	StatelessSession bool
}

func (f *Flags) RegisterCommon(fs *flag.FlagSet) {
	fs.StringVar(&f.BackendURL, "backend-url", "", "FlowDeploy backend base URL (e.g. https://api.flowdeploy.example.com)")
	fs.StringVar(&f.Token, "token", "", "Personal access token with prefix pdp_live_; can also be set via FLOWDEPLOY_TOKEN. Required in stdio mode; optional in serve mode (forwarded from client).")
	fs.StringVar(&f.LogLevel, "log-level", "info", "log level: debug, info, warn, error")
	fs.StringVar(&f.ClientID, "client", "custom:flowdeploy-mcp", "X-MCP-Client header value (stdio mode); in serve mode the agent's reported client wins")
	fs.DurationVar(&f.RequestTimeout, "request-timeout", 15*time.Second, "per-request HTTP timeout against the backend")
}

// RegisterStdio installs flags exclusive to the stdio subcommand.
func (f *Flags) RegisterStdio(fs *flag.FlagSet) {
	f.RegisterCommon(fs)
	fs.BoolVar(&f.Stdio, "stdio", true, "use stdio transport (default)")
}

// RegisterServe installs flags exclusive to the serve subcommand.
func (f *Flags) RegisterServe(fs *flag.FlagSet) {
	f.RegisterCommon(fs)
	fs.StringVar(&f.Addr, "addr", defaultServeAddr, "TCP address for the HTTP transport (TLS termination is delegated to Traefik)")
	fs.IntVar(&f.ReadRPM, "read-rpm", defaultReadRPM, "per-PAT read tool quota (requests per minute)")
	fs.IntVar(&f.MutateRPM, "mutate-rpm", defaultMutateRPM, "per-PAT mutation tool quota (requests per minute)")
	fs.DurationVar(&f.SessionMaxAge, "session-max-age", defaultSessionMaxAge, "idle timeout for streamable MCP sessions")
	fs.StringVar(&f.AllowedClients, "allowed-clients", "cursor,claude-desktop,custom:*,ci:*", "comma-separated allowlist of acceptable X-MCP-Client values; supports trailing wildcard (prefix:*)")
	fs.BoolVar(&f.StatelessSession, "stateless-session", false, "run the streamable transport in stateless mode (each request is a fresh session)")
}

func ResolveStdio(f Flags) (Config, error) {
	cfg, err := resolveCommon(f)
	if err != nil {
		return Config{}, err
	}
	cfg.Mode = ModeStdio
	cfg.Stdio = true
	if err := cfg.validateStdio(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func ResolveServe(f Flags) (Config, error) {
	cfg, err := resolveCommon(f)
	if err != nil {
		return Config{}, err
	}
	cfg.Mode = ModeServe
	cfg.Addr = firstNonEmpty(f.Addr, os.Getenv(envServeAddr), defaultServeAddr)
	cfg.ReadRPM = positiveOrDefault(f.ReadRPM, defaultReadRPM)
	cfg.MutateRPM = positiveOrDefault(f.MutateRPM, defaultMutateRPM)
	if f.SessionMaxAge > 0 {
		cfg.SessionMaxAge = f.SessionMaxAge
	} else {
		cfg.SessionMaxAge = defaultSessionMaxAge
	}
	cfg.StatelessSession = f.StatelessSession
	cfg.AllowedClients = parseClients(f.AllowedClients)
	if err := cfg.validateServe(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Resolve preserves the original entrypoint used in tests; defaults to stdio.
func Resolve(f Flags) (Config, error) {
	return ResolveStdio(f)
}

func resolveCommon(f Flags) (Config, error) {
	cfg := Config{
		BackendURL:     firstNonEmpty(f.BackendURL, os.Getenv(envBackendURL)),
		Token:          firstNonEmpty(f.Token, os.Getenv(envToken)),
		ClientID:       firstNonEmpty(f.ClientID, os.Getenv(envClient), "custom:flowdeploy-mcp"),
		RequestTimeout: f.RequestTimeout,
	}

	logLevel := firstNonEmpty(f.LogLevel, os.Getenv(envLogLevel), "info")
	level, err := parseLogLevel(logLevel)
	if err != nil {
		return Config{}, err
	}
	cfg.LogLevel = level
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 15 * time.Second
	}
	return cfg, nil
}

func (c Config) validateCommon() error {
	if c.BackendURL == "" {
		return errors.New("backend-url is required (flag --backend-url or env FLOWDEPLOY_BACKEND_URL)")
	}
	if !strings.HasPrefix(c.BackendURL, "http://") && !strings.HasPrefix(c.BackendURL, "https://") {
		return fmt.Errorf("backend-url must include http:// or https:// scheme: got %q", c.BackendURL)
	}
	if c.RequestTimeout <= 0 {
		return errors.New("request-timeout must be positive")
	}
	return nil
}

func (c Config) validateStdio() error {
	if err := c.validateCommon(); err != nil {
		return err
	}
	if c.Token == "" {
		return errors.New("token is required in stdio mode (flag --token or env FLOWDEPLOY_TOKEN)")
	}
	if !strings.HasPrefix(c.Token, tokenPrefix) {
		return fmt.Errorf("token must start with %s", tokenPrefix)
	}
	return nil
}

func (c Config) validateServe() error {
	if err := c.validateCommon(); err != nil {
		return err
	}
	if c.Addr == "" {
		return errors.New("addr is required in serve mode")
	}
	if c.Token != "" && !strings.HasPrefix(c.Token, tokenPrefix) {
		return fmt.Errorf("token must start with %s", tokenPrefix)
	}
	if c.ReadRPM <= 0 {
		return errors.New("read-rpm must be positive")
	}
	if c.MutateRPM <= 0 {
		return errors.New("mutate-rpm must be positive")
	}
	if len(c.AllowedClients) == 0 {
		return errors.New("allowed-clients must contain at least one entry")
	}
	if c.SessionMaxAge <= 0 {
		return errors.New("session-max-age must be positive")
	}
	return nil
}

// Validate keeps backward compatibility with existing call sites and tests.
func (c Config) Validate() error {
	if c.Mode == ModeServe {
		return c.validateServe()
	}
	return c.validateStdio()
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

func positiveOrDefault(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

func parseClients(raw string) []string {
	out := []string{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (c Config) RedactedToken() string {
	if c.Token == "" {
		return "(none)"
	}
	if len(c.Token) <= len(tokenPrefix)+4 {
		return tokenPrefix + "****"
	}
	return c.Token[:len(tokenPrefix)+4] + "****"
}
