package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL  = "https://api.github.com"
	defaultTimeout  = 30 * time.Second
	apiVersion      = "2022-11-28"
	acceptHeader    = "application/vnd.github+json"
	userAgentHeader = "PaaSDeploy/1.0"

	errCreateRequest   = "create request: %w"
	errSendRequest     = "send request: %w"
	errUnexpectedStatus = "unexpected status %d: %s"
	errDecodeResponse  = "decode response: %w"
)

var _ Provider = (*PATProvider)(nil)

type PATProvider struct {
	httpClient *http.Client
	token      string
	baseURL    string
}

func NewPATProvider(token string) *PATProvider {
	return &PATProvider{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		token:   token,
		baseURL: defaultBaseURL,
	}
}

func (p *PATProvider) CreateWebhook(ctx context.Context, owner, repo string, config WebhookConfig) (*Webhook, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/hooks", p.baseURL, owner, repo)

	payload := CreateWebhookRequest{
		Name:   "web",
		Active: true,
		Events: []string{"push"},
		Config: config,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal webhook request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf(errCreateRequest, err)
	}

	p.setHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errSendRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(errUnexpectedStatus, resp.StatusCode, string(respBody))
	}

	var webhook Webhook
	if err := json.NewDecoder(resp.Body).Decode(&webhook); err != nil {
		return nil, fmt.Errorf(errDecodeResponse, err)
	}

	return &webhook, nil
}

func (p *PATProvider) DeleteWebhook(ctx context.Context, owner, repo string, hookID int64) error {
	url := fmt.Sprintf("%s/repos/%s/%s/hooks/%d", p.baseURL, owner, repo, hookID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf(errCreateRequest, err)
	}

	p.setHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf(errSendRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(errUnexpectedStatus, resp.StatusCode, string(respBody))
	}

	return nil
}

func (p *PATProvider) GetWebhook(ctx context.Context, owner, repo string, hookID int64) (*Webhook, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/hooks/%d", p.baseURL, owner, repo, hookID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf(errCreateRequest, err)
	}

	p.setHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errSendRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(errUnexpectedStatus, resp.StatusCode, string(respBody))
	}

	var webhook Webhook
	if err := json.NewDecoder(resp.Body).Decode(&webhook); err != nil {
		return nil, fmt.Errorf(errDecodeResponse, err)
	}

	return &webhook, nil
}

func (p *PATProvider) ListWebhooks(ctx context.Context, owner, repo string) ([]Webhook, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/hooks", p.baseURL, owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf(errCreateRequest, err)
	}

	p.setHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errSendRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(errUnexpectedStatus, resp.StatusCode, string(respBody))
	}

	var webhooks []Webhook
	if err := json.NewDecoder(resp.Body).Decode(&webhooks); err != nil {
		return nil, fmt.Errorf(errDecodeResponse, err)
	}

	return webhooks, nil
}

func (p *PATProvider) setHeaders(req *http.Request) {
	req.Header.Set("Accept", acceptHeader)
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	req.Header.Set("User-Agent", userAgentHeader)
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
}
