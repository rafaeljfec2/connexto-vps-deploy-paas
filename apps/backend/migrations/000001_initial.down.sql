-- Rollback initial schema

DROP TRIGGER IF EXISTS update_apps_updated_at ON apps;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS deployments;
DROP TABLE IF EXISTS apps;
DROP TYPE IF EXISTS deploy_status;
DROP TYPE IF EXISTS app_status;
