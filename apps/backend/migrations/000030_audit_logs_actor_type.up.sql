ALTER TABLE audit_logs
    ADD COLUMN actor_type VARCHAR(16) NOT NULL DEFAULT 'user'
    CHECK (actor_type IN ('user', 'pat', 'system', 'webhook'));

ALTER TABLE audit_logs
    ADD COLUMN actor_id VARCHAR(64);

CREATE INDEX idx_audit_logs_actor ON audit_logs(actor_type, actor_id);

COMMENT ON COLUMN audit_logs.actor_type IS 'Origin of the action: user (session cookie), pat (personal access token), system (automated), webhook (inbound event).';
COMMENT ON COLUMN audit_logs.actor_id IS 'Identifier of the actor (user id, PAT id, or source system identifier).';
