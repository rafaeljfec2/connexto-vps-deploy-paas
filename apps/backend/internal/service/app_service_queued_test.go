package service

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/paasdeploy/backend/internal/domain"
)

type stubAppRepo struct {
	app *domain.App
}

func (s *stubAppRepo) FindAll() ([]domain.App, error)              { return nil, nil }
func (s *stubAppRepo) FindAllByUserID(string) ([]domain.App, error) { return nil, nil }
func (s *stubAppRepo) FindByID(string) (*domain.App, error)         { return s.app, nil }
func (s *stubAppRepo) FindByIDAndUserID(string, string) (*domain.App, error) {
	return s.app, nil
}
func (s *stubAppRepo) FindByName(string) (*domain.App, error) { return nil, domain.ErrNotFound }
func (s *stubAppRepo) FindByRepoURL(string) (*domain.App, error) {
	return nil, domain.ErrNotFound
}
func (s *stubAppRepo) FindAllByRepoURL(string) ([]domain.App, error) { return nil, nil }
func (s *stubAppRepo) FindByServerID(string) ([]domain.App, error)   { return nil, nil }
func (s *stubAppRepo) Create(domain.CreateAppInput) (*domain.App, error) {
	return nil, nil
}
func (s *stubAppRepo) Update(string, domain.UpdateAppInput) (*domain.App, error) {
	return nil, nil
}
func (s *stubAppRepo) Delete(string) error     { return nil }
func (s *stubAppRepo) HardDelete(string) error { return nil }
func (s *stubAppRepo) UpdateLastDeployedAt(string, time.Time) error {
	return nil
}

type stubDeploymentRepo struct {
	created *domain.Deployment
}

func (s *stubDeploymentRepo) FindByID(string) (*domain.Deployment, error) {
	return nil, domain.ErrNotFound
}
func (s *stubDeploymentRepo) FindByAppID(string, int) ([]domain.Deployment, error) {
	return nil, nil
}
func (s *stubDeploymentRepo) FindPendingByAppID(string) (*domain.Deployment, error) {
	return nil, domain.ErrNotFound
}
func (s *stubDeploymentRepo) FindLatestByAppID(string) (*domain.Deployment, error) {
	return nil, domain.ErrNotFound
}
func (s *stubDeploymentRepo) FindMostRecentByAppID(string) (*domain.Deployment, error) {
	return nil, domain.ErrNotFound
}
func (s *stubDeploymentRepo) FindMostRecentByAppIDs([]string) (map[string]*domain.Deployment, error) {
	return map[string]*domain.Deployment{}, nil
}
func (s *stubDeploymentRepo) Create(input domain.CreateDeploymentInput) (*domain.Deployment, error) {
	s.created = &domain.Deployment{
		ID:        "deploy-new",
		AppID:     input.AppID,
		CommitSHA: input.CommitSHA,
		Status:    domain.DeployStatusPending,
	}
	return s.created, nil
}
func (s *stubDeploymentRepo) Update(string, domain.UpdateDeploymentInput) (*domain.Deployment, error) {
	return nil, nil
}
func (s *stubDeploymentRepo) AppendLogs(string, string) error    { return nil }
func (s *stubDeploymentRepo) GetNextPending() (*domain.Deployment, error) {
	return nil, domain.ErrNotFound
}
func (s *stubDeploymentRepo) MarkAsRunning(string) error { return nil }
func (s *stubDeploymentRepo) MarkAsSuccess(string, string, string) error {
	return nil
}
func (s *stubDeploymentRepo) MarkAsFailed(string, string) error { return nil }
func (s *stubDeploymentRepo) DeleteByAppID(string) error        { return nil }

type stubQueuedNotifier struct {
	calls [][2]string
}

func (s *stubQueuedNotifier) EmitDeployQueued(deployID, appID string) {
	s.calls = append(s.calls, [2]string{deployID, appID})
}

func TestAppService_TriggerDeploy_EmitsQueued(t *testing.T) {
	appRepo := &stubAppRepo{app: &domain.App{ID: "app-1", Name: "demo"}}
	deployRepo := &stubDeploymentRepo{}
	queued := &stubQueuedNotifier{}

	svc := NewAppService(
		appRepo,
		deployRepo,
		nil,
		nil,
		nil,
		queued,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	deploy, err := svc.TriggerDeploy("app-1", "abc123")
	if err != nil {
		t.Fatalf("TriggerDeploy: %v", err)
	}
	if deploy == nil || deploy.ID != "deploy-new" {
		t.Fatalf("unexpected deploy: %+v", deploy)
	}
	if len(queued.calls) != 1 {
		t.Fatalf("expected EmitDeployQueued once, got %d", len(queued.calls))
	}
	if queued.calls[0][0] != "deploy-new" || queued.calls[0][1] != "app-1" {
		t.Fatalf("unexpected queued args: %+v", queued.calls[0])
	}
}
