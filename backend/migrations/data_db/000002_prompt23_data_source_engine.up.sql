CREATE EXTENSION IF NOT EXISTS "pgcrypto";

ALTER TABLE data_sources
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS encryption_key_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_error TEXT,
    ADD COLUMN IF NOT EXISTS schema_discovered_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_sync_status TEXT,
    ADD COLUMN IF NOT EXISTS last_sync_error TEXT,
    ADD COLUMN IF NOT EXISTS last_sync_duration_ms BIGINT,
    ADD COLUMN IF NOT EXISTS next_sync_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS table_count INT,
    ADD COLUMN IF NOT EXISTS total_row_count BIGINT,
    ADD COLUMN IF NOT EXISTS total_size_bytes BIGINT,
    ADD COLUMN IF NOT EXISTS tags TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'data_sources'
          AND column_name = 'connection_config'
          AND data_type = 'jsonb'
    ) THEN
        ALTER TABLE data_sources
            ALTER COLUMN connection_config DROP DEFAULT;
        ALTER TABLE data_sources
            ALTER COLUMN connection_config TYPE BYTEA
            USING convert_to(connection_config::text, 'UTF8');
    END IF;
END $$;

ALTER TABLE data_sources
    ALTER COLUMN type TYPE TEXT USING (
        CASE type::text
            WHEN 'database' THEN 'postgresql'
            WHEN 'api' THEN 'api'
            WHEN 'file' THEN 'csv'
            WHEN 'cloud_storage' THEN 's3'
            WHEN 'stream' THEN 'stream'
            ELSE type::text
        END
    ),
    ALTER COLUMN status TYPE TEXT USING (
        CASE status::text
            WHEN 'active' THEN 'active'
            WHEN 'inactive' THEN 'inactive'
            WHEN 'error' THEN 'error'
            WHEN 'syncing' THEN 'syncing'
            ELSE 'pending_test'
        END
    );

ALTER TABLE data_sources
    ALTER COLUMN connection_config SET NOT NULL,
    ALTER COLUMN schema_metadata DROP DEFAULT,
    ALTER COLUMN created_by SET NOT NULL;

ALTER TABLE data_sources
    DROP CONSTRAINT IF EXISTS data_sources_type_check,
    DROP CONSTRAINT IF EXISTS data_sources_status_check;

ALTER TABLE data_sources
    ADD CONSTRAINT data_sources_type_check CHECK (type IN ('postgresql', 'mysql', 'mssql', 'api', 'csv', 's3', 'stream')),
    ADD CONSTRAINT data_sources_status_check CHECK (status IN ('pending_test', 'active', 'inactive', 'error', 'syncing'));

DROP INDEX IF EXISTS idx_sources_tenant_status;
DROP INDEX IF EXISTS idx_sources_tenant_type;
DROP INDEX IF EXISTS idx_sources_tenant_created;
DROP INDEX IF EXISTS idx_sources_schema;

CREATE INDEX IF NOT EXISTS idx_sources_tenant ON data_sources (tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_sources_tenant_type ON data_sources (tenant_id, type) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_sources_next_sync ON data_sources (next_sync_at) WHERE sync_frequency IS NOT NULL AND status = 'active' AND deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_sources_tenant_name_unique ON data_sources (tenant_id, name) WHERE deleted_at IS NULL;

ALTER TABLE data_models
    ADD COLUMN IF NOT EXISTS display_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS source_table TEXT,
    ADD COLUMN IF NOT EXISTS quality_rules JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS data_classification TEXT NOT NULL DEFAULT 'internal',
    ADD COLUMN IF NOT EXISTS contains_pii BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS pii_columns TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS field_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS previous_version_id UUID REFERENCES data_models(id),
    ADD COLUMN IF NOT EXISTS tags TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE data_models
    ALTER COLUMN status TYPE TEXT USING (
        CASE status::text
            WHEN 'draft' THEN 'draft'
            WHEN 'active' THEN 'active'
            WHEN 'deprecated' THEN 'deprecated'
            ELSE 'archived'
        END
    );

ALTER TABLE data_models
    DROP CONSTRAINT IF EXISTS data_models_status_check;

ALTER TABLE data_models
    ADD CONSTRAINT data_models_status_check CHECK (status IN ('draft', 'active', 'deprecated', 'archived')),
    ADD CONSTRAINT data_models_classification_check CHECK (data_classification IN ('public', 'internal', 'confidential', 'restricted'));

DROP INDEX IF EXISTS idx_models_tenant_status;
DROP INDEX IF EXISTS idx_models_source;
DROP INDEX IF EXISTS idx_models_tenant_created;
DROP INDEX IF EXISTS idx_models_schema;
DROP INDEX IF EXISTS idx_models_lineage;

CREATE INDEX IF NOT EXISTS idx_models_tenant ON data_models (tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_models_source ON data_models (source_id) WHERE source_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_models_pii ON data_models (tenant_id) WHERE contains_pii = true AND deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_models_tenant_name_version_unique ON data_models (tenant_id, name, version) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS sync_history (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    source_id           UUID NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    status              TEXT NOT NULL DEFAULT 'running'
                        CHECK (status IN ('running', 'success', 'partial', 'failed', 'cancelled')),
    sync_type           TEXT NOT NULL CHECK (sync_type IN ('full', 'incremental', 'schema_only')),
    tables_synced       INT NOT NULL DEFAULT 0,
    rows_read           BIGINT NOT NULL DEFAULT 0,
    rows_written        BIGINT NOT NULL DEFAULT 0,
    bytes_transferred   BIGINT NOT NULL DEFAULT 0,
    errors              JSONB NOT NULL DEFAULT '[]',
    error_count         INT NOT NULL DEFAULT 0,
    started_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at        TIMESTAMPTZ,
    duration_ms         BIGINT,
    triggered_by        TEXT NOT NULL DEFAULT 'manual'
                        CHECK (triggered_by IN ('manual', 'schedule', 'event', 'api')),
    triggered_by_user   UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sync_history_source ON sync_history (source_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_sync_history_tenant ON sync_history (tenant_id, started_at DESC);
