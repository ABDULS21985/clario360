DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'data_quality_rules'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'quality_rules'
    ) THEN
        ALTER TABLE data_quality_rules RENAME TO quality_rules;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'data_quality_results'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'quality_results'
    ) THEN
        ALTER TABLE data_quality_results RENAME TO quality_results;
    END IF;
END $$;

ALTER TABLE pipelines
    ADD COLUMN IF NOT EXISTS last_run_id UUID,
    ADD COLUMN IF NOT EXISTS last_run_status TEXT,
    ADD COLUMN IF NOT EXISTS last_run_error TEXT,
    ADD COLUMN IF NOT EXISTS total_runs INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS successful_runs INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS failed_runs INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_records_processed BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS avg_duration_ms BIGINT,
    ADD COLUMN IF NOT EXISTS tags TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

DROP INDEX IF EXISTS idx_pipelines_tenant_status;
DROP INDEX IF EXISTS idx_pipelines_tenant_type;
DROP INDEX IF EXISTS idx_pipelines_next_run;
DROP INDEX IF EXISTS idx_pipelines_tenant_created;
DROP INDEX IF EXISTS idx_pipelines_config;
DROP INDEX IF EXISTS idx_pipelines_tenant;
DROP INDEX IF EXISTS idx_pipelines_schedule;
DROP INDEX IF EXISTS idx_pipelines_source;
DROP INDEX IF EXISTS idx_pipelines_tenant_name_unique;

ALTER TABLE pipelines
    ALTER COLUMN type TYPE TEXT USING type::text,
    ALTER COLUMN status TYPE TEXT USING (
        CASE status::text
            WHEN 'failed' THEN 'error'
            WHEN 'completed' THEN 'active'
            ELSE status::text
        END
    );

ALTER TABLE pipelines
    DROP CONSTRAINT IF EXISTS pipelines_type_check,
    DROP CONSTRAINT IF EXISTS pipelines_status_check;

ALTER TABLE pipelines
    ADD CONSTRAINT pipelines_type_check CHECK (type::text IN ('etl', 'elt', 'batch', 'streaming')),
    ADD CONSTRAINT pipelines_status_check CHECK (status::text IN ('active', 'paused', 'disabled', 'error'));

CREATE INDEX IF NOT EXISTS idx_pipelines_tenant ON pipelines (tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_pipelines_schedule ON pipelines (next_run_at) WHERE schedule IS NOT NULL AND status::text = 'active' AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_pipelines_source ON pipelines (source_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_pipelines_tenant_name_unique ON pipelines (tenant_id, name) WHERE deleted_at IS NULL;

ALTER TABLE pipeline_runs
    ADD COLUMN IF NOT EXISTS current_phase TEXT,
    ADD COLUMN IF NOT EXISTS records_extracted BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS records_transformed BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS records_loaded BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS records_filtered BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS records_deduplicated BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS bytes_read BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS bytes_written BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS quality_gate_results JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS quality_gates_passed INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS quality_gates_failed INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS quality_gates_warned INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS extract_started_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS extract_completed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS transform_started_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS transform_completed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS load_started_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS load_completed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS duration_ms BIGINT,
    ADD COLUMN IF NOT EXISTS error_phase TEXT,
    ADD COLUMN IF NOT EXISTS error_message TEXT,
    ADD COLUMN IF NOT EXISTS error_details JSONB,
    ADD COLUMN IF NOT EXISTS triggered_by TEXT NOT NULL DEFAULT 'manual',
    ADD COLUMN IF NOT EXISTS triggered_by_user UUID,
    ADD COLUMN IF NOT EXISTS retry_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS incremental_from TEXT,
    ADD COLUMN IF NOT EXISTS incremental_to TEXT;

UPDATE pipeline_runs
SET records_extracted = COALESCE(records_processed, 0),
    records_transformed = COALESCE(records_processed, 0),
    records_loaded = GREATEST(COALESCE(records_processed, 0) - COALESCE(records_failed, 0), 0),
    error_message = COALESCE(error_log, error_message),
    duration_ms = COALESCE(duration_ms, CASE
        WHEN completed_at IS NOT NULL THEN GREATEST((EXTRACT(EPOCH FROM (completed_at - started_at)) * 1000)::BIGINT, 0)
        ELSE NULL
    END),
    error_details = COALESCE(error_details, metrics),
    current_phase = COALESCE(current_phase, 'loading')
WHERE TRUE;

ALTER TABLE pipeline_runs
    ALTER COLUMN status TYPE TEXT USING status::text;

ALTER TABLE pipeline_runs
    DROP CONSTRAINT IF EXISTS pipeline_runs_status_check,
    DROP CONSTRAINT IF EXISTS pipeline_runs_triggered_by_check;

ALTER TABLE pipeline_runs
    ADD CONSTRAINT pipeline_runs_status_check CHECK (status::text IN ('running', 'completed', 'failed', 'cancelled')),
    ADD CONSTRAINT pipeline_runs_triggered_by_check CHECK (triggered_by::text IN ('manual', 'schedule', 'event', 'api', 'retry'));

DROP INDEX IF EXISTS idx_runs_tenant;
DROP INDEX IF EXISTS idx_runs_pipeline;
DROP INDEX IF EXISTS idx_runs_status;
DROP INDEX IF EXISTS idx_runs_started;
DROP INDEX IF EXISTS idx_pipeline_runs_pipeline;
DROP INDEX IF EXISTS idx_pipeline_runs_tenant;
DROP INDEX IF EXISTS idx_pipeline_runs_status;

CREATE INDEX IF NOT EXISTS idx_pipeline_runs_pipeline ON pipeline_runs (pipeline_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_tenant ON pipeline_runs (tenant_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_status ON pipeline_runs (tenant_id, status) WHERE status::text = 'running';

CREATE TABLE IF NOT EXISTS pipeline_run_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    run_id UUID NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    level TEXT NOT NULL CHECK (level IN ('debug', 'info', 'warn', 'error')),
    phase TEXT NOT NULL,
    message TEXT NOT NULL,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_pipeline_logs_run ON pipeline_run_logs (run_id, created_at ASC);

ALTER TABLE quality_rules
    ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS config JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS schedule TEXT,
    ADD COLUMN IF NOT EXISTS last_run_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_status TEXT,
    ADD COLUMN IF NOT EXISTS consecutive_failures INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS tags TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

UPDATE quality_rules
SET name = CASE
        WHEN COALESCE(name, '') <> '' THEN name
        WHEN column_name IS NULL OR column_name = '' THEN CONCAT(rule_type::text, ' rule')
        ELSE CONCAT(column_name, ' ', rule_type::text, ' rule')
    END,
    config = COALESCE(config, rule_config, '{}'::jsonb),
    last_run_at = COALESCE(last_run_at, last_check_at),
    last_status = COALESCE(last_status, last_check_result::text)
WHERE TRUE;

ALTER TABLE quality_rules
    ALTER COLUMN rule_type TYPE TEXT USING rule_type::text,
    ALTER COLUMN severity TYPE TEXT USING severity::text,
    ALTER COLUMN column_name DROP NOT NULL;

ALTER TABLE quality_rules
    DROP COLUMN IF EXISTS rule_config,
    DROP COLUMN IF EXISTS last_check_at,
    DROP COLUMN IF EXISTS last_check_result;

ALTER TABLE quality_rules
    DROP CONSTRAINT IF EXISTS quality_rules_rule_type_check,
    DROP CONSTRAINT IF EXISTS quality_rules_severity_check;

ALTER TABLE quality_rules
    ADD CONSTRAINT quality_rules_rule_type_check CHECK (rule_type::text IN (
        'not_null', 'unique', 'range', 'regex', 'referential',
        'enum', 'freshness', 'row_count', 'custom_sql', 'statistical'
    )),
    ADD CONSTRAINT quality_rules_severity_check CHECK (severity::text IN ('critical', 'high', 'medium', 'low'));

DROP INDEX IF EXISTS idx_dq_rules_tenant;
DROP INDEX IF EXISTS idx_dq_rules_model;
DROP INDEX IF EXISTS idx_dq_rules_enabled;
DROP INDEX IF EXISTS idx_dq_rules_config;
DROP INDEX IF EXISTS idx_quality_rules_model;
DROP INDEX IF EXISTS idx_quality_rules_tenant;
DROP INDEX IF EXISTS idx_quality_rules_schedule;

CREATE INDEX IF NOT EXISTS idx_quality_rules_model ON quality_rules (model_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_quality_rules_tenant ON quality_rules (tenant_id, enabled) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_quality_rules_schedule ON quality_rules (schedule) WHERE schedule IS NOT NULL AND enabled = true AND deleted_at IS NULL;

ALTER TABLE quality_results
    ADD COLUMN IF NOT EXISTS pipeline_run_id UUID REFERENCES pipeline_runs(id),
    ADD COLUMN IF NOT EXISTS records_passed BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS pass_rate DECIMAL(5,2),
    ADD COLUMN IF NOT EXISTS failure_summary TEXT,
    ADD COLUMN IF NOT EXISTS duration_ms BIGINT,
    ADD COLUMN IF NOT EXISTS error_message TEXT;

UPDATE quality_results
SET records_passed = GREATEST(COALESCE(records_checked, 0) - COALESCE(records_failed, 0), 0),
    pass_rate = CASE
        WHEN records_checked > 0 THEN ROUND(((records_checked - records_failed)::numeric / records_checked::numeric) * 100, 2)
        ELSE 100
    END
WHERE TRUE;

ALTER TABLE quality_results
    ALTER COLUMN status TYPE TEXT USING status::text;

ALTER TABLE quality_results
    DROP CONSTRAINT IF EXISTS quality_results_status_check;

ALTER TABLE quality_results
    ADD CONSTRAINT quality_results_status_check CHECK (status::text IN ('passed', 'failed', 'warning', 'error'));

DROP INDEX IF EXISTS idx_dq_results_tenant;
DROP INDEX IF EXISTS idx_dq_results_rule;
DROP INDEX IF EXISTS idx_dq_results_model;
DROP INDEX IF EXISTS idx_dq_results_status;
DROP INDEX IF EXISTS idx_quality_results_rule;
DROP INDEX IF EXISTS idx_quality_results_model;
DROP INDEX IF EXISTS idx_quality_results_status;

CREATE INDEX IF NOT EXISTS idx_quality_results_rule ON quality_results (rule_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_quality_results_model ON quality_results (model_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_quality_results_status ON quality_results (tenant_id, status, checked_at DESC);

ALTER TABLE contradictions
    ADD COLUMN IF NOT EXISTS scan_id UUID,
    ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS entity_key_column TEXT,
    ADD COLUMN IF NOT EXISTS entity_key_value TEXT,
    ADD COLUMN IF NOT EXISTS affected_records INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS sample_records JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS authoritative_source TEXT,
    ADD COLUMN IF NOT EXISTS resolution_notes TEXT,
    ADD COLUMN IF NOT EXISTS resolution_action TEXT,
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}';

UPDATE contradictions
SET title = CASE
        WHEN COALESCE(title, '') <> '' THEN title
        ELSE INITCAP(type::text) || ' contradiction'
    END
WHERE TRUE;

ALTER TABLE contradictions
    ALTER COLUMN type TYPE TEXT USING type::text,
    ALTER COLUMN severity TYPE TEXT USING severity::text,
    ALTER COLUMN status TYPE TEXT USING status::text;

ALTER TABLE contradictions
    DROP CONSTRAINT IF EXISTS contradictions_type_check,
    DROP CONSTRAINT IF EXISTS contradictions_severity_check,
    DROP CONSTRAINT IF EXISTS contradictions_status_check,
    DROP CONSTRAINT IF EXISTS contradictions_confidence_score_check,
    DROP CONSTRAINT IF EXISTS contradictions_resolution_action_check;

ALTER TABLE contradictions
    ADD CONSTRAINT contradictions_type_check CHECK (type::text IN ('logical', 'semantic', 'temporal', 'analytical')),
    ADD CONSTRAINT contradictions_severity_check CHECK (severity::text IN ('critical', 'high', 'medium', 'low')),
    ADD CONSTRAINT contradictions_status_check CHECK (status::text IN ('detected', 'investigating', 'resolved', 'accepted', 'false_positive')),
    ADD CONSTRAINT contradictions_confidence_score_check CHECK (confidence_score BETWEEN 0.00 AND 1.00),
    ADD CONSTRAINT contradictions_resolution_action_check CHECK (
        resolution_action IS NULL OR resolution_action::text IN (
                    'source_a_corrected', 'source_b_corrected', 'both_corrected',
                    'accepted_as_is', 'data_reconciled', 'false_positive'
                )
            );

DROP INDEX IF EXISTS idx_contradictions_tenant_status;
DROP INDEX IF EXISTS idx_contradictions_tenant_type;
DROP INDEX IF EXISTS idx_contradictions_tenant_created;
DROP INDEX IF EXISTS idx_contradictions_source_a;
DROP INDEX IF EXISTS idx_contradictions_source_b;
DROP INDEX IF EXISTS idx_contradictions_tenant;
DROP INDEX IF EXISTS idx_contradictions_type;
DROP INDEX IF EXISTS idx_contradictions_scan;

CREATE INDEX IF NOT EXISTS idx_contradictions_tenant ON contradictions (tenant_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_contradictions_type ON contradictions (tenant_id, type, severity);
CREATE INDEX IF NOT EXISTS idx_contradictions_scan ON contradictions (scan_id) WHERE scan_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS contradiction_scans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed', 'cancelled')),
    models_scanned INT NOT NULL DEFAULT 0,
    model_pairs_compared INT NOT NULL DEFAULT 0,
    contradictions_found INT NOT NULL DEFAULT 0,
    by_type JSONB NOT NULL DEFAULT '{}',
    by_severity JSONB NOT NULL DEFAULT '{}',
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    duration_ms BIGINT,
    triggered_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_contradiction_scans_tenant ON contradiction_scans (tenant_id, created_at DESC);
