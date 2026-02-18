DROP INDEX IF EXISTS idx_servers_user_id;
ALTER TABLE servers DROP COLUMN IF EXISTS user_id;
