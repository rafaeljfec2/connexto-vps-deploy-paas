package ghclient

import "time"

const (
	defaultBaseURL       = "https://api.github.com"
	defaultTimeout       = 30 * time.Second
	apiVersion           = "2022-11-28"
	acceptHeader         = "application/vnd.github+json"
	userAgentHeader      = "FlowDeploy/1.0"
	authSchemeBearer     = "Bearer "
	headerGitHubAPIVersion = "X-GitHub-Api-Version"

	errCreateRequest    = "create request: %w"
	errSendRequest      = "send request: %w"
	errUnexpectedStatus = "unexpected status %d: %s"
	errDecodeResponse   = "decode response: %w"
)
