-- =============================================================================
-- Clario 360 — Data Suite Database Schema
-- Database: data_db
-- Contains: data sources, models, quality rules, contradictions, pipelines,
--           lineage, dark data, data catalog
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- ENUM TYPES
-- =============================================================================

CREATE TYPE data_source_type AS ENUM ('database', 'api', 'file', 'stream', 'cloud_storage');
COMMENT ON TYPE data_source_type IS 'Types of external data sources';

CREATE TYPE data_source_status AS ENUM ('active', 'inactive', 'error', 'syncing');
COMMENT ON TYPE data_source_status IS 'Connection status of a data source';

CREATE TYPE data_model_status AS ENUM ('draft', 'active', 'deprecated');
COMMENT ON TYPE data_model_status IS 'Lifecycle status of a data model';

CREATE TYPE quality_rule_type AS ENUM (
    'not_null', 'unique', 'range', 'regex', 'referential', 'custom'
);
COMMENT ON TYPE quality_rule_type IS 'Types of data quality validation rules';

CREATE TYPE quality_severity AS ENUM ('critical', 'high', 'medium', 'low');
COMMENT ON TYPE quality_severity IS 'Severity of data quality rule violations';

CREATE TYPE quality_result_status AS ENUM ('passed', 'failed', 'warning');
COMMENT ON TYPE quality_result_status IS 'Outcome of a data quality check';

CREATE TYPE contradiction_type AS ENUM ('logical', 'semantic', 'analytical', 'temporal');
COMMENT ON TYPE contradiction_type IS 'Types of data contradictions detected';

CREATE TYPE contradiction_status AS ENUM ('detected', 'investigating', 'resolved', 'accepted');
COMMENT ON TYPE contradiction_status IS 'Lifecycle status of a detected contradiction';

CREATE TYPE pipeline_type AS ENUM ('etl', 'elt', 'streaming', 'batch');
COMMENT ON TYPE pipeline_type IS 'Types of data pipelines';

CREATE TYPE pipeline_status AS ENUM ('active', 'paused', 'failed', 'completed');
COMMENT ON TYPE pipeline_status IS 'Current status of a pipeline';

CREATE TYPE pipeline_run_status AS ENUM ('running', 'completed', 'failed', 'cancelled');
COMMENT ON TYPE pipeline_run_status IS 'Status of an individual pipeline execution';

CREATE TYPE governance_status AS ENUM ('unmanaged', 'under_review', 'governed', 'archived');
COMMENT ON TYPE governance_status IS 'Governance lifecycle status for dark data assets';

-- =============================================================================
-- TRIGGER FUNCTION
-- =============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- TABLE: data_sources
-- =============================================================================

CREATE TABLE data_sources (
    id               UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID             NOT NULL,
    name             VARCHAR(255)     NOT NULL,
    type             data_source_type NOT NULL,
    connection_config JSONB           NOT NULL DEFAULT '{}',
    status           data_source_status NOT NULL DEFAULT 'inactive',
    schema_metadata  JSONB            DEFAULT '{}',
    last_synced_at   TIMESTAMPTZ,
    sync_frequency   VARCHAR(50),
    created_by       UUID,
    created_at       TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_by       UUID
);

COMMENT ON TABLE data_sources IS 'External data source connections managed by the Data suite';
COMMENT ON COLUMN data_sources.connection_config IS 'Encrypted connection configuration (credentials, URLs, etc.)';
COMMENT ON COLUMN data_sources.schema_metadata IS 'Discovered schema information from the source';
COMMENT ON COLUMN data_sources.sync_frequency IS 'How often to sync (cron expression or interval)';

CREATE INDEX idx_sources_tenant_status ON data_sources (tenant_id, status);
CREATE INDEX idx_sources_tenant_type ON data_sources (tenant_id, type);
CREATE INDEX idx_sources_tenant_created ON data_sources (tenant_id, created_at DESC);
CREATE INDEX idx_sources_schema ON data_sources USING GIN (schema_metadata);

CREATE TRIGGER trg_data_sources_updated_at
    BEFORE UPDATE ON data_sources
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: data_models
-- =============================================================================

CREATE TABLE data_models (
    id                UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID              NOT NULL,
    name              VARCHAR(255)      NOT NULL,
    description       TEXT              NOT NULL DEFAULT '',
    version           INTEGER           NOT NULL DEFAULT 1,
    schema_definition JSONB             NOT NULL DEFAULT '{}',
    source_id         UUID              REFERENCES data_sources(id) ON DELETE SET NULL,
    status            data_model_status NOT NULL DEFAULT 'draft',
    lineage           JSONB             DEFAULT '{}',
    created_by        UUID,
    created_at        TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    updated_by        UUID
);

COMMENT ON TABLE data_models IS 'Logical data models describing structure and meaning of datasets';
COMMENT ON COLUMN data_models.schema_definition IS 'Full schema definition (columns, types, constraints)';
COMMENT ON COLUMN data_models.lineage IS 'Data lineage metadata (upstream/downstream references)';

CREATE INDEX idx_models_tenant_status ON data_models (tenant_id, status);
CREATE INDEX idx_models_source ON data_models (source_id);
CREATE INDEX idx_models_tenant_created ON data_models (tenant_id, created_at DESC);
CREATE INDEX idx_models_schema ON data_models USING GIN (schema_definition);
CREATE INDEX idx_models_lineage ON data_models USING GIN (lineage);

CREATE TRIGGER trg_data_models_updated_at
    BEFORE UPDATE ON data_models
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: data_quality_rules
-- =============================================================================

CREATE TABLE data_quality_rules (
    id               UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID             NOT NULL,
    model_id         UUID             NOT NULL REFERENCES data_models(id) ON DELETE CASCADE,
    column_name      VARCHAR(255)     NOT NULL,
    rule_type        quality_rule_type NOT NULL,
    rule_config      JSONB            NOT NULL DEFAULT '{}',
    severity         quality_severity NOT NULL DEFAULT 'medium',
    enabled          BOOLEAN          NOT NULL DEFAULT true,
    last_check_at    TIMESTAMPTZ,
    last_check_result quality_result_status,
    created_at       TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    created_by       UUID,
    updated_by       UUID
);

COMMENT ON TABLE data_quality_rules IS 'Data quality validation rules applied to data model columns';
COMMENT ON COLUMN data_quality_rules.rule_config IS 'Rule configuration (thresholds, patterns, references)';

CREATE INDEX idx_dq_rules_tenant ON data_quality_rules (tenant_id);
CREATE INDEX idx_dq_rules_model ON data_quality_rules (model_id);
CREATE INDEX idx_dq_rules_enabled ON data_quality_rules (model_id, enabled) WHERE enabled = true;
CREATE INDEX idx_dq_rules_config ON data_quality_rules USING GIN (rule_config);

CREATE TRIGGER trg_dq_rules_updated_at
    BEFORE UPDATE ON data_quality_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: data_quality_results
-- =============================================================================

CREATE TABLE data_quality_results (
    id              UUID                  PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID                  NOT NULL,
    rule_id         UUID                  NOT NULL REFERENCES data_quality_rules(id) ON DELETE CASCADE,
    model_id        UUID                  NOT NULL REFERENCES data_models(id) ON DELETE CASCADE,
    status          quality_result_status NOT NULL,
    records_checked BIGINT                NOT NULL DEFAULT 0,
    records_failed  BIGINT                NOT NULL DEFAULT 0,
    failure_samples JSONB                 DEFAULT '[]',
    checked_at      TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ           NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE data_quality_results IS 'Results of data quality rule executions';
COMMENT ON COLUMN data_quality_results.failure_samples IS 'Sample of failing records for debugging';

CREATE INDEX idx_dq_results_tenant ON data_quality_results (tenant_id);
CREATE INDEX idx_dq_results_rule ON data_quality_results (rule_id, checked_at DESC);
CREATE INDEX idx_dq_results_model ON data_quality_results (model_id, checked_at DESC);
CREATE INDEX idx_dq_results_status ON data_quality_results (tenant_id, status);

-- =============================================================================
-- TABLE: contradictions
-- =============================================================================

CREATE TABLE contradictions (
    id                   UUID                 PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID                 NOT NULL,
    type                 contradiction_type   NOT NULL,
    source_a             JSONB                NOT NULL,
    source_b             JSONB                NOT NULL,
    description          TEXT                 NOT NULL DEFAULT '',
    severity             quality_severity     NOT NULL DEFAULT 'medium',
    confidence_score     DECIMAL(5,4)         CHECK (confidence_score >= 0.0 AND confidence_score <= 1.0),
    resolution_guidance  TEXT,
    status               contradiction_status NOT NULL DEFAULT 'detected',
    resolved_by          UUID,
    resolved_at          TIMESTAMPTZ,
    created_at           TIMESTAMPTZ          NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ          NOT NULL DEFAULT NOW(),
    created_by           UUID,
    updated_by           UUID
);

COMMENT ON TABLE contradictions IS 'Data contradictions detected between different data sources';
COMMENT ON COLUMN contradictions.source_a IS 'First conflicting data reference (source, field, value)';
COMMENT ON COLUMN contradictions.source_b IS 'Second conflicting data reference';
COMMENT ON COLUMN contradictions.resolution_guidance IS 'AI-generated guidance on resolving the contradiction';

CREATE INDEX idx_contradictions_tenant_status ON contradictions (tenant_id, status);
CREATE INDEX idx_contradictions_tenant_type ON contradictions (tenant_id, type);
CREATE INDEX idx_contradictions_tenant_created ON contradictions (tenant_id, created_at DESC);
CREATE INDEX idx_contradictions_source_a ON contradictions USING GIN (source_a);
CREATE INDEX idx_contradictions_source_b ON contradictions USING GIN (source_b);

CREATE TRIGGER trg_contradictions_updated_at
    BEFORE UPDATE ON contradictions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: pipelines
-- =============================================================================

CREATE TABLE pipelines (
    id          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID            NOT NULL,
    name        VARCHAR(255)    NOT NULL,
    description TEXT            NOT NULL DEFAULT '',
    type        pipeline_type   NOT NULL,
    source_id   UUID            REFERENCES data_sources(id) ON DELETE SET NULL,
    target_id   UUID            REFERENCES data_sources(id) ON DELETE SET NULL,
    schedule    TEXT,
    config      JSONB           NOT NULL DEFAULT '{}',
    status      pipeline_status NOT NULL DEFAULT 'active',
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    created_by  UUID,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_by  UUID
);

COMMENT ON TABLE pipelines IS 'Data pipeline definitions (ETL/ELT/streaming/batch)';
COMMENT ON COLUMN pipelines.schedule IS 'Cron expression for scheduled execution';
COMMENT ON COLUMN pipelines.config IS 'Pipeline configuration (transformations, mappings, etc.)';

CREATE INDEX idx_pipelines_tenant_status ON pipelines (tenant_id, status);
CREATE INDEX idx_pipelines_tenant_type ON pipelines (tenant_id, type);
CREATE INDEX idx_pipelines_next_run ON pipelines (next_run_at) WHERE status = 'active';
CREATE INDEX idx_pipelines_tenant_created ON pipelines (tenant_id, created_at DESC);
CREATE INDEX idx_pipelines_config ON pipelines USING GIN (config);

CREATE TRIGGER trg_pipelines_updated_at
    BEFORE UPDATE ON pipelines
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: pipeline_runs
-- =============================================================================

CREATE TABLE pipeline_runs (
    id                UUID                PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID                NOT NULL,
    pipeline_id       UUID                NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    status            pipeline_run_status NOT NULL DEFAULT 'running',
    started_at        TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    completed_at      TIMESTAMPTZ,
    records_processed BIGINT              NOT NULL DEFAULT 0,
    records_failed    BIGINT              NOT NULL DEFAULT 0,
    error_log         TEXT,
    metrics           JSONB               DEFAULT '{}',
    created_at        TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE pipeline_runs IS 'Individual pipeline execution records';
COMMENT ON COLUMN pipeline_runs.metrics IS 'Execution metrics (duration, throughput, etc.)';

CREATE INDEX idx_runs_tenant ON pipeline_runs (tenant_id);
CREATE INDEX idx_runs_pipeline ON pipeline_runs (pipeline_id, created_at DESC);
CREATE INDEX idx_runs_status ON pipeline_runs (tenant_id, status);
CREATE INDEX idx_runs_started ON pipeline_runs (started_at DESC);

-- =============================================================================
-- TABLE: data_lineage
-- =============================================================================

CREATE TABLE data_lineage (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL,
    source_type     VARCHAR(100) NOT NULL,
    source_id       UUID        NOT NULL,
    target_type     VARCHAR(100) NOT NULL,
    target_id       UUID        NOT NULL,
    transformation  TEXT,
    pipeline_id     UUID        REFERENCES pipelines(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID
);

COMMENT ON TABLE data_lineage IS 'Data lineage graph tracking data flow between entities';
COMMENT ON COLUMN data_lineage.source_type IS 'Type of source entity (e.g., data_source, data_model)';
COMMENT ON COLUMN data_lineage.target_type IS 'Type of target entity';
COMMENT ON COLUMN data_lineage.transformation IS 'Description of the transformation applied';

CREATE INDEX idx_lineage_tenant ON data_lineage (tenant_id);
CREATE INDEX idx_lineage_source ON data_lineage (source_type, source_id);
CREATE INDEX idx_lineage_target ON data_lineage (target_type, target_id);
CREATE INDEX idx_lineage_pipeline ON data_lineage (pipeline_id);

-- =============================================================================
-- TABLE: dark_data_assets
-- =============================================================================

CREATE TABLE dark_data_assets (
    id                UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID              NOT NULL,
    location          TEXT              NOT NULL,
    type              VARCHAR(100)      NOT NULL,
    size_bytes        BIGINT,
    last_accessed_at  TIMESTAMPTZ,
    classification    VARCHAR(50),
    risk_score        DECIMAL(5,4)      CHECK (risk_score >= 0.0 AND risk_score <= 1.0),
    owner             UUID,
    governance_status governance_status NOT NULL DEFAULT 'unmanaged',
    discovered_at     TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    created_at        TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    created_by        UUID,
    updated_by        UUID
);

COMMENT ON TABLE dark_data_assets IS 'Unmanaged or unknown data assets discovered during scanning';
COMMENT ON COLUMN dark_data_assets.location IS 'File path, URL, or storage location';
COMMENT ON COLUMN dark_data_assets.size_bytes IS 'Size of the data asset in bytes';
COMMENT ON COLUMN dark_data_assets.governance_status IS 'Current governance lifecycle stage';

CREATE INDEX idx_dark_data_tenant ON dark_data_assets (tenant_id);
CREATE INDEX idx_dark_data_governance ON dark_data_assets (tenant_id, governance_status);
CREATE INDEX idx_dark_data_risk ON dark_data_assets (tenant_id, risk_score DESC NULLS LAST);
CREATE INDEX idx_dark_data_tenant_created ON dark_data_assets (tenant_id, created_at DESC);

CREATE TRIGGER trg_dark_data_updated_at
    BEFORE UPDATE ON dark_data_assets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: data_catalogs
-- =============================================================================

CREATE TABLE data_catalogs (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL,
    name            VARCHAR(255) NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    schema_info     JSONB       NOT NULL DEFAULT '{}',
    owner           UUID,
    tags            TEXT[]      DEFAULT '{}',
    classification  VARCHAR(50),
    access_count    BIGINT      NOT NULL DEFAULT 0,
    last_accessed_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID,
    updated_by      UUID
);

COMMENT ON TABLE data_catalogs IS 'Enterprise data catalog entries for data discovery and governance';
COMMENT ON COLUMN data_catalogs.schema_info IS 'Schema structure of the cataloged data asset';
COMMENT ON COLUMN data_catalogs.tags IS 'Searchable tags for discovery';
COMMENT ON COLUMN data_catalogs.access_count IS 'Number of times this entry has been accessed';

CREATE INDEX idx_catalogs_tenant ON data_catalogs (tenant_id);
CREATE INDEX idx_catalogs_tags ON data_catalogs USING GIN (tags);
CREATE INDEX idx_catalogs_schema ON data_catalogs USING GIN (schema_info);
CREATE INDEX idx_catalogs_classification ON data_catalogs (tenant_id, classification);
CREATE INDEX idx_catalogs_access ON data_catalogs (tenant_id, access_count DESC);
CREATE INDEX idx_catalogs_tenant_created ON data_catalogs (tenant_id, created_at DESC);

CREATE TRIGGER trg_data_catalogs_updated_at
    BEFORE UPDATE ON data_catalogs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
