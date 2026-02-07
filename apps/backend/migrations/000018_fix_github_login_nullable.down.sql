-- Revert: restore NOT NULL on github_login
-- Note: this will fail if any rows have NULL github_login
ALTER TABLE users ALTER COLUMN github_login SET NOT NULL;
