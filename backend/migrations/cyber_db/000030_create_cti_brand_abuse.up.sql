-- =============================================================================
-- Clario 360 — CTI Brand Abuse Monitoring
-- Migration 000030: Monitored brands and abuse incident tracking
-- =============================================================================

-- ---------------------------------------------------------------------------
-- 1. cti_monitored_brands
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cti_monitored_brands (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    brand_name     VARCHAR(300) NOT NULL,
    domain_pattern VARCHAR(500),
    logo_file_id   UUID,
    keywords       TEXT[] DEFAULT '{}',
    is_active      BOOLEAN NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by     UUID,
    updated_by     UUID,
    CONSTRAINT uq_cti_brand_tenant_name UNIQUE (tenant_id, brand_name)
);

COMMENT ON TABLE cti_monitored_brands IS 'Brands monitored for phishing, typosquatting, and abuse';

ALTER TABLE cti_monitored_brands ENABLE ROW LEVEL SECURITY;
ALTER TABLE cti_monitored_brands FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cti_monitored_brands
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON cti_monitored_brands
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON cti_monitored_brands
    FOR UPDATE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON cti_monitored_brands
    FOR DELETE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ---------------------------------------------------------------------------
-- 2. cti_brand_abuse_incidents
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cti_brand_abuse_incidents (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    brand_id             UUID NOT NULL REFERENCES cti_monitored_brands(id) ON DELETE CASCADE,
    malicious_domain     VARCHAR(500) NOT NULL,
    abuse_type           VARCHAR(50) NOT NULL,
    risk_level           VARCHAR(20) NOT NULL DEFAULT 'medium',
    region_id            UUID REFERENCES cti_geographic_regions(id) ON DELETE SET NULL,
    detection_count      INTEGER NOT NULL DEFAULT 1,
    source_id            UUID REFERENCES cti_data_sources(id) ON DELETE SET NULL,
    -- WHOIS / hosting
    whois_registrant     TEXT,
    whois_created_date   DATE,
    ssl_issuer           VARCHAR(300),
    hosting_ip           INET,
    hosting_asn          VARCHAR(20),
    screenshot_file_id   UUID,
    -- Takedown lifecycle
    takedown_status      VARCHAR(30) NOT NULL DEFAULT 'detected',
    takedown_requested_at TIMESTAMPTZ,
    taken_down_at        TIMESTAMPTZ,
    -- Timestamps
    first_detected_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_detected_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by           UUID,
    updated_by           UUID,
    deleted_at           TIMESTAMPTZ
);

COMMENT ON TABLE cti_brand_abuse_incidents IS 'Individual brand-abuse incidents (phishing, typosquat, etc.)';

CREATE INDEX idx_cti_brand_abuse_brand_risk
    ON cti_brand_abuse_incidents (tenant_id, brand_id, risk_level)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_cti_brand_abuse_takedown
    ON cti_brand_abuse_incidents (tenant_id, takedown_status)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_cti_brand_abuse_time
    ON cti_brand_abuse_incidents (tenant_id, first_detected_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_cti_brand_abuse_domain
    ON cti_brand_abuse_incidents (tenant_id, malicious_domain)
    WHERE deleted_at IS NULL;

ALTER TABLE cti_brand_abuse_incidents ENABLE ROW LEVEL SECURITY;
ALTER TABLE cti_brand_abuse_incidents FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cti_brand_abuse_incidents
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON cti_brand_abuse_incidents
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON cti_brand_abuse_incidents
    FOR UPDATE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON cti_brand_abuse_incidents
    FOR DELETE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
