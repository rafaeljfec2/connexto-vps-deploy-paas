package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	githubAuthorizeURL = "https://github.com/login/oauth/authorize"
	githubTokenURL     = "https://github.com/login/oauth/access_token"
	githubAPIURL       = "https://api.github.com"

	errFmtCreateRequest   = "create request: %w"
	errFmtSendRequest     = "send request: %w"
	errFmtUnexpectedStatus = "unexpected status %d: %s"
	errFmtDecodeResponse  = "decode response: %w"
)

type OAuthClient struct {
	clientID     string
	clientSecret string
	callbackURL  string
	httpClient   *http.Client
}

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
}

type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func NewOAuthClient(cfg OAuthConfig) *OAuthClient {
	return &OAuthClient{
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		callbackURL:  cfg.CallbackURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *OAuthClient) GetAuthorizationURL(state string) string {
	params := url.Values{
		"client_id":    []string{c.clientID},
		"redirect_uri": []string{c.callbackURL},
		"scope":        []string{"user:email"},
		"state":        []string{state},
	}
	return githubAuthorizeURL + "?" + params.Encode()
}

func (c *OAuthClient) ExchangeCodeForToken(ctx context.Context, code string) (*TokenResponse, error) {
	payload := map[string]string{
		"client_id":     c.clientID,
		"client_secret": c.clientSecret,
		"code":          code,
		"redirect_uri":  c.callbackURL,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubTokenURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf(errFmtCreateRequest, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errFmtSendRequest, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(errFmtUnexpectedStatus, resp.StatusCode, string(respBody))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf(errFmtDecodeResponse, err)
	}

	if tokenResp.AccessToken == "" {
		var errResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("oauth error: %s - %s", errResp.Error, errResp.ErrorDescription)
		}
		return nil, fmt.Errorf("no access token in response")
	}

	return &tokenResp, nil
}

func (c *OAuthClient) GetUser(ctx context.Context, accessToken string) (*GitHubUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPIURL+"/user", nil)
	if err != nil {
		return nil, fmt.Errorf(errFmtCreateRequest, err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errFmtSendRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(errFmtUnexpectedStatus, resp.StatusCode, string(body))
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf(errFmtDecodeResponse, err)
	}

	return &user, nil
}

func (c *OAuthClient) GetUserEmails(ctx context.Context, accessToken string) ([]GitHubEmail, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPIURL+"/user/emails", nil)
	if err != nil {
		return nil, fmt.Errorf(errFmtCreateRequest, err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errFmtSendRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(errFmtUnexpectedStatus, resp.StatusCode, string(body))
	}

	var emails []GitHubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return nil, fmt.Errorf(errFmtDecodeResponse, err)
	}

	return emails, nil
}

func (c *OAuthClient) GetPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	emails, err := c.GetUserEmails(ctx, accessToken)
	if err != nil {
		return "", err
	}

	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}

	for _, email := range emails {
		if email.Verified {
			return email.Email, nil
		}
	}

	if len(emails) > 0 {
		return emails[0].Email, nil
	}

	return "", nil
}
