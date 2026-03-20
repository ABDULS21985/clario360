CREATE TABLE IF NOT EXISTS threat_feed_configs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    name                TEXT NOT NULL,
    type                TEXT NOT NULL,
    url                 TEXT,
    auth_type           TEXT NOT NULL DEFAULT 'none',
    auth_config         JSONB NOT NULL DEFAULT '{}',
    sync_interval       TEXT NOT NULL DEFAULT 'manual',
    default_severity    TEXT NOT NULL DEFAULT 'medium',
    default_confidence  DECIMAL(3,2) NOT NULL DEFAULT 0.80,
    default_tags        TEXT[] NOT NULL DEFAULT '{}',
    indicator_types     TEXT[] NOT NULL DEFAULT '{}',
    enabled             BOOLEAN NOT NULL DEFAULT true,
    status              TEXT NOT NULL DEFAULT 'active',
    last_sync_at        TIMESTAMPTZ,
    last_sync_status    TEXT,
    last_error          TEXT,
    created_by          UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE threat_feed_configs
    DROP CONSTRAINT IF EXISTS threat_feed_configs_type_check;
ALTER TABLE threat_feed_configs
    ADD CONSTRAINT threat_feed_configs_type_check
        CHECK (type IN ('stix', 'taxii', 'misp', 'csv_url', 'manual'));

ALTER TABLE threat_feed_configs
    DROP CONSTRAINT IF EXISTS threat_feed_configs_auth_type_check;
ALTER TABLE threat_feed_configs
    ADD CONSTRAINT threat_feed_configs_auth_type_check
        CHECK (auth_type IN ('none', 'api_key', 'basic', 'certificate'));

ALTER TABLE threat_feed_configs
    DROP CONSTRAINT IF EXISTS threat_feed_configs_sync_interval_check;
ALTER TABLE threat_feed_configs
    ADD CONSTRAINT threat_feed_configs_sync_interval_check
        CHECK (sync_interval IN ('hourly', 'every_6h', 'daily', 'weekly', 'manual'));

ALTER TABLE threat_feed_configs
    DROP CONSTRAINT IF EXISTS threat_feed_configs_status_check;
ALTER TABLE threat_feed_configs
    ADD CONSTRAINT threat_feed_configs_status_check
        CHECK (status IN ('active', 'paused', 'error'));

ALTER TABLE threat_feed_configs
    DROP CONSTRAINT IF EXISTS threat_feed_configs_default_severity_check;
ALTER TABLE threat_feed_configs
    ADD CONSTRAINT threat_feed_configs_default_severity_check
        CHECK (default_severity IN ('critical', 'high', 'medium', 'low', 'info'));

ALTER TABLE threat_feed_configs
    DROP CONSTRAINT IF EXISTS threat_feed_configs_default_confidence_check;
ALTER TABLE threat_feed_configs
    ADD CONSTRAINT threat_feed_configs_default_confidence_check
        CHECK (default_confidence BETWEEN 0.00 AND 1.00);

CREATE UNIQUE INDEX IF NOT EXISTS idx_threat_feed_configs_tenant_name
    ON threat_feed_configs (tenant_id, name);
CREATE INDEX IF NOT EXISTS idx_threat_feed_configs_tenant_status
    ON threat_feed_configs (tenant_id, status, updated_at DESC);

DROP TRIGGER IF EXISTS trg_threat_feed_configs_updated_at ON threat_feed_configs;
CREATE TRIGGER trg_threat_feed_configs_updated_at
    BEFORE UPDATE ON threat_feed_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS threat_feed_sync_history (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    feed_id             UUID NOT NULL REFERENCES threat_feed_configs(id) ON DELETE CASCADE,
    status              TEXT NOT NULL,
    indicators_parsed   INT NOT NULL DEFAULT 0,
    indicators_imported INT NOT NULL DEFAULT 0,
    indicators_skipped  INT NOT NULL DEFAULT 0,
    indicators_failed   INT NOT NULL DEFAULT 0,
    duration_ms         INT NOT NULL DEFAULT 0,
    error_message       TEXT,
    metadata            JSONB NOT NULL DEFAULT '{}',
    started_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_threat_feed_history_tenant_feed
    ON threat_feed_sync_history (tenant_id, feed_id, started_at DESC);
