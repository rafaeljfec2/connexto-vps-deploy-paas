package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/paasdeploy/backend/internal/domain"
)

type PostgresInstallationRepository struct {
	db *sql.DB
}

func NewPostgresInstallationRepository(db *sql.DB) *PostgresInstallationRepository {
	return &PostgresInstallationRepository{db: db}
}

func (r *PostgresInstallationRepository) FindByID(ctx context.Context, id string) (*domain.Installation, error) {
	query := `
		SELECT id, installation_id, account_type, account_id, account_login,
		       repository_selection, permissions, suspended_at, created_at, updated_at
		FROM github_installations
		WHERE id = $1
	`

	return r.scanInstallation(r.db.QueryRowContext(ctx, query, id))
}

func (r *PostgresInstallationRepository) FindByInstallationID(ctx context.Context, installationID int64) (*domain.Installation, error) {
	query := `
		SELECT id, installation_id, account_type, account_id, account_login,
		       repository_selection, permissions, suspended_at, created_at, updated_at
		FROM github_installations
		WHERE installation_id = $1
	`

	return r.scanInstallation(r.db.QueryRowContext(ctx, query, installationID))
}

func (r *PostgresInstallationRepository) FindByAccountLogin(ctx context.Context, accountLogin string) (*domain.Installation, error) {
	query := `
		SELECT id, installation_id, account_type, account_id, account_login,
		       repository_selection, permissions, suspended_at, created_at, updated_at
		FROM github_installations
		WHERE account_login = $1
	`

	return r.scanInstallation(r.db.QueryRowContext(ctx, query, accountLogin))
}

func (r *PostgresInstallationRepository) scanInstallation(row *sql.Row) (*domain.Installation, error) {
	var inst domain.Installation
	var repoSelection sql.NullString
	var permissionsJSON []byte
	var suspendedAt sql.NullTime

	err := row.Scan(
		&inst.ID,
		&inst.InstallationID,
		&inst.AccountType,
		&inst.AccountID,
		&inst.AccountLogin,
		&repoSelection,
		&permissionsJSON,
		&suspendedAt,
		&inst.CreatedAt,
		&inst.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	inst.RepositorySelection = repoSelection.String
	if suspendedAt.Valid {
		inst.SuspendedAt = &suspendedAt.Time
	}

	if len(permissionsJSON) > 0 {
		if err := json.Unmarshal(permissionsJSON, &inst.Permissions); err != nil {
			return nil, err
		}
	} else {
		inst.Permissions = make(map[string]string)
	}

	return &inst, nil
}

func (r *PostgresInstallationRepository) Create(ctx context.Context, input domain.CreateInstallationInput) (*domain.Installation, error) {
	query := `
		INSERT INTO github_installations (installation_id, account_type, account_id, account_login, repository_selection, permissions)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, installation_id, account_type, account_id, account_login,
		          repository_selection, permissions, suspended_at, created_at, updated_at
	`

	permissionsJSON, err := json.Marshal(input.Permissions)
	if err != nil {
		return nil, err
	}

	repoSelection := input.RepositorySelection
	if repoSelection == "" {
		repoSelection = "selected"
	}

	var inst domain.Installation
	var repoSelectionOut sql.NullString
	var permissionsOut []byte
	var suspendedAt sql.NullTime

	err = r.db.QueryRowContext(ctx, query,
		input.InstallationID,
		input.AccountType,
		input.AccountID,
		input.AccountLogin,
		repoSelection,
		permissionsJSON,
	).Scan(
		&inst.ID,
		&inst.InstallationID,
		&inst.AccountType,
		&inst.AccountID,
		&inst.AccountLogin,
		&repoSelectionOut,
		&permissionsOut,
		&suspendedAt,
		&inst.CreatedAt,
		&inst.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	inst.RepositorySelection = repoSelectionOut.String
	if suspendedAt.Valid {
		inst.SuspendedAt = &suspendedAt.Time
	}

	if len(permissionsOut) > 0 {
		if err := json.Unmarshal(permissionsOut, &inst.Permissions); err != nil {
			return nil, err
		}
	} else {
		inst.Permissions = make(map[string]string)
	}

	return &inst, nil
}

func (r *PostgresInstallationRepository) Update(ctx context.Context, id string, input domain.UpdateInstallationInput) (*domain.Installation, error) {
	query := `
		UPDATE github_installations SET
			repository_selection = COALESCE($2, repository_selection),
			permissions = COALESCE($3, permissions),
			suspended_at = $4,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, installation_id, account_type, account_id, account_login,
		          repository_selection, permissions, suspended_at, created_at, updated_at
	`

	var repoSelNull sql.NullString
	if input.RepositorySelection != nil {
		repoSelNull = sql.NullString{String: *input.RepositorySelection, Valid: true}
	}

	var permissionsJSON []byte
	if input.Permissions != nil {
		var err error
		permissionsJSON, err = json.Marshal(input.Permissions)
		if err != nil {
			return nil, err
		}
	}

	var suspendedAtNull sql.NullTime
	if input.SuspendedAt != nil {
		suspendedAtNull = sql.NullTime{Time: *input.SuspendedAt, Valid: true}
	}

	var inst domain.Installation
	var repoSelection sql.NullString
	var permissionsOut []byte
	var suspendedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query,
		id,
		repoSelNull,
		permissionsJSON,
		suspendedAtNull,
	).Scan(
		&inst.ID,
		&inst.InstallationID,
		&inst.AccountType,
		&inst.AccountID,
		&inst.AccountLogin,
		&repoSelection,
		&permissionsOut,
		&suspendedAt,
		&inst.CreatedAt,
		&inst.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	inst.RepositorySelection = repoSelection.String
	if suspendedAt.Valid {
		inst.SuspendedAt = &suspendedAt.Time
	}

	if len(permissionsOut) > 0 {
		if err := json.Unmarshal(permissionsOut, &inst.Permissions); err != nil {
			return nil, err
		}
	} else {
		inst.Permissions = make(map[string]string)
	}

	return &inst, nil
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
		SELECT gi.id, gi.installation_id, gi.account_type, gi.account_id, gi.account_login,
		       gi.repository_selection, gi.permissions, gi.suspended_at, gi.created_at, gi.updated_at
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
		var inst domain.Installation
		var repoSelection sql.NullString
		var permissionsJSON []byte
		var suspendedAt sql.NullTime

		err := rows.Scan(
			&inst.ID,
			&inst.InstallationID,
			&inst.AccountType,
			&inst.AccountID,
			&inst.AccountLogin,
			&repoSelection,
			&permissionsJSON,
			&suspendedAt,
			&inst.CreatedAt,
			&inst.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		inst.RepositorySelection = repoSelection.String
		if suspendedAt.Valid {
			inst.SuspendedAt = &suspendedAt.Time
		}

		if len(permissionsJSON) > 0 {
			if err := json.Unmarshal(permissionsJSON, &inst.Permissions); err != nil {
				return nil, err
			}
		} else {
			inst.Permissions = make(map[string]string)
		}

		installations = append(installations, inst)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return installations, nil
}

func (r *PostgresInstallationRepository) FindDefaultInstallation(ctx context.Context, userID string) (*domain.Installation, error) {
	query := `
		SELECT gi.id, gi.installation_id, gi.account_type, gi.account_id, gi.account_login,
		       gi.repository_selection, gi.permissions, gi.suspended_at, gi.created_at, gi.updated_at
		FROM github_installations gi
		INNER JOIN user_installations ui ON gi.id = ui.installation_id
		WHERE ui.user_id = $1 AND ui.is_default = true
		LIMIT 1
	`

	var inst domain.Installation
	var repoSelection sql.NullString
	var permissionsJSON []byte
	var suspendedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&inst.ID,
		&inst.InstallationID,
		&inst.AccountType,
		&inst.AccountID,
		&inst.AccountLogin,
		&repoSelection,
		&permissionsJSON,
		&suspendedAt,
		&inst.CreatedAt,
		&inst.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	inst.RepositorySelection = repoSelection.String
	if suspendedAt.Valid {
		inst.SuspendedAt = &suspendedAt.Time
	}

	if len(permissionsJSON) > 0 {
		if err := json.Unmarshal(permissionsJSON, &inst.Permissions); err != nil {
			return nil, err
		}
	} else {
		inst.Permissions = make(map[string]string)
	}

	return &inst, nil
}

func (r *PostgresInstallationRepository) SetDefaultInstallation(ctx context.Context, userID, installationID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Remove default from all user installations
	_, err = tx.ExecContext(ctx, `UPDATE user_installations SET is_default = false WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	// Set new default
	_, err = tx.ExecContext(ctx, `UPDATE user_installations SET is_default = true WHERE user_id = $1 AND installation_id = $2`, userID, installationID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

var _ domain.InstallationRepository = (*PostgresInstallationRepository)(nil)
