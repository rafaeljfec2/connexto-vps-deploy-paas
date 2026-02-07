-- Fix: github_login must be nullable for email-only users
ALTER TABLE users ALTER COLUMN github_login DROP NOT NULL;
