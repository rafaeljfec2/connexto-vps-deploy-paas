ALTER TABLE servers ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;

UPDATE servers SET user_id = (SELECT id FROM users ORDER BY created_at ASC LIMIT 1) WHERE user_id IS NULL;

ALTER TABLE servers ALTER COLUMN user_id SET NOT NULL;

CREATE INDEX idx_servers_user_id ON servers(user_id);
