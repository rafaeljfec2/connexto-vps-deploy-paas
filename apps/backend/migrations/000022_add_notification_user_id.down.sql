DROP INDEX IF EXISTS idx_notification_rules_user_id;
ALTER TABLE notification_rules DROP COLUMN IF EXISTS user_id;

DROP INDEX IF EXISTS idx_notification_channels_user_id;
ALTER TABLE notification_channels DROP COLUMN IF EXISTS user_id;
