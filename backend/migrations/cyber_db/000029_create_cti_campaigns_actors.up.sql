-- =============================================================================
-- Clario 360 — CTI Campaigns & Threat Actors
-- Migration 000029: Threat actor profiles, campaign tracking, junction tables
-- =============================================================================

-- ---------------------------------------------------------------------------
-- 1. cti_threat_actors
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cti_threat_actors (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID NOT NULL,
    name                  VARCHAR(300) NOT NULL,
    aliases               TEXT[] DEFAULT '{}',
    actor_type            VARCHAR(50) NOT NULL DEFAULT 'unknown',
    origin_country_code   VARCHAR(3),
    origin_region_id      UUID REFERENCES cti_geographic_regions(id) ON DELETE SET NULL,
    sophistication_level  VARCHAR(20) NOT NULL DEFAULT 'intermediate',
    primary_motivation    VARCHAR(50) NOT NULL DEFAULT 'unknown',
    description           TEXT,
    first_observed_at     TIMESTAMPTZ,
    last_activity_at      TIMESTAMPTZ,
    mitre_group_id        VARCHAR(20),
    external_references   JSONB DEFAULT '{}',
    is_active             BOOLEAN NOT NULL DEFAULT true,
    risk_score            DECIMAL(5,2) NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by            UUID,
    updated_by            UUID,
    deleted_at            TIMESTAMPTZ
);

COMMENT ON TABLE cti_threat_actors IS 'Threat actor / APT group profiles';

CREATE INDEX idx_cti_actors_tenant_type
    ON cti_threat_actors (tenant_id, actor_type)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_cti_actors_tenant_active
    ON cti_threat_actors (tenant_id, is_active, last_activity_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_cti_actors_aliases_gin
    ON cti_threat_actors USING GIN (aliases)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_cti_actors_origin
    ON cti_threat_actors (tenant_id, origin_country_code)
    WHERE deleted_at IS NULL AND origin_country_code IS NOT NULL;

ALTER TABLE cti_threat_actors ENABLE ROW LEVEL SECURITY;
ALTER TABLE cti_threat_actors FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cti_threat_actors
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON cti_threat_actors
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON cti_threat_actors
    FOR UPDATE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON cti_threat_actors
    FOR DELETE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ---------------------------------------------------------------------------
-- 2. cti_campaigns
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cti_campaigns (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    campaign_code       VARCHAR(50) NOT NULL,
    name                VARCHAR(300) NOT NULL,
    description         TEXT,
    status              VARCHAR(20) NOT NULL DEFAULT 'active',
    severity_id         UUID REFERENCES cti_threat_severity_levels(id) ON DELETE SET NULL,
    primary_actor_id    UUID REFERENCES cti_threat_actors(id) ON DELETE SET NULL,
    target_sectors      UUID[] DEFAULT '{}',
    target_regions      UUID[] DEFAULT '{}',
    target_description  TEXT,
    mitre_technique_ids TEXT[] DEFAULT '{}',
    ttps_summary        TEXT,
    ioc_count           INTEGER NOT NULL DEFAULT 0,
    event_count         INTEGER NOT NULL DEFAULT 0,
    first_seen_at       TIMESTAMPTZ NOT NULL,
    last_seen_at        TIMESTAMPTZ,
    resolved_at         TIMESTAMPTZ,
    resolved_by         UUID,
    external_references JSONB DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID,
    updated_by          UUID,
    deleted_at          TIMESTAMPTZ,
    CONSTRAINT uq_cti_campaign_tenant_code UNIQUE (tenant_id, campaign_code)
);

COMMENT ON TABLE cti_campaigns IS 'Tracked CTI campaigns with lifecycle management';

CREATE INDEX idx_cti_campaigns_tenant_status
    ON cti_campaigns (tenant_id, status, severity_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_cti_campaigns_tenant_actor
    ON cti_campaigns (tenant_id, primary_actor_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_cti_campaigns_tenant_time
    ON cti_campaigns (tenant_id, first_seen_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_cti_campaigns_mitre_gin
    ON cti_campaigns USING GIN (mitre_technique_ids)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_cti_campaigns_sectors_gin
    ON cti_campaigns USING GIN (target_sectors)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_cti_campaigns_regions_gin
    ON cti_campaigns USING GIN (target_regions)
    WHERE deleted_at IS NULL;

ALTER TABLE cti_campaigns ENABLE ROW LEVEL SECURITY;
ALTER TABLE cti_campaigns FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cti_campaigns
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON cti_campaigns
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON cti_campaigns
    FOR UPDATE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON cti_campaigns
    FOR DELETE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ---------------------------------------------------------------------------
-- 3. cti_campaign_events — junction table
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cti_campaign_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    campaign_id UUID NOT NULL REFERENCES cti_campaigns(id) ON DELETE CASCADE,
    event_id    UUID NOT NULL REFERENCES cti_threat_events(id) ON DELETE CASCADE,
    linked_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    linked_by   UUID,
    CONSTRAINT uq_cti_campaign_event UNIQUE (tenant_id, campaign_id, event_id)
);

COMMENT ON TABLE cti_campaign_events IS 'Many-to-many link between campaigns and threat events';

CREATE INDEX idx_cti_campaign_events_campaign ON cti_campaign_events (tenant_id, campaign_id);
CREATE INDEX idx_cti_campaign_events_event    ON cti_campaign_events (tenant_id, event_id);

ALTER TABLE cti_campaign_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE cti_campaign_events FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cti_campaign_events
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON cti_campaign_events
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON cti_campaign_events
    FOR UPDATE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON cti_campaign_events
    FOR DELETE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ---------------------------------------------------------------------------
-- 4. cti_campaign_iocs — dedicated IOC list per campaign
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cti_campaign_iocs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL,
    campaign_id      UUID NOT NULL REFERENCES cti_campaigns(id) ON DELETE CASCADE,
    ioc_type         VARCHAR(50) NOT NULL,
    ioc_value        TEXT NOT NULL,
    confidence_score DECIMAL(3,2) NOT NULL DEFAULT 0.50
                     CHECK (confidence_score >= 0 AND confidence_score <= 1),
    first_seen_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active        BOOLEAN NOT NULL DEFAULT true,
    source_id        UUID REFERENCES cti_data_sources(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE cti_campaign_iocs IS 'Indicators of Compromise tied to specific campaigns';

CREATE INDEX idx_cti_campaign_iocs_campaign
    ON cti_campaign_iocs (tenant_id, campaign_id, ioc_type);

CREATE INDEX idx_cti_campaign_iocs_value
    ON cti_campaign_iocs (tenant_id, ioc_type, ioc_value);

ALTER TABLE cti_campaign_iocs ENABLE ROW LEVEL SECURITY;
ALTER TABLE cti_campaign_iocs FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cti_campaign_iocs
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON cti_campaign_iocs
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON cti_campaign_iocs
    FOR UPDATE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON cti_campaign_iocs
    FOR DELETE USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
