package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Deploy   DeployConfig
	Docker   DockerConfig
}

type ServerConfig struct {
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

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:     getEnv("HOST", "0.0.0.0"),
			Port:     getEnvInt("PORT", 8080),
			LogLevel: getEnv("LOG_LEVEL", "info"),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", "postgres://paas_deploy:paas_deploy@localhost:5432/paas_deploy?sslmode=disable"),
		},
		Deploy: DeployConfig{
			DataDir:            getEnv("DEPLOY_DATA_DIR", "/data/apps"),
			Workers:            getEnvInt("DEPLOY_WORKERS", 2),
			Timeout:            time.Duration(getEnvInt("DEPLOY_TIMEOUT", 600)) * time.Second,
			HealthCheckTimeout: time.Duration(getEnvInt("HEALTH_CHECK_TIMEOUT", 60)) * time.Second,
			HealthCheckRetries: getEnvInt("HEALTH_CHECK_RETRIES", 3),
		},
		Docker: DockerConfig{
			Host:     getEnv("DOCKER_HOST", "unix:///var/run/docker.sock"),
			Registry: getEnv("DOCKER_REGISTRY", ""),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
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
