package github

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/domain"
)

const (
	testAppID         = "app-123"
	testAppName       = "test-app"
	testRepoURL       = "https://github.com/owner/repo.git"
	testSecret        = "test-secret"
	testRefMain       = "refs/heads/main"
	testBranchMain    = "main"
	webhookPath       = "/paas-deploy/v1/webhooks/github"
	headerContentType = "Content-Type"
	contentTypeJSON   = "application/json"
	testDeliveryID    = "delivery-123"
)

type mockAppFinder struct {
	app *domain.App
	err error
}

func (m *mockAppFinder) FindByRepoURL(repoURL string) (*domain.App, error) {
	return m.app, m.err
}

type mockDeploymentCreator struct {
	deployment *domain.Deployment
	pending    *domain.Deployment
	createErr  error
	pendingErr error
}

func (m *mockDeploymentCreator) Create(input domain.CreateDeploymentInput) (*domain.Deployment, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.deployment, nil
}

func (m *mockDeploymentCreator) FindPendingByAppID(appID string) (*domain.Deployment, error) {
	if m.pendingErr != nil {
		return nil, m.pendingErr
	}
	return m.pending, nil
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
}

func assertStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status %d, got %d: %s", expected, resp.StatusCode, string(body))
	}
}

func createPushPayload(ref, after, repoURL, branch string) []byte {
	event := PushEvent{
		Ref:   ref,
		After: after,
		Repository: &Repository{
			FullName:      "owner/repo",
			CloneURL:      repoURL,
			DefaultBranch: branch,
		},
		HeadCommit: &Commit{
			Message: "Test commit message",
		},
	}
	data, _ := json.Marshal(event)
	return data
}

func TestWebhookHandlerPushToMainBranch(t *testing.T) {
	app := fiber.New()
	defer app.Shutdown()

	testApp := &domain.App{
		ID:            testAppID,
		Name:          testAppName,
		RepositoryURL: testRepoURL,
		Branch:        testBranchMain,
	}

	testDeployment := &domain.Deployment{
		ID:        "deploy-456",
		AppID:     testAppID,
		CommitSHA: "abc123",
		Status:    domain.DeployStatusPending,
	}

	handler := NewWebhookHandler(
		&mockAppFinder{app: testApp},
		&mockDeploymentCreator{deployment: testDeployment, pendingErr: domain.ErrNotFound},
		testSecret,
		newTestLogger(),
	)

	handler.Register(app)

	payload := createPushPayload(testRefMain, "abc123def456", testRepoURL, testBranchMain)
	signature := GenerateSignature(payload, testSecret)

	req := httptest.NewRequest(http.MethodPost, webhookPath, bytes.NewReader(payload))
	req.Header.Set(headerContentType, contentTypeJSON)
	req.Header.Set(HeaderGitHubEvent, EventPush)
	req.Header.Set(HeaderGitHubSignature, signature)
	req.Header.Set(HeaderGitHubDelivery, testDeliveryID)

	resp, err := app.Test(req)
	assertNoError(t, err)
	assertStatus(t, resp, fiber.StatusAccepted)
}

func TestWebhookHandlerPushToDifferentBranch(t *testing.T) {
	app := fiber.New()
	defer app.Shutdown()

	testApp := &domain.App{
		ID:            testAppID,
		Name:          testAppName,
		RepositoryURL: testRepoURL,
		Branch:        testBranchMain,
	}

	handler := NewWebhookHandler(
		&mockAppFinder{app: testApp},
		&mockDeploymentCreator{pendingErr: domain.ErrNotFound},
		testSecret,
		newTestLogger(),
	)

	handler.Register(app)

	payload := createPushPayload("refs/heads/develop", "abc123def456", testRepoURL, testBranchMain)
	signature := GenerateSignature(payload, testSecret)

	req := httptest.NewRequest(http.MethodPost, webhookPath, bytes.NewReader(payload))
	req.Header.Set(headerContentType, contentTypeJSON)
	req.Header.Set(HeaderGitHubEvent, EventPush)
	req.Header.Set(HeaderGitHubSignature, signature)
	req.Header.Set(HeaderGitHubDelivery, testDeliveryID)

	resp, err := app.Test(req)
	assertNoError(t, err)
	assertStatus(t, resp, fiber.StatusOK)
}

func TestWebhookHandlerInvalidSignature(t *testing.T) {
	app := fiber.New()
	defer app.Shutdown()

	handler := NewWebhookHandler(
		&mockAppFinder{},
		&mockDeploymentCreator{},
		testSecret,
		newTestLogger(),
	)

	handler.Register(app)

	payload := createPushPayload(testRefMain, "abc123def456", testRepoURL, testBranchMain)
	invalidSignature := "sha256=0000000000000000000000000000000000000000000000000000000000000000"

	req := httptest.NewRequest(http.MethodPost, webhookPath, bytes.NewReader(payload))
	req.Header.Set(headerContentType, contentTypeJSON)
	req.Header.Set(HeaderGitHubEvent, EventPush)
	req.Header.Set(HeaderGitHubSignature, invalidSignature)
	req.Header.Set(HeaderGitHubDelivery, testDeliveryID)

	resp, err := app.Test(req)
	assertNoError(t, err)
	assertStatus(t, resp, fiber.StatusUnauthorized)
}

func TestWebhookHandlerAppNotFound(t *testing.T) {
	app := fiber.New()
	defer app.Shutdown()

	handler := NewWebhookHandler(
		&mockAppFinder{err: domain.ErrNotFound},
		&mockDeploymentCreator{},
		testSecret,
		newTestLogger(),
	)

	handler.Register(app)

	payload := createPushPayload(testRefMain, "abc123def456", testRepoURL, testBranchMain)
	signature := GenerateSignature(payload, testSecret)

	req := httptest.NewRequest(http.MethodPost, webhookPath, bytes.NewReader(payload))
	req.Header.Set(headerContentType, contentTypeJSON)
	req.Header.Set(HeaderGitHubEvent, EventPush)
	req.Header.Set(HeaderGitHubSignature, signature)
	req.Header.Set(HeaderGitHubDelivery, testDeliveryID)

	resp, err := app.Test(req)
	assertNoError(t, err)
	assertStatus(t, resp, fiber.StatusOK)
}

func TestWebhookHandlerPingEvent(t *testing.T) {
	app := fiber.New()
	defer app.Shutdown()

	handler := NewWebhookHandler(
		&mockAppFinder{},
		&mockDeploymentCreator{},
		testSecret,
		newTestLogger(),
	)

	handler.Register(app)

	payload := []byte(`{"zen": "test"}`)
	signature := GenerateSignature(payload, testSecret)

	req := httptest.NewRequest(http.MethodPost, webhookPath, bytes.NewReader(payload))
	req.Header.Set(headerContentType, contentTypeJSON)
	req.Header.Set(HeaderGitHubEvent, EventPing)
	req.Header.Set(HeaderGitHubSignature, signature)
	req.Header.Set(HeaderGitHubDelivery, testDeliveryID)

	resp, err := app.Test(req)
	assertNoError(t, err)
	assertStatus(t, resp, fiber.StatusOK)
}

func TestWebhookHandlerUnsupportedEvent(t *testing.T) {
	app := fiber.New()
	defer app.Shutdown()

	handler := NewWebhookHandler(
		&mockAppFinder{},
		&mockDeploymentCreator{},
		testSecret,
		newTestLogger(),
	)

	handler.Register(app)

	payload := []byte(`{"action": "opened"}`)
	signature := GenerateSignature(payload, testSecret)

	req := httptest.NewRequest(http.MethodPost, webhookPath, bytes.NewReader(payload))
	req.Header.Set(headerContentType, contentTypeJSON)
	req.Header.Set(HeaderGitHubEvent, "pull_request")
	req.Header.Set(HeaderGitHubSignature, signature)
	req.Header.Set(HeaderGitHubDelivery, testDeliveryID)

	resp, err := app.Test(req)
	assertNoError(t, err)
	assertStatus(t, resp, fiber.StatusOK)
}

func TestWebhookHandlerDeploymentAlreadyPending(t *testing.T) {
	app := fiber.New()
	defer app.Shutdown()

	testApp := &domain.App{
		ID:            testAppID,
		Name:          testAppName,
		RepositoryURL: testRepoURL,
		Branch:        testBranchMain,
	}

	pendingDeployment := &domain.Deployment{
		ID:     "deploy-existing",
		AppID:  testAppID,
		Status: domain.DeployStatusPending,
	}

	handler := NewWebhookHandler(
		&mockAppFinder{app: testApp},
		&mockDeploymentCreator{pending: pendingDeployment},
		testSecret,
		newTestLogger(),
	)

	handler.Register(app)

	payload := createPushPayload(testRefMain, "abc123def456", testRepoURL, testBranchMain)
	signature := GenerateSignature(payload, testSecret)

	req := httptest.NewRequest(http.MethodPost, webhookPath, bytes.NewReader(payload))
	req.Header.Set(headerContentType, contentTypeJSON)
	req.Header.Set(HeaderGitHubEvent, EventPush)
	req.Header.Set(HeaderGitHubSignature, signature)
	req.Header.Set(HeaderGitHubDelivery, testDeliveryID)

	resp, err := app.Test(req)
	assertNoError(t, err)
	assertStatus(t, resp, fiber.StatusOK)
}

func TestExtractBranch(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want string
	}{
		{
			name: "main branch",
			ref:  testRefMain,
			want: testBranchMain,
		},
		{
			name: "feature branch",
			ref:  "refs/heads/feature/new-feature",
			want: "feature/new-feature",
		},
		{
			name: "tag ref",
			ref:  "refs/tags/v1.0.0",
			want: "",
		},
		{
			name: "empty ref",
			ref:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBranch(tt.ref)
			if got != tt.want {
				t.Errorf("extractBranch(%q) = %q, want %q", tt.ref, got, tt.want)
			}
		})
	}
}
