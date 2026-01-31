-- Rollback GitHub Authentication Schema

-- Remove columns from apps table
ALTER TABLE apps DROP COLUMN IF EXISTS github_installation_id;
ALTER TABLE apps DROP COLUMN IF EXISTS user_id;

-- Drop tables in reverse order (respect foreign keys)
DROP TABLE IF EXISTS user_installations;
DROP TABLE IF EXISTS github_installations;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
