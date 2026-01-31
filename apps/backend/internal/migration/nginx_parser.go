package migration

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	serverNameRe    = regexp.MustCompile(`server_name\s+([^;]+);`)
	listenRe        = regexp.MustCompile(`listen\s+([^;]+);`)
	locationRe      = regexp.MustCompile(`location\s+(~\s+|~\*\s+|=\s+|)([^\s{]+)\s*\{`)
	proxyHeaderRe   = regexp.MustCompile(`proxy_set_header\s+(\S+)\s+([^;]+);`)
	portRe          = regexp.MustCompile(`:(\d+)`)
)

type NginxSite struct {
	ConfigFile   string            `json:"configFile"`
	ServerNames  []string          `json:"serverNames"`
	Listen       []ListenDirective `json:"listen"`
	Root         string            `json:"root,omitempty"`
	Locations    []NginxLocation   `json:"locations"`
	SSLEnabled   bool              `json:"sslEnabled"`
	SSLCertPath  string            `json:"sslCertPath,omitempty"`
	SSLKeyPath   string            `json:"sslKeyPath,omitempty"`
	SSLProvider  string            `json:"sslProvider,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	HasWebSocket bool              `json:"hasWebSocket"`
	HasSSE       bool              `json:"hasSSE"`
	RawConfig    string            `json:"rawConfig"`
}

type ListenDirective struct {
	Port          int    `json:"port"`
	SSL           bool   `json:"ssl"`
	HTTP2         bool   `json:"http2"`
	DefaultServer bool   `json:"defaultServer"`
	Address       string `json:"address,omitempty"`
}

type NginxLocation struct {
	Path            string            `json:"path"`
	IsRegex         bool              `json:"isRegex"`
	ProxyPass       string            `json:"proxyPass,omitempty"`
	ProxyPort       int               `json:"proxyPort,omitempty"`
	Root            string            `json:"root,omitempty"`
	TryFiles        string            `json:"tryFiles,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	ProxyHeaders    map[string]string `json:"proxyHeaders,omitempty"`
	HasWebSocket    bool              `json:"hasWebSocket"`
	HasSSE          bool              `json:"hasSSE"`
	SSEConfig       *SSEConfig        `json:"sseConfig,omitempty"`
	ProxyBuffering  string            `json:"proxyBuffering,omitempty"`
	ProxyCache      string            `json:"proxyCache,omitempty"`
	ReadTimeout     string            `json:"readTimeout,omitempty"`
	SendTimeout     string            `json:"sendTimeout,omitempty"`
	ConnectTimeout  string            `json:"connectTimeout,omitempty"`
}

type SSEConfig struct {
	BufferingOff       bool   `json:"bufferingOff"`
	CacheOff           bool   `json:"cacheOff"`
	ReadTimeout        string `json:"readTimeout,omitempty"`
	SendTimeout        string `json:"sendTimeout,omitempty"`
	ChunkedEncoding    string `json:"chunkedEncoding,omitempty"`
	XAccelBuffering    string `json:"xAccelBuffering,omitempty"`
}

type NginxParser struct {
	sitesEnabledPath  string
	sitesAvailablePath string
	confDPath         string
}

func NewNginxParser() *NginxParser {
	return &NginxParser{
		sitesEnabledPath:   "/etc/nginx/sites-enabled",
		sitesAvailablePath: "/etc/nginx/sites-available",
		confDPath:          "/etc/nginx/conf.d",
	}
}

func (p *NginxParser) ParseAllSites() ([]NginxSite, error) {
	var sites []NginxSite

	paths := []string{p.sitesEnabledPath, p.confDPath}

	for _, basePath := range paths {
		if _, err := os.Stat(basePath); os.IsNotExist(err) {
			continue
		}

		files, err := os.ReadDir(basePath)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			if strings.HasPrefix(file.Name(), "default") {
				continue
			}

			filePath := filepath.Join(basePath, file.Name())
			site, err := p.ParseFile(filePath)
			if err != nil {
				continue
			}

			if len(site.ServerNames) > 0 && site.ServerNames[0] != "_" {
				sites = append(sites, *site)
			}
		}
	}

	return sites, nil
}

func (p *NginxParser) ParseFile(filePath string) (*NginxSite, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return p.ParseContent(string(content), filePath)
}

func (p *NginxParser) ParseContent(content string, filePath string) (*NginxSite, error) {
	site := &NginxSite{
		ConfigFile: filePath,
		Headers:    make(map[string]string),
		RawConfig:  content,
	}

	serverBlocks := extractServerBlocks(content)
	if len(serverBlocks) == 0 {
		return site, nil
	}

	mainBlock := serverBlocks[0]
	for _, block := range serverBlocks {
		if strings.Contains(block, "listen 443") || strings.Contains(block, "ssl_certificate") {
			mainBlock = block
			break
		}
	}

	site.ServerNames = parseServerNames(mainBlock)
	site.Listen = parseListenDirectives(mainBlock)
	site.Locations = parseLocations(mainBlock)
	site.Root = parseDirective(mainBlock, "root")

	if strings.Contains(mainBlock, "ssl_certificate") {
		site.SSLEnabled = true
		site.SSLCertPath = parseDirective(mainBlock, "ssl_certificate")
		site.SSLKeyPath = parseDirective(mainBlock, "ssl_certificate_key")
		site.SSLProvider = detectSSLProvider(site.SSLCertPath)
	}

	for _, loc := range site.Locations {
		if loc.HasWebSocket {
			site.HasWebSocket = true
		}
		if loc.HasSSE {
			site.HasSSE = true
		}
	}

	return site, nil
}

func extractServerBlocks(content string) []string {
	var blocks []string
	var currentBlock strings.Builder
	depth := 0
	inServer := false

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "server") && strings.Contains(trimmed, "{") {
			inServer = true
			depth = 1
			currentBlock.Reset()
			currentBlock.WriteString(line + "\n")
			continue
		}

		if inServer {
			currentBlock.WriteString(line + "\n")
			depth += strings.Count(line, "{")
			depth -= strings.Count(line, "}")

			if depth <= 0 {
				blocks = append(blocks, currentBlock.String())
				inServer = false
			}
		}
	}

	return blocks
}

func parseServerNames(block string) []string {
	match := serverNameRe.FindStringSubmatch(block)
	if len(match) < 2 {
		return nil
	}

	names := strings.Fields(match[1])
	var result []string
	for _, name := range names {
		if name != "_" && name != "" {
			result = append(result, name)
		}
	}
	return result
}

func parseListenDirectives(block string) []ListenDirective {
	var directives []ListenDirective
	matches := listenRe.FindAllStringSubmatch(block, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		parts := strings.Fields(match[1])
		if len(parts) == 0 {
			continue
		}

		directive := ListenDirective{}

		portStr := parts[0]
		if strings.Contains(portStr, ":") {
			addrParts := strings.Split(portStr, ":")
			directive.Address = addrParts[0]
			portStr = addrParts[1]
		}

		port, err := strconv.Atoi(portStr)
		if err == nil {
			directive.Port = port
		}

		for _, part := range parts[1:] {
			switch part {
			case "ssl":
				directive.SSL = true
			case "http2":
				directive.HTTP2 = true
			case "default_server":
				directive.DefaultServer = true
			}
		}

		directives = append(directives, directive)
	}

	return directives
}

func parseLocations(block string) []NginxLocation {
	var locations []NginxLocation
	matches := locationRe.FindAllStringSubmatchIndex(block, -1)

	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		modifier := strings.TrimSpace(block[match[2]:match[3]])
		path := block[match[4]:match[5]]

		locationBlock := extractBlock(block[match[0]:])

		loc := NginxLocation{
			Path:         path,
			IsRegex:      modifier == "~" || modifier == "~*",
			Headers:      make(map[string]string),
			ProxyHeaders: make(map[string]string),
		}

		loc.ProxyPass = parseDirective(locationBlock, "proxy_pass")
		if loc.ProxyPass != "" {
			loc.ProxyPort = extractPort(loc.ProxyPass)
		}

		loc.Root = parseDirective(locationBlock, "root")
		loc.TryFiles = parseDirective(locationBlock, "try_files")
		loc.ProxyBuffering = parseDirective(locationBlock, "proxy_buffering")
		loc.ProxyCache = parseDirective(locationBlock, "proxy_cache")
		loc.ReadTimeout = parseDirective(locationBlock, "proxy_read_timeout")
		loc.SendTimeout = parseDirective(locationBlock, "proxy_send_timeout")
		loc.ConnectTimeout = parseDirective(locationBlock, "proxy_connect_timeout")

		headerMatches := proxyHeaderRe.FindAllStringSubmatch(locationBlock, -1)
		for _, hm := range headerMatches {
			if len(hm) >= 3 {
				loc.ProxyHeaders[hm[1]] = strings.TrimSpace(hm[2])
			}
		}

		if strings.Contains(locationBlock, `$http_upgrade`) ||
			strings.Contains(locationBlock, `"upgrade"`) {
			loc.HasWebSocket = true
		}

		if loc.ProxyBuffering == "off" ||
			strings.Contains(locationBlock, "X-Accel-Buffering") ||
			strings.Contains(locationBlock, "chunked_transfer_encoding") {
			loc.HasSSE = true
			loc.SSEConfig = &SSEConfig{
				BufferingOff:    loc.ProxyBuffering == "off",
				CacheOff:        loc.ProxyCache == "off",
				ReadTimeout:     loc.ReadTimeout,
				SendTimeout:     loc.SendTimeout,
				ChunkedEncoding: parseDirective(locationBlock, "chunked_transfer_encoding"),
				XAccelBuffering: loc.ProxyHeaders["X-Accel-Buffering"],
			}
		}

		locations = append(locations, loc)
	}

	return locations
}

func extractBlock(content string) string {
	depth := 0
	start := strings.Index(content, "{")
	if start == -1 {
		return ""
	}

	for i := start; i < len(content); i++ {
		if content[i] == '{' {
			depth++
		} else if content[i] == '}' {
			depth--
			if depth == 0 {
				return content[start+1 : i]
			}
		}
	}

	return ""
}

func parseDirective(block, directive string) string {
	re := regexp.MustCompile(directive + `\s+([^;]+);`)
	match := re.FindStringSubmatch(block)
	if len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func extractPort(proxyPass string) int {
	match := portRe.FindStringSubmatch(proxyPass)
	if len(match) >= 2 {
		port, err := strconv.Atoi(match[1])
		if err == nil {
			return port
		}
	}
	return 0
}

func detectSSLProvider(certPath string) string {
	if strings.Contains(certPath, "letsencrypt") {
		return "certbot"
	}
	if strings.Contains(certPath, "cloudflare") {
		return "cloudflare"
	}
	return "manual"
}
