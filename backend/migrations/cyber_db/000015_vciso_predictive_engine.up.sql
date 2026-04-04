CREATE TABLE IF NOT EXISTS vciso_predictions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    prediction_type     TEXT NOT NULL CHECK (prediction_type IN (
        'alert_volume_forecast',
        'asset_risk_prediction',
        'vulnerability_exploit_prediction',
        'attack_technique_trend',
        'insider_threat_trajectory',
        'campaign_detection'
    )),
    model_version       TEXT NOT NULL,
    prediction_json     JSONB NOT NULL,
    confidence_score    DECIMAL(5,4) NOT NULL,
    confidence_interval JSONB NOT NULL,
    top_features        JSONB NOT NULL,
    explanation_text    TEXT NOT NULL,
    target_entity_type  TEXT,
    target_entity_id    TEXT,
    forecast_start      TIMESTAMPTZ NOT NULL,
    forecast_end        TIMESTAMPTZ NOT NULL,
    outcome_observed    BOOLEAN,
    outcome_value       JSONB,
    accuracy_score      DECIMAL(5,4),
    prediction_log_id   UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    evaluated_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_vciso_pred_tenant
    ON vciso_predictions (tenant_id, prediction_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_vciso_pred_accuracy
    ON vciso_predictions (prediction_type, accuracy_score)
    WHERE outcome_observed IS NOT NULL;

CREATE TABLE IF NOT EXISTS vciso_prediction_models (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_type                TEXT NOT NULL CHECK (model_type IN (
        'alert_volume_forecast',
        'asset_risk_prediction',
        'vulnerability_exploit_prediction',
        'attack_technique_trend',
        'insider_threat_trajectory',
        'campaign_detection'
    )),
    version                   TEXT NOT NULL,
    model_artifact_path       TEXT NOT NULL,
    model_framework           TEXT NOT NULL,
    backtest_accuracy         DECIMAL(5,4),
    backtest_precision        DECIMAL(5,4),
    backtest_recall           DECIMAL(5,4),
    backtest_f1               DECIMAL(5,4),
    backtest_mape             DECIMAL(8,4),
    feature_count             INT NOT NULL,
    training_samples          INT NOT NULL,
    training_duration_seconds INT NOT NULL,
    status                    TEXT NOT NULL DEFAULT 'training' CHECK (status IN ('training', 'validating', 'active', 'deprecated', 'failed')),
    active                    BOOLEAN NOT NULL DEFAULT false,
    last_drift_check          TIMESTAMPTZ,
    drift_score               DECIMAL(5,4),
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    activated_at              TIMESTAMPTZ,
    deprecated_at             TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_vciso_pred_model_active
    ON vciso_prediction_models (model_type, active)
    WHERE active = true;
CREATE UNIQUE INDEX IF NOT EXISTS idx_vciso_pred_model_version
    ON vciso_prediction_models (model_type, version);

CREATE TABLE IF NOT EXISTS vciso_feature_snapshots (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    feature_set TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id   TEXT,
    vector_json JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_vciso_feature_snapshots_tenant
    ON vciso_feature_snapshots (tenant_id, feature_set, created_at DESC);

ALTER TABLE vciso_predictions ENABLE ROW LEVEL SECURITY;
ALTER TABLE vciso_predictions FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON vciso_predictions;
DROP POLICY IF EXISTS tenant_insert ON vciso_predictions;
DROP POLICY IF EXISTS tenant_update ON vciso_predictions;
DROP POLICY IF EXISTS tenant_delete ON vciso_predictions;
CREATE POLICY tenant_isolation ON vciso_predictions
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON vciso_predictions
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON vciso_predictions
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON vciso_predictions
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE vciso_feature_snapshots ENABLE ROW LEVEL SECURITY;
ALTER TABLE vciso_feature_snapshots FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON vciso_feature_snapshots;
DROP POLICY IF EXISTS tenant_insert ON vciso_feature_snapshots;
DROP POLICY IF EXISTS tenant_update ON vciso_feature_snapshots;
DROP POLICY IF EXISTS tenant_delete ON vciso_feature_snapshots;
CREATE POLICY tenant_isolation ON vciso_feature_snapshots
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON vciso_feature_snapshots
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON vciso_feature_snapshots
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON vciso_feature_snapshots
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

