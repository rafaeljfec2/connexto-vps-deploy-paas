package ghclient

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
	testRepoFullName  = "owner/repo"
	testDeployID      = "deploy-456"
	testFileAPIMain   = "apps/api/main.go"
	testFileREADME    = "README.md"
	testFileHandler   = "apps/api/handler.go"
	testWorkdirAPI    = "apps/api"
	testAppIDRoot     = "app-root"
	testAppNameRoot   = "root-app"
	testAppIDAPI      = "app-api"
	testAppNameAPI    = "api-app"
)

type mockAppFinder struct {
	app  *domain.App
	apps []domain.App
	err  error
}

func (m *mockAppFinder) FindByRepoURL(repoURL string) (*domain.App, error) {
	return m.app, m.err
}

func (m *mockAppFinder) FindAllByRepoURL(repoURL string) ([]domain.App, error) {
	if m.err != nil && m.err != domain.ErrNotFound {
		return nil, m.err
	}
	if len(m.apps) > 0 {
		return m.apps, nil
	}
	if m.app != nil {
		return []domain.App{*m.app}, nil
	}
	return nil, nil
}

type mockDeploymentCreator struct {
	deployment *domain.Deployment
	createErr  error
}

func (m *mockDeploymentCreator) Create(input domain.CreateDeploymentInput) (*domain.Deployment, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.deployment, nil
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
	return createPushPayloadWithMessage(ref, after, repoURL, branch, "Test commit message")
}

func createPushPayloadWithMessage(ref, after, repoURL, branch, commitMessage string) []byte {
	event := PushEvent{
		Ref:   ref,
		After: after,
		Repository: &Repository{
			FullName:      testRepoFullName,
			CloneURL:      repoURL,
			DefaultBranch: branch,
		},
		HeadCommit: &Commit{
			Message: commitMessage,
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
		ID:        testDeployID,
		AppID:     testAppID,
		CommitSHA: "abc123",
		Status:    domain.DeployStatusPending,
	}

	handler := NewWebhookHandler(
		&mockAppFinder{app: testApp},
		&mockDeploymentCreator{deployment: testDeployment},
		nil,
		nil,
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
		&mockDeploymentCreator{},
		nil,
		nil,
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
		nil,
		nil,
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
		nil,
		nil,
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
		nil,
		nil,
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
		nil,
		nil,
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

func TestCommitMessageSkipsDeploy(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"chore: bump version [skip ci]", true},
		{"[SKIP CI] release", true},
		{"  [Skip Ci]  ", true},
		{"fix: something", false},
		{"[skipci]", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			got := commitMessageSkipsDeploy(tt.msg)
			if got != tt.want {
				t.Errorf("commitMessageSkipsDeploy(%q) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

func TestWebhookHandlerPushWithSkipCi(t *testing.T) {
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
		&mockDeploymentCreator{},
		nil,
		nil,
		testSecret,
		newTestLogger(),
	)

	handler.Register(app)

	payload := createPushPayloadWithMessage(testRefMain, "abc123", testRepoURL, testBranchMain, "chore(api): bump version to 1.0.20 [skip ci]")
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

func TestWebhookHandlerDeploymentAlreadyPending(t *testing.T) {
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
		&mockDeploymentCreator{createErr: domain.ErrDeploymentAlreadyActive},
		nil,
		nil,
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

func TestExtractChangedFiles(t *testing.T) {
	commits := []Commit{
		{Added: []string{testFileAPIMain}, Modified: []string{testFileREADME}, Removed: []string{}},
		{Added: []string{}, Modified: []string{testFileREADME, testFileHandler}, Removed: []string{"old.txt"}},
	}

	files := extractChangedFiles(commits)

	expected := map[string]bool{
		testFileAPIMain: true,
		testFileREADME:  true,
		testFileHandler: true,
		"old.txt":       true,
	}

	if len(files) != len(expected) {
		t.Fatalf("expected %d files, got %d: %v", len(expected), len(files), files)
	}

	for _, f := range files {
		if !expected[f] {
			t.Errorf("unexpected file in result: %q", f)
		}
	}
}

func TestShouldDeployApp(t *testing.T) {
	otherWorkdirs := []string{testWorkdirAPI}

	tests := []struct {
		name         string
		workdir      string
		changedFiles []string
		want         bool
	}{
		{
			name:         "root app - file outside other workdirs",
			workdir:      ".",
			changedFiles: []string{testFileREADME, "package.json"},
			want:         true,
		},
		{
			name:         "root app - all files inside other workdir",
			workdir:      ".",
			changedFiles: []string{testFileAPIMain, testFileHandler},
			want:         false,
		},
		{
			name:         "root app - mixed files",
			workdir:      ".",
			changedFiles: []string{testFileAPIMain, testFileREADME},
			want:         true,
		},
		{
			name:         "subdirectory app - matching file",
			workdir:      testWorkdirAPI,
			changedFiles: []string{testFileAPIMain},
			want:         true,
		},
		{
			name:         "subdirectory app - no matching file",
			workdir:      testWorkdirAPI,
			changedFiles: []string{testFileREADME, "package.json"},
			want:         false,
		},
		{
			name:         "subdirectory app with ./ prefix",
			workdir:      "./" + testWorkdirAPI,
			changedFiles: []string{testFileHandler},
			want:         true,
		},
		{
			name:         "empty changed files",
			workdir:      testWorkdirAPI,
			changedFiles: []string{},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &domain.App{Workdir: tt.workdir}
			got := shouldDeployApp(app, tt.changedFiles, otherWorkdirs)
			if got != tt.want {
				t.Errorf("shouldDeployApp(workdir=%q, files=%v) = %v, want %v", tt.workdir, tt.changedFiles, got, tt.want)
			}
		})
	}
}

func TestNormalizeWorkdir(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{".", ""},
		{"", ""},
		{"./" + testWorkdirAPI, testWorkdirAPI},
		{testWorkdirAPI, testWorkdirAPI},
		{testWorkdirAPI + "/", testWorkdirAPI},
		{"/" + testWorkdirAPI + "/", testWorkdirAPI},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeWorkdir(tt.input)
			if got != tt.want {
				t.Errorf("normalizeWorkdir(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func newTestMonorepoApps() (domain.App, domain.App) {
	return domain.App{
			ID:            testAppIDRoot,
			Name:          testAppNameRoot,
			RepositoryURL: testRepoURL,
			Branch:        testBranchMain,
			Workdir:       ".",
		}, domain.App{
			ID:            testAppIDAPI,
			Name:          testAppNameAPI,
			RepositoryURL: testRepoURL,
			Branch:        testBranchMain,
			Workdir:       testWorkdirAPI,
		}
}

func newTestMonorepoHandler(apps []domain.App) *WebhookHandler {
	return NewWebhookHandler(
		&mockAppFinder{apps: apps},
		&mockDeploymentCreator{
			deployment: &domain.Deployment{
				ID:        testDeployID,
				AppID:     apps[0].ID,
				CommitSHA: "abc123",
				Status:    domain.DeployStatusPending,
			},
		},
		nil,
		nil,
		testSecret,
		newTestLogger(),
	)
}

func sendMonorepoPush(t *testing.T, fiberApp *fiber.App, commits []Commit, headMsg string) *http.Response {
	t.Helper()
	event := PushEvent{
		Ref:   testRefMain,
		After: "abc123def456",
		Repository: &Repository{
			FullName: testRepoFullName,
			CloneURL: testRepoURL,
		},
		Commits:    commits,
		HeadCommit: &Commit{Message: headMsg},
	}
	payload, _ := json.Marshal(event)
	signature := GenerateSignature(payload, testSecret)

	req := httptest.NewRequest(http.MethodPost, webhookPath, bytes.NewReader(payload))
	req.Header.Set(headerContentType, contentTypeJSON)
	req.Header.Set(HeaderGitHubEvent, EventPush)
	req.Header.Set(HeaderGitHubSignature, signature)
	req.Header.Set(HeaderGitHubDelivery, testDeliveryID)

	resp, err := fiberApp.Test(req)
	assertNoError(t, err)
	return resp
}

func TestMonorepoDeployBothApps(t *testing.T) {
	app := fiber.New()
	defer app.Shutdown()

	rootApp, apiApp := newTestMonorepoApps()
	handler := newTestMonorepoHandler([]domain.App{rootApp, apiApp})
	handler.Register(app)

	commits := []Commit{{Added: []string{testFileAPIMain}, Modified: []string{testFileREADME}}}
	resp := sendMonorepoPush(t, app, commits, "update both")
	assertStatus(t, resp, fiber.StatusAccepted)
}

func TestMonorepoDeployOnlySubdir(t *testing.T) {
	app := fiber.New()
	defer app.Shutdown()

	rootApp, apiApp := newTestMonorepoApps()
	handler := newTestMonorepoHandler([]domain.App{rootApp, apiApp})
	handler.Register(app)

	commits := []Commit{{Modified: []string{testFileAPIMain, testFileHandler}}}
	resp := sendMonorepoPush(t, app, commits, "update api only")
	assertStatus(t, resp, fiber.StatusAccepted)
}

func TestMonorepoForcePushNoCommits(t *testing.T) {
	app := fiber.New()
	defer app.Shutdown()

	rootApp, apiApp := newTestMonorepoApps()
	handler := newTestMonorepoHandler([]domain.App{rootApp, apiApp})
	handler.Register(app)

	event := PushEvent{
		Ref:     testRefMain,
		After:   "abc123def456",
		Created: true,
		Forced:  true,
		Repository: &Repository{
			FullName: testRepoFullName,
			CloneURL: testRepoURL,
		},
		Commits:    []Commit{},
		HeadCommit: &Commit{Message: "force push"},
	}
	payload, _ := json.Marshal(event)
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
