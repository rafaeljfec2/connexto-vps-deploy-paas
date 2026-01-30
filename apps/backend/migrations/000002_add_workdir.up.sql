-- Add workdir column for monorepo support
ALTER TABLE apps ADD COLUMN workdir VARCHAR(255) NOT NULL DEFAULT '.';

COMMENT ON COLUMN apps.workdir IS 'Working directory relative to repository root (for monorepo apps)';
