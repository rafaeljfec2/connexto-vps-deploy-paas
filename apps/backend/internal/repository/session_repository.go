package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/paasdeploy/backend/internal/domain"
)

const sessionSelectColumns = `id, user_id, token_hash, ip_address, user_agent, expires_at, created_at`

type PostgresSessionRepository struct {
	db *sql.DB
}

func NewPostgresSessionRepository(db *sql.DB) *PostgresSessionRepository {
	return &PostgresSessionRepository{db: db}
}

type sessionScanFields struct {
	session   domain.Session
	ipAddress sql.NullString
	userAgent sql.NullString
}

func (f *sessionScanFields) scanDest() []any {
	return []any{
		&f.session.ID,
		&f.session.UserID,
		&f.session.TokenHash,
		&f.ipAddress,
		&f.userAgent,
		&f.session.ExpiresAt,
		&f.session.CreatedAt,
	}
}

func (f *sessionScanFields) toSession() domain.Session {
	f.session.IPAddress = fromNullString(f.ipAddress)
	f.session.UserAgent = fromNullString(f.userAgent)
	return f.session
}

func (r *PostgresSessionRepository) scanSession(row *sql.Row) (*domain.Session, error) {
	var f sessionScanFields
	if err := row.Scan(f.scanDest()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	s := f.toSession()
	return &s, nil
}

func (r *PostgresSessionRepository) Create(ctx context.Context, input domain.CreateSessionInput) (*domain.Session, error) {
	query := `
		INSERT INTO sessions (user_id, token_hash, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + sessionSelectColumns

	row := r.db.QueryRowContext(ctx, query,
		input.UserID,
		input.TokenHash,
		toNullStringValue(input.IPAddress),
		toNullStringValue(input.UserAgent),
		input.ExpiresAt,
	)

	var f sessionScanFields
	if err := row.Scan(f.scanDest()...); err != nil {
		return nil, err
	}
	s := f.toSession()
	return &s, nil
}

func (r *PostgresSessionRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	query := `SELECT ` + sessionSelectColumns + ` FROM sessions WHERE token_hash = $1 AND expires_at > NOW()`
	return r.scanSession(r.db.QueryRowContext(ctx, query, tokenHash))
}

func (r *PostgresSessionRepository) FindByUserID(ctx context.Context, userID string) ([]domain.Session, error) {
	query := `SELECT ` + sessionSelectColumns + ` FROM sessions WHERE user_id = $1 AND expires_at > NOW() ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []domain.Session
	for rows.Next() {
		var f sessionScanFields
		if err := rows.Scan(f.scanDest()...); err != nil {
			return nil, err
		}
		sessions = append(sessions, f.toSession())
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}

func (r *PostgresSessionRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM sessions WHERE id = $1`

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

func (r *PostgresSessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	query := `DELETE FROM sessions WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

func (r *PostgresSessionRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM sessions WHERE expires_at <= NOW()`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

var _ domain.SessionRepository = (*PostgresSessionRepository)(nil)
