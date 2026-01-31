package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/paasdeploy/backend/internal/domain"
)

const installationSelectColumns = `id, installation_id, account_type, account_id, account_login,
	repository_selection, permissions, suspended_at, created_at, updated_at`

const installationSelectColumnsWithPrefix = `gi.id, gi.installation_id, gi.account_type, gi.account_id, gi.account_login,
	gi.repository_selection, gi.permissions, gi.suspended_at, gi.created_at, gi.updated_at`

type PostgresInstallationRepository struct {
	db *sql.DB
}

func NewPostgresInstallationRepository(db *sql.DB) *PostgresInstallationRepository {
	return &PostgresInstallationRepository{db: db}
}

type installationScanFields struct {
	inst            domain.Installation
	repoSelection   sql.NullString
	permissionsJSON []byte
	suspendedAt     sql.NullTime
}

func (f *installationScanFields) scanDest() []any {
	return []any{
		&f.inst.ID,
		&f.inst.InstallationID,
		&f.inst.AccountType,
		&f.inst.AccountID,
		&f.inst.AccountLogin,
		&f.repoSelection,
		&f.permissionsJSON,
		&f.suspendedAt,
		&f.inst.CreatedAt,
		&f.inst.UpdatedAt,
	}
}

func (f *installationScanFields) toInstallation() (*domain.Installation, error) {
	f.inst.RepositorySelection = fromNullString(f.repoSelection)
	f.inst.SuspendedAt = fromNullTime(f.suspendedAt)

	if len(f.permissionsJSON) > 0 {
		if err := json.Unmarshal(f.permissionsJSON, &f.inst.Permissions); err != nil {
			return nil, err
		}
	} else {
		f.inst.Permissions = make(map[string]string)
	}

	return &f.inst, nil
}

func (r *PostgresInstallationRepository) scanInstallation(row *sql.Row) (*domain.Installation, error) {
	var f installationScanFields
	if err := row.Scan(f.scanDest()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return f.toInstallation()
}

func (r *PostgresInstallationRepository) FindByID(ctx context.Context, id string) (*domain.Installation, error) {
	query := `SELECT ` + installationSelectColumns + ` FROM github_installations WHERE id = $1`
	return r.scanInstallation(r.db.QueryRowContext(ctx, query, id))
}

func (r *PostgresInstallationRepository) FindByInstallationID(ctx context.Context, installationID int64) (*domain.Installation, error) {
	query := `SELECT ` + installationSelectColumns + ` FROM github_installations WHERE installation_id = $1`
	return r.scanInstallation(r.db.QueryRowContext(ctx, query, installationID))
}

func (r *PostgresInstallationRepository) FindByAccountLogin(ctx context.Context, accountLogin string) (*domain.Installation, error) {
	query := `SELECT ` + installationSelectColumns + ` FROM github_installations WHERE account_login = $1`
	return r.scanInstallation(r.db.QueryRowContext(ctx, query, accountLogin))
}

func (r *PostgresInstallationRepository) Create(ctx context.Context, input domain.CreateInstallationInput) (*domain.Installation, error) {
	query := `
		INSERT INTO github_installations (installation_id, account_type, account_id, account_login, repository_selection, permissions)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + installationSelectColumns

	permissionsJSON, err := json.Marshal(input.Permissions)
	if err != nil {
		return nil, err
	}

	repoSelection := input.RepositorySelection
	if repoSelection == "" {
		repoSelection = "selected"
	}

	row := r.db.QueryRowContext(ctx, query,
		input.InstallationID,
		input.AccountType,
		input.AccountID,
		input.AccountLogin,
		repoSelection,
		permissionsJSON,
	)

	var f installationScanFields
	if err := row.Scan(f.scanDest()...); err != nil {
		return nil, err
	}
	return f.toInstallation()
}

func (r *PostgresInstallationRepository) Update(ctx context.Context, id string, input domain.UpdateInstallationInput) (*domain.Installation, error) {
	query := `
		UPDATE github_installations SET
			repository_selection = COALESCE($2, repository_selection),
			permissions = COALESCE($3, permissions),
			suspended_at = $4,
			updated_at = NOW()
		WHERE id = $1
		RETURNING ` + installationSelectColumns

	var permissionsJSON []byte
	if input.Permissions != nil {
		var err error
		permissionsJSON, err = json.Marshal(input.Permissions)
		if err != nil {
			return nil, err
		}
	}

	row := r.db.QueryRowContext(ctx, query,
		id,
		toNullString(input.RepositorySelection),
		permissionsJSON,
		toNullTime(input.SuspendedAt),
	)

	return r.scanInstallation(row)
}

func (r *PostgresInstallationRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM github_installations WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
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

func (r *PostgresInstallationRepository) LinkUserToInstallation(ctx context.Context, userID, installationID string, isDefault bool) error {
	query := `
		INSERT INTO user_installations (user_id, installation_id, is_default)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, installation_id) DO UPDATE SET is_default = $3
	`
	_, err := r.db.ExecContext(ctx, query, userID, installationID, isDefault)
	return err
}

func (r *PostgresInstallationRepository) UnlinkUserFromInstallation(ctx context.Context, userID, installationID string) error {
	query := `DELETE FROM user_installations WHERE user_id = $1 AND installation_id = $2`
	_, err := r.db.ExecContext(ctx, query, userID, installationID)
	return err
}

func (r *PostgresInstallationRepository) FindUserInstallations(ctx context.Context, userID string) ([]domain.Installation, error) {
	query := `
		SELECT ` + installationSelectColumnsWithPrefix + `
		FROM github_installations gi
		INNER JOIN user_installations ui ON gi.id = ui.installation_id
		WHERE ui.user_id = $1
		ORDER BY gi.account_login
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var installations []domain.Installation
	for rows.Next() {
		var f installationScanFields
		if err := rows.Scan(f.scanDest()...); err != nil {
			return nil, err
		}
		inst, err := f.toInstallation()
		if err != nil {
			return nil, err
		}
		installations = append(installations, *inst)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return installations, nil
}

func (r *PostgresInstallationRepository) FindDefaultInstallation(ctx context.Context, userID string) (*domain.Installation, error) {
	query := `
		SELECT ` + installationSelectColumnsWithPrefix + `
		FROM github_installations gi
		INNER JOIN user_installations ui ON gi.id = ui.installation_id
		WHERE ui.user_id = $1 AND ui.is_default = true
		LIMIT 1
	`
	return r.scanInstallation(r.db.QueryRowContext(ctx, query, userID))
}

func (r *PostgresInstallationRepository) SetDefaultInstallation(ctx context.Context, userID, installationID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `UPDATE user_installations SET is_default = false WHERE user_id = $1`, userID); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, `UPDATE user_installations SET is_default = true WHERE user_id = $1 AND installation_id = $2`, userID, installationID); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

var _ domain.InstallationRepository = (*PostgresInstallationRepository)(nil)
