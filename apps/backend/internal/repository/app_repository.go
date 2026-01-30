package repository

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/paasdeploy/backend/internal/domain"
)

type PostgresAppRepository struct {
	db *sql.DB
}

func NewPostgresAppRepository(db *sql.DB) *PostgresAppRepository {
	return &PostgresAppRepository{db: db}
}

func (r *PostgresAppRepository) FindAll() ([]domain.App, error) {
	query := `
		SELECT id, name, repository_url, branch, workdir, runtime, config, status, webhook_id, last_deployed_at, created_at, updated_at
		FROM apps
		WHERE status != 'deleted'
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []domain.App
	for rows.Next() {
		var app domain.App
		var lastDeployedAt sql.NullTime
		var webhookID sql.NullInt64
		var runtime sql.NullString

		err := rows.Scan(
			&app.ID,
			&app.Name,
			&app.RepositoryURL,
			&app.Branch,
			&app.Workdir,
			&runtime,
			&app.Config,
			&app.Status,
			&webhookID,
			&lastDeployedAt,
			&app.CreatedAt,
			&app.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if webhookID.Valid {
			app.WebhookID = &webhookID.Int64
		}
		if lastDeployedAt.Valid {
			app.LastDeployedAt = &lastDeployedAt.Time
		}
		if runtime.Valid {
			app.Runtime = &runtime.String
		}

		apps = append(apps, app)
	}

	return apps, nil
}

func (r *PostgresAppRepository) FindByID(id string) (*domain.App, error) {
	query := `
		SELECT id, name, repository_url, branch, workdir, runtime, config, status, webhook_id, last_deployed_at, created_at, updated_at
		FROM apps
		WHERE id = $1 AND status != 'deleted'
	`

	var app domain.App
	var lastDeployedAt sql.NullTime
	var webhookID sql.NullInt64
	var runtime sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&app.ID,
		&app.Name,
		&app.RepositoryURL,
		&app.Branch,
		&app.Workdir,
		&runtime,
		&app.Config,
		&app.Status,
		&webhookID,
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

	if webhookID.Valid {
		app.WebhookID = &webhookID.Int64
	}
	if lastDeployedAt.Valid {
		app.LastDeployedAt = &lastDeployedAt.Time
	}
	if runtime.Valid {
		app.Runtime = &runtime.String
	}

	return &app, nil
}

func (r *PostgresAppRepository) FindByName(name string) (*domain.App, error) {
	query := `
		SELECT id, name, repository_url, branch, workdir, runtime, config, status, webhook_id, last_deployed_at, created_at, updated_at
		FROM apps
		WHERE name = $1 AND status != 'deleted'
	`

	var app domain.App
	var lastDeployedAt sql.NullTime
	var webhookID sql.NullInt64
	var runtime sql.NullString

	err := r.db.QueryRow(query, name).Scan(
		&app.ID,
		&app.Name,
		&app.RepositoryURL,
		&app.Branch,
		&app.Workdir,
		&runtime,
		&app.Config,
		&app.Status,
		&webhookID,
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

	if webhookID.Valid {
		app.WebhookID = &webhookID.Int64
	}
	if lastDeployedAt.Valid {
		app.LastDeployedAt = &lastDeployedAt.Time
	}
	if runtime.Valid {
		app.Runtime = &runtime.String
	}

	return &app, nil
}

func (r *PostgresAppRepository) FindByRepoURL(repoURL string) (*domain.App, error) {
	query := `
		SELECT id, name, repository_url, branch, workdir, runtime, config, status, webhook_id, last_deployed_at, created_at, updated_at
		FROM apps
		WHERE repository_url = $1 AND status != 'deleted'
	`

	var app domain.App
	var lastDeployedAt sql.NullTime
	var webhookID sql.NullInt64
	var runtime sql.NullString

	err := r.db.QueryRow(query, repoURL).Scan(
		&app.ID,
		&app.Name,
		&app.RepositoryURL,
		&app.Branch,
		&app.Workdir,
		&runtime,
		&app.Config,
		&app.Status,
		&webhookID,
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

	if webhookID.Valid {
		app.WebhookID = &webhookID.Int64
	}
	if lastDeployedAt.Valid {
		app.LastDeployedAt = &lastDeployedAt.Time
	}
	if runtime.Valid {
		app.Runtime = &runtime.String
	}

	return &app, nil
}

func (r *PostgresAppRepository) Create(input domain.CreateAppInput) (*domain.App, error) {
	config := input.Config
	if config == nil {
		config = json.RawMessage("{}")
	}

	branch := input.Branch
	if branch == "" {
		branch = "main"
	}

	workdir := input.Workdir
	if workdir == "" {
		workdir = "."
	}

	query := `
		INSERT INTO apps (name, repository_url, branch, workdir, config, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'active', NOW(), NOW())
		RETURNING id, name, repository_url, branch, workdir, runtime, config, status, webhook_id, last_deployed_at, created_at, updated_at
	`

	var app domain.App
	var lastDeployedAt sql.NullTime
	var webhookID sql.NullInt64
	var runtime sql.NullString

	err := r.db.QueryRow(query, input.Name, input.RepositoryURL, branch, workdir, config).Scan(
		&app.ID,
		&app.Name,
		&app.RepositoryURL,
		&app.Branch,
		&app.Workdir,
		&runtime,
		&app.Config,
		&app.Status,
		&webhookID,
		&lastDeployedAt,
		&app.CreatedAt,
		&app.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if webhookID.Valid {
		app.WebhookID = &webhookID.Int64
	}
	if lastDeployedAt.Valid {
		app.LastDeployedAt = &lastDeployedAt.Time
	}
	if runtime.Valid {
		app.Runtime = &runtime.String
	}

	return &app, nil
}

func (r *PostgresAppRepository) Update(id string, input domain.UpdateAppInput) (*domain.App, error) {
	app, err := r.FindByID(id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		app.Name = *input.Name
	}
	if input.RepositoryURL != nil {
		app.RepositoryURL = *input.RepositoryURL
	}
	if input.Branch != nil {
		app.Branch = *input.Branch
	}
	if input.Workdir != nil {
		app.Workdir = *input.Workdir
	}
	if input.Runtime != nil {
		app.Runtime = input.Runtime
	}
	if input.Config != nil {
		app.Config = *input.Config
	}
	if input.Status != nil {
		app.Status = *input.Status
	}
	if input.WebhookID != nil {
		app.WebhookID = input.WebhookID
	}

	query := `
		UPDATE apps
		SET name = $2, repository_url = $3, branch = $4, workdir = $5, runtime = $6, config = $7, status = $8, webhook_id = $9, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err = r.db.QueryRow(query, id, app.Name, app.RepositoryURL, app.Branch, app.Workdir, app.Runtime, app.Config, app.Status, app.WebhookID).Scan(&app.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (r *PostgresAppRepository) Delete(id string) error {
	query := `UPDATE apps SET status = 'deleted', updated_at = NOW() WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *PostgresAppRepository) UpdateLastDeployedAt(id string, deployedAt time.Time) error {
	query := `UPDATE apps SET last_deployed_at = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(query, id, deployedAt)
	return err
}

func (r *PostgresAppRepository) HardDelete(id string) error {
	query := `DELETE FROM apps WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}
