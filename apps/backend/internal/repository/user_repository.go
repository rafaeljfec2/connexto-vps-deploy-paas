package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/paasdeploy/backend/internal/domain"
)

type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, github_id, github_login, name, email, avatar_url,
		       access_token_encrypted, refresh_token_encrypted, token_expires_at,
		       created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user domain.User
	var name, email, avatarURL, refreshToken sql.NullString
	var tokenExpiresAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.GitHubID,
		&user.GitHubLogin,
		&name,
		&email,
		&avatarURL,
		&user.AccessTokenEncrypted,
		&refreshToken,
		&tokenExpiresAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	user.Name = name.String
	user.Email = email.String
	user.AvatarURL = avatarURL.String
	user.RefreshTokenEncrypted = refreshToken.String
	if tokenExpiresAt.Valid {
		user.TokenExpiresAt = &tokenExpiresAt.Time
	}

	return &user, nil
}

func (r *PostgresUserRepository) FindByGitHubID(ctx context.Context, githubID int64) (*domain.User, error) {
	query := `
		SELECT id, github_id, github_login, name, email, avatar_url,
		       access_token_encrypted, refresh_token_encrypted, token_expires_at,
		       created_at, updated_at
		FROM users
		WHERE github_id = $1
	`

	var user domain.User
	var name, email, avatarURL, refreshToken sql.NullString
	var tokenExpiresAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, githubID).Scan(
		&user.ID,
		&user.GitHubID,
		&user.GitHubLogin,
		&name,
		&email,
		&avatarURL,
		&user.AccessTokenEncrypted,
		&refreshToken,
		&tokenExpiresAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	user.Name = name.String
	user.Email = email.String
	user.AvatarURL = avatarURL.String
	user.RefreshTokenEncrypted = refreshToken.String
	if tokenExpiresAt.Valid {
		user.TokenExpiresAt = &tokenExpiresAt.Time
	}

	return &user, nil
}

func (r *PostgresUserRepository) Create(ctx context.Context, input domain.CreateUserInput) (*domain.User, error) {
	query := `
		INSERT INTO users (github_id, github_login, name, email, avatar_url,
		                   access_token_encrypted, refresh_token_encrypted, token_expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, github_id, github_login, name, email, avatar_url,
		          access_token_encrypted, refresh_token_encrypted, token_expires_at,
		          created_at, updated_at
	`

	var user domain.User
	var name, email, avatarURL, refreshToken sql.NullString
	var tokenExpiresAt sql.NullTime

	nameNull := sql.NullString{String: input.Name, Valid: input.Name != ""}
	emailNull := sql.NullString{String: input.Email, Valid: input.Email != ""}
	avatarNull := sql.NullString{String: input.AvatarURL, Valid: input.AvatarURL != ""}
	refreshNull := sql.NullString{String: input.RefreshTokenEncrypted, Valid: input.RefreshTokenEncrypted != ""}
	var expiresNull sql.NullTime
	if input.TokenExpiresAt != nil {
		expiresNull = sql.NullTime{Time: *input.TokenExpiresAt, Valid: true}
	}

	err := r.db.QueryRowContext(ctx, query,
		input.GitHubID,
		input.GitHubLogin,
		nameNull,
		emailNull,
		avatarNull,
		input.AccessTokenEncrypted,
		refreshNull,
		expiresNull,
	).Scan(
		&user.ID,
		&user.GitHubID,
		&user.GitHubLogin,
		&name,
		&email,
		&avatarURL,
		&user.AccessTokenEncrypted,
		&refreshToken,
		&tokenExpiresAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	user.Name = name.String
	user.Email = email.String
	user.AvatarURL = avatarURL.String
	user.RefreshTokenEncrypted = refreshToken.String
	if tokenExpiresAt.Valid {
		user.TokenExpiresAt = &tokenExpiresAt.Time
	}

	return &user, nil
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
		RETURNING id, github_id, github_login, name, email, avatar_url,
		          access_token_encrypted, refresh_token_encrypted, token_expires_at,
		          created_at, updated_at
	`

	var user domain.User
	var name, email, avatarURL, refreshToken sql.NullString
	var tokenExpiresAt sql.NullTime

	var loginNull, nameInput, emailInput, avatarInput, accessInput, refreshInput sql.NullString
	var expiresInput sql.NullTime

	if input.GitHubLogin != nil {
		loginNull = sql.NullString{String: *input.GitHubLogin, Valid: true}
	}
	if input.Name != nil {
		nameInput = sql.NullString{String: *input.Name, Valid: true}
	}
	if input.Email != nil {
		emailInput = sql.NullString{String: *input.Email, Valid: true}
	}
	if input.AvatarURL != nil {
		avatarInput = sql.NullString{String: *input.AvatarURL, Valid: true}
	}
	if input.AccessTokenEncrypted != nil {
		accessInput = sql.NullString{String: *input.AccessTokenEncrypted, Valid: true}
	}
	if input.RefreshTokenEncrypted != nil {
		refreshInput = sql.NullString{String: *input.RefreshTokenEncrypted, Valid: true}
	}
	if input.TokenExpiresAt != nil {
		expiresInput = sql.NullTime{Time: *input.TokenExpiresAt, Valid: true}
	}

	err := r.db.QueryRowContext(ctx, query,
		id,
		loginNull,
		nameInput,
		emailInput,
		avatarInput,
		accessInput,
		refreshInput,
		expiresInput,
	).Scan(
		&user.ID,
		&user.GitHubID,
		&user.GitHubLogin,
		&name,
		&email,
		&avatarURL,
		&user.AccessTokenEncrypted,
		&refreshToken,
		&tokenExpiresAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	user.Name = name.String
	user.Email = email.String
	user.AvatarURL = avatarURL.String
	user.RefreshTokenEncrypted = refreshToken.String
	if tokenExpiresAt.Valid {
		user.TokenExpiresAt = &tokenExpiresAt.Time
	}

	return &user, nil
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
