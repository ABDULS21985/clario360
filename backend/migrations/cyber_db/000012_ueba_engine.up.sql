CREATE TABLE IF NOT EXISTS ueba_profiles (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    entity_type         TEXT            NOT NULL CHECK (entity_type IN (
        'user', 'service_account', 'application', 'api_key'
    )),
    entity_id           TEXT            NOT NULL,
    entity_name         TEXT,
    entity_email        TEXT,
    baseline            JSONB           NOT NULL DEFAULT '{}',
    observation_count   BIGINT          NOT NULL DEFAULT 0,
    profile_maturity    TEXT            NOT NULL DEFAULT 'learning'
                                        CHECK (profile_maturity IN ('learning', 'baseline', 'mature')),
    first_seen_at       TIMESTAMPTZ     NOT NULL,
    last_seen_at        TIMESTAMPTZ     NOT NULL,
    days_active         INT             NOT NULL DEFAULT 0,
    risk_score          DECIMAL(5,2)    NOT NULL DEFAULT 0.0,
    risk_level          TEXT            NOT NULL DEFAULT 'low'
                                        CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    risk_factors        JSONB           NOT NULL DEFAULT '[]',
    risk_last_updated   TIMESTAMPTZ,
    risk_last_decayed   TIMESTAMPTZ,
    alert_count_7d      INT             NOT NULL DEFAULT 0,
    alert_count_30d     INT             NOT NULL DEFAULT 0,
    last_alert_at       TIMESTAMPTZ,
    status              TEXT            NOT NULL DEFAULT 'active'
                                        CHECK (status IN ('active', 'inactive', 'suppressed', 'whitelisted')),
    suppressed_until    TIMESTAMPTZ,
    suppressed_reason   TEXT,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),

    UNIQUE (tenant_id, entity_type, entity_id)
);

CREATE INDEX IF NOT EXISTS idx_ueba_profiles_risk
    ON ueba_profiles (tenant_id, risk_score DESC)
    WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_ueba_profiles_entity
    ON ueba_profiles (tenant_id, entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_ueba_profiles_maturity
    ON ueba_profiles (tenant_id, profile_maturity)
    WHERE status = 'active';

CREATE TABLE IF NOT EXISTS ueba_access_events (
    id                  UUID            NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    entity_type         TEXT            NOT NULL,
    entity_id           TEXT            NOT NULL,
    source_type         TEXT            NOT NULL,
    source_id           UUID,
    action              TEXT            NOT NULL CHECK (action IN (
        'select', 'insert', 'update', 'delete', 'create', 'alter', 'drop',
        'login', 'logout', 'export', 'download', 'api_call'
    )),
    database_name       TEXT,
    schema_name         TEXT,
    table_name          TEXT,
    query_hash          TEXT,
    rows_accessed       BIGINT,
    bytes_accessed      BIGINT,
    duration_ms         INT,
    source_ip           TEXT,
    user_agent          TEXT,
    success             BOOLEAN         NOT NULL DEFAULT true,
    error_message       TEXT,
    table_sensitivity   TEXT,
    contains_pii        BOOLEAN,
    anomaly_signals     JSONB           NOT NULL DEFAULT '[]',
    anomaly_count       INT             NOT NULL DEFAULT 0,
    event_timestamp     TIMESTAMPTZ     NOT NULL,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),

    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

DO $$
DECLARE
    partition_date DATE := date_trunc('month', CURRENT_DATE);
    partition_name TEXT;
BEGIN
    FOR i IN 0..3 LOOP
        partition_name := 'ueba_access_events_' || to_char(partition_date + (i || ' months')::interval, 'YYYY_MM');
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS %I PARTITION OF ueba_access_events
             FOR VALUES FROM (%L) TO (%L)',
            partition_name,
            date_trunc('month', partition_date + (i || ' months')::interval),
            date_trunc('month', partition_date + ((i + 1) || ' months')::interval)
        );
    END LOOP;
END $$;

CREATE INDEX IF NOT EXISTS idx_ueba_events_entity
    ON ueba_access_events (tenant_id, entity_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ueba_events_source
    ON ueba_access_events (tenant_id, source_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ueba_events_anomaly
    ON ueba_access_events (tenant_id, anomaly_count, created_at DESC)
    WHERE anomaly_count > 0;
CREATE INDEX IF NOT EXISTS idx_ueba_events_table
    ON ueba_access_events (tenant_id, database_name, table_name, created_at DESC);

CREATE TABLE IF NOT EXISTS ueba_alerts (
    id                       UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID            NOT NULL,
    cyber_alert_id           UUID,
    entity_type              TEXT            NOT NULL,
    entity_id                TEXT            NOT NULL,
    entity_name              TEXT,
    alert_type               TEXT            NOT NULL CHECK (alert_type IN (
        'possible_data_exfiltration',
        'possible_credential_compromise',
        'possible_insider_threat',
        'possible_lateral_movement',
        'possible_privilege_abuse',
        'unusual_activity',
        'data_reconnaissance',
        'policy_violation'
    )),
    severity                 TEXT            NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    confidence               DECIMAL(5,4)    NOT NULL,
    risk_score_before        DECIMAL(5,2)    NOT NULL,
    risk_score_after         DECIMAL(5,2)    NOT NULL,
    risk_score_delta         DECIMAL(5,2)    NOT NULL,
    title                    TEXT            NOT NULL,
    description              TEXT            NOT NULL,
    triggering_signals       JSONB           NOT NULL,
    triggering_event_ids     UUID[]          NOT NULL,
    baseline_comparison      JSONB           NOT NULL,
    correlated_signal_count  INT             NOT NULL DEFAULT 1,
    correlation_window_start TIMESTAMPTZ     NOT NULL,
    correlation_window_end   TIMESTAMPTZ     NOT NULL,
    mitre_technique_ids      TEXT[],
    mitre_tactic             TEXT,
    status                   TEXT            NOT NULL DEFAULT 'new'
                                          CHECK (status IN ('new', 'acknowledged', 'investigating', 'resolved', 'false_positive')),
    resolved_at              TIMESTAMPTZ,
    resolved_by              UUID,
    resolution_notes         TEXT,
    created_at               TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ueba_alerts_tenant
    ON ueba_alerts (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ueba_alerts_entity
    ON ueba_alerts (tenant_id, entity_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ueba_alerts_type
    ON ueba_alerts (tenant_id, alert_type, severity);
CREATE INDEX IF NOT EXISTS idx_ueba_alerts_status
    ON ueba_alerts (tenant_id, status)
    WHERE status != 'resolved';

CREATE OR REPLACE FUNCTION manage_ueba_event_partitions(retention_days INTEGER DEFAULT 90)
RETURNS VOID AS $$
DECLARE
    keep_after DATE := CURRENT_DATE - retention_days;
    next_month DATE := date_trunc('month', CURRENT_DATE);
    partition_date DATE;
    partition_name TEXT;
    part RECORD;
BEGIN
    FOR i IN 0..3 LOOP
        partition_date := next_month + (i || ' months')::interval;
        partition_name := 'ueba_access_events_' || to_char(partition_date, 'YYYY_MM');
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS %I PARTITION OF ueba_access_events
             FOR VALUES FROM (%L) TO (%L)',
            partition_name,
            date_trunc('month', partition_date),
            date_trunc('month', partition_date + interval '1 month')
        );
    END LOOP;

    FOR part IN
        SELECT c.relname AS name
        FROM pg_catalog.pg_inherits i
        JOIN pg_catalog.pg_class c ON c.oid = i.inhrelid
        JOIN pg_catalog.pg_class p ON p.oid = i.inhparent
        WHERE p.relname = 'ueba_access_events'
          AND c.relname ~ '^ueba_access_events_[0-9]{4}_[0-9]{2}$'
    LOOP
        BEGIN
            partition_date := to_date(replace(substr(part.name, length('ueba_access_events_') + 1), '_', '-'), 'YYYY-MM');
            IF partition_date < date_trunc('month', keep_after) THEN
                EXECUTE format('DROP TABLE IF EXISTS %I', part.name);
            END IF;
        EXCEPTION
            WHEN others THEN
                NULL;
        END;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

ALTER TABLE ueba_profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE ueba_profiles FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON ueba_profiles;
DROP POLICY IF EXISTS tenant_insert ON ueba_profiles;
DROP POLICY IF EXISTS tenant_update ON ueba_profiles;
DROP POLICY IF EXISTS tenant_delete ON ueba_profiles;
CREATE POLICY tenant_isolation ON ueba_profiles
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON ueba_profiles
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON ueba_profiles
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON ueba_profiles
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE ueba_access_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE ueba_access_events FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON ueba_access_events;
DROP POLICY IF EXISTS tenant_insert ON ueba_access_events;
DROP POLICY IF EXISTS tenant_update ON ueba_access_events;
DROP POLICY IF EXISTS tenant_delete ON ueba_access_events;
CREATE POLICY tenant_isolation ON ueba_access_events
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON ueba_access_events
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON ueba_access_events
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON ueba_access_events
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE ueba_alerts ENABLE ROW LEVEL SECURITY;
ALTER TABLE ueba_alerts FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON ueba_alerts;
DROP POLICY IF EXISTS tenant_insert ON ueba_alerts;
DROP POLICY IF EXISTS tenant_update ON ueba_alerts;
DROP POLICY IF EXISTS tenant_delete ON ueba_alerts;
CREATE POLICY tenant_isolation ON ueba_alerts
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON ueba_alerts
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON ueba_alerts
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON ueba_alerts
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
