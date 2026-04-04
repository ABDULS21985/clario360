DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'data_lineage'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'data_lineage_edges'
    ) THEN
        ALTER TABLE data_lineage RENAME TO data_lineage_edges;
    END IF;
END $$;

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

ALTER TABLE data_lineage_edges
    ADD COLUMN IF NOT EXISTS source_name TEXT DEFAULT '',
    ADD COLUMN IF NOT EXISTS target_name TEXT DEFAULT '',
    ADD COLUMN IF NOT EXISTS relationship TEXT DEFAULT 'depends_on',
    ADD COLUMN IF NOT EXISTS transformation TEXT,
    ADD COLUMN IF NOT EXISTS transformation_desc TEXT,
    ADD COLUMN IF NOT EXISTS transformation_type TEXT,
    ADD COLUMN IF NOT EXISTS columns_affected TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS pipeline_run_id UUID REFERENCES pipeline_runs(id),
    ADD COLUMN IF NOT EXISTS recorded_by TEXT NOT NULL DEFAULT 'manual',
    ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

UPDATE data_lineage_edges
SET source_name = COALESCE(NULLIF(source_name, ''), source_type || ':' || source_id::text),
    target_name = COALESCE(NULLIF(target_name, ''), target_type || ':' || target_id::text),
    relationship = COALESCE(NULLIF(relationship, ''), 'depends_on'),
    transformation_desc = COALESCE(transformation_desc, transformation),
    recorded_by = COALESCE(NULLIF(recorded_by, ''), 'manual'),
    active = COALESCE(active, true),
    first_seen_at = COALESCE(first_seen_at, created_at, now()),
    last_seen_at = COALESCE(last_seen_at, created_at, now()),
    metadata = COALESCE(metadata, '{}'::jsonb),
    updated_at = COALESCE(updated_at, created_at, now());

DELETE FROM data_lineage_edges d
USING (
    SELECT ctid,
           ROW_NUMBER() OVER (
               PARTITION BY tenant_id, source_type, source_id, target_type, target_id, relationship
               ORDER BY created_at DESC, id DESC
           ) AS row_num
    FROM data_lineage_edges
) dup
WHERE d.ctid = dup.ctid
  AND dup.row_num > 1;

ALTER TABLE data_lineage_edges
    ALTER COLUMN source_name SET NOT NULL,
    ALTER COLUMN target_name SET NOT NULL,
    ALTER COLUMN relationship SET NOT NULL,
    ALTER COLUMN columns_affected SET DEFAULT '{}',
    ALTER COLUMN recorded_by SET DEFAULT 'manual',
    ALTER COLUMN recorded_by SET NOT NULL,
    ALTER COLUMN active SET DEFAULT true,
    ALTER COLUMN active SET NOT NULL,
    ALTER COLUMN first_seen_at SET DEFAULT now(),
    ALTER COLUMN first_seen_at SET NOT NULL,
    ALTER COLUMN last_seen_at SET DEFAULT now(),
    ALTER COLUMN last_seen_at SET NOT NULL,
    ALTER COLUMN metadata SET DEFAULT '{}',
    ALTER COLUMN metadata SET NOT NULL,
    ALTER COLUMN updated_at SET DEFAULT now(),
    ALTER COLUMN updated_at SET NOT NULL;

ALTER TABLE data_lineage_edges
    DROP CONSTRAINT IF EXISTS data_lineage_edges_relationship_check,
    DROP CONSTRAINT IF EXISTS data_lineage_edges_recorded_by_check;

ALTER TABLE data_lineage_edges
    ADD CONSTRAINT data_lineage_edges_relationship_check CHECK (relationship IN (
        'feeds', 'derived_from', 'transforms_into', 'consumed_by',
        'validated_by', 'reported_in', 'queried_by', 'depends_on'
    )),
    ADD CONSTRAINT data_lineage_edges_recorded_by_check CHECK (recorded_by IN ('system', 'pipeline', 'query', 'manual', 'event'));

DROP INDEX IF EXISTS idx_lineage_tenant;
DROP INDEX IF EXISTS idx_lineage_source;
DROP INDEX IF EXISTS idx_lineage_target;
DROP INDEX IF EXISTS idx_lineage_pipeline;
CREATE UNIQUE INDEX IF NOT EXISTS idx_lineage_edge_unique
    ON data_lineage_edges (tenant_id, source_type, source_id, target_type, target_id, relationship);
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

ALTER TABLE dark_data_assets
    ADD COLUMN IF NOT EXISTS location TEXT,
    ADD COLUMN IF NOT EXISTS type TEXT,
    ADD COLUMN IF NOT EXISTS size_bytes BIGINT,
    ADD COLUMN IF NOT EXISTS classification TEXT,
    ADD COLUMN IF NOT EXISTS scan_id UUID REFERENCES dark_data_scans(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS name TEXT,
    ADD COLUMN IF NOT EXISTS asset_type TEXT,
    ADD COLUMN IF NOT EXISTS source_id UUID REFERENCES data_sources(id),
    ADD COLUMN IF NOT EXISTS source_name TEXT,
    ADD COLUMN IF NOT EXISTS schema_name TEXT,
    ADD COLUMN IF NOT EXISTS table_name TEXT,
    ADD COLUMN IF NOT EXISTS file_path TEXT,
    ADD COLUMN IF NOT EXISTS reason TEXT,
    ADD COLUMN IF NOT EXISTS estimated_row_count BIGINT,
    ADD COLUMN IF NOT EXISTS estimated_size_bytes BIGINT,
    ADD COLUMN IF NOT EXISTS column_count INT,
    ADD COLUMN IF NOT EXISTS contains_pii BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS pii_types TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS inferred_classification TEXT,
    ADD COLUMN IF NOT EXISTS last_modified_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS days_since_access INT,
    ADD COLUMN IF NOT EXISTS risk_factors JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS governance_notes TEXT,
    ADD COLUMN IF NOT EXISTS reviewed_by UUID,
    ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS linked_model_id UUID REFERENCES data_models(id),
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}';

ALTER TABLE dark_data_assets
    DROP CONSTRAINT IF EXISTS dark_data_assets_risk_score_check;

ALTER TABLE dark_data_assets
    ALTER COLUMN location DROP NOT NULL,
    ALTER COLUMN type DROP NOT NULL,
    ALTER COLUMN governance_status TYPE TEXT USING governance_status::text,
    ALTER COLUMN risk_score TYPE DECIMAL(5,2) USING (
        CASE
            WHEN risk_score IS NULL THEN 0
            WHEN risk_score <= 1 THEN ROUND((risk_score * 100)::numeric, 2)
            ELSE ROUND(risk_score::numeric, 2)
        END
    );

UPDATE dark_data_assets
SET name = COALESCE(NULLIF(name, ''), location),
    asset_type = COALESCE(NULLIF(asset_type, ''), CASE LOWER(COALESCE(type, ''))
        WHEN 'database_table' THEN 'database_table'
        WHEN 'database_view' THEN 'database_view'
        WHEN 'api_endpoint' THEN 'api_endpoint'
        WHEN 'stream_topic' THEN 'stream_topic'
        WHEN 'stream' THEN 'stream_topic'
        WHEN 'api' THEN 'api_endpoint'
        ELSE 'file'
    END),
    file_path = COALESCE(file_path, location),
    reason = COALESCE(NULLIF(reason, ''), CASE
        WHEN source_id IS NOT NULL THEN 'unmodeled'
        WHEN LOWER(COALESCE(type, '')) IN ('file', 'csv', 's3') THEN 'orphaned_file'
        ELSE 'unclassified'
    END),
    estimated_size_bytes = COALESCE(estimated_size_bytes, size_bytes),
    inferred_classification = COALESCE(
        inferred_classification,
        CASE LOWER(COALESCE(classification, ''))
            WHEN 'public' THEN 'public'
            WHEN 'internal' THEN 'internal'
            WHEN 'confidential' THEN 'confidential'
            WHEN 'restricted' THEN 'restricted'
            ELSE NULL
        END
    ),
    pii_types = COALESCE(pii_types, '{}'::text[]),
    contains_pii = COALESCE(contains_pii, false),
    risk_score = COALESCE(risk_score, 0),
    risk_factors = COALESCE(risk_factors, '[]'::jsonb),
    governance_status = COALESCE(NULLIF(governance_status, ''), 'unmanaged'),
    metadata = COALESCE(metadata, '{}'::jsonb),
    discovered_at = COALESCE(discovered_at, created_at, now()),
    updated_at = COALESCE(updated_at, created_at, now());

DELETE FROM dark_data_assets d
USING (
    SELECT ctid,
           ROW_NUMBER() OVER (
               PARTITION BY tenant_id, asset_type, COALESCE(source_id, '00000000-0000-0000-0000-000000000000'::uuid),
                            COALESCE(schema_name, ''), COALESCE(table_name, ''), COALESCE(file_path, ''), reason
               ORDER BY updated_at DESC, id DESC
           ) AS row_num
    FROM dark_data_assets
) dup
WHERE d.ctid = dup.ctid
  AND dup.row_num > 1;

ALTER TABLE dark_data_assets
    ALTER COLUMN name SET NOT NULL,
    ALTER COLUMN asset_type SET NOT NULL,
    ALTER COLUMN reason SET NOT NULL,
    ALTER COLUMN contains_pii SET DEFAULT false,
    ALTER COLUMN contains_pii SET NOT NULL,
    ALTER COLUMN pii_types SET DEFAULT '{}',
    ALTER COLUMN pii_types SET NOT NULL,
    ALTER COLUMN risk_score SET DEFAULT 0,
    ALTER COLUMN risk_score SET NOT NULL,
    ALTER COLUMN risk_factors SET DEFAULT '[]',
    ALTER COLUMN risk_factors SET NOT NULL,
    ALTER COLUMN governance_status SET DEFAULT 'unmanaged',
    ALTER COLUMN governance_status SET NOT NULL,
    ALTER COLUMN metadata SET DEFAULT '{}',
    ALTER COLUMN metadata SET NOT NULL,
    ALTER COLUMN discovered_at SET DEFAULT now(),
    ALTER COLUMN discovered_at SET NOT NULL,
    ALTER COLUMN updated_at SET DEFAULT now(),
    ALTER COLUMN updated_at SET NOT NULL;

ALTER TABLE dark_data_assets
    DROP CONSTRAINT IF EXISTS dark_data_assets_asset_type_check,
    DROP CONSTRAINT IF EXISTS dark_data_assets_reason_check,
    DROP CONSTRAINT IF EXISTS dark_data_assets_inferred_classification_check,
    DROP CONSTRAINT IF EXISTS dark_data_assets_governance_status_check;

ALTER TABLE dark_data_assets
    ADD CONSTRAINT dark_data_assets_asset_type_check CHECK (asset_type IN (
        'database_table', 'database_view', 'file', 'api_endpoint', 'stream_topic'
    )),
    ADD CONSTRAINT dark_data_assets_reason_check CHECK (reason IN (
        'unmodeled', 'orphaned_file', 'stale', 'ungoverned', 'unclassified'
    )),
    ADD CONSTRAINT dark_data_assets_inferred_classification_check CHECK (
        inferred_classification IS NULL OR inferred_classification IN ('public', 'internal', 'confidential', 'restricted')
    ),
    ADD CONSTRAINT dark_data_assets_governance_status_check CHECK (
        governance_status IN ('unmanaged', 'under_review', 'governed', 'archived', 'scheduled_deletion')
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
