-- =============================================================================
-- Clario 360 — Rollback CTI Brand Abuse (migration 000030)
-- =============================================================================

DROP TABLE IF EXISTS cti_brand_abuse_incidents CASCADE;
DROP TABLE IF EXISTS cti_monitored_brands CASCADE;
