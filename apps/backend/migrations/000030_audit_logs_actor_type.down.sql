DROP INDEX IF EXISTS idx_audit_logs_actor;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS actor_id;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS actor_type;
