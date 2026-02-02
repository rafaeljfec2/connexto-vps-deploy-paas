package repository

import (
	"database/sql"
	"errors"

	"github.com/paasdeploy/backend/internal/domain"
)

const notificationRuleSelectColumns = `id, event_type, channel_id, app_id, enabled, created_at, updated_at`

type PostgresNotificationRuleRepository struct {
	db *sql.DB
}

func NewPostgresNotificationRuleRepository(db *sql.DB) *PostgresNotificationRuleRepository {
	return &PostgresNotificationRuleRepository{db: db}
}

func (r *PostgresNotificationRuleRepository) scanRule(row *sql.Row) (*domain.NotificationRule, error) {
	var rule domain.NotificationRule
	var appID sql.NullString
	err := row.Scan(
		&rule.ID,
		&rule.EventType,
		&rule.ChannelID,
		&appID,
		&rule.Enabled,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if appID.Valid {
		rule.AppID = &appID.String
	}
	return &rule, nil
}

func (r *PostgresNotificationRuleRepository) FindAll() ([]domain.NotificationRule, error) {
	query := `SELECT ` + notificationRuleSelectColumns + ` FROM notification_rules ORDER BY created_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []domain.NotificationRule
	for rows.Next() {
		var rule domain.NotificationRule
		var appID sql.NullString
		if err := rows.Scan(
			&rule.ID,
			&rule.EventType,
			&rule.ChannelID,
			&appID,
			&rule.Enabled,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if appID.Valid {
			rule.AppID = &appID.String
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *PostgresNotificationRuleRepository) FindByID(id string) (*domain.NotificationRule, error) {
	query := `SELECT ` + notificationRuleSelectColumns + ` FROM notification_rules WHERE id = $1`
	return r.scanRule(r.db.QueryRow(query, id))
}

func (r *PostgresNotificationRuleRepository) FindByChannelID(channelID string) ([]domain.NotificationRule, error) {
	query := `SELECT ` + notificationRuleSelectColumns + ` FROM notification_rules WHERE channel_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(query, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []domain.NotificationRule
	for rows.Next() {
		var rule domain.NotificationRule
		var appID sql.NullString
		if err := rows.Scan(
			&rule.ID,
			&rule.EventType,
			&rule.ChannelID,
			&appID,
			&rule.Enabled,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if appID.Valid {
			rule.AppID = &appID.String
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *PostgresNotificationRuleRepository) FindByEventType(eventType string) ([]domain.NotificationRule, error) {
	query := `SELECT ` + notificationRuleSelectColumns + ` FROM notification_rules WHERE event_type = $1 AND enabled = true ORDER BY created_at DESC`
	rows, err := r.db.Query(query, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []domain.NotificationRule
	for rows.Next() {
		var rule domain.NotificationRule
		var appID sql.NullString
		if err := rows.Scan(
			&rule.ID,
			&rule.EventType,
			&rule.ChannelID,
			&appID,
			&rule.Enabled,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if appID.Valid {
			rule.AppID = &appID.String
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *PostgresNotificationRuleRepository) FindActiveByEventType(eventType string, appID *string) ([]domain.NotificationRule, error) {
	query := `
		SELECT nr.id, nr.event_type, nr.channel_id, nr.app_id, nr.enabled, nr.created_at, nr.updated_at
		FROM notification_rules nr
		JOIN notification_channels nc ON nr.channel_id = nc.id
		WHERE nr.event_type = $1 AND nr.enabled = true
		AND (
			($2::uuid IS NULL AND nr.app_id IS NULL AND nc.app_id IS NULL)
			OR ($2::uuid IS NOT NULL AND (nr.app_id IS NULL OR nr.app_id = $2) AND (nc.app_id IS NULL OR nc.app_id = $2))
		)
		ORDER BY nr.created_at DESC
	`
	var appIDVal interface{}
	if appID != nil && *appID != "" {
		appIDVal = *appID
	}
	rows, err := r.db.Query(query, eventType, appIDVal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []domain.NotificationRule
	for rows.Next() {
		var rule domain.NotificationRule
		var ruleAppID sql.NullString
		if err := rows.Scan(
			&rule.ID,
			&rule.EventType,
			&rule.ChannelID,
			&ruleAppID,
			&rule.Enabled,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if ruleAppID.Valid {
			rule.AppID = &ruleAppID.String
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *PostgresNotificationRuleRepository) Create(input domain.CreateNotificationRuleInput) (*domain.NotificationRule, error) {
	query := `
		INSERT INTO notification_rules (event_type, channel_id, app_id, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING ` + notificationRuleSelectColumns
	return r.scanRule(r.db.QueryRow(query, input.EventType, input.ChannelID, toNullString(input.AppID), input.Enabled))
}

func (r *PostgresNotificationRuleRepository) Update(id string, input domain.UpdateNotificationRuleInput) (*domain.NotificationRule, error) {
	query := `
		UPDATE notification_rules
		SET event_type = COALESCE($2, event_type),
		    enabled = COALESCE($3, enabled),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING ` + notificationRuleSelectColumns
	var eventTypeVal interface{}
	var enabledVal interface{}
	if input.EventType != nil {
		eventTypeVal = *input.EventType
	}
	if input.Enabled != nil {
		enabledVal = *input.Enabled
	}
	return r.scanRule(r.db.QueryRow(query, id, eventTypeVal, enabledVal))
}

func (r *PostgresNotificationRuleRepository) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM notification_rules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}
