-- =============================================================================
-- Clario 360 — CTI Reference / Lookup Tables
-- Migration 000027: Core reference data for Cyber Threat Intelligence module
-- =============================================================================

-- ---------------------------------------------------------------------------
-- 1. cti_threat_severity_levels
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cti_threat_severity_levels (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    code        VARCHAR(20) NOT NULL,
    label       VARCHAR(50) NOT NULL,
    color_hex   VARCHAR(7) NOT NULL DEFAULT '#6B7280',
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_cti_severity_tenant_code UNIQUE (tenant_id, code)
);
COMMENT ON TABLE cti_threat_severity_levels IS 'Enum-like severity lookup for CTI events';

ALTER TABLE cti_threat_severity_levels ENABLE ROW LEVEL SECURITY;
ALTER TABLE cti_threat_severity_levels FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cti_threat_severity_levels
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON cti_threat_severity_levels
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON cti_threat_severity_levels
    FOR UPDATE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON cti_threat_severity_levels
    FOR DELETE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ---------------------------------------------------------------------------
-- 2. cti_threat_categories
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cti_threat_categories (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    code            VARCHAR(50) NOT NULL,
    label           VARCHAR(100) NOT NULL,
    description     TEXT,
    mitre_tactic_ids TEXT[] DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_cti_category_tenant_code UNIQUE (tenant_id, code)
);
COMMENT ON TABLE cti_threat_categories IS 'Threat category taxonomy for CTI events';

ALTER TABLE cti_threat_categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE cti_threat_categories FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cti_threat_categories
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON cti_threat_categories
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON cti_threat_categories
    FOR UPDATE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON cti_threat_categories
    FOR DELETE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ---------------------------------------------------------------------------
-- 3. cti_geographic_regions
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cti_geographic_regions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL,
    code             VARCHAR(50) NOT NULL,
    label            VARCHAR(200) NOT NULL,
    parent_region_id UUID REFERENCES cti_geographic_regions(id) ON DELETE SET NULL,
    latitude         DECIMAL(10,7),
    longitude        DECIMAL(10,7),
    iso_country_code VARCHAR(3),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_cti_region_tenant_code UNIQUE (tenant_id, code)
);
COMMENT ON TABLE cti_geographic_regions IS 'Hierarchical geographic regions (continent > sub-region > country)';

CREATE INDEX idx_cti_geo_regions_parent ON cti_geographic_regions (tenant_id, parent_region_id);
CREATE INDEX idx_cti_geo_regions_country ON cti_geographic_regions (tenant_id, iso_country_code)
    WHERE iso_country_code IS NOT NULL;

ALTER TABLE cti_geographic_regions ENABLE ROW LEVEL SECURITY;
ALTER TABLE cti_geographic_regions FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cti_geographic_regions
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON cti_geographic_regions
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON cti_geographic_regions
    FOR UPDATE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON cti_geographic_regions
    FOR DELETE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ---------------------------------------------------------------------------
-- 4. cti_industry_sectors
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cti_industry_sectors (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    code        VARCHAR(50) NOT NULL,
    label       VARCHAR(100) NOT NULL,
    description TEXT,
    naics_code  VARCHAR(10),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_cti_sector_tenant_code UNIQUE (tenant_id, code)
);
COMMENT ON TABLE cti_industry_sectors IS 'Industry / sector classification for CTI targeting analysis';

ALTER TABLE cti_industry_sectors ENABLE ROW LEVEL SECURITY;
ALTER TABLE cti_industry_sectors FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cti_industry_sectors
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON cti_industry_sectors
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON cti_industry_sectors
    FOR UPDATE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON cti_industry_sectors
    FOR DELETE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ---------------------------------------------------------------------------
-- 5. cti_data_sources
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cti_data_sources (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    name                 VARCHAR(200) NOT NULL,
    source_type          VARCHAR(30) NOT NULL,
    url                  TEXT,
    api_endpoint         TEXT,
    api_key_vault_path   TEXT,
    reliability_score    DECIMAL(3,2) NOT NULL DEFAULT 0.50
                         CHECK (reliability_score >= 0 AND reliability_score <= 1),
    is_active            BOOLEAN NOT NULL DEFAULT true,
    last_polled_at       TIMESTAMPTZ,
    poll_interval_seconds INTEGER,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_cti_source_tenant_name UNIQUE (tenant_id, name)
);
COMMENT ON TABLE cti_data_sources IS 'External and internal CTI data sources';

ALTER TABLE cti_data_sources ENABLE ROW LEVEL SECURITY;
ALTER TABLE cti_data_sources FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cti_data_sources
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON cti_data_sources
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON cti_data_sources
    FOR UPDATE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON cti_data_sources
    FOR DELETE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
