-- =============================================================================
-- Clario 360 — Cyber Suite Database Schema Rollback
-- Database: cyber_db
-- =============================================================================

DROP TABLE IF EXISTS dspm_data_assets;
DROP TABLE IF EXISTS ctem_assessments;
DROP TABLE IF EXISTS remediation_actions;
DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS detection_rules;
DROP TABLE IF EXISTS threat_indicators;
DROP TABLE IF EXISTS threats;
DROP TABLE IF EXISTS vulnerabilities;
DROP TABLE IF EXISTS asset_relationships;
DROP TABLE IF EXISTS assets;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TYPE IF EXISTS data_classification;
DROP TYPE IF EXISTS ctem_status;
DROP TYPE IF EXISTS execution_mode;
DROP TYPE IF EXISTS remediation_status;
DROP TYPE IF EXISTS remediation_type;
DROP TYPE IF EXISTS alert_status;
DROP TYPE IF EXISTS detection_rule_type;
DROP TYPE IF EXISTS indicator_type;
DROP TYPE IF EXISTS threat_status;
DROP TYPE IF EXISTS vulnerability_status;
DROP TYPE IF EXISTS severity_level;
DROP TYPE IF EXISTS asset_status;
DROP TYPE IF EXISTS asset_criticality;
DROP TYPE IF EXISTS asset_type;
