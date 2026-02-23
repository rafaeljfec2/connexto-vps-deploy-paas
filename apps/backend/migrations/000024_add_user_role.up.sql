ALTER TABLE users ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'member';

UPDATE users SET role = 'admin'
WHERE id = (SELECT id FROM users ORDER BY created_at ASC LIMIT 1);
