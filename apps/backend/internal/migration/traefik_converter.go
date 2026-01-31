package migration

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	traefikRouterEntrypoints = "traefik.http.routers.%s.entrypoints"
	traefikRouterRule        = "traefik.http.routers.%s.rule"
)

type TraefikConfig struct {
	ServiceName  string            `json:"serviceName"`
	Domain       string            `json:"domain"`
	Port         int               `json:"port"`
	PathPrefix   string            `json:"pathPrefix,omitempty"`
	Priority     int               `json:"priority,omitempty"`
	Labels       map[string]string `json:"labels"`
	HasSSE       bool              `json:"hasSSE"`
	HasWebSocket bool              `json:"hasWebSocket"`
	Middlewares  []string          `json:"middlewares,omitempty"`
}

type TraefikConverter struct{}

func NewTraefikConverter() *TraefikConverter {
	return &TraefikConverter{}
}

func (c *TraefikConverter) ConvertSite(site NginxSite) []TraefikConfig {
	portGroups := c.groupLocationsByPort(site.Locations)
	mainDomain := c.getMainDomain(site.ServerNames)

	var configs []TraefikConfig
	for port, locations := range portGroups {
		config := c.buildConfig(mainDomain, port, locations)
		config.Labels = c.generateLabels(config, site)
		configs = append(configs, config)
	}

	return configs
}

func (c *TraefikConverter) groupLocationsByPort(locations []NginxLocation) map[int][]NginxLocation {
	portGroups := make(map[int][]NginxLocation)

	for _, loc := range locations {
		if loc.ProxyPort > 0 {
			portGroups[loc.ProxyPort] = append(portGroups[loc.ProxyPort], loc)
		}
	}

	if len(portGroups) == 0 {
		portGroups = c.extractPortsFromProxyPass(locations)
	}

	return portGroups
}

func (c *TraefikConverter) extractPortsFromProxyPass(locations []NginxLocation) map[int][]NginxLocation {
	portGroups := make(map[int][]NginxLocation)

	for _, loc := range locations {
		if loc.ProxyPass == "" {
			continue
		}
		if port := extractPort(loc.ProxyPass); port > 0 {
			portGroups[port] = append(portGroups[port], loc)
		}
	}

	return portGroups
}

func (c *TraefikConverter) getMainDomain(serverNames []string) string {
	if len(serverNames) > 0 {
		return serverNames[0]
	}
	return ""
}

func (c *TraefikConverter) buildConfig(mainDomain string, port int, locations []NginxLocation) TraefikConfig {
	analysis := c.analyzeLocations(locations, mainDomain)

	return TraefikConfig{
		ServiceName:  analysis.serviceName,
		Domain:       mainDomain,
		Port:         port,
		PathPrefix:   analysis.pathPrefix,
		Priority:     analysis.priority,
		HasSSE:       analysis.hasSSE,
		HasWebSocket: analysis.hasWebSocket,
		Labels:       make(map[string]string),
	}
}

type locationAnalysis struct {
	serviceName  string
	pathPrefix   string
	priority     int
	hasSSE       bool
	hasWebSocket bool
}

func (c *TraefikConverter) analyzeLocations(locations []NginxLocation, mainDomain string) locationAnalysis {
	analysis := locationAnalysis{
		serviceName: sanitizeServiceName(mainDomain),
	}

	for _, loc := range locations {
		if isCustomPath(loc) {
			analysis.pathPrefix = loc.Path
			analysis.priority = 100
			analysis.serviceName = sanitizeServiceName(loc.Path)
		}
		analysis.hasSSE = analysis.hasSSE || loc.HasSSE
		analysis.hasWebSocket = analysis.hasWebSocket || loc.HasWebSocket
	}

	return analysis
}

func isCustomPath(loc NginxLocation) bool {
	return loc.Path != "/" && !loc.IsRegex && loc.Path != ""
}

func (c *TraefikConverter) generateLabels(config TraefikConfig, site NginxSite) map[string]string {
	labels := make(map[string]string)
	svc := config.ServiceName

	labels["traefik.enable"] = "true"

	rule := fmt.Sprintf("Host(`%s`)", config.Domain)
	if config.PathPrefix != "" {
		rule = fmt.Sprintf("Host(`%s`) && PathPrefix(`%s`)", config.Domain, config.PathPrefix)
	}
	labels[fmt.Sprintf(traefikRouterRule, svc)] = rule

	if config.Priority > 0 {
		labels[fmt.Sprintf("traefik.http.routers.%s.priority", svc)] = fmt.Sprintf("%d", config.Priority)
	}

	if site.SSLEnabled {
		labels[fmt.Sprintf(traefikRouterEntrypoints, svc)] = "websecure"
		labels[fmt.Sprintf("traefik.http.routers.%s.tls.certresolver", svc)] = "letsencrypt"

		httpRouter := svc + "-http"
		labels[fmt.Sprintf(traefikRouterRule, httpRouter)] = rule
		labels[fmt.Sprintf(traefikRouterEntrypoints, httpRouter)] = "web"
		labels[fmt.Sprintf("traefik.http.routers.%s.middlewares", httpRouter)] = "redirect-https@docker"
	} else {
		labels[fmt.Sprintf(traefikRouterEntrypoints, svc)] = "web"
	}

	labels[fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", svc)] = fmt.Sprintf("%d", config.Port)

	if config.HasSSE {
		labels[fmt.Sprintf("traefik.http.services.%s.loadbalancer.responseforwarding.flushinterval", svc)] = "1ms"
	}

	var middlewares []string

	if len(site.Headers) > 0 || c.needsHeadersMiddleware(site) {
		headersMw := svc + "-headers"
		middlewares = append(middlewares, headersMw+"@docker")

		for key, value := range site.Headers {
			labels[fmt.Sprintf("traefik.http.middlewares.%s.headers.customrequestheaders.%s", headersMw, key)] = value
		}
	}

	if config.PathPrefix != "" && strings.HasSuffix(config.PathPrefix, "/") {
		stripMw := svc + "-strip"
		middlewares = append(middlewares, stripMw+"@docker")
		labels[fmt.Sprintf("traefik.http.middlewares.%s.stripprefix.prefixes", stripMw)] = config.PathPrefix
	}

	if len(middlewares) > 0 {
		labels[fmt.Sprintf("traefik.http.routers.%s.middlewares", svc)] = strings.Join(middlewares, ",")
	}

	labels["traefik.docker.network"] = "traefik-public"

	return labels
}

func (c *TraefikConverter) needsHeadersMiddleware(site NginxSite) bool {
	for _, loc := range site.Locations {
		if len(loc.ProxyHeaders) > 0 {
			return true
		}
	}
	return false
}

func (c *TraefikConverter) GenerateDockerComposeLabels(config TraefikConfig) string {
	var lines []string
	lines = append(lines, "labels:")

	for key, value := range config.Labels {
		lines = append(lines, fmt.Sprintf("  - \"%s=%s\"", key, value))
	}

	return strings.Join(lines, "\n")
}

func (c *TraefikConverter) GenerateYAMLLabels(configs []TraefikConfig) string {
	var result strings.Builder

	for i, config := range configs {
		if i > 0 {
			result.WriteString("\n---\n")
		}
		result.WriteString(fmt.Sprintf("# Service: %s (port %d)\n", config.ServiceName, config.Port))
		if config.PathPrefix != "" {
			result.WriteString(fmt.Sprintf("# Path: %s\n", config.PathPrefix))
		}
		if config.HasSSE {
			result.WriteString("# Features: SSE enabled\n")
		}
		if config.HasWebSocket {
			result.WriteString("# Features: WebSocket enabled\n")
		}
		result.WriteString(c.GenerateDockerComposeLabels(config))
		result.WriteString("\n")
	}

	return result.String()
}

func sanitizeServiceName(input string) string {
	input = strings.TrimPrefix(input, "/")
	input = strings.TrimSuffix(input, "/")

	reg := regexp.MustCompile(`[^a-zA-Z0-9-]`)
	result := reg.ReplaceAllString(input, "-")

	result = strings.Trim(result, "-")

	if result == "" {
		result = "default"
	}

	return strings.ToLower(result)
}
