-- Fix incomplete RLS on compute benchmark tables.
-- Adds FORCE ROW LEVEL SECURITY and per-operation policies (INSERT/UPDATE/DELETE).
-- The original migration (000010) only had ENABLE + a single USING policy (SELECT-only).

-- ═══════════════════════════════════════════════════════════════════════
-- ai_inference_servers
-- ═══════════════════════════════════════════════════════════════════════

ALTER TABLE ai_inference_servers FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_inference_servers_tenant_insert ON ai_inference_servers
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY ai_inference_servers_tenant_update ON ai_inference_servers
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY ai_inference_servers_tenant_delete ON ai_inference_servers
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ═══════════════════════════════════════════════════════════════════════
-- ai_benchmark_suites
-- ═══════════════════════════════════════════════════════════════════════

ALTER TABLE ai_benchmark_suites FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_benchmark_suites_tenant_insert ON ai_benchmark_suites
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY ai_benchmark_suites_tenant_update ON ai_benchmark_suites
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY ai_benchmark_suites_tenant_delete ON ai_benchmark_suites
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ═══════════════════════════════════════════════════════════════════════
-- ai_benchmark_runs
-- ═══════════════════════════════════════════════════════════════════════

ALTER TABLE ai_benchmark_runs FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_benchmark_runs_tenant_insert ON ai_benchmark_runs
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY ai_benchmark_runs_tenant_update ON ai_benchmark_runs
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY ai_benchmark_runs_tenant_delete ON ai_benchmark_runs
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ═══════════════════════════════════════════════════════════════════════
-- ai_compute_cost_models
-- ═══════════════════════════════════════════════════════════════════════

ALTER TABLE ai_compute_cost_models FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_compute_cost_models_tenant_insert ON ai_compute_cost_models
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY ai_compute_cost_models_tenant_update ON ai_compute_cost_models
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY ai_compute_cost_models_tenant_delete ON ai_compute_cost_models
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
