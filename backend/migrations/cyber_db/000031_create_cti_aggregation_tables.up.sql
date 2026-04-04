-- =============================================================================
-- Clario 360 — CTI Aggregation & Executive Dashboard Tables
-- Migration 000031: Pre-computed summaries for fast dashboard rendering
-- =============================================================================

-- ---------------------------------------------------------------------------
-- 1. cti_geo_threat_summary — per-country threat counts by period
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cti_geo_threat_summary (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID NOT NULL,
    country_code            VARCHAR(3) NOT NULL,
    city                    VARCHAR(200) NOT NULL DEFAULT '',
    latitude                DECIMAL(10,7),
    longitude               DECIMAL(10,7),
    region_id               UUID REFERENCES cti_geographic_regions(id) ON DELETE SET NULL,
    severity_critical_count INTEGER NOT NULL DEFAULT 0,
    severity_high_count     INTEGER NOT NULL DEFAULT 0,
    severity_medium_count   INTEGER NOT NULL DEFAULT 0,
    severity_low_count      INTEGER NOT NULL DEFAULT 0,
    total_count             INTEGER NOT NULL DEFAULT 0,
    top_category_id         UUID REFERENCES cti_threat_categories(id) ON DELETE SET NULL,
    top_threat_type         VARCHAR(100),
    period_start            TIMESTAMPTZ NOT NULL,
    period_end              TIMESTAMPTZ NOT NULL,
    computed_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_cti_geo_summary UNIQUE (tenant_id, country_code, city, period_start, period_end)
);

COMMENT ON TABLE cti_geo_threat_summary IS 'Pre-aggregated geographic threat distribution for the global threat map';

CREATE INDEX idx_cti_geo_summary_tenant_period
    ON cti_geo_threat_summary (tenant_id, period_start, period_end);

CREATE INDEX idx_cti_geo_summary_tenant_country
    ON cti_geo_threat_summary (tenant_id, country_code);

ALTER TABLE cti_geo_threat_summary ENABLE ROW LEVEL SECURITY;
ALTER TABLE cti_geo_threat_summary FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cti_geo_threat_summary
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON cti_geo_threat_summary
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON cti_geo_threat_summary
    FOR UPDATE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON cti_geo_threat_summary
    FOR DELETE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ---------------------------------------------------------------------------
-- 2. cti_sector_threat_summary — per-sector threat counts by period
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cti_sector_threat_summary (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID NOT NULL,
    sector_id               UUID NOT NULL REFERENCES cti_industry_sectors(id) ON DELETE CASCADE,
    severity_critical_count INTEGER NOT NULL DEFAULT 0,
    severity_high_count     INTEGER NOT NULL DEFAULT 0,
    severity_medium_count   INTEGER NOT NULL DEFAULT 0,
    severity_low_count      INTEGER NOT NULL DEFAULT 0,
    total_count             INTEGER NOT NULL DEFAULT 0,
    period_start            TIMESTAMPTZ NOT NULL,
    period_end              TIMESTAMPTZ NOT NULL,
    computed_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_cti_sector_summary UNIQUE (tenant_id, sector_id, period_start, period_end)
);

COMMENT ON TABLE cti_sector_threat_summary IS 'Pre-aggregated sector-level threat distribution';

CREATE INDEX idx_cti_sector_summary_tenant_period
    ON cti_sector_threat_summary (tenant_id, period_start, period_end);

ALTER TABLE cti_sector_threat_summary ENABLE ROW LEVEL SECURITY;
ALTER TABLE cti_sector_threat_summary FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cti_sector_threat_summary
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON cti_sector_threat_summary
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON cti_sector_threat_summary
    FOR UPDATE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON cti_sector_threat_summary
    FOR DELETE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ---------------------------------------------------------------------------
-- 3. cti_executive_snapshot — single KPI row per tenant
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cti_executive_snapshot (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                   UUID NOT NULL,
    total_events_24h            INTEGER NOT NULL DEFAULT 0,
    total_events_7d             INTEGER NOT NULL DEFAULT 0,
    total_events_30d            INTEGER NOT NULL DEFAULT 0,
    active_campaigns_count      INTEGER NOT NULL DEFAULT 0,
    critical_campaigns_count    INTEGER NOT NULL DEFAULT 0,
    total_iocs                  INTEGER NOT NULL DEFAULT 0,
    brand_abuse_critical_count  INTEGER NOT NULL DEFAULT 0,
    brand_abuse_total_count     INTEGER NOT NULL DEFAULT 0,
    top_targeted_sector_id      UUID REFERENCES cti_industry_sectors(id) ON DELETE SET NULL,
    top_threat_origin_country   VARCHAR(3),
    mean_time_to_detect_hours   DECIMAL(8,2),
    mean_time_to_respond_hours  DECIMAL(8,2),
    risk_score_overall          DECIMAL(5,2) NOT NULL DEFAULT 0,
    trend_direction             VARCHAR(10) NOT NULL DEFAULT 'stable',
    trend_percentage            DECIMAL(5,2) NOT NULL DEFAULT 0,
    computed_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_cti_exec_snapshot_tenant UNIQUE (tenant_id)
);

COMMENT ON TABLE cti_executive_snapshot IS 'Single-row CTI KPI snapshot per tenant for executive dashboards';

ALTER TABLE cti_executive_snapshot ENABLE ROW LEVEL SECURITY;
ALTER TABLE cti_executive_snapshot FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cti_executive_snapshot
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON cti_executive_snapshot
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON cti_executive_snapshot
    FOR UPDATE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON cti_executive_snapshot
    FOR DELETE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
