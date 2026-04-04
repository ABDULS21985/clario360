-- =============================================================================
-- Clario 360 — Rollback CTI Audit Column Backfill (migration 000032)
-- =============================================================================

ALTER TABLE IF EXISTS cti_executive_snapshot
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS created_at;

ALTER TABLE IF EXISTS cti_sector_threat_summary
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS created_at;

ALTER TABLE IF EXISTS cti_geo_threat_summary
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS created_at;

ALTER TABLE IF EXISTS cti_campaign_iocs
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by;

ALTER TABLE IF EXISTS cti_campaign_events
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS created_at;

ALTER TABLE IF EXISTS cti_threat_event_tags
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS updated_at;

ALTER TABLE IF EXISTS cti_data_sources
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by;

ALTER TABLE IF EXISTS cti_industry_sectors
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS updated_at;

ALTER TABLE IF EXISTS cti_geographic_regions
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS updated_at;

ALTER TABLE IF EXISTS cti_threat_categories
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS updated_at;

ALTER TABLE IF EXISTS cti_threat_severity_levels
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS updated_at;
