package repository

import (
	"database/sql"

	"github.com/paasdeploy/backend/internal/domain"
)

type PostgresEnvVarRepository struct {
	db *sql.DB
}

func NewPostgresEnvVarRepository(db *sql.DB) *PostgresEnvVarRepository {
	return &PostgresEnvVarRepository{db: db}
}

func (r *PostgresEnvVarRepository) FindByAppID(appID string) ([]domain.EnvVar, error) {
	query := `
		SELECT id, app_id, key, value, is_secret, created_at, updated_at
		FROM app_env_vars
		WHERE app_id = $1
		ORDER BY key ASC
	`

	rows, err := r.db.Query(query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vars []domain.EnvVar
	for rows.Next() {
		var v domain.EnvVar
		err := rows.Scan(
			&v.ID,
			&v.AppID,
			&v.Key,
			&v.Value,
			&v.IsSecret,
			&v.CreatedAt,
			&v.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		vars = append(vars, v)
	}

	return vars, nil
}

func (r *PostgresEnvVarRepository) FindByAppIDAndKey(appID, key string) (*domain.EnvVar, error) {
	query := `
		SELECT id, app_id, key, value, is_secret, created_at, updated_at
		FROM app_env_vars
		WHERE app_id = $1 AND key = $2
	`

	var v domain.EnvVar
	err := r.db.QueryRow(query, appID, key).Scan(
		&v.ID,
		&v.AppID,
		&v.Key,
		&v.Value,
		&v.IsSecret,
		&v.CreatedAt,
		&v.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &v, nil
}

func (r *PostgresEnvVarRepository) Create(appID string, input domain.CreateEnvVarInput) (*domain.EnvVar, error) {
	query := `
		INSERT INTO app_env_vars (app_id, key, value, is_secret, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, app_id, key, value, is_secret, created_at, updated_at
	`

	var v domain.EnvVar
	err := r.db.QueryRow(query, appID, input.Key, input.Value, input.IsSecret).Scan(
		&v.ID,
		&v.AppID,
		&v.Key,
		&v.Value,
		&v.IsSecret,
		&v.CreatedAt,
		&v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

func (r *PostgresEnvVarRepository) Update(id string, input domain.UpdateEnvVarInput) (*domain.EnvVar, error) {
	query := `
		SELECT id, app_id, key, value, is_secret, created_at, updated_at
		FROM app_env_vars
		WHERE id = $1
	`

	var v domain.EnvVar
	err := r.db.QueryRow(query, id).Scan(
		&v.ID,
		&v.AppID,
		&v.Key,
		&v.Value,
		&v.IsSecret,
		&v.CreatedAt,
		&v.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if input.Value != nil {
		v.Value = *input.Value
	}
	if input.IsSecret != nil {
		v.IsSecret = *input.IsSecret
	}

	updateQuery := `
		UPDATE app_env_vars
		SET value = $2, is_secret = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err = r.db.QueryRow(updateQuery, id, v.Value, v.IsSecret).Scan(&v.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

func (r *PostgresEnvVarRepository) Delete(id string) error {
	query := `DELETE FROM app_env_vars WHERE id = $1`
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

func (r *PostgresEnvVarRepository) DeleteByAppIDAndKey(appID, key string) error {
	query := `DELETE FROM app_env_vars WHERE app_id = $1 AND key = $2`
	result, err := r.db.Exec(query, appID, key)
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

func (r *PostgresEnvVarRepository) BulkUpsert(appID string, vars []domain.CreateEnvVarInput) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	query := `
		INSERT INTO app_env_vars (app_id, key, value, is_secret, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (app_id, key) DO UPDATE
		SET value = EXCLUDED.value, is_secret = EXCLUDED.is_secret, updated_at = NOW()
	`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, v := range vars {
		_, err = stmt.Exec(appID, v.Key, v.Value, v.IsSecret)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
