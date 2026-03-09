CREATE TABLE IF NOT EXISTS ai_validation_results (
    id                       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID         NOT NULL,
    model_id                 UUID         NOT NULL REFERENCES ai_models(id) ON DELETE CASCADE,
    version_id               UUID         NOT NULL REFERENCES ai_model_versions(id) ON DELETE CASCADE,
    dataset_type             TEXT         NOT NULL CHECK (dataset_type IN ('historical', 'custom', 'live_replay')),
    dataset_size             INT          NOT NULL,
    positive_count           INT          NOT NULL,
    negative_count           INT          NOT NULL,
    true_positives           INT          NOT NULL,
    false_positives          INT          NOT NULL,
    true_negatives           INT          NOT NULL,
    false_negatives          INT          NOT NULL,
    precision                DECIMAL(8,6) NOT NULL,
    recall                   DECIMAL(8,6) NOT NULL,
    f1_score                 DECIMAL(8,6) NOT NULL,
    false_positive_rate      DECIMAL(8,6) NOT NULL,
    accuracy                 DECIMAL(8,6) NOT NULL,
    auc                      DECIMAL(8,6) NOT NULL,
    roc_curve                JSONB        NOT NULL DEFAULT '[]'::jsonb,
    production_metrics       JSONB,
    deltas                   JSONB,
    by_severity              JSONB        NOT NULL DEFAULT '{}'::jsonb,
    by_rule_type             JSONB,
    false_positive_samples   JSONB        NOT NULL DEFAULT '[]'::jsonb,
    false_negative_samples   JSONB        NOT NULL DEFAULT '[]'::jsonb,
    recommendation           TEXT         NOT NULL CHECK (recommendation IN ('promote', 'keep_testing', 'reject')),
    recommendation_reason    TEXT         NOT NULL,
    warnings                 JSONB        NOT NULL DEFAULT '[]'::jsonb,
    validated_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    duration_ms              INT          NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_validation_results_version
    ON ai_validation_results (tenant_id, version_id, validated_at DESC);

CREATE INDEX IF NOT EXISTS idx_validation_results_model
    ON ai_validation_results (tenant_id, model_id, validated_at DESC);

ALTER TABLE ai_validation_results ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_validation_results FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON ai_validation_results
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON ai_validation_results
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON ai_validation_results
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON ai_validation_results
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
