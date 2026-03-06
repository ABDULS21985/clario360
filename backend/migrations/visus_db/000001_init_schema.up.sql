-- =============================================================================
-- Clario 360 — Visus Suite Database Schema
-- Database: visus_db
-- Contains: dashboards, widgets, KPI definitions/snapshots, executive alerts,
--           reports, report snapshots
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- ENUM TYPES
-- =============================================================================

CREATE TYPE widget_type AS ENUM (
    'kpi_card', 'line_chart', 'bar_chart', 'pie_chart',
    'table', 'heatmap', 'gauge', 'alert_feed', 'text'
);
COMMENT ON TYPE widget_type IS 'Types of dashboard widgets';

CREATE TYPE suite_name AS ENUM ('cyber', 'data', 'acta', 'lex', 'platform');
COMMENT ON TYPE suite_name IS 'Platform suites for KPI categorization';

CREATE TYPE kpi_calculation_type AS ENUM ('count', 'sum', 'average', 'percentage', 'custom');
COMMENT ON TYPE kpi_calculation_type IS 'How KPI values are calculated';

CREATE TYPE alert_severity AS ENUM ('critical', 'high', 'medium', 'low', 'info');
COMMENT ON TYPE alert_severity IS 'Severity levels for executive alerts';

CREATE TYPE alert_category AS ENUM (
    'risk', 'opportunity', 'anomaly', 'threshold_breach', 'compliance'
);
COMMENT ON TYPE alert_category IS 'Categories of executive alerts';

CREATE TYPE alert_status AS ENUM ('new', 'viewed', 'actioned', 'dismissed');
COMMENT ON TYPE alert_status IS 'Lifecycle status of an executive alert';

CREATE TYPE report_type AS ENUM ('scheduled', 'on_demand', 'automated');
COMMENT ON TYPE report_type IS 'Types of reports by generation trigger';

-- =============================================================================
-- TRIGGER FUNCTION
-- =============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- TABLE: dashboards
-- =============================================================================

CREATE TABLE dashboards (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL,
    name          VARCHAR(255) NOT NULL,
    description   TEXT        NOT NULL DEFAULT '',
    layout        JSONB       NOT NULL DEFAULT '{}',
    is_default    BOOLEAN     NOT NULL DEFAULT false,
    owner_user_id UUID,
    shared_with   JSONB       DEFAULT '[]',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by    UUID,
    updated_by    UUID
);

COMMENT ON TABLE dashboards IS 'User-configurable dashboards with widget layouts';
COMMENT ON COLUMN dashboards.layout IS 'Dashboard layout configuration (grid positions, breakpoints)';
COMMENT ON COLUMN dashboards.is_default IS 'Whether this is the default dashboard for the tenant';
COMMENT ON COLUMN dashboards.owner_user_id IS 'Dashboard owner (references platform_core.users)';
COMMENT ON COLUMN dashboards.shared_with IS 'JSON array of user/role IDs this dashboard is shared with';

CREATE INDEX idx_dashboards_tenant ON dashboards (tenant_id);
CREATE INDEX idx_dashboards_owner ON dashboards (owner_user_id);
CREATE INDEX idx_dashboards_default ON dashboards (tenant_id, is_default) WHERE is_default = true;
CREATE INDEX idx_dashboards_shared ON dashboards USING GIN (shared_with);
CREATE INDEX idx_dashboards_layout ON dashboards USING GIN (layout);

CREATE TRIGGER trg_dashboards_updated_at
    BEFORE UPDATE ON dashboards
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: dashboard_widgets
-- =============================================================================

CREATE TABLE dashboard_widgets (
    id                      UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID        NOT NULL,
    dashboard_id            UUID        NOT NULL REFERENCES dashboards(id) ON DELETE CASCADE,
    type                    widget_type NOT NULL,
    title                   VARCHAR(255) NOT NULL,
    config                  JSONB       NOT NULL DEFAULT '{}',
    position                JSONB       NOT NULL DEFAULT '{"x": 0, "y": 0, "w": 4, "h": 3}',
    refresh_interval_seconds INTEGER    DEFAULT 300,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID,
    updated_by              UUID
);

COMMENT ON TABLE dashboard_widgets IS 'Individual widgets displayed on dashboards';
COMMENT ON COLUMN dashboard_widgets.config IS 'Widget configuration (data source, query, visualization settings)';
COMMENT ON COLUMN dashboard_widgets.position IS 'Grid position {x, y, w, h} for responsive layout';
COMMENT ON COLUMN dashboard_widgets.refresh_interval_seconds IS 'How often the widget refreshes its data';

CREATE INDEX idx_widgets_dashboard ON dashboard_widgets (dashboard_id);
CREATE INDEX idx_widgets_tenant ON dashboard_widgets (tenant_id);
CREATE INDEX idx_widgets_type ON dashboard_widgets (type);
CREATE INDEX idx_widgets_config ON dashboard_widgets USING GIN (config);

CREATE TRIGGER trg_widgets_updated_at
    BEFORE UPDATE ON dashboard_widgets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: kpi_definitions
-- =============================================================================

CREATE TABLE kpi_definitions (
    id                  UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID               NOT NULL,
    name                VARCHAR(255)       NOT NULL,
    description         TEXT               NOT NULL DEFAULT '',
    suite               suite_name         NOT NULL,
    query_config        JSONB              NOT NULL DEFAULT '{}',
    unit                VARCHAR(50),
    target_value        DECIMAL(15,4),
    warning_threshold   DECIMAL(15,4),
    critical_threshold  DECIMAL(15,4),
    calculation_type    kpi_calculation_type NOT NULL DEFAULT 'count',
    created_at          TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    created_by          UUID,
    updated_by          UUID
);

COMMENT ON TABLE kpi_definitions IS 'KPI metric definitions with targets and thresholds';
COMMENT ON COLUMN kpi_definitions.suite IS 'Which platform suite this KPI belongs to';
COMMENT ON COLUMN kpi_definitions.query_config IS 'Configuration for calculating the KPI value';
COMMENT ON COLUMN kpi_definitions.unit IS 'Unit of measure (e.g., %, count, SAR)';
COMMENT ON COLUMN kpi_definitions.target_value IS 'Target/goal value for the KPI';
COMMENT ON COLUMN kpi_definitions.warning_threshold IS 'Value at which a warning is triggered';
COMMENT ON COLUMN kpi_definitions.critical_threshold IS 'Value at which a critical alert is triggered';

CREATE INDEX idx_kpi_defs_tenant_suite ON kpi_definitions (tenant_id, suite);
CREATE INDEX idx_kpi_defs_tenant ON kpi_definitions (tenant_id);
CREATE INDEX idx_kpi_defs_query ON kpi_definitions USING GIN (query_config);

CREATE TRIGGER trg_kpi_defs_updated_at
    BEFORE UPDATE ON kpi_definitions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: kpi_snapshots (PARTITIONED by created_at — monthly)
-- =============================================================================

CREATE TABLE kpi_snapshots (
    id           UUID        NOT NULL DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL,
    kpi_id       UUID        NOT NULL,
    value        DECIMAL(15,4) NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end   TIMESTAMPTZ NOT NULL,
    metadata     JSONB       DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

COMMENT ON TABLE kpi_snapshots IS 'Time-series KPI value snapshots (partitioned monthly)';
COMMENT ON COLUMN kpi_snapshots.kpi_id IS 'References kpi_definitions.id';
COMMENT ON COLUMN kpi_snapshots.value IS 'Calculated KPI value for the period';
COMMENT ON COLUMN kpi_snapshots.period_start IS 'Start of the measurement period';
COMMENT ON COLUMN kpi_snapshots.period_end IS 'End of the measurement period';

-- Create monthly partitions
CREATE TABLE kpi_snapshots_2025_01 PARTITION OF kpi_snapshots FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
CREATE TABLE kpi_snapshots_2025_02 PARTITION OF kpi_snapshots FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
CREATE TABLE kpi_snapshots_2025_03 PARTITION OF kpi_snapshots FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');
CREATE TABLE kpi_snapshots_2025_04 PARTITION OF kpi_snapshots FOR VALUES FROM ('2025-04-01') TO ('2025-05-01');
CREATE TABLE kpi_snapshots_2025_05 PARTITION OF kpi_snapshots FOR VALUES FROM ('2025-05-01') TO ('2025-06-01');
CREATE TABLE kpi_snapshots_2025_06 PARTITION OF kpi_snapshots FOR VALUES FROM ('2025-06-01') TO ('2025-07-01');
CREATE TABLE kpi_snapshots_2025_07 PARTITION OF kpi_snapshots FOR VALUES FROM ('2025-07-01') TO ('2025-08-01');
CREATE TABLE kpi_snapshots_2025_08 PARTITION OF kpi_snapshots FOR VALUES FROM ('2025-08-01') TO ('2025-09-01');
CREATE TABLE kpi_snapshots_2025_09 PARTITION OF kpi_snapshots FOR VALUES FROM ('2025-09-01') TO ('2025-10-01');
CREATE TABLE kpi_snapshots_2025_10 PARTITION OF kpi_snapshots FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');
CREATE TABLE kpi_snapshots_2025_11 PARTITION OF kpi_snapshots FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
CREATE TABLE kpi_snapshots_2025_12 PARTITION OF kpi_snapshots FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');
CREATE TABLE kpi_snapshots_2026_01 PARTITION OF kpi_snapshots FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE kpi_snapshots_2026_02 PARTITION OF kpi_snapshots FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE kpi_snapshots_2026_03 PARTITION OF kpi_snapshots FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE kpi_snapshots_2026_04 PARTITION OF kpi_snapshots FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE kpi_snapshots_2026_05 PARTITION OF kpi_snapshots FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE kpi_snapshots_2026_06 PARTITION OF kpi_snapshots FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE kpi_snapshots_2026_07 PARTITION OF kpi_snapshots FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
CREATE TABLE kpi_snapshots_2026_08 PARTITION OF kpi_snapshots FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');
CREATE TABLE kpi_snapshots_2026_09 PARTITION OF kpi_snapshots FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');
CREATE TABLE kpi_snapshots_2026_10 PARTITION OF kpi_snapshots FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');
CREATE TABLE kpi_snapshots_2026_11 PARTITION OF kpi_snapshots FOR VALUES FROM ('2026-11-01') TO ('2026-12-01');
CREATE TABLE kpi_snapshots_2026_12 PARTITION OF kpi_snapshots FOR VALUES FROM ('2026-12-01') TO ('2027-01-01');

-- Default partition for out-of-range dates
CREATE TABLE kpi_snapshots_default PARTITION OF kpi_snapshots DEFAULT;

CREATE INDEX idx_kpi_snap_tenant_kpi ON kpi_snapshots (tenant_id, kpi_id, created_at DESC);
CREATE INDEX idx_kpi_snap_period ON kpi_snapshots (period_start, period_end);
CREATE INDEX idx_kpi_snap_metadata ON kpi_snapshots USING GIN (metadata);

-- =============================================================================
-- TABLE: executive_alerts
-- =============================================================================

CREATE TABLE executive_alerts (
    id           UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID             NOT NULL,
    source_suite suite_name       NOT NULL,
    title        VARCHAR(500)     NOT NULL,
    description  TEXT             NOT NULL DEFAULT '',
    severity     alert_severity   NOT NULL DEFAULT 'medium',
    category     alert_category   NOT NULL,
    data         JSONB            DEFAULT '{}',
    status       alert_status     NOT NULL DEFAULT 'new',
    created_at   TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    created_by   UUID,
    updated_by   UUID
);

COMMENT ON TABLE executive_alerts IS 'High-level alerts surfaced to executives from all suites';
COMMENT ON COLUMN executive_alerts.source_suite IS 'Which suite generated this alert';
COMMENT ON COLUMN executive_alerts.category IS 'Classification of the alert';
COMMENT ON COLUMN executive_alerts.data IS 'Additional context and data for the alert';

CREATE INDEX idx_exec_alerts_tenant_status ON executive_alerts (tenant_id, status);
CREATE INDEX idx_exec_alerts_tenant_severity ON executive_alerts (tenant_id, severity);
CREATE INDEX idx_exec_alerts_suite ON executive_alerts (tenant_id, source_suite);
CREATE INDEX idx_exec_alerts_category ON executive_alerts (tenant_id, category);
CREATE INDEX idx_exec_alerts_tenant_created ON executive_alerts (tenant_id, created_at DESC);
CREATE INDEX idx_exec_alerts_data ON executive_alerts USING GIN (data);

CREATE TRIGGER trg_exec_alerts_updated_at
    BEFORE UPDATE ON executive_alerts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: reports
-- =============================================================================

CREATE TABLE reports (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    name             VARCHAR(255) NOT NULL,
    type             report_type NOT NULL,
    config           JSONB       NOT NULL DEFAULT '{}',
    schedule         TEXT,
    last_generated_at TIMESTAMPTZ,
    file_url         TEXT,
    created_by       UUID,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by       UUID
);

COMMENT ON TABLE reports IS 'Report definitions with optional scheduling';
COMMENT ON COLUMN reports.config IS 'Report configuration (data sources, filters, format)';
COMMENT ON COLUMN reports.schedule IS 'Cron expression for scheduled reports';
COMMENT ON COLUMN reports.file_url IS 'URL to the most recently generated report file';

CREATE INDEX idx_reports_tenant ON reports (tenant_id);
CREATE INDEX idx_reports_tenant_type ON reports (tenant_id, type);
CREATE INDEX idx_reports_schedule ON reports (schedule) WHERE schedule IS NOT NULL AND type = 'scheduled';
CREATE INDEX idx_reports_config ON reports USING GIN (config);
CREATE INDEX idx_reports_tenant_created ON reports (tenant_id, created_at DESC);

CREATE TRIGGER trg_reports_updated_at
    BEFORE UPDATE ON reports
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: report_snapshots
-- =============================================================================

CREATE TABLE report_snapshots (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL,
    report_id    UUID        NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    file_url     TEXT        NOT NULL,
    metadata     JSONB       DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE report_snapshots IS 'Historical snapshots of generated reports';
COMMENT ON COLUMN report_snapshots.file_url IS 'URL to the generated report file in object storage';
COMMENT ON COLUMN report_snapshots.metadata IS 'Generation metadata (duration, record count, parameters)';

CREATE INDEX idx_report_snap_tenant ON report_snapshots (tenant_id);
CREATE INDEX idx_report_snap_report ON report_snapshots (report_id, generated_at DESC);
CREATE INDEX idx_report_snap_generated ON report_snapshots (generated_at DESC);
