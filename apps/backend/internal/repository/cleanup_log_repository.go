package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/paasdeploy/backend/internal/domain"
)

type PostgresCleanupLogRepository struct {
	db *sql.DB
}

func NewPostgresCleanupLogRepository(db *sql.DB) *PostgresCleanupLogRepository {
	return &PostgresCleanupLogRepository{db: db}
}

func (r *PostgresCleanupLogRepository) Create(input domain.CreateCleanupLogInput) (*domain.CleanupLog, error) {
	id := uuid.New().String()

	query := `
		INSERT INTO cleanup_logs (id, server_id, cleanup_type, items_removed, space_reclaimed_bytes, triggered_by, status, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, server_id, cleanup_type, items_removed, space_reclaimed_bytes, triggered_by, status, error_message, created_at
	`

	var log domain.CleanupLog
	var errorMsg sql.NullString

	err := r.db.QueryRow(
		query,
		id,
		input.ServerID,
		input.CleanupType,
		input.ItemsRemoved,
		input.SpaceReclaimedBytes,
		input.Trigger,
		input.Status,
		nullString(input.ErrorMessage),
	).Scan(
		&log.ID,
		&log.ServerID,
		&log.CleanupType,
		&log.ItemsRemoved,
		&log.SpaceReclaimedBytes,
		&log.Trigger,
		&log.Status,
		&errorMsg,
		&log.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cleanup log: %w", err)
	}

	if errorMsg.Valid {
		log.ErrorMessage = errorMsg.String
	}

	return &log, nil
}

func (r *PostgresCleanupLogRepository) FindByServerID(serverID string, limit int, offset int) ([]domain.CleanupLog, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, server_id, cleanup_type, items_removed, space_reclaimed_bytes, triggered_by, status, error_message, created_at
		FROM cleanup_logs
		WHERE server_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, serverID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query cleanup logs: %w", err)
	}
	defer rows.Close()

	var logs []domain.CleanupLog
	for rows.Next() {
		var log domain.CleanupLog
		var errorMsg sql.NullString

		if err := rows.Scan(
			&log.ID,
			&log.ServerID,
			&log.CleanupType,
			&log.ItemsRemoved,
			&log.SpaceReclaimedBytes,
			&log.Trigger,
			&log.Status,
			&errorMsg,
			&log.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan cleanup log: %w", err)
		}

		if errorMsg.Valid {
			log.ErrorMessage = errorMsg.String
		}

		logs = append(logs, log)
	}

	return logs, rows.Err()
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
