package engine

import (
	"context"
	"database/sql"
	"time"

	"github.com/paasdeploy/backend/internal/domain"
)

type dbQuerier interface {
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
}

type Queue struct {
	db *sql.DB
}

func NewQueue(db *sql.DB) *Queue {
	return &Queue{db: db}
}

func (q *Queue) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return q.db.BeginTx(ctx, nil)
}

func scanPendingDeploy(row *sql.Row) (*domain.Deployment, error) {
	var d domain.Deployment
	var startedAt, finishedAt sql.NullTime
	var commitMessage, errorMessage, logs, previousImageTag, currentImageTag, appVersion sql.NullString

	err := row.Scan(
		&d.ID, &d.AppID, &d.CommitSHA, &commitMessage, &d.Status,
		&startedAt, &finishedAt, &errorMessage, &logs,
		&previousImageTag, &currentImageTag, &appVersion, &d.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if startedAt.Valid {
		d.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		d.FinishedAt = &finishedAt.Time
	}
	d.CommitMessage = commitMessage.String
	d.ErrorMessage = errorMessage.String
	d.Logs = logs.String
	d.PreviousImageTag = previousImageTag.String
	d.CurrentImageTag = currentImageTag.String
	d.AppVersion = appVersion.String

	return &d, nil
}

const pendingDeployQuery = `
	SELECT d.id, d.app_id, d.commit_sha, d.commit_message, d.status, d.started_at, d.finished_at,
	       d.error_message, d.logs, d.previous_image_tag, d.current_image_tag, d.app_version, d.created_at
	FROM deployments d
	WHERE d.status = 'pending'
	AND d.app_id NOT IN (
		SELECT app_id FROM deployments WHERE status = 'running'
	)
	ORDER BY d.created_at ASC
	LIMIT 1
	FOR UPDATE SKIP LOCKED
`

func (q *Queue) GetNextPending() (*domain.Deployment, error) {
	return scanPendingDeploy(q.db.QueryRow(pendingDeployQuery))
}

func (q *Queue) GetNextPendingTx(tx *sql.Tx) (*domain.Deployment, error) {
	return scanPendingDeploy(tx.QueryRow(pendingDeployQuery))
}

func (q *Queue) MarkAsRunning(id string) error {
	return q.markAsRunningWith(q.db, id)
}

func (q *Queue) MarkAsRunningTx(tx *sql.Tx, id string) error {
	return q.markAsRunningWith(tx, id)
}

func (q *Queue) markAsRunningWith(db dbQuerier, id string) error {
	now := time.Now()
	query := `UPDATE deployments SET status = 'running', started_at = $2 WHERE id = $1`
	_, err := db.Exec(query, id, now)
	return err
}

func (q *Queue) MarkAsSuccess(id string, imageTag string, appVersion string) error {
	now := time.Now()
	query := `UPDATE deployments SET status = 'success', finished_at = $2, current_image_tag = $3, app_version = $4 WHERE id = $1`
	_, err := q.db.Exec(query, id, now, imageTag, appVersion)
	return err
}

func (q *Queue) MarkAsFailed(id string, errorMessage string) error {
	now := time.Now()
	query := `UPDATE deployments SET status = 'failed', finished_at = $2, error_message = $3 WHERE id = $1`
	_, err := q.db.Exec(query, id, now, errorMessage)
	return err
}

func (q *Queue) AppendLogs(id string, logs string) error {
	query := `UPDATE deployments SET logs = COALESCE(logs, '') || $2 WHERE id = $1`
	_, err := q.db.Exec(query, id, logs)
	return err
}

func (q *Queue) SetPreviousImageTag(id string, tag string) error {
	query := `UPDATE deployments SET previous_image_tag = $2 WHERE id = $1`
	_, err := q.db.Exec(query, id, tag)
	return err
}

func (q *Queue) GetPendingCount() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM deployments WHERE status = 'pending'`
	err := q.db.QueryRow(query).Scan(&count)
	return count, err
}

func (q *Queue) GetRunningCount() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM deployments WHERE status = 'running'`
	err := q.db.QueryRow(query).Scan(&count)
	return count, err
}

func (q *Queue) UpdateAppLastDeployedAt(appID string) error {
	query := `UPDATE apps SET last_deployed_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := q.db.Exec(query, appID)
	return err
}

func (q *Queue) UpdateAppRuntime(appID, runtime string) error {
	if runtime == "" {
		return nil
	}
	query := `UPDATE apps SET runtime = $2, updated_at = NOW() WHERE id = $1`
	_, err := q.db.Exec(query, appID, runtime)
	return err
}

func (q *Queue) UpdateAppVersion(appID, appVersion string) error {
	if appVersion == "" {
		return nil
	}
	query := `UPDATE apps SET app_version = $2, updated_at = NOW() WHERE id = $1`
	_, err := q.db.Exec(query, appID, appVersion)
	return err
}

func (q *Queue) GetLastSuccessfulImageTag(appID string) (string, error) {
	query := `
		SELECT current_image_tag FROM deployments
		WHERE app_id = $1 AND status = 'success' AND current_image_tag IS NOT NULL AND current_image_tag != ''
		ORDER BY finished_at DESC
		LIMIT 1
	`
	var tag string
	err := q.db.QueryRow(query, appID).Scan(&tag)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return tag, nil
}

func (q *Queue) GetAppByID(appID string) (*domain.App, error) {
	query := `
		SELECT id, name, repository_url, branch, workdir, runtime, app_version, config, status, webhook_id, server_id, last_deployed_at, created_at, updated_at
		FROM apps
		WHERE id = $1 AND status != 'deleted'
	`

	var app domain.App
	var lastDeployedAt sql.NullTime
	var workdir sql.NullString
	var runtime sql.NullString
	var appVersionStr sql.NullString
	var webhookID sql.NullInt64
	var serverID sql.NullString

	err := q.db.QueryRow(query, appID).Scan(
		&app.ID,
		&app.Name,
		&app.RepositoryURL,
		&app.Branch,
		&workdir,
		&runtime,
		&appVersionStr,
		&app.Config,
		&app.Status,
		&webhookID,
		&serverID,
		&lastDeployedAt,
		&app.CreatedAt,
		&app.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	app.Workdir = workdir.String
	if lastDeployedAt.Valid {
		app.LastDeployedAt = &lastDeployedAt.Time
	}
	if runtime.Valid {
		app.Runtime = &runtime.String
	}
	if appVersionStr.Valid {
		app.AppVersion = &appVersionStr.String
	}
	if webhookID.Valid {
		app.WebhookID = &webhookID.Int64
	}
	if serverID.Valid {
		app.ServerID = &serverID.String
	}

	return &app, nil
}
