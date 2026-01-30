-- FlowDeploy Initial Schema
-- This migration creates the core tables for application and deployment management

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Application status enum
CREATE TYPE app_status AS ENUM ('active', 'inactive', 'deleted');

-- Deployment status enum
CREATE TYPE deploy_status AS ENUM ('pending', 'running', 'success', 'failed', 'cancelled');

-- Applications table
CREATE TABLE apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    repository_url VARCHAR(500) NOT NULL,
    branch VARCHAR(100) NOT NULL DEFAULT 'main',
    config JSONB NOT NULL DEFAULT '{}',
    status app_status NOT NULL DEFAULT 'active',
    last_deployed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Deployments table
CREATE TABLE deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    commit_sha VARCHAR(40) NOT NULL,
    commit_message TEXT,
    status deploy_status NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    error_message TEXT,
    logs TEXT,
    previous_image_tag VARCHAR(100),
    current_image_tag VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for apps
CREATE INDEX idx_apps_name ON apps(name);
CREATE INDEX idx_apps_status ON apps(status);

-- Indexes for deployments
CREATE INDEX idx_deployments_app_id ON deployments(app_id);
CREATE INDEX idx_deployments_status ON deployments(status);
CREATE INDEX idx_deployments_created_at ON deployments(created_at DESC);

-- Partial index for pending deployments (used by queue)
CREATE INDEX idx_deployments_pending ON deployments(app_id, created_at)
    WHERE status = 'pending';

-- Partial index for running deployments (used to check if app has running deploy)
CREATE INDEX idx_deployments_running ON deployments(app_id)
    WHERE status = 'running';

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger for apps updated_at
CREATE TRIGGER update_apps_updated_at
    BEFORE UPDATE ON apps
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE apps IS 'Registered applications for deployment';
COMMENT ON TABLE deployments IS 'Deployment history and queue';
COMMENT ON COLUMN apps.config IS 'JSON configuration from paasdeploy.json';
COMMENT ON COLUMN deployments.previous_image_tag IS 'Docker image tag before this deployment (for rollback)';
COMMENT ON COLUMN deployments.current_image_tag IS 'Docker image tag created by this deployment';
