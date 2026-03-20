-- Rollback compute benchmark RLS fixes (keep original ENABLE + tenant_isolation policy).

DROP POLICY IF EXISTS ai_compute_cost_models_tenant_delete ON ai_compute_cost_models;
DROP POLICY IF EXISTS ai_compute_cost_models_tenant_update ON ai_compute_cost_models;
DROP POLICY IF EXISTS ai_compute_cost_models_tenant_insert ON ai_compute_cost_models;
ALTER TABLE ai_compute_cost_models NO FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS ai_benchmark_runs_tenant_delete ON ai_benchmark_runs;
DROP POLICY IF EXISTS ai_benchmark_runs_tenant_update ON ai_benchmark_runs;
DROP POLICY IF EXISTS ai_benchmark_runs_tenant_insert ON ai_benchmark_runs;
ALTER TABLE ai_benchmark_runs NO FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS ai_benchmark_suites_tenant_delete ON ai_benchmark_suites;
DROP POLICY IF EXISTS ai_benchmark_suites_tenant_update ON ai_benchmark_suites;
DROP POLICY IF EXISTS ai_benchmark_suites_tenant_insert ON ai_benchmark_suites;
ALTER TABLE ai_benchmark_suites NO FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS ai_inference_servers_tenant_delete ON ai_inference_servers;
DROP POLICY IF EXISTS ai_inference_servers_tenant_update ON ai_inference_servers;
DROP POLICY IF EXISTS ai_inference_servers_tenant_insert ON ai_inference_servers;
ALTER TABLE ai_inference_servers NO FORCE ROW LEVEL SECURITY;
