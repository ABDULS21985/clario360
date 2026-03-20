ALTER TABLE ai_drift_reports DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON ai_drift_reports;
DROP POLICY IF EXISTS tenant_insert ON ai_drift_reports;
DROP POLICY IF EXISTS tenant_update ON ai_drift_reports;
DROP POLICY IF EXISTS tenant_delete ON ai_drift_reports;

ALTER TABLE ai_shadow_comparisons DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON ai_shadow_comparisons;
DROP POLICY IF EXISTS tenant_insert ON ai_shadow_comparisons;
DROP POLICY IF EXISTS tenant_update ON ai_shadow_comparisons;
DROP POLICY IF EXISTS tenant_delete ON ai_shadow_comparisons;

ALTER TABLE ai_prediction_logs DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON ai_prediction_logs;
DROP POLICY IF EXISTS tenant_insert ON ai_prediction_logs;
DROP POLICY IF EXISTS tenant_update ON ai_prediction_logs;
DROP POLICY IF EXISTS tenant_delete ON ai_prediction_logs;

ALTER TABLE ai_model_versions DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON ai_model_versions;
DROP POLICY IF EXISTS tenant_insert ON ai_model_versions;
DROP POLICY IF EXISTS tenant_update ON ai_model_versions;
DROP POLICY IF EXISTS tenant_delete ON ai_model_versions;

ALTER TABLE ai_models DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON ai_models;
DROP POLICY IF EXISTS tenant_insert ON ai_models;
DROP POLICY IF EXISTS tenant_update ON ai_models;
DROP POLICY IF EXISTS tenant_delete ON ai_models;
