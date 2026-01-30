package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Deploy   DeployConfig
	Docker   DockerConfig
	GitHub   GitHubConfig
}

type ServerConfig struct {
	Env      string
	Host     string
	Port     int
	LogLevel string
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
	PAT           string
	WebhookSecret string
	WebhookURL    string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Env:      getEnv("APP_ENV", "development"),
			Host:     getEnv("HOST", "0.0.0.0"),
			Port:     getEnvInt("PORT", 8080),
			LogLevel: getEnv("LOG_LEVEL", "info"),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", ""),
		},
		Deploy: DeployConfig{
			DataDir:            getEnvPath("DEPLOY_DATA_DIR", defaultDataDir()),
			Workers:            getEnvInt("DEPLOY_WORKERS", 2),
			Timeout:            time.Duration(getEnvInt("DEPLOY_TIMEOUT", 600)) * time.Second,
			HealthCheckTimeout: time.Duration(getEnvInt("HEALTH_CHECK_TIMEOUT", 60)) * time.Second,
			HealthCheckRetries: getEnvInt("HEALTH_CHECK_RETRIES", 3),
		},
		Docker: DockerConfig{
			Host:     getEnv("DOCKER_HOST", "unix:///var/run/docker.sock"),
			Registry: getEnv("DOCKER_REGISTRY", ""),
		},
		GitHub: GitHubConfig{
			PAT:           getEnv("GITHUB_PAT", ""),
			WebhookSecret: getEnv("GITHUB_WEBHOOK_SECRET", ""),
			WebhookURL:    getEnv("GITHUB_WEBHOOK_URL", ""),
		},
	}
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
