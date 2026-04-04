-- =============================================================================
-- Clario 360 — Rollback CTI Aggregation Tables (migration 000031)
-- =============================================================================

DROP TABLE IF EXISTS cti_executive_snapshot CASCADE;
DROP TABLE IF EXISTS cti_sector_threat_summary CASCADE;
DROP TABLE IF EXISTS cti_geo_threat_summary CASCADE;
