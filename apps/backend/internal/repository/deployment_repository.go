package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/paasdeploy/backend/internal/domain"
)

const deploymentSelectColumns = `id, app_id, commit_sha, commit_message, status, started_at, finished_at,
       error_message, logs, previous_image_tag, current_image_tag, app_version, created_at`

type PostgresDeploymentRepository struct {
	db *sql.DB
}

func NewPostgresDeploymentRepository(db *sql.DB) *PostgresDeploymentRepository {
	return &PostgresDeploymentRepository{db: db}
}

type deploymentScanTargets struct {
	d                domain.Deployment
	startedAt        sql.NullTime
	finishedAt       sql.NullTime
	commitMessage    sql.NullString
	errorMessage     sql.NullString
	logs             sql.NullString
	previousImageTag sql.NullString
	currentImageTag  sql.NullString
	appVersion       sql.NullString
}

func (t *deploymentScanTargets) scanArgs() []interface{} {
	return []interface{}{
		&t.d.ID, &t.d.AppID, &t.d.CommitSHA, &t.commitMessage, &t.d.Status,
		&t.startedAt, &t.finishedAt, &t.errorMessage, &t.logs,
		&t.previousImageTag, &t.currentImageTag, &t.appVersion, &t.d.CreatedAt,
	}
}

func (t *deploymentScanTargets) toDeployment() domain.Deployment {
	if t.startedAt.Valid {
		t.d.StartedAt = &t.startedAt.Time
	}
	if t.finishedAt.Valid {
		t.d.FinishedAt = &t.finishedAt.Time
	}
	t.d.CommitMessage = t.commitMessage.String
	t.d.ErrorMessage = t.errorMessage.String
	t.d.Logs = t.logs.String
	t.d.PreviousImageTag = t.previousImageTag.String
	t.d.CurrentImageTag = t.currentImageTag.String
	t.d.AppVersion = t.appVersion.String
	return t.d
}

func scanDeploymentRow(row *sql.Row) (*domain.Deployment, error) {
	var t deploymentScanTargets
	if err := row.Scan(t.scanArgs()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	d := t.toDeployment()
	return &d, nil
}

func scanDeploymentRowNullable(row *sql.Row) (*domain.Deployment, error) {
	var t deploymentScanTargets
	if err := row.Scan(t.scanArgs()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	d := t.toDeployment()
	return &d, nil
}

func (r *PostgresDeploymentRepository) FindByID(id string) (*domain.Deployment, error) {
	query := `SELECT ` + deploymentSelectColumns + ` FROM deployments WHERE id = $1`
	return scanDeploymentRow(r.db.QueryRow(query, id))
}

func (r *PostgresDeploymentRepository) FindByAppID(appID string, limit int) ([]domain.Deployment, error) {
	query := `SELECT ` + deploymentSelectColumns + `
		FROM deployments WHERE app_id = $1 ORDER BY created_at DESC LIMIT $2`

	rows, err := r.db.Query(query, appID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDeploymentRows(rows)
}

func (r *PostgresDeploymentRepository) FindPendingByAppID(appID string) (*domain.Deployment, error) {
	query := `SELECT ` + deploymentSelectColumns + `
		FROM deployments WHERE app_id = $1 AND status = 'pending'
		ORDER BY created_at ASC LIMIT 1`
	return scanDeploymentRow(r.db.QueryRow(query, appID))
}

func (r *PostgresDeploymentRepository) FindLatestByAppID(appID string) (*domain.Deployment, error) {
	query := `SELECT ` + deploymentSelectColumns + `
		FROM deployments WHERE app_id = $1 AND status = 'success'
		ORDER BY created_at DESC LIMIT 1`
	return scanDeploymentRow(r.db.QueryRow(query, appID))
}

func (r *PostgresDeploymentRepository) FindMostRecentByAppID(appID string) (*domain.Deployment, error) {
	query := `SELECT ` + deploymentSelectColumns + `
		FROM deployments WHERE app_id = $1
		ORDER BY created_at DESC LIMIT 1`
	return scanDeploymentRowNullable(r.db.QueryRow(query, appID))
}

func (r *PostgresDeploymentRepository) FindMostRecentByAppIDs(appIDs []string) (map[string]*domain.Deployment, error) {
	if len(appIDs) == 0 {
		return make(map[string]*domain.Deployment), nil
	}

	query := `SELECT DISTINCT ON (app_id) ` + deploymentSelectColumns + `
		FROM deployments WHERE app_id = ANY($1)
		ORDER BY app_id, created_at DESC`

	rows, err := r.db.Query(query, appIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*domain.Deployment)
	for rows.Next() {
		var t deploymentScanTargets
		if err := rows.Scan(t.scanArgs()...); err != nil {
			return nil, err
		}
		d := t.toDeployment()
		result[d.AppID] = &d
	}

	return result, nil
}

func (r *PostgresDeploymentRepository) Create(input domain.CreateDeploymentInput) (*domain.Deployment, error) {
	var deliveryID *string
	if input.DeliveryID != "" {
		deliveryID = &input.DeliveryID
	}

	query := `INSERT INTO deployments (app_id, commit_sha, commit_message, status, delivery_id, created_at)
		VALUES ($1, $2, $3, 'pending', $4, NOW())
		ON CONFLICT (app_id, commit_sha) WHERE status IN ('pending', 'running') DO NOTHING
		RETURNING ` + deploymentSelectColumns

	row := r.db.QueryRow(query, input.AppID, input.CommitSHA, input.CommitMessage, deliveryID)
	d, err := scanDeploymentRowNullable(row)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, domain.ErrDeploymentAlreadyActive
	}
	return d, nil
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

	query := `UPDATE deployments
		SET status = $2, started_at = $3, finished_at = $4, error_message = $5,
		    logs = $6, previous_image_tag = $7, current_image_tag = $8
		WHERE id = $1`

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
	query := `SELECT ` + deploymentSelectColumns + `
		FROM deployments
		WHERE status = 'pending'
		AND app_id NOT IN (SELECT app_id FROM deployments WHERE status = 'running')
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED`

	var t deploymentScanTargets
	err := r.db.QueryRow(query).Scan(t.scanArgs()...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	d := t.toDeployment()
	return &d, nil
}

func (r *PostgresDeploymentRepository) MarkAsRunning(id string) error {
	now := time.Now()
	query := `UPDATE deployments SET status = 'running', started_at = $2 WHERE id = $1`
	_, err := r.db.Exec(query, id, now)
	return err
}

func (r *PostgresDeploymentRepository) MarkAsSuccess(id string, imageTag string, appVersion string) error {
	now := time.Now()
	query := `UPDATE deployments SET status = 'success', finished_at = $2, current_image_tag = $3, app_version = $4 WHERE id = $1`
	_, err := r.db.Exec(query, id, now, imageTag, appVersion)
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

func scanDeploymentRows(rows *sql.Rows) ([]domain.Deployment, error) {
	var deployments []domain.Deployment
	for rows.Next() {
		var t deploymentScanTargets
		if err := rows.Scan(t.scanArgs()...); err != nil {
			return nil, err
		}
		deployments = append(deployments, t.toDeployment())
	}
	return deployments, nil
}
