CREATE TABLE servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    host VARCHAR(255) NOT NULL,
    ssh_port INTEGER NOT NULL DEFAULT 22,
    ssh_user VARCHAR(100) NOT NULL,
    ssh_key_encrypted TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    agent_version VARCHAR(50),
    last_heartbeat_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_servers_status ON servers(status);
CREATE INDEX idx_servers_last_heartbeat_at ON servers(last_heartbeat_at);

CREATE TRIGGER update_servers_updated_at
    BEFORE UPDATE ON servers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

ALTER TABLE apps ADD COLUMN server_id UUID REFERENCES servers(id) ON DELETE SET NULL;

CREATE INDEX idx_apps_server_id ON apps(server_id);

COMMENT ON TABLE servers IS 'Remote servers for deploy (agent targets)';
COMMENT ON COLUMN apps.server_id IS 'Target server for remote deploy (NULL = local)';
