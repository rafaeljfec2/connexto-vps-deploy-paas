-- Email + Password Authentication
-- Allow users to register with email/password in addition to GitHub OAuth

-- Make GitHub fields optional (nullable)
ALTER TABLE users ALTER COLUMN github_id DROP NOT NULL;
ALTER TABLE users ALTER COLUMN github_login DROP NOT NULL;
ALTER TABLE users ALTER COLUMN access_token_encrypted DROP NOT NULL;

-- Replace unique constraint on github_id to allow multiple NULLs
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_github_id_key;
CREATE UNIQUE INDEX idx_users_github_id_unique ON users(github_id) WHERE github_id IS NOT NULL;
DROP INDEX IF EXISTS idx_users_github_id;

-- New columns
ALTER TABLE users ADD COLUMN password_hash TEXT;
ALTER TABLE users ADD COLUMN auth_provider VARCHAR(20) NOT NULL DEFAULT 'github';

-- Unique partial index on email for email-based login
CREATE UNIQUE INDEX idx_users_email_unique ON users(email) WHERE email IS NOT NULL;

-- Constraint: user must have github_id OR password_hash (or both if linked)
ALTER TABLE users ADD CONSTRAINT chk_auth_method
  CHECK (github_id IS NOT NULL OR password_hash IS NOT NULL);

-- Comments
COMMENT ON COLUMN users.password_hash IS 'bcrypt hashed password for email-based auth';
COMMENT ON COLUMN users.auth_provider IS 'Primary auth provider: github or email';
