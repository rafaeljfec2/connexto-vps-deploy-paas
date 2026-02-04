package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultPort              = 8080
	DefaultDeployWorkers     = 2
	DefaultDeployTimeoutSec  = 600
	DefaultHealthTimeoutSec  = 180
	DefaultHealthRetries     = 5
	DefaultSessionMaxAgeSec  = 604800
	DefaultDockerHost        = "unix:///var/run/docker.sock"
	DefaultFrontendURL       = "http://localhost:3000"
	DefaultSessionCookieName = "flowdeploy_session"
	DefaultAppName           = "FlowDeploy"
)

type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Deploy     DeployConfig
	Docker     DockerConfig
	GitHub     GitHubConfig
	Auth       AuthConfig
	Cloudflare CloudflareConfig
	Traefik    TraefikConfig
	GRPC       GRPCConfig
}

type GRPCConfig struct {
	Enabled         bool
	Port            int
	ServerAddr      string
	AgentBinaryPath string
	AgentPort       int
}

type ServerConfig struct {
	Env         string
	Host        string
	Port        int
	LogLevel    string
	CorsOrigins string
	ApiBaseURL  string
}

type DatabaseConfig struct {
	URL string
}

type DeployConfig struct {
	DataDir            string
	Workers            int
	Timeout            time.Duration
	HealthCheckTimeout time.Duration
	HealthCheckRetries int
}

type DockerConfig struct {
	Host     string
	Registry string
}

type GitHubConfig struct {
	// PAT for Phase 1 (personal access token)
	PAT           string
	WebhookSecret string
	WebhookURL    string

	// OAuth (for user authentication)
	ClientID     string
	ClientSecret string
	CallbackURL  string

	// GitHub App (for repository access)
	AppID         int64
	AppName       string
	AppPrivateKey []byte
	AppInstallURL string
	AppSetupURL   string
}

type AuthConfig struct {
	TokenEncryptionKey string
	SessionCookieName  string
	SessionMaxAge      time.Duration
	SecureCookie       bool
	CookieDomain       string
	FrontendURL        string
}

type CloudflareConfig struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
	ServerIP     string
}

type TraefikConfig struct {
	URL string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Env:         getEnv("APP_ENV", "development"),
			Host:        getEnv("HOST", "0.0.0.0"),
			Port:        getEnvInt("PORT", DefaultPort),
			LogLevel:    getEnv("LOG_LEVEL", "info"),
			CorsOrigins: getEnv("CORS_ORIGINS", "*"),
			ApiBaseURL:  getEnv("API_BASE_URL", ""),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", ""),
		},
		Deploy: DeployConfig{
			DataDir:            getEnvPath("DEPLOY_DATA_DIR", defaultDataDir()),
			Workers:            getEnvInt("DEPLOY_WORKERS", DefaultDeployWorkers),
			Timeout:            time.Duration(getEnvInt("DEPLOY_TIMEOUT", DefaultDeployTimeoutSec)) * time.Second,
			HealthCheckTimeout: time.Duration(getEnvInt("HEALTH_CHECK_TIMEOUT", DefaultHealthTimeoutSec)) * time.Second,
			HealthCheckRetries: getEnvInt("HEALTH_CHECK_RETRIES", DefaultHealthRetries),
		},
		Docker: DockerConfig{
			Host:     getEnv("DOCKER_HOST", DefaultDockerHost),
			Registry: getEnv("DOCKER_REGISTRY", ""),
		},
		GitHub: GitHubConfig{
			PAT:           getEnv("GIT_HUB_PAT", ""),
			WebhookSecret: getEnv("GIT_HUB_WEBHOOK_SECRET", ""),
			WebhookURL:    getEnv("GIT_HUB_WEBHOOK_URL", ""),

			ClientID:     getEnv("GIT_HUB_CLIENT_ID", ""),
			ClientSecret: getEnv("GIT_HUB_CLIENT_SECRET", ""),
			CallbackURL:  getEnv("GIT_HUB_OAUTH_CALLBACK_URL", ""),

			AppID:         int64(getEnvInt("GIT_HUB_APP_ID", 0)),
			AppName:       getEnv("GIT_HUB_APP_NAME", DefaultAppName),
			AppPrivateKey: loadPrivateKey(),
			AppInstallURL: getEnv("GIT_HUB_APP_INSTALL_URL", ""),
			AppSetupURL:   getEnv("GIT_HUB_APP_SETUP_URL", ""),
		},
		Auth: AuthConfig{
			TokenEncryptionKey: getEnv("TOKEN_ENCRYPTION_KEY", ""),
			SessionCookieName:  getEnv("SESSION_COOKIE_NAME", DefaultSessionCookieName),
			SessionMaxAge:      time.Duration(getEnvInt("SESSION_MAX_AGE", DefaultSessionMaxAgeSec)) * time.Second,
			SecureCookie:       getEnv("SESSION_SECURE", "false") == "true",
			CookieDomain:       getEnv("COOKIE_DOMAIN", ""),
			FrontendURL:        getEnv("FRONTEND_URL", DefaultFrontendURL),
		},
		Cloudflare: CloudflareConfig{
			ClientID:     getEnv("CLOUDFLARE_CLIENT_ID", ""),
			ClientSecret: getEnv("CLOUDFLARE_CLIENT_SECRET", ""),
			CallbackURL:  getEnv("CLOUDFLARE_CALLBACK_URL", ""),
			ServerIP:     getEnv("CLOUDFLARE_SERVER_IP", ""),
		},
		Traefik: TraefikConfig{
			URL: getEnv("TRAEFIK_API_URL", "http://paasdeploy-traefik:8081"),
		},
		GRPC: GRPCConfig{
			Enabled:         getEnv("GRPC_ENABLED", "false") == "true",
			Port:            getEnvInt("GRPC_PORT", 50051),
			ServerAddr:      getEnv("GRPC_SERVER_ADDR", ""),
			AgentBinaryPath: getEnv("AGENT_BINARY_PATH", ""),
			AgentPort:       getEnvInt("AGENT_GRPC_PORT", 50052),
		},
	}
}

func loadPrivateKey() []byte {
	// Try loading from base64 first
	if keyBase64 := getEnv("GIT_HUB_APP_PRIVATE_KEY_BASE64", ""); keyBase64 != "" {
		// Don't decode here, let the app_client handle it
		return []byte(keyBase64)
	}

	// Try loading from file
	if keyPath := getEnv("GIT_HUB_APP_PRIVATE_KEY_PATH", ""); keyPath != "" {
		expandedPath := expandPath(keyPath)
		data, err := os.ReadFile(expandedPath)
		if err == nil {
			return data
		}
	}

	return nil
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/paas-deploy/apps"
	}
	return filepath.Join(home, ".paas-deploy", "apps")
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}

	if strings.HasPrefix(path, "$HOME") {
		home, err := os.UserHomeDir()
		if err == nil {
			return strings.Replace(path, "$HOME", home, 1)
		}
	}

	return os.ExpandEnv(path)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvPath(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return expandPath(value)
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func (c *Config) EnsureDirectories() error {
	dirs := []string{
		c.Deploy.DataDir,
		filepath.Join(c.Deploy.DataDir, ".locks"),
	}

	for _, dir := range dirs {
		if err := ensureWritableDir(dir); err != nil {
			return fmt.Errorf("failed to ensure directory %s: %w", dir, err)
		}
	}

	return nil
}

func ensureWritableDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	testFile := filepath.Join(dir, ".write_test")
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("directory is not writable: %w", err)
	}
	f.Close()
	os.Remove(testFile)

	return nil
}
