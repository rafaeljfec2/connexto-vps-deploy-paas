package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ContainerInfo struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Image    string   `json:"image"`
	Status   string   `json:"status"`
	State    string   `json:"state"`
	Ports    []string `json:"ports"`
	Created  string   `json:"created"`
	Uptime   string   `json:"uptime"`
	Networks []string `json:"networks,omitempty"`
}

type ProxyStatus struct {
	Type     string `json:"type"`
	Running  bool   `json:"running"`
	Version  string `json:"version,omitempty"`
	PID      int    `json:"pid,omitempty"`
}

type MigrationStatus struct {
	Proxy            ProxyStatus       `json:"proxy"`
	NginxSites       []NginxSite       `json:"nginxSites"`
	SSLCertificates  []SSLCertificate  `json:"sslCertificates"`
	Containers       []ContainerInfo   `json:"containers"`
	TraefikReady     bool              `json:"traefikReady"`
	MigrationNeeded  bool              `json:"migrationNeeded"`
	Warnings         []string          `json:"warnings"`
	LastBackupPath   string            `json:"lastBackupPath,omitempty"`
	LastBackupTime   *time.Time        `json:"lastBackupTime,omitempty"`
}

type BackupResult struct {
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"createdAt"`
	Files     []string  `json:"files"`
	Size      int64     `json:"size"`
}

type MigrationService struct {
	nginxParser      *NginxParser
	sslDetector      *SSLDetector
	traefikConverter *TraefikConverter
	logger           *slog.Logger
	backupBasePath   string
}

func NewMigrationService(logger *slog.Logger) *MigrationService {
	return &MigrationService{
		nginxParser:      NewNginxParser(),
		sslDetector:      NewSSLDetector(),
		traefikConverter: NewTraefikConverter(),
		logger:           logger.With("service", "migration"),
		backupBasePath:   "/var/backups/flowdeploy",
	}
}

func (s *MigrationService) GetStatus(ctx context.Context) (*MigrationStatus, error) {
	status := &MigrationStatus{
		Warnings: []string{},
	}

	status.Proxy = s.detectProxy()

	sites, err := s.nginxParser.ParseAllSites()
	if err != nil {
		s.logger.Warn("Failed to parse nginx sites", "error", err)
	} else {
		status.NginxSites = sites
	}

	certs, err := s.sslDetector.DetectAllCertificates()
	if err != nil {
		s.logger.Warn("Failed to detect SSL certificates", "error", err)
	} else {
		status.SSLCertificates = certs
	}

	containers, err := s.listDockerContainers(ctx)
	if err != nil {
		s.logger.Warn("Failed to list containers", "error", err)
	} else {
		status.Containers = containers
	}

	status.TraefikReady = s.isTraefikRunning(ctx)
	status.MigrationNeeded = status.Proxy.Type == "nginx" && status.Proxy.Running && !status.TraefikReady

	status.Warnings = s.generateWarnings(status)

	return status, nil
}

func (s *MigrationService) detectProxy() ProxyStatus {
	if s.isProcessRunning("nginx") {
		version := s.getProcessVersion("nginx", "-v")
		return ProxyStatus{
			Type:    "nginx",
			Running: true,
			Version: version,
		}
	}

	if s.isProcessRunning("apache2") || s.isProcessRunning("httpd") {
		return ProxyStatus{
			Type:    "apache",
			Running: true,
		}
	}

	return ProxyStatus{
		Type:    "none",
		Running: false,
	}
}

func (s *MigrationService) isProcessRunning(name string) bool {
	cmd := exec.Command("pgrep", "-x", name)
	err := cmd.Run()
	return err == nil
}

func (s *MigrationService) getProcessVersion(name, flag string) string {
	cmd := exec.Command(name, flag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	parts := strings.Split(string(output), "/")
	if len(parts) >= 2 {
		version := strings.TrimSpace(parts[1])
		if idx := strings.Index(version, " "); idx > 0 {
			version = version[:idx]
		}
		return version
	}
	return strings.TrimSpace(string(output))
}

func (s *MigrationService) isTraefikRunning(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "docker", "ps", "--filter", "name=traefik", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

func (s *MigrationService) listDockerContainers(ctx context.Context) ([]ContainerInfo, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--format", 
		"{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.State}}\t{{.Ports}}\t{{.CreatedAt}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var containers []ContainerInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 7 {
			continue
		}

		if strings.Contains(parts[1], "traefik") || 
		   strings.Contains(parts[1], "flowdeploy") {
			continue
		}

		container := ContainerInfo{
			ID:      parts[0],
			Name:    parts[1],
			Image:   parts[2],
			Status:  parts[3],
			State:   parts[4],
			Created: parts[6],
		}

		if parts[5] != "" {
			container.Ports = strings.Split(parts[5], ", ")
		}

		container.Uptime = s.parseUptime(parts[3])
		containers = append(containers, container)
	}

	return containers, nil
}

func (s *MigrationService) parseUptime(status string) string {
	if strings.Contains(status, "Up") {
		parts := strings.SplitN(status, "Up ", 2)
		if len(parts) > 1 {
			uptime := strings.Split(parts[1], " (")[0]
			return uptime
		}
	}
	return ""
}

func (s *MigrationService) generateWarnings(status *MigrationStatus) []string {
	var warnings []string

	warnings = append(warnings, s.generateCertWarnings(status.SSLCertificates)...)
	warnings = append(warnings, s.generateSiteWarnings(status.NginxSites)...)

	return warnings
}

func (s *MigrationService) generateCertWarnings(certs []SSLCertificate) []string {
	var warnings []string
	for _, cert := range certs {
		if cert.DaysUntilExpiry <= 7 {
			warnings = append(warnings, fmt.Sprintf(
				"Certificate for %s expires in %d days",
				cert.Domain, cert.DaysUntilExpiry))
		}
	}
	return warnings
}

func (s *MigrationService) generateSiteWarnings(sites []NginxSite) []string {
	var warnings []string
	for _, site := range sites {
		warnings = append(warnings, s.checkSiteWarnings(site)...)
	}
	return warnings
}

func (s *MigrationService) checkSiteWarnings(site NginxSite) []string {
	var warnings []string
	serverName := site.ServerNames[0]

	if site.Root != "" && len(site.Locations) == 0 {
		warnings = append(warnings, fmt.Sprintf("%s: Static site needs separate container", serverName))
	}

	if site.HasSSE {
		warnings = append(warnings, fmt.Sprintf("%s: Has SSE config - verify flushinterval after migration", serverName))
	}

	if portCount := countUniquePorts(site.Locations); portCount > 1 {
		warnings = append(warnings, fmt.Sprintf("%s: Multiple backends detected (%d ports) - will create separate apps", serverName, portCount))
	}

	return warnings
}

func countUniquePorts(locations []NginxLocation) int {
	ports := make(map[int]bool)
	for _, loc := range locations {
		if loc.ProxyPort > 0 {
			ports[loc.ProxyPort] = true
		}
	}
	return len(ports)
}

func (s *MigrationService) CreateBackup(ctx context.Context) (*BackupResult, error) {
	timestamp := time.Now().Format("2006-01-02-150405")
	backupPath := filepath.Join(s.backupBasePath, "migration-"+timestamp)

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	var files []string
	var totalSize int64

	nginxBackup := filepath.Join(backupPath, "nginx")
	if _, err := os.Stat("/etc/nginx"); err == nil {
		if err := s.copyDir("/etc/nginx", nginxBackup); err != nil {
			s.logger.Warn("Failed to backup nginx config", "error", err)
		} else {
			files = append(files, "nginx/")
		}
	}

	leBackup := filepath.Join(backupPath, "letsencrypt")
	if _, err := os.Stat("/etc/letsencrypt"); err == nil {
		if err := s.copyDir("/etc/letsencrypt", leBackup); err != nil {
			s.logger.Warn("Failed to backup letsencrypt", "error", err)
		} else {
			files = append(files, "letsencrypt/")
		}
	}

	containerList := filepath.Join(backupPath, "containers.txt")
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--format", "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}")
	output, _ := cmd.Output()
	if err := os.WriteFile(containerList, output, 0644); err == nil {
		files = append(files, "containers.txt")
	}

	filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	s.logger.Info("Backup created", "path", backupPath, "files", len(files))

	return &BackupResult{
		Path:      backupPath,
		CreatedAt: time.Now(),
		Files:     files,
		Size:      totalSize,
	}, nil
}

func (s *MigrationService) copyDir(src, dst string) error {
	cmd := exec.Command("cp", "-r", src, dst)
	return cmd.Run()
}

func (s *MigrationService) StopContainers(ctx context.Context, containerIDs []string) error {
	for _, id := range containerIDs {
		s.logger.Info("Stopping container", "id", id)
		cmd := exec.CommandContext(ctx, "docker", "stop", id)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to stop container %s: %w", id, err)
		}
	}
	return nil
}

func (s *MigrationService) StartContainers(ctx context.Context, containerIDs []string) error {
	for _, id := range containerIDs {
		s.logger.Info("Starting container", "id", id)
		cmd := exec.CommandContext(ctx, "docker", "start", id)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to start container %s: %w", id, err)
		}
	}
	return nil
}

func (s *MigrationService) StopNginx(ctx context.Context) error {
	s.logger.Info("Stopping nginx")

	cmd := exec.CommandContext(ctx, "systemctl", "stop", "nginx")
	if err := cmd.Run(); err != nil {
		cmd = exec.CommandContext(ctx, "service", "nginx", "stop")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to stop nginx: %w", err)
		}
	}

	cmd = exec.CommandContext(ctx, "systemctl", "disable", "nginx")
	cmd.Run()

	return nil
}

func (s *MigrationService) GetTraefikConfigs(site NginxSite) []TraefikConfig {
	return s.traefikConverter.ConvertSite(site)
}

func (s *MigrationService) GetTraefikLabelsYAML(site NginxSite) string {
	configs := s.traefikConverter.ConvertSite(site)
	return s.traefikConverter.GenerateYAMLLabels(configs)
}

type MigrateResult struct {
	ContainerID   string   `json:"containerId"`
	ContainerName string   `json:"containerName"`
	Domain        string   `json:"domain"`
	Labels        []string `json:"labels"`
	Success       bool     `json:"success"`
	Message       string   `json:"message"`
}

func (s *MigrationService) MigrateContainer(ctx context.Context, site NginxSite, containerID string) (*MigrateResult, error) {
	s.logger.Info("Starting container migration", "container", containerID, "domain", site.ServerNames[0])

	configs := s.traefikConverter.ConvertSite(site)
	if len(configs) == 0 {
		return nil, fmt.Errorf("no traefik config generated for site")
	}

	inspectCmd := exec.CommandContext(ctx, "docker", "inspect", containerID)
	inspectOutput, err := inspectCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	var containerInfo []map[string]interface{}
	if err := parseJSON(inspectOutput, &containerInfo); err != nil {
		return nil, fmt.Errorf("failed to parse container info: %w", err)
	}

	if len(containerInfo) == 0 {
		return nil, fmt.Errorf("container not found")
	}

	info := containerInfo[0]
	containerName := strings.TrimPrefix(info["Name"].(string), "/")

	config := info["Config"].(map[string]interface{})
	hostConfig := info["HostConfig"].(map[string]interface{})
	networkSettings := info["NetworkSettings"].(map[string]interface{})

	image := config["Image"].(string)

	var labels []string
	labels = append(labels, "traefik.enable=true")
	for _, cfg := range configs {
		labels = append(labels, fmt.Sprintf("traefik.http.routers.%s.rule=Host(`%s`)", cfg.ServiceName, cfg.Domain))
		labels = append(labels, fmt.Sprintf("traefik.http.routers.%s.entrypoints=websecure", cfg.ServiceName))
		labels = append(labels, fmt.Sprintf("traefik.http.routers.%s.tls=true", cfg.ServiceName))
		labels = append(labels, fmt.Sprintf("traefik.http.routers.%s.tls.certresolver=letsencrypt", cfg.ServiceName))
		labels = append(labels, fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port=%d", cfg.ServiceName, cfg.Port))

		labels = append(labels, fmt.Sprintf("traefik.http.routers.%s-http.rule=Host(`%s`)", cfg.ServiceName, cfg.Domain))
		labels = append(labels, fmt.Sprintf("traefik.http.routers.%s-http.entrypoints=web", cfg.ServiceName))
		labels = append(labels, fmt.Sprintf("traefik.http.routers.%s-http.middlewares=redirect-to-https", cfg.ServiceName))
	}
	labels = append(labels, "traefik.http.middlewares.redirect-to-https.redirectscheme.scheme=https")

	s.logger.Info("Stopping container", "name", containerName)
	stopCmd := exec.CommandContext(ctx, "docker", "stop", containerID)
	if err := stopCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to stop container: %w", err)
	}

	renameCmd := exec.CommandContext(ctx, "docker", "rename", containerName, containerName+"-backup")
	if err := renameCmd.Run(); err != nil {
		startCmd := exec.CommandContext(ctx, "docker", "start", containerID)
		startCmd.Run()
		return nil, fmt.Errorf("failed to rename container: %w", err)
	}

	args := []string{"run", "-d", "--name", containerName}

	if restartPolicy, ok := hostConfig["RestartPolicy"].(map[string]interface{}); ok {
		if name, ok := restartPolicy["Name"].(string); ok && name != "" {
			args = append(args, "--restart", name)
		}
	}

	networks := networkSettings["Networks"].(map[string]interface{})
	for netName := range networks {
		if netName != "bridge" {
			args = append(args, "--network", netName)
		}
	}

	if binds, ok := hostConfig["Binds"].([]interface{}); ok {
		for _, bind := range binds {
			args = append(args, "-v", bind.(string))
		}
	}

	if env, ok := config["Env"].([]interface{}); ok {
		for _, e := range env {
			envStr := e.(string)
			if !strings.HasPrefix(envStr, "PATH=") && !strings.HasPrefix(envStr, "HOME=") {
				args = append(args, "-e", envStr)
			}
		}
	}

	for _, label := range labels {
		args = append(args, "--label", label)
	}

	args = append(args, image)

	s.logger.Info("Creating new container with Traefik labels", "args", args)
	createCmd := exec.CommandContext(ctx, "docker", args...)
	output, err := createCmd.CombinedOutput()
	if err != nil {
		s.logger.Error("Failed to create container, rolling back", "error", err, "output", string(output))
		rollbackRename := exec.CommandContext(ctx, "docker", "rename", containerName+"-backup", containerName)
		rollbackRename.Run()
		startCmd := exec.CommandContext(ctx, "docker", "start", containerID)
		startCmd.Run()
		return nil, fmt.Errorf("failed to create container: %s", string(output))
	}

	newContainerID := strings.TrimSpace(string(output))
	s.logger.Info("Container migrated successfully", "newId", newContainerID)

	removeBackup := exec.CommandContext(ctx, "docker", "rm", containerName+"-backup")
	removeBackup.Run()

	return &MigrateResult{
		ContainerID:   newContainerID,
		ContainerName: containerName,
		Domain:        site.ServerNames[0],
		Labels:        labels,
		Success:       true,
		Message:       "Container migrated successfully with Traefik labels",
	}, nil
}

func parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
