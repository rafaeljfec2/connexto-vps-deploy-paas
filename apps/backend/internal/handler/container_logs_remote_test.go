package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/requestctx"
)

func TestGetAppContainerLogs_RemoteFailureIsNotEmptySuccess(t *testing.T) {
	serverID := "srv-1"
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewContainerHealthHandler(
		logsAppRepo{app: &domain.App{
			ID:       "app-1",
			UserID:   "user-1",
			Name:     "api",
			ServerID: &serverID,
		}},
		logsServerRepo{host: "10.0.0.8"},
		nil,
		nil,
		50052,
		logger,
	)
	h.remoteLogs = stubRemoteLogs{err: errors.New("agent unreachable")}

	app := fiber.New()
	app.Get("/apps/:id/container/logs", func(c *fiber.Ctx) error {
		requestctx.SetUserInContext(c, &domain.User{ID: "user-1"})
		return h.GetContainerLogs(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/apps/app-1/container/logs", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Fatalf("remote log failure returned %d", resp.StatusCode)
	}
	var body struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Success {
		t.Fatal("expected success=false when the agent cannot return app logs")
	}
}

func TestGetContainerLogs_RemoteFailureIsNotEmptySuccess(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewContainerHandler(ContainerHandlerConfig{
		ServerRepo: logsServerRepo{host: "10.0.0.8"},
		AgentPort:  50052,
		Logger:     logger,
	})
	h.remoteLogs = stubRemoteLogs{err: errors.New("agent unreachable")}

	app := fiber.New()
	app.Get("/containers/:id/logs", func(c *fiber.Ctx) error {
		requestctx.SetUserInContext(c, &domain.User{ID: "user-1"})
		return h.GetContainerLogs(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/containers/api/logs?serverId=srv-1", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Fatalf("remote log failure returned %d", resp.StatusCode)
	}

	var body struct {
		Success bool `json:"success"`
		Data    *struct {
			Logs string `json:"logs"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Success {
		t.Fatal("expected success=false when the agent cannot return logs")
	}
	if body.Data != nil && body.Data.Logs == "" && resp.StatusCode == http.StatusOK {
		t.Fatal("empty logs were reported as a successful read")
	}
}

type stubRemoteLogs struct {
	err error
}

func (s stubRemoteLogs) GetContainerLogs(context.Context, string, int, string, int, bool, *time.Time, agentclient.ContainerLogHandler) error {
	return s.err
}

type logsServerRepo struct {
	host string
}

func (r logsServerRepo) Create(domain.CreateServerInput) (*domain.Server, error) {
	return nil, errors.New("not implemented")
}

func (r logsServerRepo) FindByID(id string) (*domain.Server, error) {
	return &domain.Server{ID: id, Host: r.host}, nil
}

func (r logsServerRepo) FindByIDForUser(id string, userID string) (*domain.Server, error) {
	return &domain.Server{ID: id, UserID: userID, Host: r.host}, nil
}

func (r logsServerRepo) FindAll() ([]domain.Server, error) {
	return nil, errors.New("not implemented")
}

func (r logsServerRepo) FindAllByUserID(string) ([]domain.Server, error) {
	return nil, errors.New("not implemented")
}

func (r logsServerRepo) Update(string, domain.UpdateServerInput) (*domain.Server, error) {
	return nil, errors.New("not implemented")
}

func (r logsServerRepo) UpdateHeartbeat(string, string) error {
	return errors.New("not implemented")
}

func (r logsServerRepo) UpdateSSHHostKey(string, string) error {
	return errors.New("not implemented")
}

func (r logsServerRepo) Delete(string) error {
	return errors.New("not implemented")
}

type logsAppRepo struct {
	app *domain.App
}

func (r logsAppRepo) FindAll() ([]domain.App, error) { return nil, domain.ErrNotFound }

func (r logsAppRepo) FindAllByUserID(string) ([]domain.App, error) {
	return nil, domain.ErrNotFound
}

func (r logsAppRepo) FindByID(id string) (*domain.App, error) {
	if r.app != nil && r.app.ID == id {
		return r.app, nil
	}
	return nil, domain.ErrNotFound
}

func (r logsAppRepo) FindByIDAndUserID(id, userID string) (*domain.App, error) {
	app, err := r.FindByID(id)
	if err != nil {
		return nil, err
	}
	if app.UserID != userID {
		return nil, domain.ErrNotFound
	}
	return app, nil
}

func (r logsAppRepo) FindByName(string) (*domain.App, error) { return nil, domain.ErrNotFound }

func (r logsAppRepo) FindByRepoURL(string) (*domain.App, error) { return nil, domain.ErrNotFound }

func (r logsAppRepo) FindAllByRepoURL(string) ([]domain.App, error) {
	return nil, domain.ErrNotFound
}

func (r logsAppRepo) FindByServerID(string) ([]domain.App, error) {
	return nil, domain.ErrNotFound
}

func (r logsAppRepo) Create(domain.CreateAppInput) (*domain.App, error) {
	return nil, domain.ErrNotFound
}

func (r logsAppRepo) Update(string, domain.UpdateAppInput) (*domain.App, error) {
	return nil, domain.ErrNotFound
}

func (r logsAppRepo) Delete(string) error { return domain.ErrNotFound }

func (r logsAppRepo) HardDelete(string) error { return domain.ErrNotFound }

func (r logsAppRepo) UpdateLastDeployedAt(string, time.Time) error { return domain.ErrNotFound }
