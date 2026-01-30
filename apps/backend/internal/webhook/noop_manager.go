package webhook

import "context"

var _ Manager = (*NoOpManager)(nil)

type NoOpManager struct{}

func NewNoOpManager() *NoOpManager {
	return &NoOpManager{}
}

func (m *NoOpManager) Setup(ctx context.Context, input SetupInput) (*SetupResult, error) {
	return nil, nil
}

func (m *NoOpManager) Remove(ctx context.Context, input RemoveInput) error {
	return nil
}

func (m *NoOpManager) Status(ctx context.Context, repoURL string, webhookID int64) (*Status, error) {
	return &Status{
		Exists: false,
		Error:  "webhook management not configured",
	}, nil
}
