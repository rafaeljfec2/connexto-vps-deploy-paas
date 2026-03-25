package engine

import (
	"context"
	"log/slog"
	"time"

	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/shared/pkg/lock"
)

type Dispatcher struct {
	queue    *Queue
	locker   *lock.Locker
	logger   *slog.Logger
	pollTime time.Duration
}

func NewDispatcher(queue *Queue, locker *lock.Locker, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{
		queue:    queue,
		locker:   locker,
		logger:   logger,
		pollTime: 5 * time.Second,
	}
}

func (d *Dispatcher) Next(ctx context.Context) (*domain.Deployment, *domain.App, error) {
	tx, err := d.queue.BeginTx(ctx)
	if err != nil {
		d.logger.Error("Failed to begin transaction", "error", err)
		return nil, nil, err
	}
	defer tx.Rollback()

	deploy, err := d.queue.GetNextPendingTx(tx)
	if err != nil {
		d.logger.Error("Failed to get next pending deployment", "error", err)
		return nil, nil, err
	}
	if deploy == nil {
		return nil, nil, nil
	}

	if d.locker.IsLocked(deploy.AppID) {
		d.logger.Debug("App is locked, skipping", "appId", deploy.AppID)
		return nil, nil, nil
	}

	if err := d.locker.Acquire(deploy.AppID); err != nil {
		d.logger.Warn("Failed to acquire lock", "appId", deploy.AppID, "error", err)
		return nil, nil, nil
	}

	app, err := d.queue.GetAppByID(deploy.AppID)
	if err != nil {
		d.locker.Release(deploy.AppID)
		d.logger.Error("Failed to get app for deployment", "deployId", deploy.ID, "appId", deploy.AppID, "error", err)
		return nil, nil, err
	}

	if err := d.queue.MarkAsRunningTx(tx, deploy.ID); err != nil {
		d.locker.Release(deploy.AppID)
		d.logger.Error("Failed to mark deployment as running", "deployId", deploy.ID, "error", err)
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		d.locker.Release(deploy.AppID)
		d.logger.Error("Failed to commit dispatch transaction", "deployId", deploy.ID, "error", err)
		return nil, nil, err
	}

	deploy.Status = domain.DeployStatusRunning
	now := time.Now()
	deploy.StartedAt = &now

	d.logger.Info("Dispatched deployment",
		"deployId", deploy.ID,
		"appId", deploy.AppID,
		"appName", app.Name,
		"commitSha", deploy.CommitSHA,
	)

	return deploy, app, nil
}

func (d *Dispatcher) Release(appID string) error {
	return d.locker.Release(appID)
}

func (d *Dispatcher) MarkSuccess(deployID, imageTag, appVersion string) error {
	err := d.queue.MarkAsSuccess(deployID, imageTag, appVersion)
	if err == nil {
		d.logger.Info("Deployment marked as success", "deployId", deployID, "imageTag", imageTag)
	}
	return err
}

func (d *Dispatcher) MarkFailed(deployID, errorMessage string) error {
	err := d.queue.MarkAsFailed(deployID, errorMessage)
	if err == nil {
		d.logger.Info("Deployment marked as failed", "deployId", deployID, "error", errorMessage)
	} else {
		d.logger.Error("Failed to mark deployment as failed", "deployId", deployID, "error", err)
	}
	return err
}

func (d *Dispatcher) AppendLogs(deployID, logs string) error {
	return d.queue.AppendLogs(deployID, logs)
}

func (d *Dispatcher) SetPreviousImageTag(deployID, tag string) error {
	return d.queue.SetPreviousImageTag(deployID, tag)
}

func (d *Dispatcher) GetLastSuccessfulImageTag(appID string) (string, error) {
	return d.queue.GetLastSuccessfulImageTag(appID)
}

func (d *Dispatcher) UpdateAppLastDeployedAt(appID string) error {
	return d.queue.UpdateAppLastDeployedAt(appID)
}

func (d *Dispatcher) UpdateAppRuntime(appID, runtime string) error {
	return d.queue.UpdateAppRuntime(appID, runtime)
}

func (d *Dispatcher) UpdateAppVersion(appID, appVersion string) error {
	return d.queue.UpdateAppVersion(appID, appVersion)
}

func (d *Dispatcher) SetPollTime(duration time.Duration) {
	d.pollTime = duration
}

func (d *Dispatcher) PollTime() time.Duration {
	return d.pollTime
}
