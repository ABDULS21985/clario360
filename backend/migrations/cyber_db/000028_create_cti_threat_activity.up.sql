-- =============================================================================
-- Clario 360 — CTI Threat Activity Tables
-- Migration 000028: Core threat event stream and tagging
-- =============================================================================

-- ---------------------------------------------------------------------------
-- 1. cti_threat_events — every observed threat / indicator sighting
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cti_threat_events (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    event_type          VARCHAR(50) NOT NULL,
    title               VARCHAR(500) NOT NULL,
    description         TEXT,
    severity_id         UUID REFERENCES cti_threat_severity_levels(id) ON DELETE SET NULL,
    category_id         UUID REFERENCES cti_threat_categories(id) ON DELETE SET NULL,
    source_id           UUID REFERENCES cti_data_sources(id) ON DELETE SET NULL,
    source_reference    VARCHAR(500),
    confidence_score    DECIMAL(3,2) NOT NULL DEFAULT 0.50
                        CHECK (confidence_score >= 0 AND confidence_score <= 1),
    -- Origin geography
    origin_latitude     DECIMAL(10,7),
    origin_longitude    DECIMAL(10,7),
    origin_country_code VARCHAR(3),
    origin_city         VARCHAR(200),
    origin_region_id    UUID REFERENCES cti_geographic_regions(id) ON DELETE SET NULL,
    -- Target
    target_sector_id    UUID REFERENCES cti_industry_sectors(id) ON DELETE SET NULL,
    target_org_name     VARCHAR(300),
    target_country_code VARCHAR(3),
    -- IOC
    ioc_type            VARCHAR(50),
    ioc_value           TEXT,
    -- MITRE
    mitre_technique_ids TEXT[] DEFAULT '{}',
    -- Raw data
    raw_payload         JSONB DEFAULT '{}',
    -- Resolution
    is_false_positive   BOOLEAN NOT NULL DEFAULT false,
    resolved_at         TIMESTAMPTZ,
    resolved_by         UUID,
    -- Timestamps
    first_seen_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID,
    updated_by          UUID,
    deleted_at          TIMESTAMPTZ
);

COMMENT ON TABLE cti_threat_events IS 'Individual CTI threat events / indicator sightings';

-- Performance indexes
CREATE INDEX idx_cti_events_tenant_severity
    ON cti_threat_events (tenant_id, severity_id, first_seen_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_cti_events_tenant_category
    ON cti_threat_events (tenant_id, category_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_cti_events_tenant_sector
    ON cti_threat_events (tenant_id, target_sector_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_cti_events_ioc
    ON cti_threat_events (tenant_id, ioc_type, ioc_value)
    WHERE deleted_at IS NULL AND ioc_type IS NOT NULL;

CREATE INDEX idx_cti_events_origin_country
    ON cti_threat_events (tenant_id, origin_country_code)
    WHERE deleted_at IS NULL AND origin_country_code IS NOT NULL;

CREATE INDEX idx_cti_events_time
    ON cti_threat_events (tenant_id, first_seen_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_cti_events_target_country
    ON cti_threat_events (tenant_id, target_country_code)
    WHERE deleted_at IS NULL AND target_country_code IS NOT NULL;

CREATE INDEX idx_cti_events_mitre_gin
    ON cti_threat_events USING GIN (mitre_technique_ids)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_cti_events_payload_gin
    ON cti_threat_events USING GIN (raw_payload)
    WHERE deleted_at IS NULL;

-- RLS
ALTER TABLE cti_threat_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE cti_threat_events FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cti_threat_events
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON cti_threat_events
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON cti_threat_events
    FOR UPDATE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON cti_threat_events
    FOR DELETE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ---------------------------------------------------------------------------
-- 2. cti_threat_event_tags
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cti_threat_event_tags (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL,
    event_id   UUID NOT NULL REFERENCES cti_threat_events(id) ON DELETE CASCADE,
    tag        VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_cti_event_tag UNIQUE (tenant_id, event_id, tag)
);

COMMENT ON TABLE cti_threat_event_tags IS 'Free-form tags attached to CTI threat events';

CREATE INDEX idx_cti_event_tags_event ON cti_threat_event_tags (tenant_id, event_id);
CREATE INDEX idx_cti_event_tags_tag   ON cti_threat_event_tags (tenant_id, tag);

ALTER TABLE cti_threat_event_tags ENABLE ROW LEVEL SECURITY;
ALTER TABLE cti_threat_event_tags FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cti_threat_event_tags
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON cti_threat_event_tags
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON cti_threat_event_tags
    FOR UPDATE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON cti_threat_event_tags
    FOR DELETE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
