package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/paasdeploy/backend/internal/domain"
)

const patSelectColumns = `id, user_id, name, token_hash, token_prefix, scopes, last_used_at, expires_at, revoked_at, created_at, updated_at`

type PostgresPersonalAccessTokenRepository struct {
	db *sql.DB
}

func NewPostgresPersonalAccessTokenRepository(db *sql.DB) *PostgresPersonalAccessTokenRepository {
	return &PostgresPersonalAccessTokenRepository{db: db}
}

type patScanFields struct {
	token      domain.PersonalAccessToken
	scopesJSON []byte
	lastUsedAt sql.NullTime
	expiresAt  sql.NullTime
	revokedAt  sql.NullTime
}

func (f *patScanFields) scanDest() []any {
	return []any{
		&f.token.ID,
		&f.token.UserID,
		&f.token.Name,
		&f.token.TokenHash,
		&f.token.TokenPrefix,
		&f.scopesJSON,
		&f.lastUsedAt,
		&f.expiresAt,
		&f.revokedAt,
		&f.token.CreatedAt,
		&f.token.UpdatedAt,
	}
}

func (f *patScanFields) toToken() (domain.PersonalAccessToken, error) {
	scopes := []string{}
	if len(f.scopesJSON) > 0 {
		if err := json.Unmarshal(f.scopesJSON, &scopes); err != nil {
			return domain.PersonalAccessToken{}, err
		}
	}
	f.token.Scopes = scopes
	f.token.LastUsedAt = fromNullTime(f.lastUsedAt)
	f.token.ExpiresAt = fromNullTime(f.expiresAt)
	f.token.RevokedAt = fromNullTime(f.revokedAt)
	return f.token, nil
}

func (r *PostgresPersonalAccessTokenRepository) scanRow(row *sql.Row) (*domain.PersonalAccessToken, error) {
	var f patScanFields
	if err := row.Scan(f.scanDest()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	token, err := f.toToken()
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *PostgresPersonalAccessTokenRepository) Create(ctx context.Context, input domain.CreatePersonalAccessTokenInput) (*domain.PersonalAccessToken, error) {
	scopesJSON, err := json.Marshal(input.Scopes)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO personal_access_tokens (user_id, name, token_hash, token_prefix, scopes, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + patSelectColumns

	row := r.db.QueryRowContext(ctx, query,
		input.UserID,
		input.Name,
		input.TokenHash,
		input.TokenPrefix,
		scopesJSON,
		toNullTime(input.ExpiresAt),
	)
	return r.scanRow(row)
}

func (r *PostgresPersonalAccessTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.PersonalAccessToken, error) {
	query := `SELECT ` + patSelectColumns + ` FROM personal_access_tokens WHERE token_hash = $1`
	return r.scanRow(r.db.QueryRowContext(ctx, query, tokenHash))
}

func (r *PostgresPersonalAccessTokenRepository) FindByID(ctx context.Context, id string) (*domain.PersonalAccessToken, error) {
	query := `SELECT ` + patSelectColumns + ` FROM personal_access_tokens WHERE id = $1`
	return r.scanRow(r.db.QueryRowContext(ctx, query, id))
}

func (r *PostgresPersonalAccessTokenRepository) ListByUserID(ctx context.Context, userID string) ([]domain.PersonalAccessToken, error) {
	query := `SELECT ` + patSelectColumns + ` FROM personal_access_tokens WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := []domain.PersonalAccessToken{}
	for rows.Next() {
		var f patScanFields
		if err := rows.Scan(f.scanDest()...); err != nil {
			return nil, err
		}
		token, err := f.toToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tokens, nil
}

func (r *PostgresPersonalAccessTokenRepository) Revoke(ctx context.Context, id string, userID string) error {
	query := `UPDATE personal_access_tokens SET revoked_at = NOW() WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PostgresPersonalAccessTokenRepository) TouchLastUsed(ctx context.Context, id string) error {
	query := `UPDATE personal_access_tokens SET last_used_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

var _ domain.PersonalAccessTokenRepository = (*PostgresPersonalAccessTokenRepository)(nil)
