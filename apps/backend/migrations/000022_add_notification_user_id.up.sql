ALTER TABLE notification_channels ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;

UPDATE notification_channels SET user_id = (SELECT id FROM users ORDER BY created_at ASC LIMIT 1) WHERE user_id IS NULL;

ALTER TABLE notification_channels ALTER COLUMN user_id SET NOT NULL;

CREATE INDEX idx_notification_channels_user_id ON notification_channels(user_id);

ALTER TABLE notification_rules ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;

UPDATE notification_rules SET user_id = (SELECT id FROM users ORDER BY created_at ASC LIMIT 1) WHERE user_id IS NULL;

ALTER TABLE notification_rules ALTER COLUMN user_id SET NOT NULL;

CREATE INDEX idx_notification_rules_user_id ON notification_rules(user_id);
