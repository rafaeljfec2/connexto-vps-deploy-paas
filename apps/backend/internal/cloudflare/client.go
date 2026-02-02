package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	baseURL            = "https://api.cloudflare.com/client/v4"
	errRequestFailed   = "request failed: %w"
	errDecodeResponse  = "decode response: %w"
	errCloudflareAPI   = "cloudflare API error: %v"
)

type Client struct {
	apiToken   string
	httpClient *http.Client
	logger     *slog.Logger
}

func NewClient(apiToken string, logger *slog.Logger) *Client {
	return &Client{
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.With("component", "cloudflare"),
	}
}

type Zone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type DNSRecord struct {
	ID       string `json:"id,omitempty"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Proxied  bool   `json:"proxied"`
	ZoneID   string `json:"zone_id,omitempty"`
	ZoneName string `json:"zone_name,omitempty"`
}

type UserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type apiResponse[T any] struct {
	Success  bool         `json:"success"`
	Errors   []apiMessage `json:"errors"`
	Messages []apiMessage `json:"messages"`
	Result   T            `json:"result"`
}

type apiMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Type    any    `json:"type"`
}

func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

func (c *Client) GetUserInfo(ctx context.Context) (*UserInfo, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/user", nil)
	if err != nil {
		return nil, fmt.Errorf(errRequestFailed, err)
	}
	defer resp.Body.Close()

	var result apiResponse[UserInfo]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf(errDecodeResponse, err)
	}

	if !result.Success {
		return nil, fmt.Errorf(errCloudflareAPI, result.Errors)
	}

	return &result.Result, nil
}

func (c *Client) GetZoneID(ctx context.Context, domain string) (string, error) {
	path := fmt.Sprintf("/zones?name=%s", domain)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", fmt.Errorf(errRequestFailed, err)
	}
	defer resp.Body.Close()

	var result apiResponse[[]Zone]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf(errDecodeResponse, err)
	}

	if !result.Success {
		return "", fmt.Errorf(errCloudflareAPI, result.Errors)
	}

	if len(result.Result) == 0 {
		return "", fmt.Errorf("zone not found for domain: %s", domain)
	}

	return result.Result[0].ID, nil
}

func (c *Client) CreateCNAME(ctx context.Context, zoneID, name, target string) (string, error) {
	record := DNSRecord{
		Type:    "CNAME",
		Name:    name,
		Content: target,
		TTL:     1,
		Proxied: false,
	}

	return c.createRecord(ctx, zoneID, record)
}

func (c *Client) CreateARecord(ctx context.Context, zoneID, name, ip string) (string, error) {
	record := DNSRecord{
		Type:    "A",
		Name:    name,
		Content: ip,
		TTL:     1,
		Proxied: false,
	}

	return c.createRecord(ctx, zoneID, record)
}

func (c *Client) CreateOrGetARecord(ctx context.Context, zoneID, name, ip string) (string, error) {
	recordID, err := c.CreateARecord(ctx, zoneID, name, ip)
	if err == nil {
		return recordID, nil
	}

	if !strings.Contains(err.Error(), "81058") {
		return "", err
	}

	c.logger.Info("DNS record already exists, fetching existing record", "name", name)

	records, err := c.ListRecords(ctx, zoneID, name)
	if err != nil {
		return "", fmt.Errorf("failed to list existing records: %w", err)
	}

	for _, r := range records {
		if r.Type == "A" && r.Name == name {
			c.logger.Info("Found existing DNS record", "record_id", r.ID, "name", r.Name)
			return r.ID, nil
		}
	}

	return "", fmt.Errorf("record exists but could not be found")
}

func (c *Client) createRecord(ctx context.Context, zoneID string, record DNSRecord) (string, error) {
	body, err := json.Marshal(record)
	if err != nil {
		return "", fmt.Errorf("marshal record: %w", err)
	}

	path := fmt.Sprintf("/zones/%s/dns_records", zoneID)
	resp, err := c.doRequest(ctx, http.MethodPost, path, strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf(errRequestFailed, err)
	}
	defer resp.Body.Close()

	var result apiResponse[DNSRecord]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf(errDecodeResponse, err)
	}

	if !result.Success {
		return "", fmt.Errorf(errCloudflareAPI, result.Errors)
	}

	c.logger.Info("DNS record created",
		"record_id", result.Result.ID,
		"type", record.Type,
		"name", record.Name,
		"content", record.Content,
	)

	return result.Result.ID, nil
}

func (c *Client) DeleteRecord(ctx context.Context, zoneID, recordID string) error {
	path := fmt.Sprintf("/zones/%s/dns_records/%s", zoneID, recordID)
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf(errRequestFailed, err)
	}
	defer resp.Body.Close()

	var result apiResponse[struct {
		ID string `json:"id"`
	}]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf(errDecodeResponse, err)
	}

	if !result.Success {
		return fmt.Errorf(errCloudflareAPI, result.Errors)
	}

	c.logger.Info("DNS record deleted", "record_id", recordID)

	return nil
}

func (c *Client) ListRecords(ctx context.Context, zoneID, name string) ([]DNSRecord, error) {
	path := fmt.Sprintf("/zones/%s/dns_records", zoneID)
	if name != "" {
		path += fmt.Sprintf("?name=%s", name)
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf(errRequestFailed, err)
	}
	defer resp.Body.Close()

	var result apiResponse[[]DNSRecord]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf(errDecodeResponse, err)
	}

	if !result.Success {
		return nil, fmt.Errorf(errCloudflareAPI, result.Errors)
	}

	return result.Result, nil
}

type TokenInfo struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (c *Client) VerifyToken(ctx context.Context) (*TokenInfo, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/user/tokens/verify", nil)
	if err != nil {
		return nil, fmt.Errorf(errRequestFailed, err)
	}
	defer resp.Body.Close()

	var result apiResponse[TokenInfo]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf(errDecodeResponse, err)
	}

	if !result.Success {
		return nil, fmt.Errorf("token verification failed: %v", result.Errors)
	}

	if result.Result.Status != "active" {
		return nil, fmt.Errorf("token is not active: %s", result.Result.Status)
	}

	return &result.Result, nil
}
