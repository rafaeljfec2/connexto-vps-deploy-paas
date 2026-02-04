CREATE TABLE webhook_payloads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id VARCHAR(255) NOT NULL UNIQUE,
    event_type VARCHAR(100) NOT NULL,
    provider VARCHAR(50) NOT NULL DEFAULT 'github',
    payload JSONB NOT NULL,
    outcome VARCHAR(50),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_payloads_created_at ON webhook_payloads(created_at DESC);
CREATE INDEX idx_webhook_payloads_event_type ON webhook_payloads(event_type);
CREATE INDEX idx_webhook_payloads_delivery_id ON webhook_payloads(delivery_id);

COMMENT ON TABLE webhook_payloads IS 'Stores incoming webhook payloads for audit and debugging';
