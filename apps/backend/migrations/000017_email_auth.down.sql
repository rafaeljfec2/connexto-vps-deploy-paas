-- Revert Email + Password Authentication

-- Remove constraint and columns
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_auth_method;
ALTER TABLE users DROP COLUMN IF EXISTS auth_provider;
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;

-- Remove email unique index
DROP INDEX IF EXISTS idx_users_email_unique;

-- Restore github_id unique constraint
DROP INDEX IF EXISTS idx_users_github_id_unique;
CREATE INDEX idx_users_github_id ON users(github_id);
ALTER TABLE users ADD CONSTRAINT users_github_id_key UNIQUE (github_id);

-- Restore NOT NULL on github fields
ALTER TABLE users ALTER COLUMN access_token_encrypted SET NOT NULL;
ALTER TABLE users ALTER COLUMN github_id SET NOT NULL;
