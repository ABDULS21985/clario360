-- Removes Row-Level Security from all tenant-scoped tables in data_db.

-- TABLE: contradiction_scans
ALTER TABLE contradiction_scans DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON contradiction_scans;
DROP POLICY IF EXISTS tenant_insert ON contradiction_scans;
DROP POLICY IF EXISTS tenant_update ON contradiction_scans;
DROP POLICY IF EXISTS tenant_delete ON contradiction_scans;

-- TABLE: analytics_audit_log
ALTER TABLE analytics_audit_log DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON analytics_audit_log;
DROP POLICY IF EXISTS tenant_insert ON analytics_audit_log;
DROP POLICY IF EXISTS tenant_update ON analytics_audit_log;
DROP POLICY IF EXISTS tenant_delete ON analytics_audit_log;

-- TABLE: saved_queries
ALTER TABLE saved_queries DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON saved_queries;
DROP POLICY IF EXISTS tenant_insert ON saved_queries;
DROP POLICY IF EXISTS tenant_update ON saved_queries;
DROP POLICY IF EXISTS tenant_delete ON saved_queries;

-- TABLE: data_catalogs
ALTER TABLE data_catalogs DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON data_catalogs;
DROP POLICY IF EXISTS tenant_insert ON data_catalogs;
DROP POLICY IF EXISTS tenant_update ON data_catalogs;
DROP POLICY IF EXISTS tenant_delete ON data_catalogs;

-- TABLE: dark_data_scans
ALTER TABLE dark_data_scans DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON dark_data_scans;
DROP POLICY IF EXISTS tenant_insert ON dark_data_scans;
DROP POLICY IF EXISTS tenant_update ON dark_data_scans;
DROP POLICY IF EXISTS tenant_delete ON dark_data_scans;

-- TABLE: dark_data_assets
ALTER TABLE dark_data_assets DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON dark_data_assets;
DROP POLICY IF EXISTS tenant_insert ON dark_data_assets;
DROP POLICY IF EXISTS tenant_update ON dark_data_assets;
DROP POLICY IF EXISTS tenant_delete ON dark_data_assets;

-- TABLE: data_lineage_edges
ALTER TABLE data_lineage_edges DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON data_lineage_edges;
DROP POLICY IF EXISTS tenant_insert ON data_lineage_edges;
DROP POLICY IF EXISTS tenant_update ON data_lineage_edges;
DROP POLICY IF EXISTS tenant_delete ON data_lineage_edges;

-- TABLE: pipeline_run_logs
ALTER TABLE pipeline_run_logs DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON pipeline_run_logs;
DROP POLICY IF EXISTS tenant_insert ON pipeline_run_logs;
DROP POLICY IF EXISTS tenant_update ON pipeline_run_logs;
DROP POLICY IF EXISTS tenant_delete ON pipeline_run_logs;

-- TABLE: pipeline_runs
ALTER TABLE pipeline_runs DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON pipeline_runs;
DROP POLICY IF EXISTS tenant_insert ON pipeline_runs;
DROP POLICY IF EXISTS tenant_update ON pipeline_runs;
DROP POLICY IF EXISTS tenant_delete ON pipeline_runs;

-- TABLE: pipelines
ALTER TABLE pipelines DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON pipelines;
DROP POLICY IF EXISTS tenant_insert ON pipelines;
DROP POLICY IF EXISTS tenant_update ON pipelines;
DROP POLICY IF EXISTS tenant_delete ON pipelines;

-- TABLE: contradictions
ALTER TABLE contradictions DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON contradictions;
DROP POLICY IF EXISTS tenant_insert ON contradictions;
DROP POLICY IF EXISTS tenant_update ON contradictions;
DROP POLICY IF EXISTS tenant_delete ON contradictions;

-- TABLE: quality_results
ALTER TABLE quality_results DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON quality_results;
DROP POLICY IF EXISTS tenant_insert ON quality_results;
DROP POLICY IF EXISTS tenant_update ON quality_results;
DROP POLICY IF EXISTS tenant_delete ON quality_results;

-- TABLE: quality_rules
ALTER TABLE quality_rules DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON quality_rules;
DROP POLICY IF EXISTS tenant_insert ON quality_rules;
DROP POLICY IF EXISTS tenant_update ON quality_rules;
DROP POLICY IF EXISTS tenant_delete ON quality_rules;

-- TABLE: data_models
ALTER TABLE data_models DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON data_models;
DROP POLICY IF EXISTS tenant_insert ON data_models;
DROP POLICY IF EXISTS tenant_update ON data_models;
DROP POLICY IF EXISTS tenant_delete ON data_models;

-- TABLE: data_sources
ALTER TABLE data_sources DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON data_sources;
DROP POLICY IF EXISTS tenant_insert ON data_sources;
DROP POLICY IF EXISTS tenant_update ON data_sources;
DROP POLICY IF EXISTS tenant_delete ON data_sources;
