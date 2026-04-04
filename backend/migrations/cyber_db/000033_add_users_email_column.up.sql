-- Add email and deleted_at columns to the lightweight users lookup table so
-- alert enrichment can display user emails and filter soft-deleted users.
ALTER TABLE users ADD COLUMN IF NOT EXISTS email TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
