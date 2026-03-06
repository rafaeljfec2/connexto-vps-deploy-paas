package traefik

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type Router struct {
	Name        string   `json:"name"`
	EntryPoints []string `json:"entryPoints"`
	Service     string   `json:"service"`
	Rule        string   `json:"rule"`
	TLS         *TLS     `json:"tls,omitempty"`
	Status      string   `json:"status"`
	Provider    string   `json:"provider"`
}

type TLS struct {
	CertResolver string `json:"certResolver,omitempty"`
}

type Certificate struct {
	Domain      string    `json:"domain"`
	Certificate string    `json:"certificate,omitempty"`
	Key         string    `json:"key,omitempty"`
	Store       string    `json:"store,omitempty"`
	Sans        []string  `json:"sans,omitempty"`
	NotAfter    time.Time `json:"notAfter,omitempty"`
	NotBefore   time.Time `json:"notBefore,omitempty"`
	Issuer      string    `json:"issuer,omitempty"`
}

type CertificateStatus struct {
	Domain    string     `json:"domain"`
	Status    string     `json:"status"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	IssuedAt  *time.Time `json:"issuedAt,omitempty"`
	Issuer    string     `json:"issuer,omitempty"`
	Error     string     `json:"error,omitempty"`
}

func (c *Client) GetRouters(ctx context.Context) ([]Router, error) {
	url := fmt.Sprintf("%s/api/http/routers", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch routers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("traefik API error: %s - %s", resp.Status, string(body))
	}

	var routers []Router
	if err := json.NewDecoder(resp.Body).Decode(&routers); err != nil {
		return nil, fmt.Errorf("failed to decode routers: %w", err)
	}

	return routers, nil
}

func (c *Client) GetRouterByName(ctx context.Context, name string) (*Router, error) {
	url := fmt.Sprintf("%s/api/http/routers/%s", c.baseURL, name)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch router: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("traefik API error: %s - %s", resp.Status, string(body))
	}

	var router Router
	if err := json.NewDecoder(resp.Body).Decode(&router); err != nil {
		return nil, fmt.Errorf("failed to decode router: %w", err)
	}

	return &router, nil
}

func (c *Client) GetCertificateStatus(ctx context.Context, domain string) (*CertificateStatus, error) {
	routers, err := c.GetRouters(ctx)
	if err != nil {
		return nil, err
	}

	status := &CertificateStatus{
		Domain: domain,
		Status: "unknown",
	}

	for _, router := range routers {
		if containsDomain(router.Rule, domain) {
			if router.TLS != nil && router.TLS.CertResolver != "" {
				if router.Status == "enabled" {
					status.Status = "active"
				} else {
					status.Status = "pending"
				}
			} else {
				status.Status = "no_tls"
			}
			break
		}
	}

	return status, nil
}

func (c *Client) GetAllCertificatesStatus(ctx context.Context) ([]CertificateStatus, error) {
	routers, err := c.GetRouters(ctx)
	if err != nil {
		return nil, err
	}

	var certificates []CertificateStatus
	seen := make(map[string]bool)

	for _, router := range routers {
		certs := extractCertificatesFromRouter(router, seen)
		certificates = append(certificates, certs...)
	}

	return certificates, nil
}

func extractCertificatesFromRouter(router Router, seen map[string]bool) []CertificateStatus {
	if router.TLS == nil || router.TLS.CertResolver == "" {
		return nil
	}

	var certificates []CertificateStatus
	domains := extractDomainsFromRule(router.Rule)

	for _, domain := range domains {
		if seen[domain] {
			continue
		}
		seen[domain] = true

		certificates = append(certificates, CertificateStatus{
			Domain: domain,
			Status: resolveRouterStatus(router.Status),
		})
	}

	return certificates
}

func resolveRouterStatus(routerStatus string) string {
	if routerStatus == "enabled" {
		return "active"
	}
	return "pending"
}

func containsDomain(rule, domain string) bool {
	return len(rule) > 0 && len(domain) > 0 && 
		(contains(rule, fmt.Sprintf("Host(`%s`)", domain)) ||
		 contains(rule, fmt.Sprintf("host(`%s`)", domain)))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsString(s, substr))
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func extractDomainsFromRule(rule string) []string {
	var domains []string
	
	i := 0
	for i < len(rule) {
		hostStart := findString(rule[i:], "Host(`")
		if hostStart == -1 {
			hostStart = findString(rule[i:], "host(`")
		}
		if hostStart == -1 {
			break
		}
		
		hostStart += i + 6
		hostEnd := findString(rule[hostStart:], "`)")
		if hostEnd == -1 {
			break
		}
		
		domain := rule[hostStart : hostStart+hostEnd]
		if domain != "" && domain != "localhost" && !containsString(domain, ".localhost") {
			domains = append(domains, domain)
		}
		
		i = hostStart + hostEnd + 2
	}
	
	return domains
}

func findString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
