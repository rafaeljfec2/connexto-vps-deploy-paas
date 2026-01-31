-- GitHub Authentication Schema
-- This migration adds user authentication via GitHub OAuth and GitHub App installations

-- Users table (GitHub-based authentication)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    github_id BIGINT NOT NULL UNIQUE,
    github_login VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    email VARCHAR(255),
    avatar_url VARCHAR(500),
    access_token_encrypted TEXT NOT NULL,
    refresh_token_encrypted TEXT,
    token_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_github_id ON users(github_id);
CREATE INDEX idx_users_github_login ON users(github_login);

-- Trigger for users updated_at
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Sessions table (user login sessions)
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    ip_address VARCHAR(45),
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- GitHub App installations table
CREATE TABLE github_installations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    installation_id BIGINT NOT NULL UNIQUE,
    account_type VARCHAR(50) NOT NULL CHECK (account_type IN ('User', 'Organization')),
    account_id BIGINT NOT NULL,
    account_login VARCHAR(255) NOT NULL,
    repository_selection VARCHAR(50) DEFAULT 'selected' CHECK (repository_selection IN ('all', 'selected')),
    permissions JSONB DEFAULT '{}',
    suspended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_github_installations_installation_id ON github_installations(installation_id);
CREATE INDEX idx_github_installations_account ON github_installations(account_id, account_login);

-- Trigger for github_installations updated_at
CREATE TRIGGER update_github_installations_updated_at
    BEFORE UPDATE ON github_installations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- User installations junction table (users can have access to multiple installations)
CREATE TABLE user_installations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    installation_id UUID NOT NULL REFERENCES github_installations(id) ON DELETE CASCADE,
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, installation_id)
);

CREATE INDEX idx_user_installations_user ON user_installations(user_id);
CREATE INDEX idx_user_installations_installation ON user_installations(installation_id);

-- Add user_id to apps table (app owner)
ALTER TABLE apps ADD COLUMN user_id UUID REFERENCES users(id);
CREATE INDEX idx_apps_user_id ON apps(user_id);

-- Add github_installation_id to apps table (which installation to use for this app)
ALTER TABLE apps ADD COLUMN github_installation_id UUID REFERENCES github_installations(id);
CREATE INDEX idx_apps_github_installation_id ON apps(github_installation_id);

-- Comments
COMMENT ON TABLE users IS 'Users authenticated via GitHub OAuth';
COMMENT ON TABLE sessions IS 'User login sessions with secure token hashes';
COMMENT ON TABLE github_installations IS 'GitHub App installations for repository access';
COMMENT ON TABLE user_installations IS 'Links users to their GitHub App installations';
COMMENT ON COLUMN users.access_token_encrypted IS 'AES-256-GCM encrypted GitHub access token';
COMMENT ON COLUMN users.refresh_token_encrypted IS 'AES-256-GCM encrypted GitHub refresh token';
COMMENT ON COLUMN sessions.token_hash IS 'SHA-256 hash of the session token';
COMMENT ON COLUMN github_installations.repository_selection IS 'Whether all repos or selected repos are accessible';
