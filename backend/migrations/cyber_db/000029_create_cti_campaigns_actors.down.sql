-- =============================================================================
-- Clario 360 — Rollback CTI Campaigns & Actors (migration 000029)
-- =============================================================================

DROP TABLE IF EXISTS cti_campaign_iocs CASCADE;
DROP TABLE IF EXISTS cti_campaign_events CASCADE;
DROP TABLE IF EXISTS cti_campaigns CASCADE;
DROP TABLE IF EXISTS cti_threat_actors CASCADE;
