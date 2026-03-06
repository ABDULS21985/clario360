-- =============================================================================
-- Clario 360 — Visus Suite Database Schema Rollback
-- Database: visus_db
-- =============================================================================

DROP TABLE IF EXISTS report_snapshots;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS executive_alerts;

-- Drop partitioned table (cascades to all partitions)
DROP TABLE IF EXISTS kpi_snapshots CASCADE;

DROP TABLE IF EXISTS kpi_definitions;
DROP TABLE IF EXISTS dashboard_widgets;
DROP TABLE IF EXISTS dashboards;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TYPE IF EXISTS report_type;
DROP TYPE IF EXISTS alert_status;
DROP TYPE IF EXISTS alert_category;
DROP TYPE IF EXISTS alert_severity;
DROP TYPE IF EXISTS kpi_calculation_type;
DROP TYPE IF EXISTS suite_name;
DROP TYPE IF EXISTS widget_type;
