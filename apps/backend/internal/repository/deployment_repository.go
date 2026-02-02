package repository

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
	"github.com/paasdeploy/backend/internal/domain"
)

type PostgresDeploymentRepository struct {
	db *sql.DB
}

func NewPostgresDeploymentRepository(db *sql.DB) *PostgresDeploymentRepository {
	return &PostgresDeploymentRepository{db: db}
}

func (r *PostgresDeploymentRepository) FindByID(id string) (*domain.Deployment, error) {
	query := `
		SELECT id, app_id, commit_sha, commit_message, status, started_at, finished_at,
		       error_message, logs, previous_image_tag, current_image_tag, created_at
		FROM deployments
		WHERE id = $1
	`

	var d domain.Deployment
	var startedAt, finishedAt sql.NullTime
	var commitMessage, errorMessage, logs, previousImageTag, currentImageTag sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&d.ID, &d.AppID, &d.CommitSHA, &commitMessage, &d.Status,
		&startedAt, &finishedAt, &errorMessage, &logs,
		&previousImageTag, &currentImageTag, &d.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
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

func (r *PostgresDeploymentRepository) FindByAppID(appID string, limit int) ([]domain.Deployment, error) {
	query := `
		SELECT id, app_id, commit_sha, commit_message, status, started_at, finished_at,
		       error_message, logs, previous_image_tag, current_image_tag, created_at
		FROM deployments
		WHERE app_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(query, appID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deployments []domain.Deployment
	for rows.Next() {
		var d domain.Deployment
		var startedAt, finishedAt sql.NullTime
		var commitMessage, errorMessage, logs, previousImageTag, currentImageTag sql.NullString

		err := rows.Scan(
			&d.ID, &d.AppID, &d.CommitSHA, &commitMessage, &d.Status,
			&startedAt, &finishedAt, &errorMessage, &logs,
			&previousImageTag, &currentImageTag, &d.CreatedAt,
		)
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

		deployments = append(deployments, d)
	}

	return deployments, nil
}

func (r *PostgresDeploymentRepository) FindPendingByAppID(appID string) (*domain.Deployment, error) {
	query := `
		SELECT id, app_id, commit_sha, commit_message, status, started_at, finished_at,
		       error_message, logs, previous_image_tag, current_image_tag, created_at
		FROM deployments
		WHERE app_id = $1 AND status = 'pending'
		ORDER BY created_at ASC
		LIMIT 1
	`

	var d domain.Deployment
	var startedAt, finishedAt sql.NullTime
	var commitMessage, errorMessage, logs, previousImageTag, currentImageTag sql.NullString

	err := r.db.QueryRow(query, appID).Scan(
		&d.ID, &d.AppID, &d.CommitSHA, &commitMessage, &d.Status,
		&startedAt, &finishedAt, &errorMessage, &logs,
		&previousImageTag, &currentImageTag, &d.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
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

func (r *PostgresDeploymentRepository) FindLatestByAppID(appID string) (*domain.Deployment, error) {
	query := `
		SELECT id, app_id, commit_sha, commit_message, status, started_at, finished_at,
		       error_message, logs, previous_image_tag, current_image_tag, created_at
		FROM deployments
		WHERE app_id = $1 AND status = 'success'
		ORDER BY created_at DESC
		LIMIT 1
	`

	var d domain.Deployment
	var startedAt, finishedAt sql.NullTime
	var commitMessage, errorMessage, logs, previousImageTag, currentImageTag sql.NullString

	err := r.db.QueryRow(query, appID).Scan(
		&d.ID, &d.AppID, &d.CommitSHA, &commitMessage, &d.Status,
		&startedAt, &finishedAt, &errorMessage, &logs,
		&previousImageTag, &currentImageTag, &d.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
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

func (r *PostgresDeploymentRepository) FindMostRecentByAppID(appID string) (*domain.Deployment, error) {
	query := `
		SELECT id, app_id, commit_sha, commit_message, status, started_at, finished_at,
		       error_message, logs, previous_image_tag, current_image_tag, created_at
		FROM deployments
		WHERE app_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var d domain.Deployment
	var startedAt, finishedAt sql.NullTime
	var commitMessage, errorMessage, logs, previousImageTag, currentImageTag sql.NullString

	err := r.db.QueryRow(query, appID).Scan(
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

func (r *PostgresDeploymentRepository) FindMostRecentByAppIDs(appIDs []string) (map[string]*domain.Deployment, error) {
	if len(appIDs) == 0 {
		return make(map[string]*domain.Deployment), nil
	}

	query := `
		SELECT DISTINCT ON (app_id) 
		       id, app_id, commit_sha, commit_message, status, started_at, finished_at,
		       error_message, logs, previous_image_tag, current_image_tag, created_at
		FROM deployments
		WHERE app_id = ANY($1)
		ORDER BY app_id, created_at DESC
	`

	rows, err := r.db.Query(query, pq.Array(appIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*domain.Deployment)
	for rows.Next() {
		var d domain.Deployment
		var startedAt, finishedAt sql.NullTime
		var commitMessage, errorMessage, logs, previousImageTag, currentImageTag sql.NullString

		err := rows.Scan(
			&d.ID, &d.AppID, &d.CommitSHA, &commitMessage, &d.Status,
			&startedAt, &finishedAt, &errorMessage, &logs,
			&previousImageTag, &currentImageTag, &d.CreatedAt,
		)
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

		result[d.AppID] = &d
	}

	return result, nil
}

func (r *PostgresDeploymentRepository) Create(input domain.CreateDeploymentInput) (*domain.Deployment, error) {
	query := `
		INSERT INTO deployments (app_id, commit_sha, commit_message, status, created_at)
		VALUES ($1, $2, $3, 'pending', NOW())
		RETURNING id, app_id, commit_sha, commit_message, status, started_at, finished_at,
		          error_message, logs, previous_image_tag, current_image_tag, created_at
	`

	var d domain.Deployment
	var startedAt, finishedAt sql.NullTime
	var commitMessage, errorMessage, logs, previousImageTag, currentImageTag sql.NullString

	err := r.db.QueryRow(query, input.AppID, input.CommitSHA, input.CommitMessage).Scan(
		&d.ID, &d.AppID, &d.CommitSHA, &commitMessage, &d.Status,
		&startedAt, &finishedAt, &errorMessage, &logs,
		&previousImageTag, &currentImageTag, &d.CreatedAt,
	)
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

func (r *PostgresDeploymentRepository) Update(id string, input domain.UpdateDeploymentInput) (*domain.Deployment, error) {
	d, err := r.FindByID(id)
	if err != nil {
		return nil, err
	}

	if input.Status != nil {
		d.Status = *input.Status
	}
	if input.StartedAt != nil {
		d.StartedAt = input.StartedAt
	}
	if input.FinishedAt != nil {
		d.FinishedAt = input.FinishedAt
	}
	if input.ErrorMessage != nil {
		d.ErrorMessage = *input.ErrorMessage
	}
	if input.Logs != nil {
		d.Logs = *input.Logs
	}
	if input.PreviousImageTag != nil {
		d.PreviousImageTag = *input.PreviousImageTag
	}
	if input.CurrentImageTag != nil {
		d.CurrentImageTag = *input.CurrentImageTag
	}

	query := `
		UPDATE deployments
		SET status = $2, started_at = $3, finished_at = $4, error_message = $5,
		    logs = $6, previous_image_tag = $7, current_image_tag = $8
		WHERE id = $1
	`

	_, err = r.db.Exec(query, id, d.Status, d.StartedAt, d.FinishedAt, d.ErrorMessage, d.Logs, d.PreviousImageTag, d.CurrentImageTag)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (r *PostgresDeploymentRepository) AppendLogs(id string, logs string) error {
	query := `UPDATE deployments SET logs = COALESCE(logs, '') || $2 WHERE id = $1`
	_, err := r.db.Exec(query, id, logs)
	return err
}

func (r *PostgresDeploymentRepository) GetNextPending() (*domain.Deployment, error) {
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

	err := r.db.QueryRow(query).Scan(
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

func (r *PostgresDeploymentRepository) MarkAsRunning(id string) error {
	now := time.Now()
	query := `UPDATE deployments SET status = 'running', started_at = $2 WHERE id = $1`
	_, err := r.db.Exec(query, id, now)
	return err
}

func (r *PostgresDeploymentRepository) MarkAsSuccess(id string, imageTag string) error {
	now := time.Now()
	query := `UPDATE deployments SET status = 'success', finished_at = $2, current_image_tag = $3 WHERE id = $1`
	_, err := r.db.Exec(query, id, now, imageTag)
	return err
}

func (r *PostgresDeploymentRepository) MarkAsFailed(id string, errorMessage string) error {
	now := time.Now()
	query := `UPDATE deployments SET status = 'failed', finished_at = $2, error_message = $3 WHERE id = $1`
	_, err := r.db.Exec(query, id, now, errorMessage)
	return err
}

func (r *PostgresDeploymentRepository) DeleteByAppID(appID string) error {
	query := `DELETE FROM deployments WHERE app_id = $1`
	_, err := r.db.Exec(query, appID)
	return err
}
