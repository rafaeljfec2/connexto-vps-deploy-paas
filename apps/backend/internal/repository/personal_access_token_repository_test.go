package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/paasdeploy/backend/internal/domain"
)

func newPATRepoWithMock(t *testing.T) (*PostgresPersonalAccessTokenRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("open sqlmock: %v", err)
	}
	repo := NewPostgresPersonalAccessTokenRepository(db)
	cleanup := func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
		_ = db.Close()
	}
	return repo, mock, cleanup
}

func TestPostgresPATRepositoryCreatePersistsScopesAsJSONB(t *testing.T) {
	repo, mock, cleanup := newPATRepoWithMock(t)
	defer cleanup()

	expiry := time.Now().Add(72 * time.Hour).UTC()
	input := domain.CreatePersonalAccessTokenInput{
		UserID:      "11111111-1111-1111-1111-111111111111",
		Name:        "ci-bot",
		TokenHash:   "deadbeef",
		TokenPrefix: "pdp_live_",
		Scopes:      []string{"read", "deploy"},
		ExpiresAt:   &expiry,
	}

	now := time.Now().UTC()
	scopesPayload, _ := json.Marshal(input.Scopes)

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "name", "token_hash", "token_prefix",
		"scopes", "last_used_at", "expires_at", "revoked_at", "created_at", "updated_at",
	}).AddRow(
		"22222222-2222-2222-2222-222222222222",
		input.UserID,
		input.Name,
		input.TokenHash,
		input.TokenPrefix,
		scopesPayload,
		nil,
		expiry,
		nil,
		now,
		now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`INSERT INTO personal_access_tokens (user_id, name, token_hash, token_prefix, scopes, expires_at)`,
	)).
		WithArgs(input.UserID, input.Name, input.TokenHash, input.TokenPrefix, scopesPayload, sqlmock.AnyArg()).
		WillReturnRows(rows)

	got, err := repo.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if got.ID != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("unexpected id: %s", got.ID)
	}
	if len(got.Scopes) != 2 || got.Scopes[0] != "read" || got.Scopes[1] != "deploy" {
		t.Fatalf("scopes round-trip failed: %#v", got.Scopes)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(expiry) {
		t.Fatalf("expires_at round-trip failed: %v vs %v", got.ExpiresAt, expiry)
	}
}

func TestPostgresPATRepositoryCreateMarshalsEmptyScopesAsEmptyJSONArray(t *testing.T) {
	repo, mock, cleanup := newPATRepoWithMock(t)
	defer cleanup()

	input := domain.CreatePersonalAccessTokenInput{
		UserID:      "11111111-1111-1111-1111-111111111111",
		Name:        "no-scopes",
		TokenHash:   "deadbeef",
		TokenPrefix: "pdp_live_",
		Scopes:      []string{},
	}

	expectedScopes := []byte(`[]`)
	rows := sqlmock.NewRows([]string{
		"id", "user_id", "name", "token_hash", "token_prefix",
		"scopes", "last_used_at", "expires_at", "revoked_at", "created_at", "updated_at",
	}).AddRow(
		"22222222-2222-2222-2222-222222222222",
		input.UserID,
		input.Name,
		input.TokenHash,
		input.TokenPrefix,
		expectedScopes,
		nil, nil, nil,
		time.Now().UTC(), time.Now().UTC(),
	)

	mock.ExpectQuery(`INSERT INTO personal_access_tokens`).
		WithArgs(input.UserID, input.Name, input.TokenHash, input.TokenPrefix, expectedScopes, sqlmock.AnyArg()).
		WillReturnRows(rows)

	got, err := repo.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if got.Scopes == nil || len(got.Scopes) != 0 {
		t.Fatalf("expected non-nil empty scopes, got %#v", got.Scopes)
	}
}

func TestPostgresPATRepositoryFindByTokenHashReturnsNotFound(t *testing.T) {
	repo, mock, cleanup := newPATRepoWithMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT .* FROM personal_access_tokens WHERE token_hash`).
		WithArgs("missing-hash").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.FindByTokenHash(context.Background(), "missing-hash")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestPostgresPATRepositoryListByUserIDReturnsRows(t *testing.T) {
	repo, mock, cleanup := newPATRepoWithMock(t)
	defer cleanup()

	now := time.Now().UTC()
	scopesA, _ := json.Marshal([]string{"read"})
	scopesB, _ := json.Marshal([]string{"deploy"})

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "name", "token_hash", "token_prefix",
		"scopes", "last_used_at", "expires_at", "revoked_at", "created_at", "updated_at",
	}).
		AddRow("aaaa", "user-1", "first", "hash-a", "pdp_live_", scopesA, nil, nil, nil, now, now).
		AddRow("bbbb", "user-1", "second", "hash-b", "pdp_live_", scopesB, nil, nil, nil, now, now)

	mock.ExpectQuery(`SELECT .* FROM personal_access_tokens WHERE user_id`).
		WithArgs("user-1").
		WillReturnRows(rows)

	tokens, err := repo.ListByUserID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}
	if tokens[0].Scopes[0] != "read" || tokens[1].Scopes[0] != "deploy" {
		t.Fatalf("scopes mismatch: %#v / %#v", tokens[0].Scopes, tokens[1].Scopes)
	}
}

func TestPostgresPATRepositoryRevokeReturnsNotFoundOnZeroRows(t *testing.T) {
	repo, mock, cleanup := newPATRepoWithMock(t)
	defer cleanup()

	mock.ExpectExec(`UPDATE personal_access_tokens SET revoked_at = NOW\(\)`).
		WithArgs("missing-id", "user-1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Revoke(context.Background(), "missing-id", "user-1")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected domain.ErrNotFound when no rows affected, got %v", err)
	}
}

func TestPostgresPATRepositoryRevokeSucceedsOnAffectedRow(t *testing.T) {
	repo, mock, cleanup := newPATRepoWithMock(t)
	defer cleanup()

	mock.ExpectExec(`UPDATE personal_access_tokens SET revoked_at = NOW\(\)`).
		WithArgs("tok-1", "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.Revoke(context.Background(), "tok-1", "user-1"); err != nil {
		t.Fatalf("expected revoke success, got %v", err)
	}
}

func TestPostgresPATRepositoryTouchLastUsedExecutesUpdate(t *testing.T) {
	repo, mock, cleanup := newPATRepoWithMock(t)
	defer cleanup()

	mock.ExpectExec(`UPDATE personal_access_tokens SET last_used_at = NOW\(\)`).
		WithArgs("tok-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.TouchLastUsed(context.Background(), "tok-1"); err != nil {
		t.Fatalf("expected touch success, got %v", err)
	}
}
