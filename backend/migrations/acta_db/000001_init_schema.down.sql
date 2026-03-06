-- =============================================================================
-- Clario 360 — Acta Suite Database Schema Rollback
-- Database: acta_db
-- =============================================================================

DROP TABLE IF EXISTS compliance_checks;
DROP TABLE IF EXISTS workflow_instances;
DROP TABLE IF EXISTS governance_workflows;
DROP TABLE IF EXISTS action_items;
DROP TABLE IF EXISTS meeting_minutes;
DROP TABLE IF EXISTS agenda_items;
DROP TABLE IF EXISTS meetings;
DROP TABLE IF EXISTS committees;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TYPE IF EXISTS compliance_check_status;
DROP TYPE IF EXISTS workflow_instance_status;
DROP TYPE IF EXISTS workflow_status;
DROP TYPE IF EXISTS action_item_status;
DROP TYPE IF EXISTS minutes_status;
DROP TYPE IF EXISTS agenda_item_status;
DROP TYPE IF EXISTS meeting_status;
DROP TYPE IF EXISTS committee_status;
