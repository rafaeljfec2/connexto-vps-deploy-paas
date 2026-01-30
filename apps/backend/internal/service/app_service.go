package service

import (
	"errors"
	"strings"

	"github.com/paasdeploy/backend/internal/domain"
)

type AppService struct {
	appRepo        domain.AppRepository
	deploymentRepo domain.DeploymentRepository
}

func NewAppService(appRepo domain.AppRepository, deploymentRepo domain.DeploymentRepository) *AppService {
	return &AppService{
		appRepo:        appRepo,
		deploymentRepo: deploymentRepo,
	}
}

func (s *AppService) ListApps() ([]domain.App, error) {
	apps, err := s.appRepo.FindAll()
	if err != nil {
		return nil, err
	}
	if apps == nil {
		apps = []domain.App{}
	}
	return apps, nil
}

func (s *AppService) GetApp(id string) (*domain.App, error) {
	return s.appRepo.FindByID(id)
}

func (s *AppService) CreateApp(input domain.CreateAppInput) (*domain.App, error) {
	if err := s.validateCreateInput(input); err != nil {
		return nil, err
	}

	existing, err := s.appRepo.FindByName(input.Name)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrAlreadyExists
	}

	return s.appRepo.Create(input)
}

func (s *AppService) UpdateApp(id string, input domain.UpdateAppInput) (*domain.App, error) {
	return s.appRepo.Update(id, input)
}

func (s *AppService) DeleteApp(id string) error {
	return s.appRepo.Delete(id)
}

func (s *AppService) ListDeployments(appID string) ([]domain.Deployment, error) {
	_, err := s.appRepo.FindByID(appID)
	if err != nil {
		return nil, err
	}

	deployments, err := s.deploymentRepo.FindByAppID(appID, 50)
	if err != nil {
		return nil, err
	}
	if deployments == nil {
		deployments = []domain.Deployment{}
	}
	return deployments, nil
}

func (s *AppService) TriggerDeploy(appID string, commitSHA string) (*domain.Deployment, error) {
	app, err := s.appRepo.FindByID(appID)
	if err != nil {
		return nil, err
	}

	pending, err := s.deploymentRepo.FindPendingByAppID(appID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if pending != nil {
		return nil, domain.ErrDeployInProgress
	}

	if commitSHA == "" {
		commitSHA = "HEAD"
	}

	input := domain.CreateDeploymentInput{
		AppID:         app.ID,
		CommitSHA:     commitSHA,
		CommitMessage: "Manual deploy triggered",
	}

	return s.deploymentRepo.Create(input)
}

func (s *AppService) TriggerRollback(appID string) (*domain.Deployment, error) {
	_, err := s.appRepo.FindByID(appID)
	if err != nil {
		return nil, err
	}

	latestSuccess, err := s.deploymentRepo.FindLatestByAppID(appID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNoDeployAvailable
		}
		return nil, err
	}

	input := domain.CreateDeploymentInput{
		AppID:         appID,
		CommitSHA:     latestSuccess.CommitSHA,
		CommitMessage: "Rollback to " + latestSuccess.CommitSHA[:7],
	}

	return s.deploymentRepo.Create(input)
}

func (s *AppService) validateCreateInput(input domain.CreateAppInput) error {
	if input.Name == "" {
		return domain.ErrInvalidInput
	}

	if len(input.Name) < 2 || len(input.Name) > 63 {
		return domain.ErrInvalidInput
	}

	if input.RepositoryURL == "" {
		return domain.ErrInvalidInput
	}

	if !strings.HasPrefix(input.RepositoryURL, "https://github.com/") &&
		!strings.HasPrefix(input.RepositoryURL, "git@github.com:") {
		return domain.ErrInvalidInput
	}

	return nil
}
