DROP POLICY IF EXISTS tenant_isolation ON ai_validation_results;
DROP POLICY IF EXISTS tenant_insert ON ai_validation_results;
DROP POLICY IF EXISTS tenant_update ON ai_validation_results;
DROP POLICY IF EXISTS tenant_delete ON ai_validation_results;
ALTER TABLE IF EXISTS ai_validation_results DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS ai_validation_results;
