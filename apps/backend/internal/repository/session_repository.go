package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/paasdeploy/backend/internal/domain"
)

type PostgresSessionRepository struct {
	db *sql.DB
}

func NewPostgresSessionRepository(db *sql.DB) *PostgresSessionRepository {
	return &PostgresSessionRepository{db: db}
}

func (r *PostgresSessionRepository) Create(ctx context.Context, input domain.CreateSessionInput) (*domain.Session, error) {
	query := `
		INSERT INTO sessions (user_id, token_hash, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, token_hash, ip_address, user_agent, expires_at, created_at
	`

	var session domain.Session
	var ipAddress, userAgent sql.NullString

	ipNull := sql.NullString{String: input.IPAddress, Valid: input.IPAddress != ""}
	uaNull := sql.NullString{String: input.UserAgent, Valid: input.UserAgent != ""}

	err := r.db.QueryRowContext(ctx, query,
		input.UserID,
		input.TokenHash,
		ipNull,
		uaNull,
		input.ExpiresAt,
	).Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&ipAddress,
		&userAgent,
		&session.ExpiresAt,
		&session.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	session.IPAddress = ipAddress.String
	session.UserAgent = userAgent.String

	return &session, nil
}

func (r *PostgresSessionRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	query := `
		SELECT id, user_id, token_hash, ip_address, user_agent, expires_at, created_at
		FROM sessions
		WHERE token_hash = $1 AND expires_at > NOW()
	`

	var session domain.Session
	var ipAddress, userAgent sql.NullString

	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&ipAddress,
		&userAgent,
		&session.ExpiresAt,
		&session.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	session.IPAddress = ipAddress.String
	session.UserAgent = userAgent.String

	return &session, nil
}

func (r *PostgresSessionRepository) FindByUserID(ctx context.Context, userID string) ([]domain.Session, error) {
	query := `
		SELECT id, user_id, token_hash, ip_address, user_agent, expires_at, created_at
		FROM sessions
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []domain.Session
	for rows.Next() {
		var session domain.Session
		var ipAddress, userAgent sql.NullString

		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.TokenHash,
			&ipAddress,
			&userAgent,
			&session.ExpiresAt,
			&session.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		session.IPAddress = ipAddress.String
		session.UserAgent = userAgent.String
		sessions = append(sessions, session)
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
