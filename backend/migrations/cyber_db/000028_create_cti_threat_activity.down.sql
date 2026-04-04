-- =============================================================================
-- Clario 360 — Rollback CTI Threat Activity (migration 000028)
-- =============================================================================

DROP TABLE IF EXISTS cti_threat_event_tags CASCADE;
DROP TABLE IF EXISTS cti_threat_events CASCADE;
