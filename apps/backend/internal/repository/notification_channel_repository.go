package repository

import (
	"database/sql"
	"errors"

	"github.com/paasdeploy/backend/internal/domain"
)

const notificationChannelSelectColumns = `id, user_id, type, name, config, app_id, created_at, updated_at`

type PostgresNotificationChannelRepository struct {
	db *sql.DB
}

func NewPostgresNotificationChannelRepository(db *sql.DB) *PostgresNotificationChannelRepository {
	return &PostgresNotificationChannelRepository{db: db}
}

func (r *PostgresNotificationChannelRepository) scanChannel(row *sql.Row) (*domain.NotificationChannel, error) {
	var ch domain.NotificationChannel
	var appID sql.NullString
	err := row.Scan(
		&ch.ID,
		&ch.UserID,
		&ch.Type,
		&ch.Name,
		&ch.Config,
		&appID,
		&ch.CreatedAt,
		&ch.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if appID.Valid {
		ch.AppID = &appID.String
	}
	return &ch, nil
}

func (r *PostgresNotificationChannelRepository) scanChannelRows(rows *sql.Rows) ([]domain.NotificationChannel, error) {
	defer rows.Close()
	var channels []domain.NotificationChannel
	for rows.Next() {
		var ch domain.NotificationChannel
		var appID sql.NullString
		if err := rows.Scan(
			&ch.ID,
			&ch.UserID,
			&ch.Type,
			&ch.Name,
			&ch.Config,
			&appID,
			&ch.CreatedAt,
			&ch.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if appID.Valid {
			ch.AppID = &appID.String
		}
		channels = append(channels, ch)
	}
	return channels, rows.Err()
}

func (r *PostgresNotificationChannelRepository) FindAll() ([]domain.NotificationChannel, error) {
	query := `SELECT ` + notificationChannelSelectColumns + ` FROM notification_channels ORDER BY created_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	return r.scanChannelRows(rows)
}

func (r *PostgresNotificationChannelRepository) FindAllByUserID(userID string) ([]domain.NotificationChannel, error) {
	query := `SELECT ` + notificationChannelSelectColumns + ` FROM notification_channels WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	return r.scanChannelRows(rows)
}

func (r *PostgresNotificationChannelRepository) FindByID(id string) (*domain.NotificationChannel, error) {
	query := `SELECT ` + notificationChannelSelectColumns + ` FROM notification_channels WHERE id = $1`
	return r.scanChannel(r.db.QueryRow(query, id))
}

func (r *PostgresNotificationChannelRepository) FindByIDAndUserID(id string, userID string) (*domain.NotificationChannel, error) {
	query := `SELECT ` + notificationChannelSelectColumns + ` FROM notification_channels WHERE id = $1 AND user_id = $2`
	return r.scanChannel(r.db.QueryRow(query, id, userID))
}

func (r *PostgresNotificationChannelRepository) FindByAppID(appID string) ([]domain.NotificationChannel, error) {
	query := `SELECT ` + notificationChannelSelectColumns + ` FROM notification_channels WHERE app_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(query, appID)
	if err != nil {
		return nil, err
	}
	return r.scanChannelRows(rows)
}

func (r *PostgresNotificationChannelRepository) FindGlobal() ([]domain.NotificationChannel, error) {
	query := `SELECT ` + notificationChannelSelectColumns + ` FROM notification_channels WHERE app_id IS NULL ORDER BY created_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	return r.scanChannelRows(rows)
}

func (r *PostgresNotificationChannelRepository) Create(input domain.CreateNotificationChannelInput) (*domain.NotificationChannel, error) {
	query := `
		INSERT INTO notification_channels (user_id, type, name, config, app_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING ` + notificationChannelSelectColumns
	return r.scanChannel(r.db.QueryRow(query, input.UserID, input.Type, input.Name, input.Config, toNullString(input.AppID)))
}

func (r *PostgresNotificationChannelRepository) Update(id string, input domain.UpdateNotificationChannelInput) (*domain.NotificationChannel, error) {
	query := `
		UPDATE notification_channels
		SET name = COALESCE($2, name),
		    config = COALESCE($3, config),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING ` + notificationChannelSelectColumns
	var nameVal, configVal interface{}
	if input.Name != nil {
		nameVal = *input.Name
	}
	if input.Config != nil {
		configVal = *input.Config
	}
	return r.scanChannel(r.db.QueryRow(query, id, nameVal, configVal))
}

func (r *PostgresNotificationChannelRepository) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM notification_channels WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}
