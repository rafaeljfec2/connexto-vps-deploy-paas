package migration

import (
	"fmt"
	"regexp"
	"strings"
)

type TraefikConfig struct {
	ServiceName    string            `json:"serviceName"`
	Domain         string            `json:"domain"`
	Port           int               `json:"port"`
	PathPrefix     string            `json:"pathPrefix,omitempty"`
	Priority       int               `json:"priority,omitempty"`
	Labels         map[string]string `json:"labels"`
	HasSSE         bool              `json:"hasSSE"`
	HasWebSocket   bool              `json:"hasWebSocket"`
	Middlewares    []string          `json:"middlewares,omitempty"`
}

type TraefikConverter struct{}

func NewTraefikConverter() *TraefikConverter {
	return &TraefikConverter{}
}

func (c *TraefikConverter) ConvertSite(site NginxSite) []TraefikConfig {
	var configs []TraefikConfig

	portGroups := make(map[int][]NginxLocation)
	for _, loc := range site.Locations {
		if loc.ProxyPort > 0 {
			portGroups[loc.ProxyPort] = append(portGroups[loc.ProxyPort], loc)
		}
	}

	if len(portGroups) == 0 && len(site.Locations) > 0 {
		for _, loc := range site.Locations {
			if loc.ProxyPass != "" {
				port := extractPort(loc.ProxyPass)
				if port > 0 {
					portGroups[port] = append(portGroups[port], loc)
				}
			}
		}
	}

	mainDomain := ""
	if len(site.ServerNames) > 0 {
		mainDomain = site.ServerNames[0]
	}

	for port, locations := range portGroups {
		serviceName := sanitizeServiceName(mainDomain)
		
		var pathPrefix string
		priority := 0
		hasSSE := false
		hasWebSocket := false

		for _, loc := range locations {
			if loc.Path != "/" && !loc.IsRegex && loc.Path != "" {
				pathPrefix = loc.Path
				priority = 100
				serviceName = sanitizeServiceName(pathPrefix)
			}
			if loc.HasSSE {
				hasSSE = true
			}
			if loc.HasWebSocket {
				hasWebSocket = true
			}
		}

		config := TraefikConfig{
			ServiceName:  serviceName,
			Domain:       mainDomain,
			Port:         port,
			PathPrefix:   pathPrefix,
			Priority:     priority,
			HasSSE:       hasSSE,
			HasWebSocket: hasWebSocket,
			Labels:       make(map[string]string),
		}

		config.Labels = c.generateLabels(config, site)
		configs = append(configs, config)
	}

	return configs
}

func (c *TraefikConverter) generateLabels(config TraefikConfig, site NginxSite) map[string]string {
	labels := make(map[string]string)
	svc := config.ServiceName

	labels["traefik.enable"] = "true"

	rule := fmt.Sprintf("Host(`%s`)", config.Domain)
	if config.PathPrefix != "" {
		rule = fmt.Sprintf("Host(`%s`) && PathPrefix(`%s`)", config.Domain, config.PathPrefix)
	}
	labels[fmt.Sprintf("traefik.http.routers.%s.rule", svc)] = rule

	if config.Priority > 0 {
		labels[fmt.Sprintf("traefik.http.routers.%s.priority", svc)] = fmt.Sprintf("%d", config.Priority)
	}

	if site.SSLEnabled {
		labels[fmt.Sprintf("traefik.http.routers.%s.entrypoints", svc)] = "websecure"
		labels[fmt.Sprintf("traefik.http.routers.%s.tls.certresolver", svc)] = "letsencrypt"

		httpRouter := svc + "-http"
		labels[fmt.Sprintf("traefik.http.routers.%s.rule", httpRouter)] = rule
		labels[fmt.Sprintf("traefik.http.routers.%s.entrypoints", httpRouter)] = "web"
		labels[fmt.Sprintf("traefik.http.routers.%s.middlewares", httpRouter)] = "redirect-https@docker"
	} else {
		labels[fmt.Sprintf("traefik.http.routers.%s.entrypoints", svc)] = "web"
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
