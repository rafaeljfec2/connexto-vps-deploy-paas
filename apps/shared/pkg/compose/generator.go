package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type GenerateParams struct {
	AppName  string
	ImageTag string
	Config   *Config
	Domains  []DomainRoute
	EnvVars  map[string]string
}

func GenerateContent(params GenerateParams) string {
	cfg := params.Config
	envYAML := BuildEnvVarsYAML(cfg, params.EnvVars)
	labels := BuildLabelsYAML(params.AppName, params.Domains, cfg.Port)
	portMapping := BuildPortMapping(cfg.HostPort, cfg.Port)
	healthCmd := BuildHealthCheckCommand(cfg.Runtime, cfg.Port, cfg.Healthcheck.Path)

	return fmt.Sprintf("services:\n"+
		"  %s:\n"+
		"    image: %s\n"+
		"    container_name: %s\n"+
		"    restart: unless-stopped\n"+
		"    ports:\n"+
		"      - \"%s\"\n"+
		"%s"+
		"%s"+
		"    healthcheck:\n"+
		"      test:\n"+
		"        - CMD-SHELL\n"+
		"        - %s\n"+
		"      interval: %s\n"+
		"      timeout: %s\n"+
		"      retries: %d\n"+
		"      start_period: %s\n"+
		"    deploy:\n"+
		"      resources:\n"+
		"        limits:\n"+
		"          memory: %s\n"+
		"          cpus: '%s'\n"+
		"    networks:\n"+
		"      - paasdeploy\n\n"+
		"networks:\n"+
		"  paasdeploy:\n"+
		"    external: true\n",
		params.AppName, params.ImageTag, params.AppName, portMapping, envYAML, labels,
		healthCmd,
		cfg.Healthcheck.Interval, cfg.Healthcheck.Timeout,
		cfg.Healthcheck.Retries, cfg.Healthcheck.StartPeriod,
		cfg.Resources.Memory, cfg.Resources.CPU,
	)
}

func WriteComposeFile(appDir string, params GenerateParams) error {
	content := GenerateContent(params)
	composePath := filepath.Join(appDir, "docker-compose.yml")
	return os.WriteFile(composePath, []byte(content), 0644)
}

func BuildEnvVarsYAML(cfg *Config, appEnvVars map[string]string) string {
	allEnvVars := make(map[string]string)

	for k, v := range cfg.Env {
		allEnvVars[k] = v
	}

	for k, v := range appEnvVars {
		allEnvVars[k] = v
	}

	if len(allEnvVars) == 0 {
		return ""
	}

	envVars := "    environment:\n"
	for k, v := range allEnvVars {
		escapedValue := EscapeEnvValue(v)
		envVars += fmt.Sprintf("      - %s=%s\n", k, escapedValue)
	}
	return envVars
}

func EscapeEnvValue(value string) string {
	return strings.ReplaceAll(value, "$", "$$")
}

func BuildHealthCheckCommand(runtime string, port int, path string) string {
	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)

	switch strings.ToLower(runtime) {
	case "node", "nodejs", "node.js":
		return fmt.Sprintf(
			`node -e 'const h=require("http");h.get("%s",(r)=>process.exit(r.statusCode>=200&&r.statusCode<400?0:1)).on("error",()=>process.exit(1))'`,
			url,
		)
	case "python", "python3":
		return fmt.Sprintf(
			`python3 -c 'import urllib.request,sys;urllib.request.urlopen("%s");sys.exit(0)' 2>/dev/null || python -c 'import urllib.request,sys;urllib.request.urlopen("%s");sys.exit(0)'`,
			url, url,
		)
	case "go", "golang":
		return fmt.Sprintf("curl -sf %s || wget -q --spider %s || exit 1", url, url)
	case "ruby":
		return fmt.Sprintf(
			`ruby -e 'require "net/http";exit(Net::HTTP.get_response(URI("%s")).is_a?(Net::HTTPSuccess)?0:1)'`,
			url,
		)
	case "php":
		return fmt.Sprintf(
			`php -r 'exit(file_get_contents("%s")!==false?0:1);'`,
			url,
		)
	default:
		return fmt.Sprintf("curl -sf %s || wget -q --spider %s || exit 1", url, url)
	}
}

func BuildLabelsYAML(appName string, domains []DomainRoute, port int) string {
	if len(domains) > 0 {
		var labels strings.Builder
		labels.WriteString("    labels:\n")
		labels.WriteString(fmt.Sprintf("      - \"paasdeploy.app=%s\"\n", appName))
		labels.WriteString("      - \"traefik.enable=true\"\n")
		labels.WriteString("      - \"traefik.docker.network=paasdeploy\"\n")
		labels.WriteString(fmt.Sprintf("      - \"traefik.http.services.%s.loadbalancer.server.port=%d\"\n", appName, port))

		for i, d := range domains {
			routerName := appName
			if i > 0 {
				routerName = fmt.Sprintf("%s-%d", appName, i)
			}

			priority := 1
			if d.PathPrefix != "" {
				priority = 100 + len(d.PathPrefix)
			}

			rule := fmt.Sprintf("Host(`%s`)", d.Domain)
			if d.PathPrefix != "" {
				rule = fmt.Sprintf("Host(`%s`) && PathPrefix(`%s`)", d.Domain, d.PathPrefix)
			}

			labels.WriteString(fmt.Sprintf("      - \"traefik.http.routers.%s.rule=%s\"\n", routerName, rule))
			labels.WriteString(fmt.Sprintf("      - \"traefik.http.routers.%s.priority=%d\"\n", routerName, priority))
			labels.WriteString(fmt.Sprintf("      - \"traefik.http.routers.%s.tls=true\"\n", routerName))
			labels.WriteString(fmt.Sprintf("      - \"traefik.http.routers.%s.tls.certresolver=letsencrypt\"\n", routerName))
			labels.WriteString(fmt.Sprintf("      - \"traefik.http.routers.%s.service=%s\"\n", routerName, appName))
		}

		return labels.String()
	}

	return fmt.Sprintf("    labels:\n"+
		"      - \"paasdeploy.app=%s\"\n"+
		"      - \"traefik.enable=true\"\n"+
		"      - \"traefik.docker.network=paasdeploy\"\n"+
		"      - \"traefik.http.routers.%s.rule=Host(`%s.localhost`)\"\n"+
		"      - \"traefik.http.services.%s.loadbalancer.server.port=%d\"\n",
		appName, appName, appName, appName, port)
}

func BuildPortMapping(hostPort, port int) string {
	if hostPort > 0 {
		return fmt.Sprintf("%d:%d", hostPort, port)
	}
	return fmt.Sprintf("%d", port)
}
