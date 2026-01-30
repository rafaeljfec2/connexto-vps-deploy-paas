package engine

import (
	"database/sql"
	"time"

	"github.com/paasdeploy/backend/internal/domain"
)

type Queue struct {
	db *sql.DB
}

func NewQueue(db *sql.DB) *Queue {
	return &Queue{db: db}
}

func (q *Queue) GetNextPending() (*domain.Deployment, error) {
	query := `
		SELECT d.id, d.app_id, d.commit_sha, d.commit_message, d.status, d.started_at, d.finished_at,
		       d.error_message, d.logs, d.previous_image_tag, d.current_image_tag, d.created_at
		FROM deployments d
		WHERE d.status = 'pending'
		AND d.app_id NOT IN (
			SELECT app_id FROM deployments WHERE status = 'running'
		)
		ORDER BY d.created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`

	var d domain.Deployment
	var startedAt, finishedAt sql.NullTime
	var commitMessage, errorMessage, logs, previousImageTag, currentImageTag sql.NullString

	err := q.db.QueryRow(query).Scan(
		&d.ID, &d.AppID, &d.CommitSHA, &commitMessage, &d.Status,
		&startedAt, &finishedAt, &errorMessage, &logs,
		&previousImageTag, &currentImageTag, &d.CreatedAt,
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

	return &d, nil
}

func (q *Queue) MarkAsRunning(id string) error {
	now := time.Now()
	query := `UPDATE deployments SET status = 'running', started_at = $2 WHERE id = $1`
	_, err := q.db.Exec(query, id, now)
	return err
}

func (q *Queue) MarkAsSuccess(id string, imageTag string) error {
	now := time.Now()
	query := `UPDATE deployments SET status = 'success', finished_at = $2, current_image_tag = $3 WHERE id = $1`
	_, err := q.db.Exec(query, id, now, imageTag)
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

func (q *Queue) GetAppByID(appID string) (*domain.App, error) {
	query := `
		SELECT id, name, repository_url, branch, config, status, last_deployed_at, created_at, updated_at
		FROM apps
		WHERE id = $1 AND status != 'deleted'
	`

	var app domain.App
	var lastDeployedAt sql.NullTime

	err := q.db.QueryRow(query, appID).Scan(
		&app.ID,
		&app.Name,
		&app.RepositoryURL,
		&app.Branch,
		&app.Config,
		&app.Status,
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

	if lastDeployedAt.Valid {
		app.LastDeployedAt = &lastDeployedAt.Time
	}

	return &app, nil
}
