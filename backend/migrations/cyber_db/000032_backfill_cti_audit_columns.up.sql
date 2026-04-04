-- =============================================================================
-- Clario 360 — Backfill CTI Audit Columns (migration 000032)
-- Adds missing audit columns required across CTI tables introduced in 000027-000031.
-- =============================================================================

-- ---------------------------------------------------------------------------
-- Reference tables
-- ---------------------------------------------------------------------------

ALTER TABLE cti_threat_severity_levels
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS created_by UUID,
    ADD COLUMN IF NOT EXISTS updated_by UUID;
UPDATE cti_threat_severity_levels
SET updated_at = COALESCE(updated_at, created_at, NOW())
WHERE updated_at IS NULL;
ALTER TABLE cti_threat_severity_levels
    ALTER COLUMN updated_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at SET NOT NULL;

ALTER TABLE cti_threat_categories
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS created_by UUID,
    ADD COLUMN IF NOT EXISTS updated_by UUID;
UPDATE cti_threat_categories
SET updated_at = COALESCE(updated_at, created_at, NOW())
WHERE updated_at IS NULL;
ALTER TABLE cti_threat_categories
    ALTER COLUMN updated_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at SET NOT NULL;

ALTER TABLE cti_geographic_regions
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS created_by UUID,
    ADD COLUMN IF NOT EXISTS updated_by UUID;
UPDATE cti_geographic_regions
SET updated_at = COALESCE(updated_at, created_at, NOW())
WHERE updated_at IS NULL;
ALTER TABLE cti_geographic_regions
    ALTER COLUMN updated_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at SET NOT NULL;

ALTER TABLE cti_industry_sectors
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS created_by UUID,
    ADD COLUMN IF NOT EXISTS updated_by UUID;
UPDATE cti_industry_sectors
SET updated_at = COALESCE(updated_at, created_at, NOW())
WHERE updated_at IS NULL;
ALTER TABLE cti_industry_sectors
    ALTER COLUMN updated_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at SET NOT NULL;

ALTER TABLE cti_data_sources
    ADD COLUMN IF NOT EXISTS created_by UUID,
    ADD COLUMN IF NOT EXISTS updated_by UUID;

-- ---------------------------------------------------------------------------
-- Threat activity / campaign support tables
-- ---------------------------------------------------------------------------

ALTER TABLE cti_threat_event_tags
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS created_by UUID,
    ADD COLUMN IF NOT EXISTS updated_by UUID;
UPDATE cti_threat_event_tags
SET updated_at = COALESCE(updated_at, created_at, NOW())
WHERE updated_at IS NULL;
ALTER TABLE cti_threat_event_tags
    ALTER COLUMN updated_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at SET NOT NULL;

ALTER TABLE cti_campaign_events
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS created_by UUID,
    ADD COLUMN IF NOT EXISTS updated_by UUID;
UPDATE cti_campaign_events
SET created_at = COALESCE(created_at, linked_at, NOW()),
    updated_at = COALESCE(updated_at, linked_at, NOW()),
    created_by = COALESCE(created_by, linked_by),
    updated_by = COALESCE(updated_by, linked_by)
WHERE created_at IS NULL
   OR updated_at IS NULL
   OR created_by IS NULL
   OR updated_by IS NULL;
ALTER TABLE cti_campaign_events
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN updated_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at SET NOT NULL;

ALTER TABLE cti_campaign_iocs
    ADD COLUMN IF NOT EXISTS created_by UUID,
    ADD COLUMN IF NOT EXISTS updated_by UUID;

-- ---------------------------------------------------------------------------
-- Aggregation / executive dashboard tables
-- ---------------------------------------------------------------------------

ALTER TABLE cti_geo_threat_summary
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS created_by UUID,
    ADD COLUMN IF NOT EXISTS updated_by UUID;
UPDATE cti_geo_threat_summary
SET created_at = COALESCE(created_at, computed_at, NOW()),
    updated_at = COALESCE(updated_at, computed_at, NOW())
WHERE created_at IS NULL
   OR updated_at IS NULL;
ALTER TABLE cti_geo_threat_summary
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN updated_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at SET NOT NULL;

ALTER TABLE cti_sector_threat_summary
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS created_by UUID,
    ADD COLUMN IF NOT EXISTS updated_by UUID;
UPDATE cti_sector_threat_summary
SET created_at = COALESCE(created_at, computed_at, NOW()),
    updated_at = COALESCE(updated_at, computed_at, NOW())
WHERE created_at IS NULL
   OR updated_at IS NULL;
ALTER TABLE cti_sector_threat_summary
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN updated_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at SET NOT NULL;

ALTER TABLE cti_executive_snapshot
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS created_by UUID,
    ADD COLUMN IF NOT EXISTS updated_by UUID;
UPDATE cti_executive_snapshot
SET created_at = COALESCE(created_at, computed_at, NOW()),
    updated_at = COALESCE(updated_at, computed_at, NOW())
WHERE created_at IS NULL
   OR updated_at IS NULL;
ALTER TABLE cti_executive_snapshot
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN updated_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at SET NOT NULL;
