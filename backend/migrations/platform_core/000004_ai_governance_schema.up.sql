-- =============================================================================
-- AI GOVERNANCE CONTROL PLANE
-- =============================================================================

CREATE TABLE IF NOT EXISTS ai_models (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL,
    name            TEXT        NOT NULL,
    slug            TEXT        NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    model_type      TEXT        NOT NULL CHECK (model_type IN (
        'rule_based', 'statistical', 'ml_classifier', 'ml_regressor',
        'nlp_extractor', 'anomaly_detector', 'scorer', 'recommender'
    )),
    suite           TEXT        NOT NULL CHECK (suite IN ('cyber', 'data', 'acta', 'lex', 'visus', 'platform')),
    owner_user_id   UUID,
    owner_team      TEXT,
    risk_tier       TEXT        NOT NULL DEFAULT 'medium' CHECK (risk_tier IN ('low', 'medium', 'high', 'critical')),
    status          TEXT        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'deprecated', 'retired')),
    tags            TEXT[]      NOT NULL DEFAULT '{}',
    metadata        JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_by      UUID        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_models_tenant_slug_unique
    ON ai_models (tenant_id, slug)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ai_models_tenant
    ON ai_models (tenant_id, suite, status)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS ai_model_versions (
    id                          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                   UUID        NOT NULL,
    model_id                    UUID        NOT NULL REFERENCES ai_models(id) ON DELETE CASCADE,
    version_number              INT         NOT NULL,
    status                      TEXT        NOT NULL DEFAULT 'development' CHECK (status IN (
        'development', 'staging', 'shadow', 'production', 'retired', 'failed', 'rolled_back'
    )),
    description                 TEXT        NOT NULL DEFAULT '',
    artifact_type               TEXT        NOT NULL CHECK (artifact_type IN (
        'go_function', 'rule_set', 'statistical_config', 'template_config', 'serialized_model'
    )),
    artifact_config             JSONB       NOT NULL,
    artifact_hash               TEXT        NOT NULL,
    explainability_type         TEXT        NOT NULL CHECK (explainability_type IN (
        'rule_trace', 'feature_importance', 'statistical_deviation', 'template_based'
    )),
    explanation_template        TEXT,
    training_data_desc          TEXT,
    training_data_hash          TEXT,
    training_metrics            JSONB       NOT NULL DEFAULT '{}'::jsonb,
    prediction_count            BIGINT      NOT NULL DEFAULT 0,
    avg_latency_ms              DECIMAL(10,2),
    avg_confidence              DECIMAL(5,4),
    accuracy_metric             DECIMAL(5,4),
    false_positive_rate         DECIMAL(5,4),
    false_negative_rate         DECIMAL(5,4),
    feedback_count              INT         NOT NULL DEFAULT 0,
    promoted_to_staging_at      TIMESTAMPTZ,
    promoted_to_shadow_at       TIMESTAMPTZ,
    promoted_to_production_at   TIMESTAMPTZ,
    promoted_by                 UUID,
    retired_at                  TIMESTAMPTZ,
    retired_by                  UUID,
    retirement_reason           TEXT,
    rolled_back_at              TIMESTAMPTZ,
    rolled_back_by              UUID,
    rollback_reason             TEXT,
    replaced_version_id         UUID REFERENCES ai_model_versions(id),
    created_by                  UUID        NOT NULL,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (model_id, version_number)
);

CREATE INDEX IF NOT EXISTS idx_ai_versions_model
    ON ai_model_versions (model_id, status);

CREATE INDEX IF NOT EXISTS idx_ai_versions_production
    ON ai_model_versions (tenant_id, model_id)
    WHERE status = 'production';

CREATE INDEX IF NOT EXISTS idx_ai_versions_shadow
    ON ai_model_versions (tenant_id, model_id)
    WHERE status = 'shadow';

CREATE TABLE IF NOT EXISTS ai_prediction_logs (
    id                           UUID        NOT NULL DEFAULT gen_random_uuid(),
    tenant_id                    UUID        NOT NULL,
    model_id                     UUID        NOT NULL REFERENCES ai_models(id) ON DELETE CASCADE,
    model_version_id             UUID        NOT NULL REFERENCES ai_model_versions(id) ON DELETE CASCADE,
    input_hash                   TEXT        NOT NULL,
    input_summary                JSONB,
    prediction                   JSONB       NOT NULL,
    confidence                   DECIMAL(5,4),
    explanation_structured       JSONB       NOT NULL,
    explanation_text             TEXT        NOT NULL,
    explanation_factors          JSONB       NOT NULL DEFAULT '[]'::jsonb,
    suite                        TEXT        NOT NULL,
    use_case                     TEXT        NOT NULL,
    entity_type                  TEXT,
    entity_id                    UUID,
    is_shadow                    BOOLEAN     NOT NULL DEFAULT false,
    shadow_production_version_id UUID,
    shadow_divergence            JSONB,
    feedback_correct             BOOLEAN,
    feedback_by                  UUID,
    feedback_at                  TIMESTAMPTZ,
    feedback_notes               TEXT,
    feedback_corrected_output    JSONB,
    latency_ms                   INT         NOT NULL DEFAULT 0,
    created_at                   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

DO $$
DECLARE
    start_date DATE := date_trunc('month', CURRENT_DATE);
    partition_date DATE;
    partition_name TEXT;
BEGIN
    FOR i IN 0..3 LOOP
        partition_date := start_date + (i || ' months')::interval;
        partition_name := 'ai_prediction_logs_' || to_char(partition_date, 'YYYY_MM');
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS %I PARTITION OF ai_prediction_logs FOR VALUES FROM (%L) TO (%L)',
            partition_name,
            partition_date,
            partition_date + interval '1 month'
        );
    END LOOP;
END $$;

CREATE TABLE IF NOT EXISTS ai_prediction_logs_default
    PARTITION OF ai_prediction_logs DEFAULT;

CREATE INDEX IF NOT EXISTS idx_pred_logs_model
    ON ai_prediction_logs (tenant_id, model_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_pred_logs_entity
    ON ai_prediction_logs (tenant_id, entity_type, entity_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_pred_logs_shadow
    ON ai_prediction_logs (tenant_id, model_version_id, created_at DESC)
    WHERE is_shadow = true;

CREATE INDEX IF NOT EXISTS idx_pred_logs_feedback
    ON ai_prediction_logs (tenant_id, model_id, created_at DESC)
    WHERE feedback_correct IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_pred_logs_use_case
    ON ai_prediction_logs (tenant_id, suite, use_case, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_pred_logs_input_hash
    ON ai_prediction_logs (tenant_id, model_id, input_hash, created_at DESC);

CREATE TABLE IF NOT EXISTS ai_shadow_comparisons (
    id                      UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID        NOT NULL,
    model_id                UUID        NOT NULL REFERENCES ai_models(id) ON DELETE CASCADE,
    production_version_id   UUID        NOT NULL REFERENCES ai_model_versions(id) ON DELETE CASCADE,
    shadow_version_id       UUID        NOT NULL REFERENCES ai_model_versions(id) ON DELETE CASCADE,
    period_start            TIMESTAMPTZ NOT NULL,
    period_end              TIMESTAMPTZ NOT NULL,
    total_predictions       INT         NOT NULL,
    agreement_count         INT         NOT NULL,
    disagreement_count      INT         NOT NULL,
    agreement_rate          DECIMAL(5,4) NOT NULL,
    production_metrics      JSONB       NOT NULL,
    shadow_metrics          JSONB       NOT NULL,
    metrics_delta           JSONB       NOT NULL,
    divergence_samples      JSONB       NOT NULL DEFAULT '[]'::jsonb,
    divergence_by_use_case  JSONB       NOT NULL DEFAULT '{}'::jsonb,
    recommendation          TEXT        NOT NULL CHECK (recommendation IN ('promote', 'keep_shadow', 'reject', 'needs_review')),
    recommendation_reason   TEXT        NOT NULL,
    recommendation_factors  JSONB       NOT NULL DEFAULT '[]'::jsonb,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_shadow_comparisons_model
    ON ai_shadow_comparisons (model_id, created_at DESC);

CREATE TABLE IF NOT EXISTS ai_drift_reports (
    id                        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID        NOT NULL,
    model_id                  UUID        NOT NULL REFERENCES ai_models(id) ON DELETE CASCADE,
    model_version_id          UUID        NOT NULL REFERENCES ai_model_versions(id) ON DELETE CASCADE,
    period                    TEXT        NOT NULL,
    period_start              TIMESTAMPTZ NOT NULL,
    period_end                TIMESTAMPTZ NOT NULL,
    output_psi                DECIMAL(8,4),
    output_drift_level        TEXT CHECK (output_drift_level IN ('none', 'low', 'moderate', 'significant')),
    confidence_psi            DECIMAL(8,4),
    confidence_drift_level    TEXT CHECK (confidence_drift_level IN ('none', 'low', 'moderate', 'significant')),
    current_volume            BIGINT      NOT NULL,
    reference_volume          BIGINT      NOT NULL,
    volume_change_pct         DECIMAL(8,2),
    current_p95_latency_ms    DECIMAL(10,2),
    reference_p95_latency_ms  DECIMAL(10,2),
    latency_change_pct        DECIMAL(8,2),
    current_accuracy          DECIMAL(5,4),
    reference_accuracy        DECIMAL(5,4),
    accuracy_change           DECIMAL(5,4),
    alerts                    JSONB       NOT NULL DEFAULT '[]'::jsonb,
    alert_count               INT         NOT NULL DEFAULT 0,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_drift_reports_model
    ON ai_drift_reports (model_id, created_at DESC);
