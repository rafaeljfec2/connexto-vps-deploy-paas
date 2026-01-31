package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/paasdeploy/backend/internal/domain"
)

const cloudflareConnectionSelectColumns = `id, user_id, cloudflare_account_id, cloudflare_email,
	access_token_encrypted, refresh_token_encrypted, token_expires_at,
	created_at, updated_at`

type PostgresCloudflareConnectionRepository struct {
	db *sql.DB
}

func NewPostgresCloudflareConnectionRepository(db *sql.DB) *PostgresCloudflareConnectionRepository {
	return &PostgresCloudflareConnectionRepository{db: db}
}

type cloudflareConnectionScanFields struct {
	conn           domain.CloudflareConnection
	email          sql.NullString
	refreshToken   sql.NullString
	tokenExpiresAt sql.NullTime
}

func (f *cloudflareConnectionScanFields) scanDest() []any {
	return []any{
		&f.conn.ID,
		&f.conn.UserID,
		&f.conn.CloudflareAccountID,
		&f.email,
		&f.conn.AccessTokenEncrypted,
		&f.refreshToken,
		&f.tokenExpiresAt,
		&f.conn.CreatedAt,
		&f.conn.UpdatedAt,
	}
}

func (f *cloudflareConnectionScanFields) toConnection() *domain.CloudflareConnection {
	f.conn.CloudflareEmail = fromNullString(f.email)
	f.conn.RefreshTokenEncrypted = fromNullString(f.refreshToken)
	f.conn.TokenExpiresAt = fromNullTime(f.tokenExpiresAt)
	return &f.conn
}

func (r *PostgresCloudflareConnectionRepository) scanConnection(row *sql.Row) (*domain.CloudflareConnection, error) {
	var f cloudflareConnectionScanFields
	if err := row.Scan(f.scanDest()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return f.toConnection(), nil
}

func (r *PostgresCloudflareConnectionRepository) Create(ctx context.Context, input domain.CreateCloudflareConnectionInput) (*domain.CloudflareConnection, error) {
	query := `
		INSERT INTO cloudflare_connections (user_id, cloudflare_account_id, cloudflare_email,
			access_token_encrypted, refresh_token_encrypted, token_expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + cloudflareConnectionSelectColumns

	return r.scanConnection(r.db.QueryRowContext(ctx, query,
		input.UserID,
		input.CloudflareAccountID,
		toNullStringValue(input.CloudflareEmail),
		input.AccessTokenEncrypted,
		toNullStringValue(input.RefreshTokenEncrypted),
		toNullTime(input.TokenExpiresAt),
	))
}

func (r *PostgresCloudflareConnectionRepository) FindByUserID(ctx context.Context, userID string) (*domain.CloudflareConnection, error) {
	query := `SELECT ` + cloudflareConnectionSelectColumns + ` FROM cloudflare_connections WHERE user_id = $1`
	return r.scanConnection(r.db.QueryRowContext(ctx, query, userID))
}

func (r *PostgresCloudflareConnectionRepository) Update(ctx context.Context, conn *domain.CloudflareConnection) error {
	query := `
		UPDATE cloudflare_connections
		SET cloudflare_account_id = $2,
			cloudflare_email = $3,
			access_token_encrypted = $4,
			refresh_token_encrypted = $5,
			token_expires_at = $6,
			updated_at = NOW()
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		conn.ID,
		conn.CloudflareAccountID,
		toNullStringValue(conn.CloudflareEmail),
		conn.AccessTokenEncrypted,
		toNullStringValue(conn.RefreshTokenEncrypted),
		toNullTime(conn.TokenExpiresAt),
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PostgresCloudflareConnectionRepository) Upsert(ctx context.Context, input domain.CreateCloudflareConnectionInput) (*domain.CloudflareConnection, error) {
	query := `
		INSERT INTO cloudflare_connections (user_id, cloudflare_account_id, cloudflare_email,
			access_token_encrypted, refresh_token_encrypted, token_expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			cloudflare_account_id = EXCLUDED.cloudflare_account_id,
			cloudflare_email = EXCLUDED.cloudflare_email,
			access_token_encrypted = EXCLUDED.access_token_encrypted,
			refresh_token_encrypted = EXCLUDED.refresh_token_encrypted,
			token_expires_at = EXCLUDED.token_expires_at,
			updated_at = NOW()
		RETURNING ` + cloudflareConnectionSelectColumns

	return r.scanConnection(r.db.QueryRowContext(ctx, query,
		input.UserID,
		input.CloudflareAccountID,
		toNullStringValue(input.CloudflareEmail),
		input.AccessTokenEncrypted,
		toNullStringValue(input.RefreshTokenEncrypted),
		toNullTime(input.TokenExpiresAt),
	))
}

func (r *PostgresCloudflareConnectionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	query := `DELETE FROM cloudflare_connections WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

var _ domain.CloudflareConnectionRepository = (*PostgresCloudflareConnectionRepository)(nil)
