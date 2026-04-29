CREATE TABLE personal_access_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(120) NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    token_prefix VARCHAR(32) NOT NULL,
    scopes JSONB NOT NULL DEFAULT '[]'::jsonb,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_personal_access_tokens_user_id ON personal_access_tokens(user_id);
CREATE INDEX idx_personal_access_tokens_token_hash ON personal_access_tokens(token_hash);
CREATE INDEX idx_personal_access_tokens_active
    ON personal_access_tokens(user_id)
    WHERE revoked_at IS NULL;

CREATE TRIGGER update_personal_access_tokens_updated_at
    BEFORE UPDATE ON personal_access_tokens
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE personal_access_tokens IS 'Long-lived API tokens used for programmatic access (CLI, CI, MCP).';
COMMENT ON COLUMN personal_access_tokens.token_hash IS 'SHA-256 hash of the plaintext token; plaintext shown only once at creation.';
COMMENT ON COLUMN personal_access_tokens.token_prefix IS 'Non-secret prefix displayed in the UI to identify the token (e.g. pdp_live_abc12345).';
COMMENT ON COLUMN personal_access_tokens.scopes IS 'Permission scopes: read, deploy, containers:write, config:write, resources:write, servers:write, destructive, admin.';
