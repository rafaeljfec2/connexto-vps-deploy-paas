CREATE TABLE app_env_vars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    is_secret BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(app_id, key)
);

CREATE INDEX idx_app_env_vars_app_id ON app_env_vars(app_id);

COMMENT ON TABLE app_env_vars IS 'Environment variables for each application';
COMMENT ON COLUMN app_env_vars.is_secret IS 'If true, value is masked in UI responses';
