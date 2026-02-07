package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/paasdeploy/backend/internal/domain"
)

const userSelectColumns = `id, github_id, github_login, name, email, avatar_url,
	access_token_encrypted, refresh_token_encrypted, token_expires_at,
	password_hash, auth_provider, created_at, updated_at`

type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

type userScanFields struct {
	user           domain.User
	githubID       sql.NullInt64
	name           sql.NullString
	email          sql.NullString
	avatarURL      sql.NullString
	accessToken    sql.NullString
	refreshToken   sql.NullString
	tokenExpiresAt sql.NullTime
	passwordHash   sql.NullString
}

func (f *userScanFields) scanDest() []any {
	return []any{
		&f.user.ID,
		&f.githubID,
		&f.user.GitHubLogin,
		&f.name,
		&f.email,
		&f.avatarURL,
		&f.accessToken,
		&f.refreshToken,
		&f.tokenExpiresAt,
		&f.passwordHash,
		&f.user.AuthProvider,
		&f.user.CreatedAt,
		&f.user.UpdatedAt,
	}
}

func (f *userScanFields) toUser() *domain.User {
	if f.githubID.Valid {
		f.user.GitHubID = &f.githubID.Int64
	}
	f.user.Name = fromNullString(f.name)
	f.user.Email = fromNullString(f.email)
	f.user.AvatarURL = fromNullString(f.avatarURL)
	f.user.AccessTokenEncrypted = fromNullString(f.accessToken)
	f.user.RefreshTokenEncrypted = fromNullString(f.refreshToken)
	f.user.TokenExpiresAt = fromNullTime(f.tokenExpiresAt)
	f.user.PasswordHash = fromNullString(f.passwordHash)
	return &f.user
}

func (r *PostgresUserRepository) scanUser(row *sql.Row) (*domain.User, error) {
	var f userScanFields
	if err := row.Scan(f.scanDest()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return f.toUser(), nil
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE id = $1`
	return r.scanUser(r.db.QueryRowContext(ctx, query, id))
}

func (r *PostgresUserRepository) FindByGitHubID(ctx context.Context, githubID int64) (*domain.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE github_id = $1`
	return r.scanUser(r.db.QueryRowContext(ctx, query, githubID))
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE email = $1`
	return r.scanUser(r.db.QueryRowContext(ctx, query, email))
}

func (r *PostgresUserRepository) Create(ctx context.Context, input domain.CreateUserInput) (*domain.User, error) {
	query := `
		INSERT INTO users (github_id, github_login, name, email, avatar_url,
		                   access_token_encrypted, refresh_token_encrypted, token_expires_at,
		                   auth_provider)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'github')
		RETURNING ` + userSelectColumns

	row := r.db.QueryRowContext(ctx, query,
		input.GitHubID,
		input.GitHubLogin,
		toNullStringValue(input.Name),
		toNullStringValue(input.Email),
		toNullStringValue(input.AvatarURL),
		input.AccessTokenEncrypted,
		toNullStringValue(input.RefreshTokenEncrypted),
		toNullTime(input.TokenExpiresAt),
	)

	var f userScanFields
	if err := row.Scan(f.scanDest()...); err != nil {
		return nil, err
	}
	return f.toUser(), nil
}

func (r *PostgresUserRepository) CreateEmailUser(ctx context.Context, input domain.CreateEmailUserInput) (*domain.User, error) {
	query := `
		INSERT INTO users (email, name, password_hash, auth_provider)
		VALUES ($1, $2, $3, 'email')
		RETURNING ` + userSelectColumns

	row := r.db.QueryRowContext(ctx, query,
		input.Email,
		toNullStringValue(input.Name),
		input.PasswordHash,
	)

	var f userScanFields
	if err := row.Scan(f.scanDest()...); err != nil {
		return nil, err
	}
	return f.toUser(), nil
}

func (r *PostgresUserRepository) LinkGitHub(ctx context.Context, userID string, input domain.LinkGitHubInput) (*domain.User, error) {
	query := `
		UPDATE users SET
			github_id = $2,
			github_login = $3,
			name = COALESCE(NULLIF(name, ''), $4),
			avatar_url = COALESCE(avatar_url, $5),
			access_token_encrypted = $6,
			refresh_token_encrypted = $7,
			token_expires_at = $8,
			updated_at = NOW()
		WHERE id = $1
		RETURNING ` + userSelectColumns

	row := r.db.QueryRowContext(ctx, query,
		userID,
		input.GitHubID,
		input.GitHubLogin,
		toNullStringValue(input.Name),
		toNullStringValue(input.AvatarURL),
		input.AccessTokenEncrypted,
		toNullStringValue(input.RefreshTokenEncrypted),
		toNullTime(input.TokenExpiresAt),
	)

	return r.scanUser(row)
}

func (r *PostgresUserRepository) Update(ctx context.Context, id string, input domain.UpdateUserInput) (*domain.User, error) {
	query := `
		UPDATE users SET
			github_login = COALESCE($2, github_login),
			name = COALESCE($3, name),
			email = COALESCE($4, email),
			avatar_url = COALESCE($5, avatar_url),
			access_token_encrypted = COALESCE($6, access_token_encrypted),
			refresh_token_encrypted = COALESCE($7, refresh_token_encrypted),
			token_expires_at = COALESCE($8, token_expires_at),
			updated_at = NOW()
		WHERE id = $1
		RETURNING ` + userSelectColumns

	row := r.db.QueryRowContext(ctx, query,
		id,
		toNullString(input.GitHubLogin),
		toNullString(input.Name),
		toNullString(input.Email),
		toNullString(input.AvatarURL),
		toNullString(input.AccessTokenEncrypted),
		toNullString(input.RefreshTokenEncrypted),
		toNullTime(input.TokenExpiresAt),
	)

	return r.scanUser(row)
}

func (r *PostgresUserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`

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

var _ domain.UserRepository = (*PostgresUserRepository)(nil)
