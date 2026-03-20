CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS visus_dashboards (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL,
    name            TEXT        NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    grid_columns    INT         NOT NULL DEFAULT 12,
    visibility      TEXT        NOT NULL DEFAULT 'private'
                                CHECK (visibility IN ('private', 'team', 'organization', 'public')),
    shared_with     UUID[]      NOT NULL DEFAULT '{}',
    is_default      BOOLEAN     NOT NULL DEFAULT false,
    is_system       BOOLEAN     NOT NULL DEFAULT false,
    tags            TEXT[]      NOT NULL DEFAULT '{}',
    metadata        JSONB       NOT NULL DEFAULT '{}',
    created_by      UUID        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_visus_dashboards_tenant_name_created_by
    ON visus_dashboards (tenant_id, name, created_by)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_visus_dashboards_tenant
    ON visus_dashboards (tenant_id, visibility)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_visus_dashboards_user
    ON visus_dashboards (created_by)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS visus_widgets (
    id                        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID        NOT NULL,
    dashboard_id              UUID        NOT NULL REFERENCES visus_dashboards(id) ON DELETE CASCADE,
    title                     TEXT        NOT NULL,
    subtitle                  TEXT,
    type                      TEXT        NOT NULL CHECK (type IN (
                                'kpi_card', 'line_chart', 'bar_chart', 'area_chart', 'pie_chart',
                                'gauge', 'table', 'alert_feed', 'text', 'sparkline',
                                'heatmap', 'status_grid', 'trend_indicator'
                              )),
    config                    JSONB       NOT NULL,
    pos_x                     INT         NOT NULL CHECK (pos_x BETWEEN 0 AND 11),
    pos_y                     INT         NOT NULL CHECK (pos_y >= 0),
    pos_w                     INT         NOT NULL CHECK (pos_w BETWEEN 1 AND 12),
    pos_h                     INT         NOT NULL CHECK (pos_h BETWEEN 1 AND 8),
    refresh_interval_seconds  INT         NOT NULL DEFAULT 60,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_visus_widget_x_bound CHECK (pos_x + pos_w <= 12)
);

CREATE INDEX IF NOT EXISTS idx_visus_widgets_dashboard
    ON visus_widgets (dashboard_id);

CREATE TABLE IF NOT EXISTS visus_kpi_definitions (
    id                    UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID            NOT NULL,
    name                  TEXT            NOT NULL,
    description           TEXT            NOT NULL DEFAULT '',
    category              TEXT            NOT NULL DEFAULT 'general'
                                          CHECK (category IN ('security', 'data', 'governance', 'legal', 'operations', 'general')),
    suite                 TEXT            NOT NULL CHECK (suite IN ('cyber', 'data', 'acta', 'lex', 'platform', 'custom')),
    icon                  TEXT,
    query_endpoint        TEXT            NOT NULL,
    query_params          JSONB           NOT NULL DEFAULT '{}',
    value_path            TEXT            NOT NULL,
    unit                  TEXT            NOT NULL DEFAULT 'count'
                                          CHECK (unit IN ('count', 'percentage', 'hours', 'minutes', 'score', 'currency', 'ratio', 'bytes')),
    format_pattern        TEXT,
    target_value          DECIMAL(15,2),
    warning_threshold     DECIMAL(15,2),
    critical_threshold    DECIMAL(15,2),
    direction             TEXT            NOT NULL DEFAULT 'lower_is_better'
                                          CHECK (direction IN ('higher_is_better', 'lower_is_better')),
    calculation_type      TEXT            NOT NULL DEFAULT 'direct'
                                          CHECK (calculation_type IN ('direct', 'delta', 'percentage_change', 'average_over_period', 'sum_over_period')),
    calculation_window    TEXT,
    snapshot_frequency    TEXT            NOT NULL DEFAULT 'hourly'
                                          CHECK (snapshot_frequency IN ('every_15m', 'hourly', 'every_4h', 'daily', 'weekly')),
    enabled               BOOLEAN         NOT NULL DEFAULT true,
    is_default            BOOLEAN         NOT NULL DEFAULT false,
    last_snapshot_at      TIMESTAMPTZ,
    last_value            DECIMAL(15,2),
    last_status           TEXT            CHECK (last_status IN ('normal', 'warning', 'critical', 'unknown')),
    tags                  TEXT[]          NOT NULL DEFAULT '{}',
    created_by            UUID            NOT NULL,
    created_at            TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ     NOT NULL DEFAULT now(),
    deleted_at            TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_visus_kpi_definitions_tenant_name
    ON visus_kpi_definitions (tenant_id, name)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_visus_kpis_tenant
    ON visus_kpi_definitions (tenant_id, enabled)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_visus_kpis_suite
    ON visus_kpi_definitions (tenant_id, suite)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_visus_kpis_schedule
    ON visus_kpi_definitions (snapshot_frequency, last_snapshot_at)
    WHERE enabled = true AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS visus_kpi_snapshots (
    id                UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID            NOT NULL,
    kpi_id            UUID            NOT NULL REFERENCES visus_kpi_definitions(id) ON DELETE CASCADE,
    value             DECIMAL(15,2)   NOT NULL,
    previous_value    DECIMAL(15,2),
    delta             DECIMAL(15,2),
    delta_percent     DECIMAL(15,2),
    status            TEXT            NOT NULL CHECK (status IN ('normal', 'warning', 'critical', 'unknown')),
    period_start      TIMESTAMPTZ     NOT NULL,
    period_end        TIMESTAMPTZ     NOT NULL,
    fetch_success     BOOLEAN         NOT NULL DEFAULT true,
    fetch_latency_ms  INT,
    fetch_error       TEXT,
    created_at        TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_kpi_snapshots_kpi
    ON visus_kpi_snapshots (kpi_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_kpi_snapshots_tenant
    ON visus_kpi_snapshots (tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS visus_executive_alerts (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID        NOT NULL,
    title               TEXT        NOT NULL,
    description         TEXT        NOT NULL,
    category            TEXT        NOT NULL CHECK (category IN (
                                'risk', 'compliance', 'data_quality', 'governance', 'legal',
                                'operational', 'financial', 'strategic'
                              )),
    severity            TEXT        NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info')),
    source_suite        TEXT        NOT NULL,
    source_type         TEXT        NOT NULL,
    source_entity_id    UUID,
    source_event_type   TEXT,
    status              TEXT        NOT NULL DEFAULT 'new'
                                CHECK (status IN ('new', 'viewed', 'acknowledged', 'actioned', 'dismissed', 'escalated')),
    viewed_at           TIMESTAMPTZ,
    viewed_by           UUID,
    actioned_at         TIMESTAMPTZ,
    actioned_by         UUID,
    action_notes        TEXT,
    dismissed_at        TIMESTAMPTZ,
    dismissed_by        UUID,
    dismiss_reason      TEXT,
    dedup_key           TEXT,
    occurrence_count    INT         NOT NULL DEFAULT 1,
    first_seen_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    linked_kpi_id       UUID        REFERENCES visus_kpi_definitions(id),
    linked_dashboard_id UUID,
    metadata            JSONB       NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_visus_alerts_tenant
    ON visus_executive_alerts (tenant_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_visus_alerts_category
    ON visus_executive_alerts (tenant_id, category, severity);
CREATE INDEX IF NOT EXISTS idx_visus_alerts_dedup
    ON visus_executive_alerts (tenant_id, dedup_key, last_seen_at DESC)
    WHERE status IN ('new', 'viewed', 'acknowledged');

CREATE TABLE IF NOT EXISTS visus_report_definitions (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID        NOT NULL,
    name                TEXT        NOT NULL,
    description         TEXT        NOT NULL DEFAULT '',
    report_type         TEXT        NOT NULL DEFAULT 'executive_summary'
                                CHECK (report_type IN ('executive_summary', 'security_posture',
                                    'data_intelligence', 'governance', 'legal', 'custom')),
    sections            TEXT[]      NOT NULL,
    period              TEXT        NOT NULL DEFAULT '30d'
                                CHECK (period IN ('7d', '14d', '30d', '90d', 'quarterly', 'annual', 'custom')),
    custom_period_start DATE,
    custom_period_end   DATE,
    schedule            TEXT,
    next_run_at         TIMESTAMPTZ,
    recipients          UUID[]      NOT NULL DEFAULT '{}',
    auto_send           BOOLEAN     NOT NULL DEFAULT false,
    last_generated_at   TIMESTAMPTZ,
    total_generated     INT         NOT NULL DEFAULT 0,
    created_by          UUID        NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_visus_report_definitions_tenant_name
    ON visus_report_definitions (tenant_id, name)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_visus_reports_tenant
    ON visus_report_definitions (tenant_id)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_visus_reports_schedule
    ON visus_report_definitions (next_run_at)
    WHERE schedule IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS visus_report_snapshots (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID        NOT NULL,
    report_id           UUID        NOT NULL REFERENCES visus_report_definitions(id) ON DELETE CASCADE,
    report_data         JSONB       NOT NULL,
    narrative           TEXT,
    file_id             UUID,
    file_format         TEXT        NOT NULL DEFAULT 'json'
                                CHECK (file_format IN ('json', 'pdf', 'html')),
    period_start        DATE        NOT NULL,
    period_end          DATE        NOT NULL,
    sections_included   TEXT[]      NOT NULL,
    generation_time_ms  BIGINT,
    suite_fetch_errors  JSONB       NOT NULL DEFAULT '{}',
    generated_by        UUID,
    generated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_visus_report_snapshots_idempotent
    ON visus_report_snapshots (tenant_id, report_id, period_start, period_end);
CREATE INDEX IF NOT EXISTS idx_report_snapshots_report
    ON visus_report_snapshots (report_id, generated_at DESC);

CREATE TABLE IF NOT EXISTS visus_suite_cache (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    suite             TEXT        NOT NULL,
    endpoint          TEXT        NOT NULL,
    response_data     JSONB       NOT NULL,
    fetched_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    ttl_seconds       INT         NOT NULL DEFAULT 60,
    fetch_latency_ms  INT,
    UNIQUE (tenant_id, suite, endpoint)
);

CREATE INDEX IF NOT EXISTS idx_suite_cache_tenant
    ON visus_suite_cache (tenant_id, suite);

DROP TRIGGER IF EXISTS trg_visus_dashboards_updated_at ON visus_dashboards;
CREATE TRIGGER trg_visus_dashboards_updated_at
    BEFORE UPDATE ON visus_dashboards
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trg_visus_widgets_updated_at ON visus_widgets;
CREATE TRIGGER trg_visus_widgets_updated_at
    BEFORE UPDATE ON visus_widgets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trg_visus_kpi_definitions_updated_at ON visus_kpi_definitions;
CREATE TRIGGER trg_visus_kpi_definitions_updated_at
    BEFORE UPDATE ON visus_kpi_definitions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trg_visus_executive_alerts_updated_at ON visus_executive_alerts;
CREATE TRIGGER trg_visus_executive_alerts_updated_at
    BEFORE UPDATE ON visus_executive_alerts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trg_visus_report_definitions_updated_at ON visus_report_definitions;
CREATE TRIGGER trg_visus_report_definitions_updated_at
    BEFORE UPDATE ON visus_report_definitions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
