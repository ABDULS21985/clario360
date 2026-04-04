WITH seeded_sources AS (
    SELECT
        gs,
        uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'data-source-' || gs) AS source_id,
        CASE (gs - 1) % 7
            WHEN 0 THEN 'postgresql'
            WHEN 1 THEN 'mysql'
            WHEN 2 THEN 'mssql'
            WHEN 3 THEN 'api'
            WHEN 4 THEN 'csv'
            WHEN 5 THEN 's3'
            ELSE 'stream'
        END AS source_type
    FROM generate_series(1, {{ .Scale.DataSourceCount }}) gs
)
INSERT INTO data_sources (
    id, tenant_id, name, description, type, connection_config, encryption_key_id, status,
    schema_metadata, schema_discovered_at, last_synced_at, last_sync_status, last_sync_error,
    last_sync_duration_ms, next_sync_at, sync_frequency, table_count, total_row_count,
    total_size_bytes, tags, metadata, created_by, created_at, updated_at
)
SELECT
    source_id,
    '{{ .MainTenantID }}'::uuid,
    format('Seeded Data Source %s', lpad(gs::text, 2, '0')),
    format('Seeded source %s for lineage, quality, contradiction, and analytics demonstrations.', gs),
    source_type,
    convert_to(jsonb_build_object('host', format('seeded-source-%s.local', gs), 'database', format('db_%s', gs))::text, 'UTF8'),
    'demo-data-key',
    CASE WHEN gs % 9 = 0 THEN 'syncing' WHEN gs % 13 = 0 THEN 'error' ELSE 'active' END,
    jsonb_build_object(
        'schemas', jsonb_build_array('public', 'analytics'),
        'sample_tables', jsonb_build_array(format('table_%s_a', gs), format('table_%s_b', gs))
    ),
    now() - interval '10 days',
    now() - make_interval(hours => gs),
    CASE WHEN gs % 13 = 0 THEN 'failed' WHEN gs % 11 = 0 THEN 'partial' ELSE 'success' END,
    CASE WHEN gs % 13 = 0 THEN 'Seeded connector timeout' ELSE NULL END,
    1500 + (gs * 37),
    now() + make_interval(hours => 1 + (gs % 24)),
    CASE WHEN gs % 2 = 0 THEN '0 * * * *' ELSE '0 */6 * * *' END,
    8 + (gs % 20),
    250000 + (gs * 7000),
    134217728 + (gs * 1048576),
    ARRAY['seeded','governance', CASE WHEN gs % 2 = 0 THEN 'gold' ELSE 'silver' END],
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs, 'module', 'data'),
    '{{ .DataStewardUserID }}'::uuid,
    now() - make_interval(days => (30 - gs)),
    now() - make_interval(hours => gs)
FROM seeded_sources
ON CONFLICT (id) DO UPDATE SET
    description = EXCLUDED.description,
    type = EXCLUDED.type,
    connection_config = EXCLUDED.connection_config,
    encryption_key_id = EXCLUDED.encryption_key_id,
    status = EXCLUDED.status,
    schema_metadata = EXCLUDED.schema_metadata,
    schema_discovered_at = EXCLUDED.schema_discovered_at,
    last_synced_at = EXCLUDED.last_synced_at,
    last_sync_status = EXCLUDED.last_sync_status,
    last_sync_error = EXCLUDED.last_sync_error,
    last_sync_duration_ms = EXCLUDED.last_sync_duration_ms,
    next_sync_at = EXCLUDED.next_sync_at,
    sync_frequency = EXCLUDED.sync_frequency,
    table_count = EXCLUDED.table_count,
    total_row_count = EXCLUDED.total_row_count,
    total_size_bytes = EXCLUDED.total_size_bytes,
    tags = EXCLUDED.tags,
    metadata = EXCLUDED.metadata,
    created_by = EXCLUDED.created_by,
    updated_at = EXCLUDED.updated_at,
    deleted_at = NULL;

WITH seeded_sources AS (
    SELECT gs, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'data-source-' || gs) AS source_id
    FROM generate_series(1, {{ .Scale.DataSourceCount }}) gs
)
INSERT INTO sync_history (
    id, tenant_id, source_id, status, sync_type, tables_synced, rows_read, rows_written,
    bytes_transferred, errors, error_count, started_at, completed_at, duration_ms,
    triggered_by, triggered_by_user, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'sync-history-' || gs),
    '{{ .MainTenantID }}'::uuid,
    source_id,
    CASE WHEN gs % 11 = 0 THEN 'partial' WHEN gs % 13 = 0 THEN 'failed' ELSE 'success' END,
    CASE WHEN gs % 3 = 0 THEN 'incremental' ELSE 'full' END,
    8 + (gs % 20),
    180000 + (gs * 2500),
    176000 + (gs * 2400),
    10485760 + (gs * 400000),
    CASE WHEN gs % 13 = 0 THEN '["seeded connector timeout"]'::jsonb ELSE '[]'::jsonb END,
    CASE WHEN gs % 13 = 0 THEN 1 ELSE 0 END,
    now() - make_interval(days => 7, hours => gs),
    now() - make_interval(days => 7, hours => gs) + make_interval(mins => 8 + (gs % 20)),
    480000 + (gs * 1200),
    CASE WHEN gs % 2 = 0 THEN 'schedule' ELSE 'manual' END,
    '{{ .DataStewardUserID }}'::uuid,
    now() - make_interval(days => 7, hours => gs)
FROM seeded_sources
ON CONFLICT (id) DO UPDATE SET
    status = EXCLUDED.status,
    rows_read = EXCLUDED.rows_read,
    rows_written = EXCLUDED.rows_written,
    bytes_transferred = EXCLUDED.bytes_transferred,
    errors = EXCLUDED.errors,
    error_count = EXCLUDED.error_count,
    completed_at = EXCLUDED.completed_at,
    duration_ms = EXCLUDED.duration_ms;

WITH seeded_models AS (
    SELECT
        gs,
        uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'data-model-' || gs) AS model_id,
        uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'data-source-' || (((gs - 1) % {{ .Scale.DataSourceCount }}) + 1)) AS source_id
    FROM generate_series(1, {{ .Scale.DataModelCount }}) gs
)
INSERT INTO data_models (
    id, tenant_id, name, display_name, description, version, schema_definition, source_id,
    source_table, status, lineage, quality_rules, data_classification, contains_pii, pii_columns,
    field_count, previous_version_id, tags, metadata, created_by, created_at, updated_at
)
SELECT
    model_id,
    '{{ .MainTenantID }}'::uuid,
    format('seeded_model_%s', lpad(gs::text, 2, '0')),
    format('Seeded Model %s', lpad(gs::text, 2, '0')),
    format('Seeded enterprise data model %s for cross-suite demonstrations.', gs),
    1,
    jsonb_build_object(
        'fields', jsonb_build_array(
            jsonb_build_object('name', 'entity_id', 'type', 'uuid'),
            jsonb_build_object('name', 'owner_name', 'type', 'text'),
            jsonb_build_object('name', 'score', 'type', 'numeric')
        )
    ),
    source_id,
    format('seeded_table_%s', gs),
    CASE WHEN gs % 17 = 0 THEN 'deprecated' ELSE 'active' END,
    jsonb_build_object('upstream', jsonb_build_array(source_id::text)),
    jsonb_build_array(jsonb_build_object('name', 'freshness', 'severity', 'high')),
    CASE (gs - 1) % 4
        WHEN 0 THEN 'public'
        WHEN 1 THEN 'internal'
        WHEN 2 THEN 'confidential'
        ELSE 'restricted'
    END,
    gs % 3 = 0,
    CASE WHEN gs % 3 = 0 THEN ARRAY['email', 'phone_number'] ELSE ARRAY[]::text[] END,
    12 + (gs % 8),
    NULL,
    ARRAY['seeded','catalog', CASE WHEN gs % 2 = 0 THEN 'gold' ELSE 'standard' END],
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'domain', CASE WHEN gs % 2 = 0 THEN 'customer' ELSE 'operations' END),
    '{{ .DataStewardUserID }}'::uuid,
    now() - make_interval(days => 30 - gs),
    now() - make_interval(hours => gs)
FROM seeded_models
ON CONFLICT (id) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    schema_definition = EXCLUDED.schema_definition,
    source_id = EXCLUDED.source_id,
    source_table = EXCLUDED.source_table,
    status = EXCLUDED.status,
    lineage = EXCLUDED.lineage,
    quality_rules = EXCLUDED.quality_rules,
    data_classification = EXCLUDED.data_classification,
    contains_pii = EXCLUDED.contains_pii,
    pii_columns = EXCLUDED.pii_columns,
    field_count = EXCLUDED.field_count,
    tags = EXCLUDED.tags,
    metadata = EXCLUDED.metadata,
    updated_at = EXCLUDED.updated_at,
    deleted_at = NULL;

WITH seeded_models AS (
    SELECT gs, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'data-model-' || gs) AS model_id
    FROM generate_series(1, {{ .Scale.DataModelCount }}) gs
)
INSERT INTO data_catalogs (
    id, tenant_id, name, description, schema_info, owner, tags, classification,
    access_count, last_accessed_at, created_at, updated_at, created_by
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'data-catalog-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('Catalog Entry %s', lpad(gs::text, 2, '0')),
    'Seeded catalog entry with discoverability metadata.',
    jsonb_build_object('model_id', model_id, 'columns', 12 + (gs % 8)),
    '{{ .DataStewardUserID }}'::uuid,
    ARRAY['seeded','catalog'],
    CASE (gs - 1) % 4
        WHEN 0 THEN 'public'
        WHEN 1 THEN 'internal'
        WHEN 2 THEN 'confidential'
        ELSE 'restricted'
    END,
    50 + (gs * 3),
    now() - make_interval(hours => gs),
    now() - make_interval(days => 20 - (gs % 10)),
    now() - make_interval(hours => gs),
    '{{ .DataStewardUserID }}'::uuid
FROM seeded_models
ON CONFLICT (id) DO UPDATE SET
    description = EXCLUDED.description,
    schema_info = EXCLUDED.schema_info,
    access_count = EXCLUDED.access_count,
    last_accessed_at = EXCLUDED.last_accessed_at,
    updated_at = EXCLUDED.updated_at;

WITH seeded_pipelines AS (
    SELECT
        gs,
        uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'pipeline-' || gs) AS pipeline_id,
        uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'data-source-' || (((gs - 1) % {{ .Scale.DataSourceCount }}) + 1)) AS source_id,
        uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'data-source-' || (((gs) % {{ .Scale.DataSourceCount }}) + 1)) AS target_id
    FROM generate_series(1, {{ .Scale.PipelineCount }}) gs
)
INSERT INTO pipelines (
    id, tenant_id, name, description, type, source_id, target_id, schedule, config, status,
    last_run_at, next_run_at, created_by, created_at, updated_at, last_run_id, last_run_status,
    last_run_error, total_runs, successful_runs, failed_runs, total_records_processed, avg_duration_ms, tags
)
SELECT
    pipeline_id,
    '{{ .MainTenantID }}'::uuid,
    format('Seeded Pipeline %s', lpad(gs::text, 2, '0')),
    'Seeded pipeline for ETL, quality, and lineage demonstrations.',
    CASE (gs - 1) % 4
        WHEN 0 THEN 'etl'
        WHEN 1 THEN 'elt'
        WHEN 2 THEN 'batch'
        ELSE 'streaming'
    END,
    source_id,
    target_id,
    CASE WHEN gs % 2 = 0 THEN '0 * * * *' ELSE '15 */6 * * *' END,
    jsonb_build_object('transformations', jsonb_build_array('normalize', 'enrich', 'mask_pii')),
    CASE WHEN gs % 13 = 0 THEN 'error' WHEN gs % 5 = 0 THEN 'paused' ELSE 'active' END,
    now() - make_interval(hours => gs),
    now() + make_interval(hours => (gs % 12) + 1),
    '{{ .DataStewardUserID }}'::uuid,
    now() - make_interval(days => 30 - gs),
    now() - make_interval(hours => gs),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'pipeline-run-' || gs),
    CASE WHEN gs % 13 = 0 THEN 'failed' WHEN gs % 7 = 0 THEN 'running' ELSE 'completed' END,
    CASE WHEN gs % 13 = 0 THEN 'Seeded pipeline validation failure' ELSE NULL END,
    12 + (gs * 2),
    10 + (gs * 2),
    CASE WHEN gs % 13 = 0 THEN 2 ELSE 1 END,
    400000 + (gs * 12000),
    620000 + (gs * 2000),
    ARRAY['seeded','pipeline']
FROM seeded_pipelines
ON CONFLICT (id) DO UPDATE SET
    description = EXCLUDED.description,
    type = EXCLUDED.type,
    source_id = EXCLUDED.source_id,
    target_id = EXCLUDED.target_id,
    schedule = EXCLUDED.schedule,
    config = EXCLUDED.config,
    status = EXCLUDED.status,
    last_run_at = EXCLUDED.last_run_at,
    next_run_at = EXCLUDED.next_run_at,
    last_run_id = EXCLUDED.last_run_id,
    last_run_status = EXCLUDED.last_run_status,
    last_run_error = EXCLUDED.last_run_error,
    total_runs = EXCLUDED.total_runs,
    successful_runs = EXCLUDED.successful_runs,
    failed_runs = EXCLUDED.failed_runs,
    total_records_processed = EXCLUDED.total_records_processed,
    avg_duration_ms = EXCLUDED.avg_duration_ms,
    tags = EXCLUDED.tags,
    updated_at = EXCLUDED.updated_at,
    deleted_at = NULL;

WITH seeded_runs AS (
    SELECT
        gs,
        uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'pipeline-run-' || gs) AS run_id,
        uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'pipeline-' || (((gs - 1) % {{ .Scale.PipelineCount }}) + 1)) AS pipeline_id
    FROM generate_series(1, {{ .Scale.PipelineRunCount }}) gs
)
INSERT INTO pipeline_runs (
    id, tenant_id, pipeline_id, status, started_at, completed_at, records_processed, records_failed,
    error_log, metrics, created_at, current_phase, records_extracted, records_transformed, records_loaded,
    records_filtered, records_deduplicated, bytes_read, bytes_written, quality_gate_results,
    quality_gates_passed, quality_gates_failed, quality_gates_warned, extract_started_at, extract_completed_at,
    transform_started_at, transform_completed_at, load_started_at, load_completed_at, duration_ms,
    error_phase, error_message, error_details, triggered_by, triggered_by_user, retry_count,
    incremental_from, incremental_to
)
SELECT
    run_id,
    '{{ .MainTenantID }}'::uuid,
    pipeline_id,
    CASE WHEN gs % 17 = 0 THEN 'failed' WHEN gs % 11 = 0 THEN 'running' ELSE 'completed' END,
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320)),
    CASE WHEN gs % 11 = 0 THEN NULL ELSE date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 18 + (gs % 20)) END,
    120000 + (gs * 150),
    CASE WHEN gs % 17 = 0 THEN 120 + (gs % 30) ELSE 5 + (gs % 8) END,
    CASE WHEN gs % 17 = 0 THEN 'Seeded transform step failed.' ELSE NULL END,
    jsonb_build_object('throughput_rps', 900 + (gs % 120), 'seeded', true),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320)),
    CASE WHEN gs % 17 = 0 THEN 'transform' WHEN gs % 11 = 0 THEN 'load' ELSE 'completed' END,
    120000 + (gs * 150),
    118000 + (gs * 145),
    117500 + (gs * 143),
    80 + (gs % 30),
    30 + (gs % 15),
    5242880 + (gs * 20000),
    4194304 + (gs * 18000),
    jsonb_build_array(jsonb_build_object('name', 'freshness', 'status', 'passed')),
    3,
    CASE WHEN gs % 17 = 0 THEN 1 ELSE 0 END,
    CASE WHEN gs % 5 = 0 THEN 1 ELSE 0 END,
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320)),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 4),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 4),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 10),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 10),
    CASE WHEN gs % 11 = 0 THEN NULL ELSE date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 18 + (gs % 20)) END,
    CASE WHEN gs % 11 = 0 THEN NULL ELSE (18 + (gs % 20)) * 60000 END,
    CASE WHEN gs % 17 = 0 THEN 'transform' ELSE NULL END,
    CASE WHEN gs % 17 = 0 THEN 'Seeded transform step failed.' ELSE NULL END,
    CASE WHEN gs % 17 = 0 THEN jsonb_build_object('failed_step', 'transform') ELSE NULL END,
    CASE WHEN gs % 3 = 0 THEN 'schedule' WHEN gs % 5 = 0 THEN 'retry' ELSE 'manual' END,
    '{{ .DataStewardUserID }}'::uuid,
    CASE WHEN gs % 5 = 0 THEN 1 ELSE 0 END,
    CASE WHEN gs % 3 = 0 THEN to_char(now() - interval '1 day', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') ELSE NULL END,
    CASE WHEN gs % 3 = 0 THEN to_char(now(), 'YYYY-MM-DD"T"HH24:MI:SS"Z"') ELSE NULL END
FROM seeded_runs
ON CONFLICT (id) DO UPDATE SET
    status = EXCLUDED.status,
    completed_at = EXCLUDED.completed_at,
    records_processed = EXCLUDED.records_processed,
    records_failed = EXCLUDED.records_failed,
    error_log = EXCLUDED.error_log,
    metrics = EXCLUDED.metrics,
    current_phase = EXCLUDED.current_phase,
    duration_ms = EXCLUDED.duration_ms,
    error_message = EXCLUDED.error_message,
    error_details = EXCLUDED.error_details;

INSERT INTO pipeline_run_logs (
    id, tenant_id, run_id, level, phase, message, details, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'pipeline-run-log-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'pipeline-run-' || (((gs - 1) % {{ .Scale.PipelineRunCount }}) + 1)),
    CASE WHEN gs % 19 = 0 THEN 'error' WHEN gs % 7 = 0 THEN 'warn' WHEN gs % 5 = 0 THEN 'debug' ELSE 'info' END,
    CASE (gs - 1) % 4
        WHEN 0 THEN 'extract'
        WHEN 1 THEN 'transform'
        WHEN 2 THEN 'quality'
        ELSE 'load'
    END,
    format('Seeded pipeline log line %s', gs),
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'log_index', gs),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320))
FROM generate_series(1, {{ .Scale.PipelineRunLogCount }}) gs
ON CONFLICT (id) DO NOTHING;

WITH seeded_rules AS (
    SELECT
        gs,
        uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'quality-rule-' || gs) AS rule_id,
        uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'data-model-' || (((gs - 1) % {{ .Scale.DataModelCount }}) + 1)) AS model_id
    FROM generate_series(1, {{ .Scale.QualityRuleCount }}) gs
)
INSERT INTO quality_rules (
    id, tenant_id, model_id, column_name, rule_type, severity, enabled, created_at, updated_at,
    created_by, name, description, config, schedule, last_run_at, last_status, consecutive_failures, tags
)
SELECT
    rule_id,
    '{{ .MainTenantID }}'::uuid,
    model_id,
    CASE (gs - 1) % 4
        WHEN 0 THEN 'entity_id'
        WHEN 1 THEN 'email'
        WHEN 2 THEN 'updated_at'
        ELSE 'score'
    END,
    CASE (gs - 1) % 6
        WHEN 0 THEN 'not_null'
        WHEN 1 THEN 'unique'
        WHEN 2 THEN 'range'
        WHEN 3 THEN 'regex'
        WHEN 4 THEN 'freshness'
        ELSE 'statistical'
    END,
    CASE (gs - 1) % 4
        WHEN 0 THEN 'critical'
        WHEN 1 THEN 'high'
        WHEN 2 THEN 'medium'
        ELSE 'low'
    END,
    true,
    now() - make_interval(days => 20 - (gs % 10)),
    now() - make_interval(hours => gs),
    '{{ .DataStewardUserID }}'::uuid,
    format('Seeded Quality Rule %s', lpad(gs::text, 3, '0')),
    'Seeded data quality rule for quality scorecards and failure drilldowns.',
    jsonb_build_object('threshold', 95, 'seeded', true),
    CASE WHEN gs % 2 = 0 THEN '0 * * * *' ELSE '15 */6 * * *' END,
    now() - make_interval(hours => gs),
    CASE WHEN gs % 11 = 0 THEN 'failed' WHEN gs % 5 = 0 THEN 'warning' ELSE 'passed' END,
    CASE WHEN gs % 11 = 0 THEN 2 ELSE 0 END,
    ARRAY['seeded','quality']
FROM seeded_rules
ON CONFLICT (id) DO UPDATE SET
    model_id = EXCLUDED.model_id,
    column_name = EXCLUDED.column_name,
    rule_type = EXCLUDED.rule_type,
    severity = EXCLUDED.severity,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    config = EXCLUDED.config,
    schedule = EXCLUDED.schedule,
    last_run_at = EXCLUDED.last_run_at,
    last_status = EXCLUDED.last_status,
    consecutive_failures = EXCLUDED.consecutive_failures,
    tags = EXCLUDED.tags,
    updated_at = EXCLUDED.updated_at,
    deleted_at = NULL;

INSERT INTO quality_results (
    id, tenant_id, rule_id, model_id, pipeline_run_id, status, records_checked, records_failed,
    failure_samples, checked_at, created_at, records_passed, pass_rate, failure_summary, duration_ms, error_message
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'quality-result-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'quality-rule-' || (((gs - 1) % {{ .Scale.QualityRuleCount }}) + 1)),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'data-model-' || (((gs - 1) % {{ .Scale.DataModelCount }}) + 1)),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'pipeline-run-' || (((gs - 1) % {{ .Scale.PipelineRunCount }}) + 1)),
    CASE WHEN gs % 13 = 0 THEN 'failed' WHEN gs % 7 = 0 THEN 'warning' ELSE 'passed' END,
    10000 + (gs * 12),
    CASE WHEN gs % 13 = 0 THEN 120 + (gs % 40) WHEN gs % 7 = 0 THEN 25 + (gs % 10) ELSE 0 END,
    CASE
        WHEN gs % 13 = 0 THEN jsonb_build_array(jsonb_build_object('row_id', gs, 'reason', 'threshold breach'))
        ELSE '[]'::jsonb
    END,
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320)),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320)),
    CASE WHEN gs % 13 = 0 THEN (10000 + (gs * 12)) - (120 + (gs % 40)) WHEN gs % 7 = 0 THEN (10000 + (gs * 12)) - (25 + (gs % 10)) ELSE 10000 + (gs * 12) END,
    CASE WHEN gs % 13 = 0 THEN 98.20 WHEN gs % 7 = 0 THEN 99.60 ELSE 100.00 END,
    CASE WHEN gs % 13 = 0 THEN 'Seeded quality failures detected.' WHEN gs % 7 = 0 THEN 'Seeded warning threshold reached.' ELSE NULL END,
    200 + (gs % 800),
    CASE WHEN gs % 13 = 0 THEN 'Seeded failing sample' ELSE NULL END
FROM generate_series(1, {{ .Scale.QualityResultCount }}) gs
ON CONFLICT (id) DO NOTHING;

INSERT INTO contradiction_scans (
    id, tenant_id, status, models_scanned, model_pairs_compared, contradictions_found,
    by_type, by_severity, started_at, completed_at, duration_ms, triggered_by, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'contradiction-scan-' || gs),
    '{{ .MainTenantID }}'::uuid,
    'completed',
    {{ .Scale.DataModelCount }},
    {{ .Scale.DataModelCount }} * 2,
    15 + gs,
    '{"logical":6,"semantic":4,"temporal":3,"analytical":2}'::jsonb,
    '{"critical":1,"high":4,"medium":7,"low":3}'::jsonb,
    now() - make_interval(days => gs),
    now() - make_interval(days => gs) + interval '22 minutes',
    1320000,
    '{{ .DataStewardUserID }}'::uuid,
    now() - make_interval(days => gs)
FROM generate_series(1, 12) gs
ON CONFLICT (id) DO NOTHING;

INSERT INTO contradictions (
    id, tenant_id, type, source_a, source_b, description, severity, confidence_score, resolution_guidance,
    status, created_at, updated_at, created_by, scan_id, title, entity_key_column, entity_key_value,
    affected_records, sample_records, authoritative_source, resolution_notes, resolution_action, metadata
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'contradiction-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE (gs - 1) % 4
        WHEN 0 THEN 'logical'
        WHEN 1 THEN 'semantic'
        WHEN 2 THEN 'temporal'
        ELSE 'analytical'
    END,
    jsonb_build_object('source', format('Seeded Data Source %s', (((gs - 1) % {{ .Scale.DataSourceCount }}) + 1)), 'field', 'status', 'value', 'active'),
    jsonb_build_object('source', format('Seeded Data Source %s', (((gs) % {{ .Scale.DataSourceCount }}) + 1)), 'field', 'status', 'value', 'inactive'),
    'Seeded contradiction across overlapping records for RCA and governance workflows.',
    CASE WHEN gs % 9 = 0 THEN 'critical' WHEN gs % 5 = 0 THEN 'high' WHEN gs % 3 = 0 THEN 'medium' ELSE 'low' END,
    round(((70 + (gs % 25))::numeric / 100), 4),
    'Review source precedence and reconcile canonical values.',
    CASE WHEN gs % 11 = 0 THEN 'investigating' WHEN gs % 17 = 0 THEN 'resolved' ELSE 'detected' END,
    now() - make_interval(days => (gs % 18)),
    now() - make_interval(hours => gs),
    '{{ .DataStewardUserID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'contradiction-scan-' || (((gs - 1) % 12) + 1)),
    format('Seeded contradiction %s', lpad(gs::text, 4, '0')),
    'entity_id',
    format('ENT-%s', lpad(gs::text, 6, '0')),
    5 + (gs % 40),
    jsonb_build_array(jsonb_build_object('entity_id', format('ENT-%s', lpad(gs::text, 6, '0')))),
    CASE WHEN gs % 2 = 0 THEN 'seeded_source_a' ELSE 'seeded_source_b' END,
    CASE WHEN gs % 17 = 0 THEN 'Marked reconciled during seeded governance review.' ELSE NULL END,
    CASE WHEN gs % 17 = 0 THEN 'data_reconciled' ELSE NULL END,
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs)
FROM generate_series(1, {{ .Scale.ContradictionCount }}) gs
ON CONFLICT (id) DO NOTHING;

INSERT INTO data_lineage_edges (
    id, tenant_id, source_type, source_id, source_name, target_type, target_id, target_name,
    relationship, transformation_desc, transformation_type, columns_affected, pipeline_id, pipeline_run_id,
    recorded_by, active, first_seen_at, last_seen_at, metadata, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'lineage-edge-' || gs),
    '{{ .MainTenantID }}'::uuid,
    'data_model',
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'data-model-' || (((gs - 1) % {{ .Scale.DataModelCount }}) + 1)),
    format('Seeded Model %s', lpad((((gs - 1) % {{ .Scale.DataModelCount }}) + 1)::text, 2, '0')),
    'data_model',
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'data-model-' || (((gs) % {{ .Scale.DataModelCount }}) + 1)),
    format('Seeded Model %s', lpad((((gs) % {{ .Scale.DataModelCount }}) + 1)::text, 2, '0')),
    CASE WHEN gs % 2 = 0 THEN 'derived_from' ELSE 'feeds' END,
    'Seeded transformation from curated source model to downstream analytics model.',
    CASE WHEN gs % 3 = 0 THEN 'batch_transform' ELSE 'enrichment' END,
    ARRAY['entity_id', 'score', 'updated_at'],
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'pipeline-' || (((gs - 1) % {{ .Scale.PipelineCount }}) + 1)),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'pipeline-run-' || (((gs - 1) % {{ .Scale.PipelineRunCount }}) + 1)),
    'pipeline',
    true,
    now() - make_interval(days => 20 - (gs % 10)),
    now() - make_interval(hours => gs),
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs),
    now() - make_interval(days => 20 - (gs % 10)),
    now() - make_interval(hours => gs)
FROM generate_series(1, {{ .Scale.DataModelCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    relationship = EXCLUDED.relationship,
    transformation_desc = EXCLUDED.transformation_desc,
    transformation_type = EXCLUDED.transformation_type,
    columns_affected = EXCLUDED.columns_affected,
    pipeline_id = EXCLUDED.pipeline_id,
    pipeline_run_id = EXCLUDED.pipeline_run_id,
    last_seen_at = EXCLUDED.last_seen_at,
    metadata = EXCLUDED.metadata,
    updated_at = EXCLUDED.updated_at;

INSERT INTO dark_data_scans (
    id, tenant_id, status, sources_scanned, storage_scanned, assets_discovered, by_reason, by_type,
    pii_assets_found, high_risk_found, total_size_bytes, started_at, completed_at, duration_ms, triggered_by, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dark-data-scan-' || gs),
    '{{ .MainTenantID }}'::uuid,
    'completed',
    {{ .Scale.DataSourceCount }},
    true,
    20 + (gs * 5),
    '{"unmodeled":8,"orphaned_file":6,"stale":4,"ungoverned":2}'::jsonb,
    '{"database_table":10,"file":8,"api_endpoint":2}'::jsonb,
    4 + gs,
    2 + (gs % 4),
    2147483648 + (gs * 1048576),
    now() - make_interval(days => gs),
    now() - make_interval(days => gs) + interval '15 minutes',
    900000,
    '{{ .DataStewardUserID }}'::uuid,
    now() - make_interval(days => gs)
FROM generate_series(1, 8) gs
ON CONFLICT (id) DO NOTHING;

INSERT INTO dark_data_assets (
    id, tenant_id, scan_id, name, asset_type, source_id, source_name, schema_name, table_name,
    file_path, reason, estimated_row_count, estimated_size_bytes, column_count, contains_pii,
    pii_types, inferred_classification, last_accessed_at, last_modified_at, days_since_access,
    risk_score, risk_factors, governance_status, governance_notes, reviewed_by, reviewed_at,
    linked_model_id, metadata, discovered_at, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dark-data-asset-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dark-data-scan-' || (((gs - 1) % 8) + 1)),
    format('Seeded Dark Asset %s', lpad(gs::text, 4, '0')),
    CASE (gs - 1) % 5
        WHEN 0 THEN 'database_table'
        WHEN 1 THEN 'database_view'
        WHEN 2 THEN 'file'
        WHEN 3 THEN 'api_endpoint'
        ELSE 'stream_topic'
    END,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'data-source-' || (((gs - 1) % {{ .Scale.DataSourceCount }}) + 1)),
    format('Seeded Data Source %s', (((gs - 1) % {{ .Scale.DataSourceCount }}) + 1)),
    'public',
    format('shadow_table_%s', gs),
    format('/darkdata/seeded/file_%s.parquet', gs),
    CASE (gs - 1) % 5
        WHEN 0 THEN 'unmodeled'
        WHEN 1 THEN 'orphaned_file'
        WHEN 2 THEN 'stale'
        WHEN 3 THEN 'ungoverned'
        ELSE 'unclassified'
    END,
    50000 + (gs * 90),
    1048576 + (gs * 8192),
    6 + (gs % 18),
    gs % 3 = 0,
    CASE WHEN gs % 3 = 0 THEN ARRAY['email', 'phone_number'] ELSE ARRAY[]::text[] END,
    CASE (gs - 1) % 4
        WHEN 0 THEN 'public'
        WHEN 1 THEN 'internal'
        WHEN 2 THEN 'confidential'
        ELSE 'restricted'
    END,
    now() - make_interval(days => (gs % 90)),
    now() - make_interval(days => (gs % 30)),
    5 + (gs % 120),
    CASE WHEN gs % 11 = 0 THEN 92.5 WHEN gs % 5 = 0 THEN 78.0 ELSE 42.0 + (gs % 35) END,
    jsonb_build_array(jsonb_build_object('factor', 'unguarded_copy'), jsonb_build_object('factor', 'stale_access_pattern')),
    CASE WHEN gs % 13 = 0 THEN 'under_review' WHEN gs % 17 = 0 THEN 'governed' ELSE 'unmanaged' END,
    CASE WHEN gs % 17 = 0 THEN 'Seeded review completed.' ELSE NULL END,
    CASE WHEN gs % 17 = 0 THEN '{{ .DataStewardUserID }}'::uuid ELSE NULL END,
    CASE WHEN gs % 17 = 0 THEN now() - interval '1 day' ELSE NULL END,
    CASE WHEN gs % 2 = 0 THEN uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'data-model-' || (((gs - 1) % {{ .Scale.DataModelCount }}) + 1)) ELSE NULL END,
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs),
    now() - make_interval(days => (gs % 14)),
    now() - make_interval(days => (gs % 14)),
    now() - make_interval(hours => gs)
FROM generate_series(1, {{ .Scale.DarkDataAssetCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    scan_id = EXCLUDED.scan_id,
    name = EXCLUDED.name,
    asset_type = EXCLUDED.asset_type,
    source_id = EXCLUDED.source_id,
    source_name = EXCLUDED.source_name,
    schema_name = EXCLUDED.schema_name,
    table_name = EXCLUDED.table_name,
    file_path = EXCLUDED.file_path,
    reason = EXCLUDED.reason,
    estimated_row_count = EXCLUDED.estimated_row_count,
    estimated_size_bytes = EXCLUDED.estimated_size_bytes,
    column_count = EXCLUDED.column_count,
    contains_pii = EXCLUDED.contains_pii,
    pii_types = EXCLUDED.pii_types,
    inferred_classification = EXCLUDED.inferred_classification,
    last_accessed_at = EXCLUDED.last_accessed_at,
    last_modified_at = EXCLUDED.last_modified_at,
    days_since_access = EXCLUDED.days_since_access,
    risk_score = EXCLUDED.risk_score,
    risk_factors = EXCLUDED.risk_factors,
    governance_status = EXCLUDED.governance_status,
    governance_notes = EXCLUDED.governance_notes,
    reviewed_by = EXCLUDED.reviewed_by,
    reviewed_at = EXCLUDED.reviewed_at,
    linked_model_id = EXCLUDED.linked_model_id,
    metadata = EXCLUDED.metadata,
    discovered_at = EXCLUDED.discovered_at,
    updated_at = EXCLUDED.updated_at;

INSERT INTO saved_queries (
    id, tenant_id, name, description, model_id, query_definition, last_run_at, run_count,
    visibility, tags, created_by, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'saved-query-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('Seeded Saved Query %s', lpad(gs::text, 3, '0')),
    'Seeded saved query for analytics and audit demonstrations.',
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'data-model-' || (((gs - 1) % {{ .Scale.DataModelCount }}) + 1)),
    jsonb_build_object('select', jsonb_build_array('entity_id', 'score'), 'filters', jsonb_build_array(jsonb_build_object('field', 'status', 'op', '=', 'value', 'active'))),
    now() - make_interval(hours => gs),
    5 + (gs * 2),
    CASE WHEN gs % 3 = 0 THEN 'organization' WHEN gs % 2 = 0 THEN 'team' ELSE 'private' END,
    ARRAY['seeded','analytics'],
    '{{ .DataStewardUserID }}'::uuid,
    now() - make_interval(days => 15 - (gs % 10)),
    now() - make_interval(hours => gs)
FROM generate_series(1, {{ .Scale.SavedQueryCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    description = EXCLUDED.description,
    query_definition = EXCLUDED.query_definition,
    last_run_at = EXCLUDED.last_run_at,
    run_count = EXCLUDED.run_count,
    visibility = EXCLUDED.visibility,
    tags = EXCLUDED.tags,
    updated_at = EXCLUDED.updated_at,
    deleted_at = NULL;

INSERT INTO analytics_audit_log (
    id, tenant_id, user_id, model_id, source_id, query_definition, columns_accessed, filters_applied,
    data_classification, pii_columns_accessed, pii_masking_applied, rows_returned, truncated,
    execution_time_ms, error_occurred, error_message, saved_query_id, ip_address, user_agent, executed_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'analytics-audit-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE ((gs - 1) % 7) + 1
        WHEN 1 THEN '{{ .MainAdminUserID }}'::uuid
        WHEN 2 THEN '{{ .SecurityManagerUserID }}'::uuid
        WHEN 3 THEN '{{ .DataStewardUserID }}'::uuid
        WHEN 4 THEN '{{ .LegalManagerUserID }}'::uuid
        WHEN 5 THEN '{{ .BoardSecretaryUserID }}'::uuid
        WHEN 6 THEN '{{ .ExecutiveUserID }}'::uuid
        ELSE '{{ .AuditorUserID }}'::uuid
    END,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'data-model-' || (((gs - 1) % {{ .Scale.DataModelCount }}) + 1)),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'data-source-' || (((gs - 1) % {{ .Scale.DataSourceCount }}) + 1)),
    jsonb_build_object('query', format('select * from seeded_model_%s limit %s', (((gs - 1) % {{ .Scale.DataModelCount }}) + 1), 100 + (gs % 200))),
    ARRAY['entity_id', 'owner_name', 'score'],
    jsonb_build_array(jsonb_build_object('field', 'status', 'op', '=', 'value', 'active')),
    CASE (gs - 1) % 4
        WHEN 0 THEN 'public'
        WHEN 1 THEN 'internal'
        WHEN 2 THEN 'confidential'
        ELSE 'restricted'
    END,
    CASE WHEN gs % 9 = 0 THEN ARRAY['email'] ELSE ARRAY[]::text[] END,
    gs % 9 = 0,
    100 + (gs % 900),
    gs % 19 = 0,
    35 + (gs % 800),
    gs % 23 = 0,
    CASE WHEN gs % 23 = 0 THEN 'Seeded analytics query error.' ELSE NULL END,
    CASE WHEN gs % 3 = 0 THEN uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'saved-query-' || (((gs - 1) % {{ .Scale.SavedQueryCount }}) + 1)) ELSE NULL END,
    format('10.40.%s.%s', ((gs - 1) % 255), (gs % 255)),
    format('Clario Analytics/%s', 1 + (gs % 4)),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320))
FROM generate_series(1, {{ .Scale.AnalyticsAuditLogCount }}) gs
ON CONFLICT (id) DO NOTHING;
