package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	httpClient *http.Client
	token      string
	baseURL    string
}

func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		token:   token,
		baseURL: defaultBaseURL,
	}
}

func (c *Client) CreateWebhook(ctx context.Context, owner, repo, webhookURL, secret string) (*Webhook, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/hooks", c.baseURL, owner, repo)

	payload := CreateWebhookRequest{
		Name:   "web",
		Active: true,
		Events: []string{"push"},
		Config: WebhookConfig{
			URL:         webhookURL,
			ContentType: "json",
			Secret:      secret,
			InsecureSSL: "0",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal webhook request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var webhook Webhook
	if err := json.NewDecoder(resp.Body).Decode(&webhook); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &webhook, nil
}

func (c *Client) DeleteWebhook(ctx context.Context, owner, repo string, hookID int64) error {
	url := fmt.Sprintf("%s/repos/%s/%s/hooks/%d", c.baseURL, owner, repo, hookID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (c *Client) ListWebhooks(ctx context.Context, owner, repo string) ([]Webhook, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/hooks", c.baseURL, owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var webhooks []Webhook
	if err := json.NewDecoder(resp.Body).Decode(&webhooks); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return webhooks, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Accept", acceptHeader)
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	req.Header.Set("User-Agent", userAgentHeader)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}
