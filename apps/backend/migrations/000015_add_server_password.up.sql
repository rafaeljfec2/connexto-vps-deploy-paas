ALTER TABLE servers
    ADD COLUMN IF NOT EXISTS ssh_password_encrypted TEXT;
