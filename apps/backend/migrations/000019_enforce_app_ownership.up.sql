UPDATE apps SET user_id = (SELECT id FROM users ORDER BY created_at ASC LIMIT 1) WHERE user_id IS NULL;
ALTER TABLE apps ALTER COLUMN user_id SET NOT NULL;
