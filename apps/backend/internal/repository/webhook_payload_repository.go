package repository

import (
	"context"
	"database/sql"

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
