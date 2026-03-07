CREATE TABLE IF NOT EXISTS data_lineage_edges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    source_type TEXT NOT NULL CHECK (source_type IN (
        'data_source', 'data_model', 'pipeline', 'quality_rule',
        'suite_consumer', 'report', 'analytics_query', 'external'
    )),
    source_id UUID NOT NULL,
    source_name TEXT NOT NULL,
    target_type TEXT NOT NULL CHECK (target_type IN (
        'data_source', 'data_model', 'pipeline', 'quality_rule',
        'suite_consumer', 'report', 'analytics_query', 'external'
    )),
    target_id UUID NOT NULL,
    target_name TEXT NOT NULL,
    relationship TEXT NOT NULL CHECK (relationship IN (
        'feeds', 'derived_from', 'transforms_into', 'consumed_by',
        'validated_by', 'reported_in', 'queried_by', 'depends_on'
    )),
    transformation_desc TEXT,
    transformation_type TEXT,
    columns_affected TEXT[] NOT NULL DEFAULT '{}',
    pipeline_id UUID REFERENCES pipelines(id),
    pipeline_run_id UUID REFERENCES pipeline_runs(id),
    recorded_by TEXT NOT NULL DEFAULT 'system'
        CHECK (recorded_by IN ('system', 'pipeline', 'query', 'manual', 'event')),
    active BOOLEAN NOT NULL DEFAULT true,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, source_type, source_id, target_type, target_id, relationship)
);

CREATE INDEX IF NOT EXISTS idx_lineage_tenant
    ON data_lineage_edges (tenant_id)
    WHERE active = true;
CREATE INDEX IF NOT EXISTS idx_lineage_source
    ON data_lineage_edges (tenant_id, source_type, source_id)
    WHERE active = true;
CREATE INDEX IF NOT EXISTS idx_lineage_target
    ON data_lineage_edges (tenant_id, target_type, target_id)
    WHERE active = true;
CREATE INDEX IF NOT EXISTS idx_lineage_pipeline
    ON data_lineage_edges (pipeline_id)
    WHERE pipeline_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS dark_data_scans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'running'
        CHECK (status IN ('running', 'completed', 'failed')),
    sources_scanned INT NOT NULL DEFAULT 0,
    storage_scanned BOOLEAN NOT NULL DEFAULT false,
    assets_discovered INT NOT NULL DEFAULT 0,
    by_reason JSONB NOT NULL DEFAULT '{}',
    by_type JSONB NOT NULL DEFAULT '{}',
    pii_assets_found INT NOT NULL DEFAULT 0,
    high_risk_found INT NOT NULL DEFAULT 0,
    total_size_bytes BIGINT NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    duration_ms BIGINT,
    triggered_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_darkdata_scans_tenant
    ON dark_data_scans (tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS dark_data_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    scan_id UUID REFERENCES dark_data_scans(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    asset_type TEXT NOT NULL CHECK (asset_type IN (
        'database_table', 'database_view', 'file', 'api_endpoint', 'stream_topic'
    )),
    source_id UUID REFERENCES data_sources(id),
    source_name TEXT,
    schema_name TEXT,
    table_name TEXT,
    file_path TEXT,
    reason TEXT NOT NULL CHECK (reason IN (
        'unmodeled',
        'orphaned_file',
        'stale',
        'ungoverned',
        'unclassified'
    )),
    estimated_row_count BIGINT,
    estimated_size_bytes BIGINT,
    column_count INT,
    contains_pii BOOLEAN NOT NULL DEFAULT false,
    pii_types TEXT[] NOT NULL DEFAULT '{}',
    inferred_classification TEXT
        CHECK (inferred_classification IN ('public', 'internal', 'confidential', 'restricted')),
    last_accessed_at TIMESTAMPTZ,
    last_modified_at TIMESTAMPTZ,
    days_since_access INT,
    risk_score DECIMAL(5,2) NOT NULL DEFAULT 0,
    risk_factors JSONB NOT NULL DEFAULT '[]',
    governance_status TEXT NOT NULL DEFAULT 'unmanaged'
        CHECK (governance_status IN (
            'unmanaged', 'under_review', 'governed', 'archived', 'scheduled_deletion'
        )),
    governance_notes TEXT,
    reviewed_by UUID,
    reviewed_at TIMESTAMPTZ,
    linked_model_id UUID REFERENCES data_models(id),
    metadata JSONB NOT NULL DEFAULT '{}',
    discovered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_darkdata_identity_unique
    ON dark_data_assets (
        tenant_id,
        asset_type,
        COALESCE(source_id, '00000000-0000-0000-0000-000000000000'::uuid),
        COALESCE(schema_name, ''),
        COALESCE(table_name, ''),
        COALESCE(file_path, ''),
        reason
    );
CREATE INDEX IF NOT EXISTS idx_darkdata_tenant
    ON dark_data_assets (tenant_id, governance_status);
CREATE INDEX IF NOT EXISTS idx_darkdata_risk
    ON dark_data_assets (tenant_id, risk_score DESC);
CREATE INDEX IF NOT EXISTS idx_darkdata_source
    ON dark_data_assets (source_id)
    WHERE source_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_darkdata_pii
    ON dark_data_assets (tenant_id)
    WHERE contains_pii = true;
CREATE INDEX IF NOT EXISTS idx_darkdata_scan
    ON dark_data_assets (scan_id)
    WHERE scan_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS saved_queries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    model_id UUID NOT NULL REFERENCES data_models(id),
    query_definition JSONB NOT NULL,
    last_run_at TIMESTAMPTZ,
    run_count INT NOT NULL DEFAULT 0,
    visibility TEXT NOT NULL DEFAULT 'private'
        CHECK (visibility IN ('private', 'team', 'organization')),
    tags TEXT[] NOT NULL DEFAULT '{}',
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_saved_queries_tenant_name_unique
    ON saved_queries (tenant_id, name, created_by)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_saved_queries_tenant
    ON saved_queries (tenant_id, visibility)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_saved_queries_model
    ON saved_queries (model_id)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS analytics_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    model_id UUID NOT NULL REFERENCES data_models(id),
    source_id UUID NOT NULL REFERENCES data_sources(id),
    query_definition JSONB NOT NULL,
    columns_accessed TEXT[] NOT NULL DEFAULT '{}',
    filters_applied JSONB NOT NULL DEFAULT '[]',
    data_classification TEXT NOT NULL,
    pii_columns_accessed TEXT[] NOT NULL DEFAULT '{}',
    pii_masking_applied BOOLEAN NOT NULL DEFAULT false,
    rows_returned INT NOT NULL DEFAULT 0,
    truncated BOOLEAN NOT NULL DEFAULT false,
    execution_time_ms BIGINT,
    error_occurred BOOLEAN NOT NULL DEFAULT false,
    error_message TEXT,
    saved_query_id UUID REFERENCES saved_queries(id),
    ip_address TEXT,
    user_agent TEXT,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_analytics_audit_tenant
    ON analytics_audit_log (tenant_id, executed_at DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_audit_user
    ON analytics_audit_log (user_id, executed_at DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_audit_model
    ON analytics_audit_log (model_id, executed_at DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_audit_pii
    ON analytics_audit_log (tenant_id, executed_at DESC)
    WHERE pii_columns_accessed != '{}';

CREATE INDEX IF NOT EXISTS idx_quality_results_latest
    ON quality_results (rule_id, checked_at DESC);

CREATE INDEX IF NOT EXISTS idx_contradictions_open
    ON contradictions (tenant_id, status)
    WHERE status IN ('detected', 'investigating');
