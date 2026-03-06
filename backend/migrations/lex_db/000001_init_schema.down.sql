-- =============================================================================
-- Clario 360 — Lex Suite Database Schema Rollback
-- Database: lex_db
-- =============================================================================

DROP TABLE IF EXISTS legal_workflow_instances;
DROP TABLE IF EXISTS legal_workflows;
DROP TABLE IF EXISTS compliance_alerts;
DROP TABLE IF EXISTS compliance_rules;
DROP TABLE IF EXISTS legal_documents;
DROP TABLE IF EXISTS contract_clauses;
DROP TABLE IF EXISTS contracts;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TYPE IF EXISTS legal_instance_status;
DROP TYPE IF EXISTS legal_workflow_status;
DROP TYPE IF EXISTS legal_workflow_type;
DROP TYPE IF EXISTS compliance_alert_status;
DROP TYPE IF EXISTS compliance_severity;
DROP TYPE IF EXISTS legal_doc_status;
DROP TYPE IF EXISTS clause_status;
DROP TYPE IF EXISTS clause_risk_level;
DROP TYPE IF EXISTS contract_status;
DROP TYPE IF EXISTS contract_type;
