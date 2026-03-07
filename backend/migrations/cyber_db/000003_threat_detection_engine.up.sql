-- =============================================================================
-- Clario 360 — Cyber Suite Threat Detection Engine (PROMPT 17)
-- Normalizes the legacy threat/alert schema into the production detection
-- engine model and adds alert investigation and security event storage tables.
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =============================================================================
-- ALERTS
-- =============================================================================

DROP INDEX IF EXISTS idx_alerts_assigned;
DROP INDEX IF EXISTS idx_alerts_tenant_status;
DROP INDEX IF EXISTS idx_alerts_tenant_severity;
DROP INDEX IF EXISTS idx_alerts_tenant_created;
DROP INDEX IF EXISTS idx_alerts_rule;
DROP INDEX IF EXISTS idx_alerts_affected_assets;
DROP INDEX IF EXISTS idx_alerts_explanation;
DROP INDEX IF EXISTS idx_alerts_factors;

ALTER TABLE alerts
    ALTER COLUMN title TYPE TEXT,
    ALTER COLUMN description TYPE TEXT,
    ALTER COLUMN severity DROP DEFAULT,
    ALTER COLUMN severity TYPE TEXT USING severity::text,
    ALTER COLUMN status DROP DEFAULT,
    ALTER COLUMN status TYPE TEXT USING status::text;

ALTER TABLE alerts
    RENAME COLUMN affected_assets TO asset_ids;

ALTER TABLE alerts
    ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'manual',
    ADD COLUMN IF NOT EXISTS asset_id UUID REFERENCES assets(id),
    ADD COLUMN IF NOT EXISTS assigned_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS escalated_to UUID,
    ADD COLUMN IF NOT EXISTS escalated_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS mitre_tactic_id TEXT,
    ADD COLUMN IF NOT EXISTS mitre_tactic_name TEXT,
    ADD COLUMN IF NOT EXISTS mitre_technique_id TEXT,
    ADD COLUMN IF NOT EXISTS mitre_technique_name TEXT,
    ADD COLUMN IF NOT EXISTS event_count INT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS first_event_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS last_event_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS false_positive_reason TEXT,
    ADD COLUMN IF NOT EXISTS tags TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE alerts
    ALTER COLUMN asset_ids SET DEFAULT '{}',
    ALTER COLUMN asset_ids SET NOT NULL,
    ALTER COLUMN explanation SET DEFAULT '{}',
    ALTER COLUMN explanation TYPE JSONB USING COALESCE(explanation, '{}'::jsonb),
    ALTER COLUMN confidence_score TYPE DECIMAL(3,2) USING COALESCE(confidence_score, 0.00),
    ALTER COLUMN confidence_score SET DEFAULT 0.00,
    ALTER COLUMN severity SET DEFAULT 'medium',
    ALTER COLUMN status SET DEFAULT 'new';

UPDATE alerts
SET
    asset_ids = COALESCE(asset_ids, '{}'),
    first_event_at = COALESCE(first_event_at, created_at, now()),
    last_event_at = COALESCE(last_event_at, updated_at, created_at, now()),
    assigned_at = COALESCE(assigned_at, acknowledged_at),
    tags = COALESCE(tags, '{}'),
    metadata = COALESCE(metadata, '{}'::jsonb);

ALTER TABLE alerts
    DROP CONSTRAINT IF EXISTS alerts_severity_check;
ALTER TABLE alerts
    ADD CONSTRAINT alerts_severity_check
        CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info'));

ALTER TABLE alerts
    DROP CONSTRAINT IF EXISTS alerts_status_check;
ALTER TABLE alerts
    ADD CONSTRAINT alerts_status_check
        CHECK (status IN (
            'new', 'acknowledged', 'investigating', 'in_progress',
            'resolved', 'closed', 'false_positive', 'escalated', 'merged'
        ));

ALTER TABLE alerts
    DROP CONSTRAINT IF EXISTS alerts_confidence_score_check;
ALTER TABLE alerts
    ADD CONSTRAINT alerts_confidence_score_check
        CHECK (confidence_score BETWEEN 0.00 AND 1.00);

CREATE INDEX IF NOT EXISTS idx_alerts_tenant_status
    ON alerts (tenant_id, status, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_alerts_tenant_severity
    ON alerts (tenant_id, severity, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_alerts_tenant_assigned
    ON alerts (tenant_id, assigned_to) WHERE assigned_to IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_alerts_tenant_asset
    ON alerts (tenant_id, asset_id) WHERE asset_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_alerts_tenant_rule
    ON alerts (tenant_id, rule_id) WHERE rule_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_alerts_tenant_mitre
    ON alerts (tenant_id, mitre_technique_id) WHERE mitre_technique_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_alerts_confidence
    ON alerts (tenant_id, confidence_score DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_alerts_tenant_created
    ON alerts (tenant_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_alerts_tags
    ON alerts USING GIN (tags) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_alerts_fts
    ON alerts USING GIN (
        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, ''))
    ) WHERE deleted_at IS NULL;

-- =============================================================================
-- ALERT COMMENTS
-- =============================================================================

CREATE TABLE IF NOT EXISTS alert_comments (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID            NOT NULL,
    alert_id        UUID            NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
    user_id         UUID            NOT NULL,
    user_name       TEXT            NOT NULL,
    user_email      TEXT            NOT NULL,
    content         TEXT            NOT NULL,
    is_system       BOOLEAN         NOT NULL DEFAULT false,
    metadata        JSONB           NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_comments_alert
    ON alert_comments (alert_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_comments_tenant
    ON alert_comments (tenant_id, created_at DESC);

DROP TRIGGER IF EXISTS trg_alert_comments_updated_at ON alert_comments;
CREATE TRIGGER trg_alert_comments_updated_at
    BEFORE UPDATE ON alert_comments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- ALERT TIMELINE
-- =============================================================================

CREATE TABLE IF NOT EXISTS alert_timeline (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID            NOT NULL,
    alert_id        UUID            NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
    action          TEXT            NOT NULL,
    actor_id        UUID,
    actor_name      TEXT,
    old_value       TEXT,
    new_value       TEXT,
    description     TEXT            NOT NULL,
    metadata        JSONB           NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_timeline_alert
    ON alert_timeline (alert_id, created_at ASC);

-- =============================================================================
-- DETECTION RULES
-- =============================================================================

ALTER TABLE detection_rules
    ALTER COLUMN tenant_id DROP NOT NULL,
    ALTER COLUMN name TYPE TEXT,
    ALTER COLUMN description TYPE TEXT,
    ALTER COLUMN severity DROP DEFAULT,
    ALTER COLUMN severity TYPE TEXT USING severity::text,
    ALTER COLUMN rule_type DROP DEFAULT,
    ALTER COLUMN rule_type TYPE TEXT USING rule_type::text;

ALTER TABLE detection_rules
    RENAME COLUMN mitre_techniques TO mitre_technique_ids;

ALTER TABLE detection_rules
    ADD COLUMN IF NOT EXISTS mitre_tactic_ids TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS base_confidence DECIMAL(3,2) NOT NULL DEFAULT 0.70,
    ADD COLUMN IF NOT EXISTS false_positive_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS true_positive_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS tags TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS is_template BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS template_id TEXT,
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE detection_rules
    ALTER COLUMN rule_content TYPE JSONB USING
        CASE
            WHEN rule_content IS NULL OR btrim(rule_content) = '' THEN '{}'::jsonb
            WHEN left(ltrim(rule_content), 1) IN ('{', '[') THEN rule_content::jsonb
            ELSE jsonb_build_object('legacy_rule_content', rule_content)
        END,
    ALTER COLUMN rule_content SET NOT NULL,
    ALTER COLUMN severity SET DEFAULT 'medium',
    ALTER COLUMN enabled SET DEFAULT true,
    ALTER COLUMN last_triggered_at DROP DEFAULT;

UPDATE detection_rules
SET
    mitre_technique_ids = COALESCE(mitre_technique_ids, '{}'),
    mitre_tactic_ids = COALESCE(mitre_tactic_ids, '{}'),
    tags = COALESCE(tags, '{}');

ALTER TABLE detection_rules
    DROP CONSTRAINT IF EXISTS detection_rules_rule_type_check;
ALTER TABLE detection_rules
    ADD CONSTRAINT detection_rules_rule_type_check
        CHECK (rule_type IN ('sigma', 'threshold', 'correlation', 'anomaly'));

ALTER TABLE detection_rules
    DROP CONSTRAINT IF EXISTS detection_rules_severity_check;
ALTER TABLE detection_rules
    ADD CONSTRAINT detection_rules_severity_check
        CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info'));

ALTER TABLE detection_rules
    DROP CONSTRAINT IF EXISTS detection_rules_base_confidence_check;
ALTER TABLE detection_rules
    ADD CONSTRAINT detection_rules_base_confidence_check
        CHECK (base_confidence BETWEEN 0.00 AND 1.00);

DROP INDEX IF EXISTS idx_rules_tenant;
DROP INDEX IF EXISTS idx_rules_type;
DROP INDEX IF EXISTS idx_rules_severity;
DROP INDEX IF EXISTS idx_rules_mitre;

CREATE INDEX IF NOT EXISTS idx_rules_tenant_enabled
    ON detection_rules (tenant_id, enabled) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_rules_tenant_type
    ON detection_rules (tenant_id, rule_type) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_rules_mitre
    ON detection_rules USING GIN (mitre_technique_ids) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_rules_tenant_name_unique
    ON detection_rules (tenant_id, name) WHERE deleted_at IS NULL AND tenant_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_rules_template_unique
    ON detection_rules (template_id) WHERE is_template = true AND template_id IS NOT NULL;

-- =============================================================================
-- THREATS
-- =============================================================================

ALTER TABLE threats
    RENAME COLUMN title TO name;

ALTER TABLE threats
    ALTER COLUMN name TYPE TEXT,
    ALTER COLUMN description TYPE TEXT,
    ALTER COLUMN severity DROP DEFAULT,
    ALTER COLUMN severity TYPE TEXT USING severity::text,
    ALTER COLUMN status DROP DEFAULT,
    ALTER COLUMN status TYPE TEXT USING status::text;

ALTER TABLE threats
    ADD COLUMN IF NOT EXISTS threat_actor TEXT,
    ADD COLUMN IF NOT EXISTS campaign TEXT,
    ADD COLUMN IF NOT EXISTS mitre_tactic_ids TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS mitre_technique_ids TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS affected_asset_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS alert_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS contained_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS tags TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

UPDATE threats
SET
    first_seen_at = COALESCE(first_seen_at, detected_at, created_at, now()),
    last_seen_at = COALESCE(last_seen_at, resolved_at, updated_at, detected_at, now()),
    mitre_technique_ids = COALESCE(mitre_technique_ids,
        CASE
            WHEN mitre_technique_id IS NULL OR mitre_technique_id = '' THEN '{}'
            ELSE ARRAY[mitre_technique_id]
        END
    ),
    mitre_tactic_ids = COALESCE(mitre_tactic_ids,
        CASE
            WHEN mitre_tactic IS NULL OR mitre_tactic = '' THEN '{}'
            ELSE ARRAY[mitre_tactic]
        END
    ),
    tags = COALESCE(tags, '{}'),
    metadata = COALESCE(metadata, '{}'::jsonb);

ALTER TABLE threats
    DROP CONSTRAINT IF EXISTS threats_type_check;
ALTER TABLE threats
    ADD CONSTRAINT threats_type_check
        CHECK (type IN (
            'malware', 'phishing', 'apt', 'ransomware', 'ddos',
            'insider_threat', 'supply_chain', 'zero_day', 'brute_force', 'other'
        ));

ALTER TABLE threats
    DROP CONSTRAINT IF EXISTS threats_severity_check;
ALTER TABLE threats
    ADD CONSTRAINT threats_severity_check
        CHECK (severity IN ('critical', 'high', 'medium', 'low'));

ALTER TABLE threats
    DROP CONSTRAINT IF EXISTS threats_status_check;
ALTER TABLE threats
    ADD CONSTRAINT threats_status_check
        CHECK (status IN ('active', 'contained', 'eradicated', 'monitoring', 'closed'));

CREATE INDEX IF NOT EXISTS idx_threats_tenant_status
    ON threats (tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_threats_tenant_type
    ON threats (tenant_id, type) WHERE deleted_at IS NULL;

-- =============================================================================
-- THREAT INDICATORS
-- =============================================================================

ALTER TABLE threat_indicators
    ALTER COLUMN threat_id DROP NOT NULL,
    ALTER COLUMN type TYPE TEXT USING type::text;

UPDATE threat_indicators
SET type = CASE type
    WHEN 'hash_md5' THEN 'file_hash_md5'
    WHEN 'hash_sha1' THEN 'file_hash_sha1'
    WHEN 'hash_sha256' THEN 'file_hash_sha256'
    ELSE type
END;

ALTER TABLE threat_indicators
    RENAME COLUMN first_seen TO first_seen_at;
ALTER TABLE threat_indicators
    RENAME COLUMN last_seen TO last_seen_at;

ALTER TABLE threat_indicators
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS severity TEXT NOT NULL DEFAULT 'medium',
    ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS tags TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS created_by UUID,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE threat_indicators
    ALTER COLUMN confidence TYPE DECIMAL(3,2) USING COALESCE(confidence, 0.80),
    ALTER COLUMN confidence SET DEFAULT 0.80,
    ALTER COLUMN source TYPE TEXT USING COALESCE(source, 'manual'),
    ALTER COLUMN source SET DEFAULT 'manual';

UPDATE threat_indicators
SET
    source = COALESCE(NULLIF(source, ''), 'manual'),
    first_seen_at = COALESCE(first_seen_at, created_at, now()),
    last_seen_at = COALESCE(last_seen_at, created_at, now()),
    tags = COALESCE(tags, '{}'),
    metadata = COALESCE(metadata, '{}'::jsonb);

ALTER TABLE threat_indicators
    DROP CONSTRAINT IF EXISTS threat_indicators_type_check;
ALTER TABLE threat_indicators
    ADD CONSTRAINT threat_indicators_type_check
        CHECK (type IN (
            'ip', 'domain', 'url', 'email', 'file_hash_md5', 'file_hash_sha1',
            'file_hash_sha256', 'certificate', 'registry_key', 'user_agent', 'cidr'
        ));

ALTER TABLE threat_indicators
    DROP CONSTRAINT IF EXISTS threat_indicators_severity_check;
ALTER TABLE threat_indicators
    ADD CONSTRAINT threat_indicators_severity_check
        CHECK (severity IN ('critical', 'high', 'medium', 'low'));

ALTER TABLE threat_indicators
    DROP CONSTRAINT IF EXISTS threat_indicators_source_check;
ALTER TABLE threat_indicators
    ADD CONSTRAINT threat_indicators_source_check
        CHECK (source IN ('manual', 'stix_feed', 'osint', 'internal', 'vendor'));

ALTER TABLE threat_indicators
    DROP CONSTRAINT IF EXISTS threat_indicators_confidence_check;
ALTER TABLE threat_indicators
    ADD CONSTRAINT threat_indicators_confidence_check
        CHECK (confidence BETWEEN 0.00 AND 1.00);

DROP INDEX IF EXISTS idx_indicators_tenant;
DROP INDEX IF EXISTS idx_indicators_type_value;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'threat_indicators_tenant_type_value_key'
    ) THEN
        ALTER TABLE threat_indicators
            ADD CONSTRAINT threat_indicators_tenant_type_value_key
                UNIQUE (tenant_id, type, value);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_indicators_tenant_type
    ON threat_indicators (tenant_id, type) WHERE active = true;
CREATE INDEX IF NOT EXISTS idx_indicators_value
    ON threat_indicators (value) WHERE active = true;
CREATE INDEX IF NOT EXISTS idx_indicators_threat
    ON threat_indicators (threat_id) WHERE threat_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_indicators_tenant_active
    ON threat_indicators (tenant_id, active, expires_at);

DROP TRIGGER IF EXISTS trg_threat_indicators_updated_at ON threat_indicators;
CREATE TRIGGER trg_threat_indicators_updated_at
    BEFORE UPDATE ON threat_indicators
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- SECURITY EVENT LOG
-- =============================================================================

CREATE TABLE IF NOT EXISTS security_events (
    id              UUID            NOT NULL DEFAULT gen_random_uuid(),
    tenant_id       UUID            NOT NULL,
    timestamp       TIMESTAMPTZ     NOT NULL,
    source          TEXT            NOT NULL,
    type            TEXT            NOT NULL,
    severity        TEXT            NOT NULL DEFAULT 'info',
    source_ip       INET,
    dest_ip         INET,
    dest_port       INT,
    protocol        TEXT,
    username        TEXT,
    process         TEXT,
    parent_process  TEXT,
    command_line    TEXT,
    file_path       TEXT,
    file_hash       TEXT,
    asset_id        UUID,
    raw_event       JSONB           NOT NULL DEFAULT '{}',
    matched_rules   UUID[]          NOT NULL DEFAULT '{}',
    processed_at    TIMESTAMPTZ     NOT NULL DEFAULT now(),

    PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

CREATE INDEX IF NOT EXISTS idx_events_tenant_time
    ON security_events (tenant_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_events_source_ip
    ON security_events (tenant_id, source_ip) WHERE source_ip IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_events_dest_ip
    ON security_events (tenant_id, dest_ip) WHERE dest_ip IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_events_asset
    ON security_events (tenant_id, asset_id) WHERE asset_id IS NOT NULL;

CREATE OR REPLACE FUNCTION create_security_events_partition(start_date DATE)
RETURNS VOID AS $$
DECLARE
    end_date DATE := (start_date + INTERVAL '1 month')::DATE;
    partition_name TEXT := format('security_events_%s', to_char(start_date, 'YYYY_MM'));
BEGIN
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF security_events FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date
    );
END;
$$ LANGUAGE plpgsql;

SELECT create_security_events_partition((date_trunc('month', now()) - INTERVAL '1 month')::DATE);
SELECT create_security_events_partition(date_trunc('month', now())::DATE);
SELECT create_security_events_partition((date_trunc('month', now()) + INTERVAL '1 month')::DATE);
