ALTER TABLE sessions ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
COMMENT ON COLUMN sessions.last_active_at IS 'Last time the session was actively used (e.g. token refresh or API call)';
