package github

import (
	"encoding/json"
	"time"
)

type PushEvent struct {
	Ref        string      `json:"ref"`
	Before     string      `json:"before"`
	After      string      `json:"after"`
	Created    bool        `json:"created"`
	Deleted    bool        `json:"deleted"`
	Forced     bool        `json:"forced"`
	Compare    string      `json:"compare"`
	Commits    []Commit    `json:"commits"`
	HeadCommit *Commit     `json:"head_commit"`
	Repository *Repository `json:"repository"`
	Pusher     *Pusher     `json:"pusher"`
	Sender     *User       `json:"sender"`
}

type Repository struct {
	ID            int64           `json:"id"`
	NodeID        string          `json:"node_id"`
	Name          string          `json:"name"`
	FullName      string          `json:"full_name"`
	Private       bool            `json:"private"`
	Owner         *User           `json:"owner"`
	HTMLURL       string          `json:"html_url"`
	Description   string          `json:"description"`
	Fork          bool            `json:"fork"`
	URL           string          `json:"url"`
	CloneURL      string          `json:"clone_url"`
	GitURL        string          `json:"git_url"`
	SSHURL        string          `json:"ssh_url"`
	DefaultBranch string          `json:"default_branch"`
	CreatedAt     FlexibleTime    `json:"created_at"`
	UpdatedAt     FlexibleTime    `json:"updated_at"`
	PushedAt      FlexibleTime    `json:"pushed_at"`
}

type Commit struct {
	ID        string       `json:"id"`
	TreeID    string       `json:"tree_id"`
	Distinct  bool         `json:"distinct"`
	Message   string       `json:"message"`
	Timestamp string       `json:"timestamp"`
	URL       string       `json:"url"`
	Author    *CommitUser  `json:"author"`
	Committer *CommitUser  `json:"committer"`
	Added     []string     `json:"added"`
	Removed   []string     `json:"removed"`
	Modified  []string     `json:"modified"`
}

type FlexibleTime struct {
	time.Time
}

func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
	var timestamp int64
	if err := json.Unmarshal(data, &timestamp); err == nil {
		ft.Time = time.Unix(timestamp, 0)
		return nil
	}

	var timeStr string
	if err := json.Unmarshal(data, &timeStr); err == nil {
		parsed, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			parsed, err = time.Parse("2006-01-02T15:04:05Z", timeStr)
		}
		if err != nil {
			ft.Time = time.Time{}
			return nil
		}
		ft.Time = parsed
		return nil
	}

	ft.Time = time.Time{}
	return nil
}

type CommitUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type Pusher struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type User struct {
	Login     string `json:"login"`
	ID        int64  `json:"id"`
	NodeID    string `json:"node_id"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Type      string `json:"type"`
}

type CreateWebhookRequest struct {
	Name   string        `json:"name"`
	Active bool          `json:"active"`
	Events []string      `json:"events"`
	Config WebhookConfig `json:"config"`
}
