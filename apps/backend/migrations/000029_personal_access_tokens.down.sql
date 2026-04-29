DROP TRIGGER IF EXISTS update_personal_access_tokens_updated_at ON personal_access_tokens;
DROP INDEX IF EXISTS idx_personal_access_tokens_active;
DROP INDEX IF EXISTS idx_personal_access_tokens_token_hash;
DROP INDEX IF EXISTS idx_personal_access_tokens_user_id;
DROP TABLE IF EXISTS personal_access_tokens;
