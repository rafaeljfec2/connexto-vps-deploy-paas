package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/paasdeploy/backend/internal/domain"
)

const appSelectColumns = `id, name, repository_url, branch, workdir, runtime, config, status, webhook_id, server_id, last_deployed_at, created_at, updated_at`

type PostgresAppRepository struct {
	db *sql.DB
}

func NewPostgresAppRepository(db *sql.DB) *PostgresAppRepository {
	return &PostgresAppRepository{db: db}
}

type appScanFields struct {
	app            domain.App
	webhookID      sql.NullInt64
	serverID       sql.NullString
	lastDeployedAt sql.NullTime
	runtime        sql.NullString
}

func (f *appScanFields) scanDest() []any {
	return []any{
		&f.app.ID,
		&f.app.Name,
		&f.app.RepositoryURL,
		&f.app.Branch,
		&f.app.Workdir,
		&f.runtime,
		&f.app.Config,
		&f.app.Status,
		&f.webhookID,
		&f.serverID,
		&f.lastDeployedAt,
		&f.app.CreatedAt,
		&f.app.UpdatedAt,
	}
}

func (f *appScanFields) toApp() *domain.App {
	if f.webhookID.Valid {
		f.app.WebhookID = &f.webhookID.Int64
	}
	if f.serverID.Valid {
		f.app.ServerID = &f.serverID.String
	}
	if f.lastDeployedAt.Valid {
		f.app.LastDeployedAt = &f.lastDeployedAt.Time
	}
	if f.runtime.Valid {
		f.app.Runtime = &f.runtime.String
	}
	return &f.app
}

func (r *PostgresAppRepository) scanApp(row *sql.Row) (*domain.App, error) {
	var f appScanFields
	if err := row.Scan(f.scanDest()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return f.toApp(), nil
}

func (r *PostgresAppRepository) FindAll() ([]domain.App, error) {
	query := `SELECT ` + appSelectColumns + ` FROM apps WHERE status != 'deleted' ORDER BY created_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []domain.App
	for rows.Next() {
		var f appScanFields
		if err := rows.Scan(f.scanDest()...); err != nil {
			return nil, err
		}
		apps = append(apps, *f.toApp())
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return apps, nil
}

func (r *PostgresAppRepository) FindByID(id string) (*domain.App, error) {
	query := `SELECT ` + appSelectColumns + ` FROM apps WHERE id = $1 AND status != 'deleted'`
	return r.scanApp(r.db.QueryRow(query, id))
}

func (r *PostgresAppRepository) FindByName(name string) (*domain.App, error) {
	query := `SELECT ` + appSelectColumns + ` FROM apps WHERE name = $1 AND status != 'deleted'`
	return r.scanApp(r.db.QueryRow(query, name))
}

func (r *PostgresAppRepository) FindByRepoURL(repoURL string) (*domain.App, error) {
	query := `SELECT ` + appSelectColumns + ` FROM apps WHERE repository_url = $1 AND status != 'deleted'`
	return r.scanApp(r.db.QueryRow(query, repoURL))
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
		INSERT INTO apps (name, repository_url, branch, workdir, config, server_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'active', NOW(), NOW())
		RETURNING ` + appSelectColumns

	var serverID interface{}
	if input.ServerID != nil && *input.ServerID != "" {
		serverID = *input.ServerID
	}

	row := r.db.QueryRow(query, input.Name, input.RepositoryURL, branch, workdir, config, serverID)

	var f appScanFields
	if err := row.Scan(f.scanDest()...); err != nil {
		return nil, err
	}
	return f.toApp(), nil
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
	if input.ServerID != nil {
		app.ServerID = input.ServerID
	}

	query := `
		UPDATE apps
		SET name = $2, repository_url = $3, branch = $4, workdir = $5, runtime = $6, config = $7, status = $8, webhook_id = $9, server_id = $10, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err = r.db.QueryRow(query, id, app.Name, app.RepositoryURL, app.Branch, app.Workdir, app.Runtime, app.Config, app.Status, app.WebhookID, app.ServerID).Scan(&app.UpdatedAt)
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
