-- =============================================================================
-- Clario 360 — Data Suite Database Schema Rollback
-- Database: data_db
-- =============================================================================

DROP TABLE IF EXISTS data_catalogs;
DROP TABLE IF EXISTS dark_data_assets;
DROP TABLE IF EXISTS data_lineage;
DROP TABLE IF EXISTS pipeline_runs;
DROP TABLE IF EXISTS pipelines;
DROP TABLE IF EXISTS contradictions;
DROP TABLE IF EXISTS data_quality_results;
DROP TABLE IF EXISTS data_quality_rules;
DROP TABLE IF EXISTS data_models;
DROP TABLE IF EXISTS data_sources;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TYPE IF EXISTS governance_status;
DROP TYPE IF EXISTS pipeline_run_status;
DROP TYPE IF EXISTS pipeline_status;
DROP TYPE IF EXISTS pipeline_type;
DROP TYPE IF EXISTS contradiction_status;
DROP TYPE IF EXISTS contradiction_type;
DROP TYPE IF EXISTS quality_result_status;
DROP TYPE IF EXISTS quality_severity;
DROP TYPE IF EXISTS quality_rule_type;
DROP TYPE IF EXISTS data_model_status;
DROP TYPE IF EXISTS data_source_status;
DROP TYPE IF EXISTS data_source_type;
