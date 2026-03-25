ALTER TABLE deployments ADD COLUMN IF NOT EXISTS delivery_id TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_deployments_app_commit_active
    ON deployments (app_id, commit_sha)
    WHERE status IN ('pending', 'running');

CREATE UNIQUE INDEX IF NOT EXISTS idx_deployments_delivery_id
    ON deployments (delivery_id)
    WHERE delivery_id IS NOT NULL;
