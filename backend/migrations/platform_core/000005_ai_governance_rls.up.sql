ALTER TABLE ai_models ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_models FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON ai_models
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON ai_models
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON ai_models
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON ai_models
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE ai_model_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_model_versions FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON ai_model_versions
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON ai_model_versions
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON ai_model_versions
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON ai_model_versions
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE ai_prediction_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_prediction_logs FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON ai_prediction_logs
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON ai_prediction_logs
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON ai_prediction_logs
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON ai_prediction_logs
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE ai_shadow_comparisons ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_shadow_comparisons FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON ai_shadow_comparisons
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON ai_shadow_comparisons
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON ai_shadow_comparisons
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON ai_shadow_comparisons
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE ai_drift_reports ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_drift_reports FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON ai_drift_reports
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON ai_drift_reports
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON ai_drift_reports
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON ai_drift_reports
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
