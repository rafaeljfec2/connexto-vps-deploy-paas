package handler

const (
	APIPrefix = "/paas-deploy/v1"

	MsgNotAuthenticated     = "not authenticated"
	MsgAppNotFound          = "app not found"
	MsgChannelNotFound      = "channel not found"
	MsgRuleNotFound         = "rule not found"
	MsgEnvVarNotFound       = "environment variable not found"
	MsgInvalidRequestBody   = "invalid request body"
	MsgKeyRequired          = "key is required"
	MsgAtLeastOneVariable   = "at least one variable is required"
	MsgAllVarsMustHaveKey   = "all variables must have a key"
	MsgEnvVarDeleted        = "environment variable deleted"
	MsgNoGitHubInstallation = "no GitHub App installation found"
	MsgOwnerRepoRequired    = "owner and repo are required"
	MsgInstallGitHubApp     = "Please install the GitHub App to access your repositories"
	MsgServerNotFound       = "Server not found"
	MsgFailedPruneImages    = "Failed to prune images"

	DateTimeFormatISO8601 = "2006-01-02T15:04:05Z"
)
