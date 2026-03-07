DROP INDEX IF EXISTS idx_contradiction_scans_tenant;
DROP TABLE IF EXISTS contradiction_scans;

DROP INDEX IF EXISTS idx_contradictions_scan;
DROP INDEX IF EXISTS idx_contradictions_type;
DROP INDEX IF EXISTS idx_contradictions_tenant;

ALTER TABLE contradictions
    DROP CONSTRAINT IF EXISTS contradictions_type_check,
    DROP CONSTRAINT IF EXISTS contradictions_severity_check,
    DROP CONSTRAINT IF EXISTS contradictions_status_check,
    DROP CONSTRAINT IF EXISTS contradictions_confidence_score_check,
    DROP CONSTRAINT IF EXISTS contradictions_resolution_action_check,
    DROP COLUMN IF EXISTS scan_id,
    DROP COLUMN IF EXISTS title,
    DROP COLUMN IF EXISTS entity_key_column,
    DROP COLUMN IF EXISTS entity_key_value,
    DROP COLUMN IF EXISTS affected_records,
    DROP COLUMN IF EXISTS sample_records,
    DROP COLUMN IF EXISTS authoritative_source,
    DROP COLUMN IF EXISTS resolution_notes,
    DROP COLUMN IF EXISTS resolution_action,
    DROP COLUMN IF EXISTS metadata;

CREATE INDEX IF NOT EXISTS idx_contradictions_tenant_status ON contradictions (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_contradictions_tenant_type ON contradictions (tenant_id, type);
CREATE INDEX IF NOT EXISTS idx_contradictions_tenant_created ON contradictions (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_contradictions_source_a ON contradictions USING GIN (source_a);
CREATE INDEX IF NOT EXISTS idx_contradictions_source_b ON contradictions USING GIN (source_b);

DROP INDEX IF EXISTS idx_quality_results_rule;
DROP INDEX IF EXISTS idx_quality_results_model;
DROP INDEX IF EXISTS idx_quality_results_status;

ALTER TABLE quality_results
    DROP CONSTRAINT IF EXISTS quality_results_status_check,
    DROP COLUMN IF EXISTS pipeline_run_id,
    DROP COLUMN IF EXISTS records_passed,
    DROP COLUMN IF EXISTS pass_rate,
    DROP COLUMN IF EXISTS failure_summary,
    DROP COLUMN IF EXISTS duration_ms,
    DROP COLUMN IF EXISTS error_message;

CREATE INDEX IF NOT EXISTS idx_dq_results_tenant ON quality_results (tenant_id);
CREATE INDEX IF NOT EXISTS idx_dq_results_rule ON quality_results (rule_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_dq_results_model ON quality_results (model_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_dq_results_status ON quality_results (tenant_id, status);

DROP INDEX IF EXISTS idx_quality_rules_model;
DROP INDEX IF EXISTS idx_quality_rules_tenant;
DROP INDEX IF EXISTS idx_quality_rules_schedule;

ALTER TABLE quality_rules
    DROP CONSTRAINT IF EXISTS quality_rules_rule_type_check,
    DROP CONSTRAINT IF EXISTS quality_rules_severity_check,
    DROP COLUMN IF EXISTS name,
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS config,
    DROP COLUMN IF EXISTS schedule,
    DROP COLUMN IF EXISTS last_run_at,
    DROP COLUMN IF EXISTS last_status,
    DROP COLUMN IF EXISTS consecutive_failures,
    DROP COLUMN IF EXISTS tags,
    DROP COLUMN IF EXISTS deleted_at;

CREATE INDEX IF NOT EXISTS idx_dq_rules_tenant ON quality_rules (tenant_id);
CREATE INDEX IF NOT EXISTS idx_dq_rules_model ON quality_rules (model_id);
CREATE INDEX IF NOT EXISTS idx_dq_rules_enabled ON quality_rules (model_id, enabled) WHERE enabled = true;

DROP INDEX IF EXISTS idx_pipeline_logs_run;
DROP TABLE IF EXISTS pipeline_run_logs;

DROP INDEX IF EXISTS idx_pipeline_runs_pipeline;
DROP INDEX IF EXISTS idx_pipeline_runs_tenant;
DROP INDEX IF EXISTS idx_pipeline_runs_status;

ALTER TABLE pipeline_runs
    DROP CONSTRAINT IF EXISTS pipeline_runs_status_check,
    DROP CONSTRAINT IF EXISTS pipeline_runs_triggered_by_check,
    DROP COLUMN IF EXISTS current_phase,
    DROP COLUMN IF EXISTS records_extracted,
    DROP COLUMN IF EXISTS records_transformed,
    DROP COLUMN IF EXISTS records_loaded,
    DROP COLUMN IF EXISTS records_filtered,
    DROP COLUMN IF EXISTS records_deduplicated,
    DROP COLUMN IF EXISTS bytes_read,
    DROP COLUMN IF EXISTS bytes_written,
    DROP COLUMN IF EXISTS quality_gate_results,
    DROP COLUMN IF EXISTS quality_gates_passed,
    DROP COLUMN IF EXISTS quality_gates_failed,
    DROP COLUMN IF EXISTS quality_gates_warned,
    DROP COLUMN IF EXISTS extract_started_at,
    DROP COLUMN IF EXISTS extract_completed_at,
    DROP COLUMN IF EXISTS transform_started_at,
    DROP COLUMN IF EXISTS transform_completed_at,
    DROP COLUMN IF EXISTS load_started_at,
    DROP COLUMN IF EXISTS load_completed_at,
    DROP COLUMN IF EXISTS duration_ms,
    DROP COLUMN IF EXISTS error_phase,
    DROP COLUMN IF EXISTS error_message,
    DROP COLUMN IF EXISTS error_details,
    DROP COLUMN IF EXISTS triggered_by,
    DROP COLUMN IF EXISTS triggered_by_user,
    DROP COLUMN IF EXISTS retry_count,
    DROP COLUMN IF EXISTS incremental_from,
    DROP COLUMN IF EXISTS incremental_to;

CREATE INDEX IF NOT EXISTS idx_runs_tenant ON pipeline_runs (tenant_id);
CREATE INDEX IF NOT EXISTS idx_runs_pipeline ON pipeline_runs (pipeline_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_runs_status ON pipeline_runs (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_runs_started ON pipeline_runs (started_at DESC);

DROP INDEX IF EXISTS idx_pipelines_tenant;
DROP INDEX IF EXISTS idx_pipelines_schedule;
DROP INDEX IF EXISTS idx_pipelines_source;
DROP INDEX IF EXISTS idx_pipelines_tenant_name_unique;

ALTER TABLE pipelines
    DROP CONSTRAINT IF EXISTS pipelines_type_check,
    DROP CONSTRAINT IF EXISTS pipelines_status_check,
    DROP COLUMN IF EXISTS last_run_id,
    DROP COLUMN IF EXISTS last_run_status,
    DROP COLUMN IF EXISTS last_run_error,
    DROP COLUMN IF EXISTS total_runs,
    DROP COLUMN IF EXISTS successful_runs,
    DROP COLUMN IF EXISTS failed_runs,
    DROP COLUMN IF EXISTS total_records_processed,
    DROP COLUMN IF EXISTS avg_duration_ms,
    DROP COLUMN IF EXISTS tags,
    DROP COLUMN IF EXISTS deleted_at;

CREATE INDEX IF NOT EXISTS idx_pipelines_tenant_status ON pipelines (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_pipelines_tenant_type ON pipelines (tenant_id, type);
CREATE INDEX IF NOT EXISTS idx_pipelines_next_run ON pipelines (next_run_at) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_pipelines_tenant_created ON pipelines (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pipelines_config ON pipelines USING GIN (config);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'quality_results'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'data_quality_results'
    ) THEN
        ALTER TABLE quality_results RENAME TO data_quality_results;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'quality_rules'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'data_quality_rules'
    ) THEN
        ALTER TABLE quality_rules RENAME TO data_quality_rules;
    END IF;
END $$;
