# GitHub Integration Architecture

This document describes the two-phase approach for GitHub webhook integration, designed to be **completely decoupled** from the deployment process.

## Architecture Principles

1. **Separation of Concerns**: Webhook management is independent of deployment
2. **Interface-Based Design**: Use abstractions to allow implementation swapping
3. **Optional Integration**: Deploy process works with or without automatic webhook setup
4. **No Breaking Changes**: Phase 2 implementation should not require changes to Phase 1 consumers

## System Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         PaaS Deploy System                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────┐    ┌──────────────────┐    ┌──────────────────┐  │
│  │              │    │                  │    │                  │  │
│  │  App Service │───▶│ Webhook Manager  │───▶│ GitHub Provider  │  │
│  │              │    │   (Interface)    │    │  (Interface)     │  │
│  └──────────────┘    └──────────────────┘    └──────────────────┘  │
│         │                                            │              │
│         │                                            ▼              │
│         │                                   ┌──────────────────┐   │
│         │                                   │  Phase 1: PAT    │   │
│         │                                   │  Phase 2: App    │   │
│         │                                   └──────────────────┘   │
│         │                                                          │
│         ▼                                                          │
│  ┌──────────────┐    ┌──────────────────┐                         │
│  │   Deploy     │◀───│ Webhook Handler  │◀── GitHub Push Event    │
│  │   Engine     │    │  (Receives)      │                         │
│  └──────────────┘    └──────────────────┘                         │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## Key Interfaces

### WebhookManager Interface

```go
// WebhookManager handles webhook lifecycle independent of provider
type WebhookManager interface {
    // Setup creates a webhook for the given repository
    // Returns webhook ID for future reference
    Setup(ctx context.Context, input WebhookSetupInput) (*WebhookResult, error)

    // Remove deletes a webhook from the repository
    Remove(ctx context.Context, input WebhookRemoveInput) error

    // Verify checks if webhook exists and is properly configured
    Verify(ctx context.Context, input WebhookVerifyInput) (*WebhookStatus, error)
}

type WebhookSetupInput struct {
    RepositoryURL string
    TargetURL     string
    Secret        string
    Events        []string
}

type WebhookResult struct {
    ID         string
    Provider   string // "github", "gitlab", etc.
    WebhookURL string
    Active     bool
}

type WebhookRemoveInput struct {
    RepositoryURL string
    WebhookID     string
}

type WebhookVerifyInput struct {
    RepositoryURL string
    WebhookID     string
}

type WebhookStatus struct {
    Exists     bool
    Active     bool
    LastPingAt *time.Time
    Error      string
}
```

### GitHubProvider Interface

```go
// GitHubProvider abstracts GitHub API operations
// Implementations: PATProvider (Phase 1), AppProvider (Phase 2)
type GitHubProvider interface {
    // CreateWebhook creates a webhook on the repository
    CreateWebhook(ctx context.Context, owner, repo string, config WebhookConfig) (*Webhook, error)

    // DeleteWebhook removes a webhook from the repository
    DeleteWebhook(ctx context.Context, owner, repo string, webhookID int64) error

    // GetWebhook retrieves webhook details
    GetWebhook(ctx context.Context, owner, repo string, webhookID int64) (*Webhook, error)

    // ListWebhooks lists all webhooks for a repository
    ListWebhooks(ctx context.Context, owner, repo string) ([]Webhook, error)
}
```

## Decoupling from Deploy Process

The deploy process must remain completely independent:

```
┌─────────────────────────────────────────────────────────────────┐
│                     DEPLOY PROCESS (Independent)                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   Webhook Handler                    Deploy Engine              │
│   (Receives Events)                  (Executes Deploy)          │
│         │                                  ▲                    │
│         │                                  │                    │
│         └──────────────────────────────────┘                    │
│                    Creates Deployment                           │
│                    (via DeploymentRepository)                   │
│                                                                 │
│   NO DEPENDENCY ON:                                             │
│   - WebhookManager                                              │
│   - GitHubProvider                                              │
│   - PAT or GitHub App tokens                                    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### What Deploy Process Needs

1. **Webhook Handler**: Receives POST from GitHub, validates signature, creates deployment
2. **Webhook Secret**: For signature validation (environment variable)

### What Deploy Process Does NOT Need

1. GitHub PAT
2. GitHub App credentials
3. WebhookManager
4. Any knowledge of how webhooks were created

---

## Phase 1: Personal Access Token (PAT)

### Overview

Use a single PAT for personal/team repositories where the token owner has admin access.

### Scope

- Personal repositories
- Organization repositories where user is admin
- Team internal projects

### Implementation

```go
// PATProvider implements GitHubProvider using Personal Access Token
type PATProvider struct {
    client *http.Client
    token  string
}

func NewPATProvider(token string) *PATProvider {
    return &PATProvider{
        client: &http.Client{Timeout: 30 * time.Second},
        token:  token,
    }
}

func (p *PATProvider) CreateWebhook(ctx context.Context, owner, repo string, config WebhookConfig) (*Webhook, error) {
    // Uses existing github.Client implementation
    // Authorization: Bearer <PAT>
}
```

### Configuration

```bash
# .env
GITHUB_PAT=ghp_xxxxxxxxxxxxxxxxxxxx
GITHUB_WEBHOOK_SECRET=your-webhook-secret
GITHUB_WEBHOOK_URL=https://deploy.yourdomain.com/webhooks/github
```

### Required PAT Permissions

| Permission        | Scope        | Purpose                     |
| ----------------- | ------------ | --------------------------- |
| `repo`            | Full control | Access private repositories |
| `admin:repo_hook` | Full control | Create/delete webhooks      |

### Limitations

- Only works for repositories where PAT owner has admin access
- Single token for all operations
- Not suitable for multi-tenant with external users

### File Structure

```
internal/
├── github/
│   ├── provider.go          # GitHubProvider interface
│   ├── pat_provider.go      # Phase 1: PAT implementation
│   ├── client.go            # HTTP client (existing)
│   └── ...
├── webhook/
│   ├── manager.go           # WebhookManager interface
│   ├── github_manager.go    # GitHub implementation
│   └── ...
```

---

## Phase 2: GitHub App

### Overview

GitHub App allows external users to install the app on their repositories, granting specific permissions without sharing tokens.

### Scope

- Multi-tenant platform
- External users with their own repositories
- Production SaaS deployment

### How GitHub Apps Work

```
┌─────────────────────────────────────────────────────────────────┐
│                      GitHub App Flow                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. User clicks "Install App"                                   │
│           │                                                     │
│           ▼                                                     │
│  2. GitHub shows permission dialog                              │
│     - Repository webhooks (read/write)                          │
│     - Repository contents (read) [optional]                     │
│           │                                                     │
│           ▼                                                     │
│  3. User selects repositories                                   │
│           │                                                     │
│           ▼                                                     │
│  4. GitHub sends installation webhook to your app               │
│           │                                                     │
│           ▼                                                     │
│  5. Your app stores installation_id                             │
│           │                                                     │
│           ▼                                                     │
│  6. Your app generates installation token (expires 1h)          │
│           │                                                     │
│           ▼                                                     │
│  7. Use token to create webhooks on user's repos                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Implementation

```go
// AppProvider implements GitHubProvider using GitHub App
type AppProvider struct {
    appID          int64
    privateKey     []byte
    installationID int64
    client         *http.Client
}

func NewAppProvider(appID int64, privateKey []byte) *AppProvider {
    return &AppProvider{
        appID:      appID,
        privateKey: privateKey,
        client:     &http.Client{Timeout: 30 * time.Second},
    }
}

// GetInstallationToken generates a short-lived token for API calls
func (p *AppProvider) GetInstallationToken(ctx context.Context, installationID int64) (string, error) {
    // 1. Create JWT signed with private key
    // 2. Call POST /app/installations/{installation_id}/access_tokens
    // 3. Return token (valid for 1 hour)
}

func (p *AppProvider) CreateWebhook(ctx context.Context, owner, repo string, config WebhookConfig) (*Webhook, error) {
    // 1. Get installation ID for this repo
    // 2. Generate installation token
    // 3. Create webhook using token
}
```

### Configuration

```bash
# .env
GITHUB_APP_ID=123456
GITHUB_APP_PRIVATE_KEY_PATH=/path/to/private-key.pem
# Or base64 encoded
GITHUB_APP_PRIVATE_KEY_BASE64=LS0tLS1CRUdJTi...

GITHUB_WEBHOOK_SECRET=your-webhook-secret
GITHUB_WEBHOOK_URL=https://deploy.yourdomain.com/webhooks/github
```

### Required App Permissions

| Permission          | Access       | Purpose                |
| ------------------- | ------------ | ---------------------- |
| Repository webhooks | Read & Write | Create/manage webhooks |
| Metadata            | Read         | Access repository info |

### Database Schema Addition

```sql
-- Store GitHub App installations
CREATE TABLE github_installations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    installation_id BIGINT NOT NULL UNIQUE,
    account_type VARCHAR(50) NOT NULL, -- 'User' or 'Organization'
    account_login VARCHAR(255) NOT NULL,
    account_id BIGINT NOT NULL,
    repositories JSONB, -- List of repos if not all
    permissions JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Link apps to installations
ALTER TABLE apps ADD COLUMN github_installation_id UUID REFERENCES github_installations(id);
```

### File Structure

```
internal/
├── github/
│   ├── provider.go          # GitHubProvider interface
│   ├── pat_provider.go      # Phase 1: PAT implementation
│   ├── app_provider.go      # Phase 2: GitHub App implementation
│   ├── jwt.go               # JWT generation for App auth
│   └── ...
├── webhook/
│   ├── manager.go           # WebhookManager interface
│   ├── github_manager.go    # GitHub implementation (uses provider)
│   └── ...
```

---

## Migration Path: Phase 1 → Phase 2

### Strategy: Feature Flag

```go
type WebhookManagerFactory struct {
    config *config.Config
}

func (f *WebhookManagerFactory) Create() WebhookManager {
    if f.config.GitHub.AppID != 0 {
        // Phase 2: GitHub App
        provider := github.NewAppProvider(
            f.config.GitHub.AppID,
            f.config.GitHub.PrivateKey,
        )
        return webhook.NewGitHubManager(provider, f.config)
    }

    if f.config.GitHub.PAT != "" {
        // Phase 1: PAT
        provider := github.NewPATProvider(f.config.GitHub.PAT)
        return webhook.NewGitHubManager(provider, f.config)
    }

    // No automatic webhook management
    return webhook.NewNoOpManager()
}
```

### No Breaking Changes

1. **Phase 1 users**: Continue using PAT, no changes required
2. **Phase 2 users**: Configure GitHub App, system auto-detects
3. **No webhook management**: System works, users configure webhooks manually

### Coexistence

Both phases can coexist:

- Personal repos: Use PAT (faster, simpler)
- External users: Use GitHub App (secure, proper)

---

## Integration with App Service

### App Creation Flow

```go
func (s *AppService) CreateApp(ctx context.Context, input CreateAppInput) (*App, error) {
    // 1. Validate input
    if err := s.validateCreateInput(input); err != nil {
        return nil, err
    }

    // 2. Check for duplicates
    existing, err := s.appRepo.FindByName(input.Name)
    if err != nil && !errors.Is(err, domain.ErrNotFound) {
        return nil, err
    }
    if existing != nil {
        return nil, domain.ErrAlreadyExists
    }

    // 3. Create app in database
    app, err := s.appRepo.Create(input)
    if err != nil {
        return nil, err
    }

    // 4. Setup webhook (async, non-blocking, optional)
    if s.webhookManager != nil {
        go s.setupWebhookAsync(ctx, app)
    }

    return app, nil
}

func (s *AppService) setupWebhookAsync(ctx context.Context, app *App) {
    result, err := s.webhookManager.Setup(ctx, WebhookSetupInput{
        RepositoryURL: app.RepositoryURL,
        TargetURL:     s.config.GitHub.WebhookURL,
        Secret:        s.config.GitHub.WebhookSecret,
        Events:        []string{"push"},
    })

    if err != nil {
        s.logger.Error("failed to setup webhook",
            "app_id", app.ID,
            "error", err,
        )
        // App is created, webhook setup failed
        // User can retry or setup manually
        return
    }

    // Update app with webhook info
    s.appRepo.Update(app.ID, UpdateAppInput{
        WebhookID:       &result.ID,
        WebhookProvider: &result.Provider,
    })
}
```

### Key Points

1. **Webhook setup is async**: App creation succeeds even if webhook fails
2. **Webhook is optional**: System works without automatic webhook
3. **Retry mechanism**: User can trigger webhook setup again
4. **Manual fallback**: User can always configure webhook manually

---

## API Endpoints

### Webhook Management (Optional)

```
POST   /api/apps/{id}/webhook          # Setup webhook for app
DELETE /api/apps/{id}/webhook          # Remove webhook from app
GET    /api/apps/{id}/webhook/status   # Check webhook status
POST   /api/apps/{id}/webhook/verify   # Verify webhook is working
```

### Response Examples

```json
// POST /api/apps/{id}/webhook
{
    "status": "success",
    "data": {
        "webhook_id": "12345678",
        "provider": "github",
        "url": "https://deploy.example.com/webhooks/github",
        "active": true
    }
}

// GET /api/apps/{id}/webhook/status
{
    "status": "success",
    "data": {
        "exists": true,
        "active": true,
        "last_ping_at": "2024-01-15T10:30:00Z",
        "provider": "github"
    }
}
```

---

## Error Handling

### Webhook Setup Errors

| Error            | Cause                                   | User Action                  |
| ---------------- | --------------------------------------- | ---------------------------- |
| `unauthorized`   | Invalid PAT or insufficient permissions | Check PAT permissions        |
| `not_found`      | Repository doesn't exist or no access   | Verify repository URL        |
| `already_exists` | Webhook already configured              | Use existing or delete first |
| `rate_limited`   | GitHub API rate limit                   | Wait and retry               |

### Graceful Degradation

```go
func (s *AppService) CreateApp(ctx context.Context, input CreateAppInput) (*App, error) {
    app, err := s.createAppInDB(input)
    if err != nil {
        return nil, err
    }

    // Webhook setup failure should NOT fail app creation
    if s.webhookManager != nil {
        if err := s.setupWebhook(ctx, app); err != nil {
            s.logger.Warn("webhook setup failed, manual configuration required",
                "app_id", app.ID,
                "error", err,
            )
            // Continue - app is created, webhook can be setup later
        }
    }

    return app, nil
}
```

---

## Testing Strategy

### Unit Tests

```go
// Mock provider for testing
type MockGitHubProvider struct {
    CreateWebhookFunc func(ctx context.Context, owner, repo string, config WebhookConfig) (*Webhook, error)
    DeleteWebhookFunc func(ctx context.Context, owner, repo string, webhookID int64) error
}

func TestAppService_CreateApp_WithWebhook(t *testing.T) {
    mockProvider := &MockGitHubProvider{
        CreateWebhookFunc: func(ctx context.Context, owner, repo string, config WebhookConfig) (*Webhook, error) {
            return &Webhook{ID: 123}, nil
        },
    }

    manager := webhook.NewGitHubManager(mockProvider, testConfig)
    service := NewAppService(appRepo, deployRepo, manager, logger)

    app, err := service.CreateApp(ctx, input)

    assert.NoError(t, err)
    assert.NotNil(t, app)
}

func TestAppService_CreateApp_WebhookFailure_StillCreatesApp(t *testing.T) {
    mockProvider := &MockGitHubProvider{
        CreateWebhookFunc: func(ctx context.Context, owner, repo string, config WebhookConfig) (*Webhook, error) {
            return nil, errors.New("GitHub API error")
        },
    }

    manager := webhook.NewGitHubManager(mockProvider, testConfig)
    service := NewAppService(appRepo, deployRepo, manager, logger)

    app, err := service.CreateApp(ctx, input)

    // App should be created even if webhook fails
    assert.NoError(t, err)
    assert.NotNil(t, app)
}
```

### Integration Tests

```go
func TestGitHubPATProvider_CreateWebhook_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    token := os.Getenv("GITHUB_PAT_TEST")
    if token == "" {
        t.Skip("GITHUB_PAT_TEST not set")
    }

    provider := github.NewPATProvider(token)

    webhook, err := provider.CreateWebhook(ctx, "owner", "test-repo", WebhookConfig{
        URL:    "https://example.com/webhook",
        Secret: "test-secret",
    })

    assert.NoError(t, err)
    assert.NotZero(t, webhook.ID)

    // Cleanup
    _ = provider.DeleteWebhook(ctx, "owner", "test-repo", webhook.ID)
}
```

---

## Checklist

### Phase 1 Implementation

- [ ] Create `GitHubProvider` interface
- [ ] Implement `PATProvider`
- [ ] Create `WebhookManager` interface
- [ ] Implement `GitHubWebhookManager`
- [ ] Add `GITHUB_WEBHOOK_URL` to config
- [ ] Add `webhook_id` column to apps table
- [ ] Integrate with `AppService.CreateApp`
- [ ] Integrate with `AppService.DeleteApp`
- [ ] Add webhook management API endpoints
- [ ] Write unit tests
- [ ] Write integration tests
- [ ] Update documentation

### Phase 2 Implementation

- [ ] Implement `AppProvider` (GitHub App)
- [ ] Add JWT generation for App authentication
- [ ] Create `github_installations` table
- [ ] Add installation webhook handler
- [ ] Add OAuth flow for app installation
- [ ] Update `WebhookManagerFactory` for auto-detection
- [ ] Write unit tests
- [ ] Write integration tests
- [ ] Update documentation

---

## References

- [GitHub REST API - Webhooks](https://docs.github.com/en/rest/webhooks)
- [GitHub Apps Documentation](https://docs.github.com/en/apps)
- [Creating a GitHub App](https://docs.github.com/en/apps/creating-github-apps)
- [Authenticating as a GitHub App](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app)
