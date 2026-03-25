DROP INDEX IF EXISTS idx_deployments_delivery_id;
DROP INDEX IF EXISTS idx_deployments_app_commit_active;
ALTER TABLE deployments DROP COLUMN IF EXISTS delivery_id;
