package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/paasdeploy/backend/internal/ghclient"
)

var _ ghclient.WebhookPayloadStore = (*PostgresWebhookPayloadRepository)(nil)

type PostgresWebhookPayloadRepository struct {
	db *sql.DB
}

func NewPostgresWebhookPayloadRepository(db *sql.DB) *PostgresWebhookPayloadRepository {
	return &PostgresWebhookPayloadRepository{db: db}
}

type SaveWebhookPayloadInput struct {
	DeliveryID   string
	EventType    string
	Provider     string
	Payload      []byte
	Outcome      string
	ErrorMessage *string
}

func (r *PostgresWebhookPayloadRepository) SavePayload(ctx context.Context, deliveryID, eventType, provider string, payload []byte, outcome string, errMsg *string) error {
	return r.Save(ctx, SaveWebhookPayloadInput{
		DeliveryID:   deliveryID,
		EventType:    eventType,
		Provider:     provider,
		Payload:      payload,
		Outcome:      outcome,
		ErrorMessage: errMsg,
	})
}

func (r *PostgresWebhookPayloadRepository) Save(ctx context.Context, input SaveWebhookPayloadInput) error {
	var errMsg any
	if input.ErrorMessage != nil {
		errMsg = *input.ErrorMessage
	}

	query := `
		INSERT INTO webhook_payloads (delivery_id, event_type, provider, payload, outcome, error_message)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (delivery_id) DO UPDATE SET
			event_type = EXCLUDED.event_type,
			payload = EXCLUDED.payload,
			outcome = EXCLUDED.outcome,
			error_message = EXCLUDED.error_message
	`
	_, err := r.db.ExecContext(ctx, query,
		input.DeliveryID,
		input.EventType,
		input.Provider,
		input.Payload,
		input.Outcome,
		errMsg,
	)
	if err != nil {
		return err
	}
	return nil
}

type WebhookPayloadFilter struct {
	Limit  int
	Offset int
}

type WebhookPayloadResult struct {
	ID           string
	DeliveryID   string
	EventType    string
	Provider     string
	Payload      []byte
	Outcome      string
	ErrorMessage *string
	CreatedAt    string
}

func (r *PostgresWebhookPayloadRepository) FindAll(ctx context.Context, filter WebhookPayloadFilter) ([]WebhookPayloadResult, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM webhook_payloads").Scan(&total); err != nil {
		return nil, 0, err
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, delivery_id, event_type, provider, payload, outcome, error_message, created_at
		 FROM webhook_payloads ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var results []WebhookPayloadResult
	for rows.Next() {
		var row WebhookPayloadResult
		var errMsg sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&row.ID, &row.DeliveryID, &row.EventType, &row.Provider,
			&row.Payload, &row.Outcome, &errMsg, &createdAt); err != nil {
			return nil, 0, err
		}
		if errMsg.Valid {
			row.ErrorMessage = &errMsg.String
		}
		row.CreatedAt = createdAt.Format(time.RFC3339)
		results = append(results, row)
	}
	return results, total, nil
}
