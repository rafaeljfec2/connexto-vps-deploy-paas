CREATE TABLE IF NOT EXISTS cleanup_logs (
    id TEXT PRIMARY KEY,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    cleanup_type TEXT NOT NULL CHECK (cleanup_type IN ('containers', 'volumes', 'images')),
    items_removed INTEGER NOT NULL DEFAULT 0,
    space_reclaimed_bytes BIGINT NOT NULL DEFAULT 0,
    triggered_by TEXT NOT NULL DEFAULT 'scheduled' CHECK (triggered_by IN ('scheduled', 'manual')),
    status TEXT NOT NULL DEFAULT 'success' CHECK (status IN ('success', 'failed')),
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_cleanup_logs_server_created ON cleanup_logs(server_id, created_at DESC);
