-- =============================================================================
-- DSPM Advanced Data Intelligence: Lineage, AI Usage, Classification History,
-- Compliance Posture, Financial Impact
-- =============================================================================

-- =============================================================================
-- DSPM DATA LINEAGE — End-to-end data flow tracking
-- =============================================================================

CREATE TABLE IF NOT EXISTS dspm_data_lineage (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    -- Source
    source_asset_id     UUID            NOT NULL REFERENCES dspm_data_assets(id),
    source_asset_name   TEXT,
    source_table        TEXT,
    -- Destination
    target_asset_id     UUID            NOT NULL REFERENCES dspm_data_assets(id),
    target_asset_name   TEXT,
    target_table        TEXT,
    -- Lineage details
    edge_type           TEXT            NOT NULL CHECK (edge_type IN (
        'etl_pipeline', 'replication', 'api_transfer', 'manual_copy',
        'query_derived', 'stream', 'export', 'inferred'
    )),
    transformation      TEXT,
    pipeline_id         TEXT,
    pipeline_name       TEXT,
    -- Classification flow
    source_classification TEXT,
    target_classification TEXT,
    classification_changed BOOLEAN      NOT NULL DEFAULT false,
    pii_types_transferred TEXT[]        NOT NULL DEFAULT '{}',
    -- Confidence (for inferred lineage)
    confidence          FLOAT           NOT NULL DEFAULT 1.0,
    evidence            JSONB           NOT NULL DEFAULT '{}',
    -- Status
    status              TEXT            NOT NULL DEFAULT 'active'
                                        CHECK (status IN ('active', 'inactive', 'broken', 'deprecated')),
    last_transfer_at    TIMESTAMPTZ,
    transfer_count_30d  INT             NOT NULL DEFAULT 0,
    -- Metadata
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),

    UNIQUE (tenant_id, source_asset_id, target_asset_id, COALESCE(source_table, ''), COALESCE(target_table, ''), edge_type)
);

CREATE INDEX idx_dspm_lineage_source    ON dspm_data_lineage (tenant_id, source_asset_id);
CREATE INDEX idx_dspm_lineage_target    ON dspm_data_lineage (tenant_id, target_asset_id);
CREATE INDEX idx_dspm_lineage_pii       ON dspm_data_lineage (tenant_id) WHERE array_length(pii_types_transferred, 1) > 0;

-- =============================================================================
-- DSPM AI DATA USAGE — Track data assets used for AI/ML
-- =============================================================================

CREATE TABLE IF NOT EXISTS dspm_ai_data_usage (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    -- Data asset
    data_asset_id       UUID            NOT NULL REFERENCES dspm_data_assets(id),
    data_asset_name     TEXT,
    data_classification TEXT,
    contains_pii        BOOLEAN         NOT NULL DEFAULT false,
    pii_types           TEXT[]          NOT NULL DEFAULT '{}',
    -- AI/ML usage
    usage_type          TEXT            NOT NULL CHECK (usage_type IN (
        'training_data', 'evaluation_data', 'inference_input',
        'rag_knowledge_base', 'prompt_context', 'feature_store', 'embedding_source'
    )),
    -- Model/Pipeline reference
    model_id            UUID,
    model_name          TEXT,
    model_slug          TEXT,
    pipeline_id         TEXT,
    pipeline_name       TEXT,
    -- Risk assessment
    ai_risk_score       FLOAT           NOT NULL DEFAULT 0.0,
    ai_risk_level       TEXT            NOT NULL DEFAULT 'low'
                                        CHECK (ai_risk_level IN ('low', 'medium', 'high', 'critical')),
    risk_factors        JSONB           NOT NULL DEFAULT '[]',
    -- Governance
    consent_verified    BOOLEAN         NOT NULL DEFAULT false,
    data_minimization   BOOLEAN         NOT NULL DEFAULT false,
    anonymization_level TEXT            CHECK (anonymization_level IN (
        'none', 'pseudonymized', 'anonymized', 'differential_privacy'
    )),
    retention_compliant BOOLEAN         NOT NULL DEFAULT true,
    -- Status
    status              TEXT            NOT NULL DEFAULT 'active'
                                        CHECK (status IN ('active', 'inactive', 'blocked', 'under_review')),
    first_detected_at   TIMESTAMPTZ     NOT NULL DEFAULT now(),
    last_detected_at    TIMESTAMPTZ     NOT NULL DEFAULT now(),
    -- Metadata
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),

    UNIQUE (tenant_id, data_asset_id, usage_type, COALESCE(model_slug, ''))
);

CREATE INDEX idx_dspm_ai_usage_asset    ON dspm_ai_data_usage (tenant_id, data_asset_id);
CREATE INDEX idx_dspm_ai_usage_model    ON dspm_ai_data_usage (tenant_id, model_slug);
CREATE INDEX idx_dspm_ai_usage_risk     ON dspm_ai_data_usage (tenant_id, ai_risk_score DESC) WHERE status = 'active';

-- =============================================================================
-- DSPM CLASSIFICATION HISTORY — Track classification changes over time
-- =============================================================================

CREATE TABLE IF NOT EXISTS dspm_classification_history (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    data_asset_id       UUID            NOT NULL REFERENCES dspm_data_assets(id),
    -- Change
    old_classification  TEXT,
    new_classification  TEXT            NOT NULL,
    old_pii_types       TEXT[]          NOT NULL DEFAULT '{}',
    new_pii_types       TEXT[]          NOT NULL DEFAULT '{}',
    change_type         TEXT            NOT NULL CHECK (change_type IN (
        'initial', 'escalation', 'deescalation',
        'pii_added', 'pii_removed', 'reclassification'
    )),
    -- Source
    detected_by         TEXT            NOT NULL,
    confidence          FLOAT           NOT NULL DEFAULT 1.0,
    evidence            JSONB           NOT NULL DEFAULT '{}',
    -- Actor
    actor_id            UUID,
    actor_type          TEXT            NOT NULL DEFAULT 'system',
    -- Timestamp
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX idx_dspm_class_history_asset ON dspm_classification_history (tenant_id, data_asset_id, created_at DESC);

-- =============================================================================
-- DSPM COMPLIANCE POSTURE — Per-framework compliance scoring
-- =============================================================================

CREATE TABLE IF NOT EXISTS dspm_compliance_posture (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    -- Framework
    framework           TEXT            NOT NULL CHECK (framework IN (
        'gdpr', 'hipaa', 'soc2', 'pci_dss', 'saudi_pdpl', 'iso27001'
    )),
    -- Scores
    overall_score       FLOAT           NOT NULL DEFAULT 0.0,
    -- Per-control breakdown
    controls_total      INT             NOT NULL DEFAULT 0,
    controls_compliant  INT             NOT NULL DEFAULT 0,
    controls_partial    INT             NOT NULL DEFAULT 0,
    controls_non_compliant INT          NOT NULL DEFAULT 0,
    controls_not_applicable INT         NOT NULL DEFAULT 0,
    -- Detail
    control_details     JSONB           NOT NULL DEFAULT '[]',
    -- Trend
    score_7d_ago        FLOAT,
    score_30d_ago       FLOAT,
    score_90d_ago       FLOAT,
    trend_direction     TEXT            CHECK (trend_direction IN ('improving', 'stable', 'declining')),
    -- Financial exposure
    estimated_fine_exposure FLOAT       NOT NULL DEFAULT 0.0,
    fine_currency       TEXT            NOT NULL DEFAULT 'USD',
    -- Snapshot timestamp
    evaluated_at        TIMESTAMPTZ     NOT NULL DEFAULT now(),
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),

    UNIQUE (tenant_id, framework)
);

CREATE INDEX idx_dspm_compliance_tenant  ON dspm_compliance_posture (tenant_id, overall_score);

-- =============================================================================
-- DSPM FINANCIAL IMPACT — Per-asset financial risk quantification
-- =============================================================================

CREATE TABLE IF NOT EXISTS dspm_financial_impact (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    data_asset_id       UUID            NOT NULL REFERENCES dspm_data_assets(id),
    -- Impact estimates
    estimated_breach_cost   FLOAT       NOT NULL DEFAULT 0.0,
    cost_per_record         FLOAT       NOT NULL DEFAULT 0.0,
    record_count            BIGINT      NOT NULL DEFAULT 0,
    -- Cost breakdown
    cost_breakdown          JSONB       NOT NULL DEFAULT '{}',
    -- Methodology
    methodology             TEXT        NOT NULL DEFAULT 'ibm_ponemon',
    methodology_details     JSONB       NOT NULL DEFAULT '{}',
    -- Applicable regulations
    applicable_regulations  TEXT[]      NOT NULL DEFAULT '{}',
    max_regulatory_fine     FLOAT       NOT NULL DEFAULT 0.0,
    -- Risk context
    breach_probability_annual FLOAT     NOT NULL DEFAULT 0.0,
    annual_expected_loss    FLOAT       NOT NULL DEFAULT 0.0,
    -- Timestamp
    calculated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (tenant_id, data_asset_id)
);

CREATE INDEX idx_dspm_financial_tenant   ON dspm_financial_impact (tenant_id, estimated_breach_cost DESC);
CREATE INDEX idx_dspm_financial_risk     ON dspm_financial_impact (tenant_id, annual_expected_loss DESC);
