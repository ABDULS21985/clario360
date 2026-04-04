CREATE TABLE IF NOT EXISTS vciso_llm_audit_log (
    id                 UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id         UUID            NOT NULL,
    conversation_id    UUID            NOT NULL,
    tenant_id          UUID            NOT NULL,
    user_id            UUID            NOT NULL,
    provider           TEXT            NOT NULL,
    model              TEXT            NOT NULL,
    prompt_tokens      INT             NOT NULL DEFAULT 0,
    completion_tokens  INT             NOT NULL DEFAULT 0,
    total_tokens       INT             NOT NULL DEFAULT 0,
    estimated_cost_usd DECIMAL(10,6)   NOT NULL DEFAULT 0,
    llm_latency_ms     INT             NOT NULL DEFAULT 0,
    total_latency_ms   INT             NOT NULL DEFAULT 0,
    system_prompt_hash TEXT            NOT NULL,
    system_prompt_version TEXT         NOT NULL DEFAULT 'v1.0',
    user_message       TEXT            NOT NULL,
    context_turns      INT             NOT NULL DEFAULT 0,
    raw_completion     TEXT            NOT NULL DEFAULT '',
    tool_calls_json    JSONB           NOT NULL DEFAULT '[]'::jsonb,
    tool_call_count    INT             NOT NULL DEFAULT 0,
    reasoning_trace    JSONB           NOT NULL DEFAULT '[]'::jsonb,
    grounding_result   TEXT            NOT NULL DEFAULT 'passed'
                                     CHECK (grounding_result IN ('passed', 'corrected', 'blocked')),
    pii_detections     INT             NOT NULL DEFAULT 0,
    injection_flags    INT             NOT NULL DEFAULT 0,
    final_response     TEXT            NOT NULL DEFAULT '',
    prediction_log_id  UUID,
    engine_used        TEXT            NOT NULL DEFAULT 'llm'
                                     CHECK (engine_used IN ('llm', 'rule_based', 'fallback')),
    routing_reason     TEXT,
    created_at         TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_vciso_llm_audit_conv
    ON vciso_llm_audit_log (conversation_id, created_at);
CREATE INDEX IF NOT EXISTS idx_vciso_llm_audit_tenant
    ON vciso_llm_audit_log (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_vciso_llm_audit_grounding
    ON vciso_llm_audit_log (grounding_result)
    WHERE grounding_result != 'passed';
CREATE INDEX IF NOT EXISTS idx_vciso_llm_audit_cost
    ON vciso_llm_audit_log (tenant_id, estimated_cost_usd);

CREATE TABLE IF NOT EXISTS vciso_llm_system_prompts (
    id               UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    version          TEXT            NOT NULL UNIQUE,
    prompt_text      TEXT            NOT NULL,
    prompt_hash      TEXT            NOT NULL,
    tool_schemas     JSONB           NOT NULL DEFAULT '[]'::jsonb,
    description      TEXT,
    created_by       TEXT            NOT NULL,
    active           BOOLEAN         NOT NULL DEFAULT false,
    created_at       TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_vciso_llm_prompts_active
    ON vciso_llm_system_prompts (active)
    WHERE active = true;

CREATE TABLE IF NOT EXISTS vciso_llm_rate_limits (
    id                    UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID            NOT NULL,
    max_calls_per_minute  INT             NOT NULL DEFAULT 10,
    max_calls_per_hour    INT             NOT NULL DEFAULT 100,
    max_calls_per_day     INT             NOT NULL DEFAULT 500,
    max_tokens_per_day    INT             NOT NULL DEFAULT 500000,
    max_cost_per_day_usd  DECIMAL(10,2)   NOT NULL DEFAULT 50.00,
    current_calls_minute  INT             NOT NULL DEFAULT 0,
    current_calls_hour    INT             NOT NULL DEFAULT 0,
    current_calls_day     INT             NOT NULL DEFAULT 0,
    current_tokens_day    INT             NOT NULL DEFAULT 0,
    current_cost_day_usd  DECIMAL(10,2)   NOT NULL DEFAULT 0.00,
    minute_reset_at       TIMESTAMPTZ     NOT NULL DEFAULT now(),
    hour_reset_at         TIMESTAMPTZ     NOT NULL DEFAULT now(),
    day_reset_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_vciso_llm_rate_tenant
    ON vciso_llm_rate_limits (tenant_id);

ALTER TABLE vciso_llm_audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE vciso_llm_audit_log FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON vciso_llm_audit_log;
DROP POLICY IF EXISTS tenant_insert ON vciso_llm_audit_log;
DROP POLICY IF EXISTS tenant_update ON vciso_llm_audit_log;
DROP POLICY IF EXISTS tenant_delete ON vciso_llm_audit_log;
CREATE POLICY tenant_isolation ON vciso_llm_audit_log
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON vciso_llm_audit_log
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON vciso_llm_audit_log
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON vciso_llm_audit_log
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE vciso_llm_rate_limits ENABLE ROW LEVEL SECURITY;
ALTER TABLE vciso_llm_rate_limits FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON vciso_llm_rate_limits;
DROP POLICY IF EXISTS tenant_insert ON vciso_llm_rate_limits;
DROP POLICY IF EXISTS tenant_update ON vciso_llm_rate_limits;
DROP POLICY IF EXISTS tenant_delete ON vciso_llm_rate_limits;
CREATE POLICY tenant_isolation ON vciso_llm_rate_limits
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON vciso_llm_rate_limits
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON vciso_llm_rate_limits
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON vciso_llm_rate_limits
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
