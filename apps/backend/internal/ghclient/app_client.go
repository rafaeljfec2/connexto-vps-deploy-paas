package ghclient

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AppClient struct {
	appID      int64
	privateKey *rsa.PrivateKey
	httpClient *http.Client
}

type AppConfig struct {
	AppID      int64
	PrivateKey []byte
}

type InstallationToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type AppRepository struct {
	ID            int64    `json:"id"`
	NodeID        string   `json:"node_id"`
	Name          string   `json:"name"`
	FullName      string   `json:"full_name"`
	Owner         AppOwner `json:"owner"`
	Private       bool     `json:"private"`
	Description   string   `json:"description"`
	Fork          bool     `json:"fork"`
	HTMLURL       string   `json:"html_url"`
	CloneURL      string   `json:"clone_url"`
	SSHURL        string   `json:"ssh_url"`
	DefaultBranch string   `json:"default_branch"`
	Language      string   `json:"language"`
	UpdatedAt     string   `json:"updated_at"`
	PushedAt      string   `json:"pushed_at"`
}

type AppOwner struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	Type      string `json:"type"`
}

type InstallationRepositoriesResponse struct {
	TotalCount   int             `json:"total_count"`
	Repositories []AppRepository `json:"repositories"`
}

func NewAppClient(cfg AppConfig) (*AppClient, error) {
	privateKey, err := ParsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	return &AppClient{
		appID:      cfg.AppID,
		privateKey: privateKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *AppClient) GenerateJWT() (string, error) {
	return GenerateAppJWT(c.appID, c.privateKey)
}

func (c *AppClient) GetInstallationToken(ctx context.Context, installationID int64) (*InstallationToken, error) {
	jwt, err := c.GenerateJWT()
	if err != nil {
		return nil, fmt.Errorf("generate JWT: %w", err)
	}

	url := fmt.Sprintf("%s/app/installations/%d/access_tokens", githubAPIURL, installationID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf(errCreateRequest, err)
	}

	req.Header.Set("Authorization", authSchemeBearer+jwt)
	req.Header.Set("Accept", acceptHeader)
	req.Header.Set(headerGitHubAPIVersion, apiVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errSendRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(errUnexpectedStatus, resp.StatusCode, string(body))
	}

	var token InstallationToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf(errDecodeResponse, err)
	}

	return &token, nil
}

func (c *AppClient) ListInstallationRepos(ctx context.Context, installationID int64) ([]AppRepository, error) {
	token, err := c.GetInstallationToken(ctx, installationID)
	if err != nil {
		return nil, err
	}

	return c.listReposWithToken(ctx, token.Token)
}

func (c *AppClient) listReposWithToken(ctx context.Context, accessToken string) ([]AppRepository, error) {
	var allRepos []AppRepository
	page := 1
	perPage := 100

	for {
		url := fmt.Sprintf("%s/installation/repositories?page=%d&per_page=%d", githubAPIURL, page, perPage)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf(errCreateRequest, err)
		}

		req.Header.Set("Authorization", authSchemeBearer+accessToken)
		req.Header.Set("Accept", acceptHeader)
		req.Header.Set(headerGitHubAPIVersion, apiVersion)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf(errSendRequest, err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf(errUnexpectedStatus, resp.StatusCode, string(body))
		}

		var reposResp InstallationRepositoriesResponse
		if err := json.NewDecoder(resp.Body).Decode(&reposResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf(errDecodeResponse, err)
		}
		resp.Body.Close()

		allRepos = append(allRepos, reposResp.Repositories...)

		if len(reposResp.Repositories) < perPage {
			break
		}
		page++
	}

	return allRepos, nil
}

func (c *AppClient) GetRepository(ctx context.Context, installationID int64, owner, repo string) (*AppRepository, error) {
	token, err := c.GetInstallationToken(ctx, installationID)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/repos/%s/%s", githubAPIURL, owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf(errCreateRequest, err)
	}

	req.Header.Set("Authorization", authSchemeBearer+token.Token)
	req.Header.Set("Accept", acceptHeader)
	req.Header.Set(headerGitHubAPIVersion, apiVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errSendRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("repository not found or not accessible")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(errUnexpectedStatus, resp.StatusCode, string(body))
	}

	var repository AppRepository
	if err := json.NewDecoder(resp.Body).Decode(&repository); err != nil {
		return nil, fmt.Errorf(errDecodeResponse, err)
	}

	return &repository, nil
}

func (c *AppClient) GetInstallation(ctx context.Context, installationID int64) (*InstallationInfo, error) {
	jwt, err := c.GenerateJWT()
	if err != nil {
		return nil, fmt.Errorf("generate JWT: %w", err)
	}

	url := fmt.Sprintf("%s/app/installations/%d", githubAPIURL, installationID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf(errCreateRequest, err)
	}

	req.Header.Set("Authorization", authSchemeBearer+jwt)
	req.Header.Set("Accept", acceptHeader)
	req.Header.Set(headerGitHubAPIVersion, apiVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errSendRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(errUnexpectedStatus, resp.StatusCode, string(body))
	}

	var info InstallationInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf(errDecodeResponse, err)
	}

	return &info, nil
}

type InstallationInfo struct {
	ID                  int64             `json:"id"`
	Account             Account           `json:"account"`
	RepositorySelection string            `json:"repository_selection"`
	Permissions         map[string]string `json:"permissions"`
	Events              []string          `json:"events"`
	SuspendedAt         *time.Time        `json:"suspended_at"`
	SuspendedBy         *Account          `json:"suspended_by"`
}

type Account struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Type      string `json:"type"`
	AvatarURL string `json:"avatar_url"`
}
