package compose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultAppPort = 8080

type Config struct {
	Name    string `json:"name"`
	Runtime string `json:"runtime,omitempty"`
	Build   struct {
		Type       string            `json:"type"`
		Dockerfile string            `json:"dockerfile"`
		Context    string            `json:"context"`
		Args       map[string]string `json:"args,omitempty"`
		Target     string            `json:"target,omitempty"`
	} `json:"build"`
	Healthcheck struct {
		Path        string `json:"path"`
		Interval    string `json:"interval"`
		Timeout     string `json:"timeout"`
		Retries     int    `json:"retries"`
		StartPeriod string `json:"startPeriod"`
	} `json:"healthcheck"`
	Port      int               `json:"port"`
	HostPort  int               `json:"hostPort,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Resources struct {
		Memory string `json:"memory"`
		CPU    string `json:"cpu"`
	} `json:"resources"`
	Domains []string `json:"domains,omitempty"`
}

type DomainRoute struct {
	Domain     string
	PathPrefix string
}

func LoadConfig(appDir string) (*Config, error) {
	configPath := filepath.Join(appDir, "paasdeploy.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("paasdeploy.json not found in %s - this file is required for deployment", appDir)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read paasdeploy.json: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid paasdeploy.json: %w", err)
	}

	if config.Name == "" {
		return nil, fmt.Errorf("paasdeploy.json: 'name' field is required")
	}

	ApplyDefaults(&config)

	return &config, nil
}

func ValidateDockerfile(appDir string, config *Config) error {
	dockerfilePath := filepath.Join(appDir, config.Build.Dockerfile)
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("Dockerfile not found at %s - this file is required for deployment", config.Build.Dockerfile)
	}
	return nil
}

func ApplyDefaults(config *Config) {
	if config.Port == 0 {
		config.Port = DefaultAppPort
	}
	if config.Build.Type == "" {
		config.Build.Type = "dockerfile"
	}
	if config.Build.Dockerfile == "" {
		config.Build.Dockerfile = "./Dockerfile"
	}
	if config.Build.Context == "" {
		config.Build.Context = "."
	}
	if config.Healthcheck.Path == "" {
		config.Healthcheck.Path = "/health"
	}
	if config.Healthcheck.Interval == "" {
		config.Healthcheck.Interval = "30s"
	}
	if config.Healthcheck.Timeout == "" {
		config.Healthcheck.Timeout = "5s"
	}
	if config.Healthcheck.Retries == 0 {
		config.Healthcheck.Retries = 3
	}
	if config.Healthcheck.StartPeriod == "" {
		config.Healthcheck.StartPeriod = "10s"
	}
	if config.Resources.Memory == "" {
		config.Resources.Memory = "512m"
	}
	if config.Resources.CPU == "" {
		config.Resources.CPU = "0.5"
	}
}
