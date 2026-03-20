CREATE TABLE IF NOT EXISTS vciso_conversations (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID            NOT NULL,
    user_id         UUID            NOT NULL,
    title           TEXT,
    status          TEXT            NOT NULL DEFAULT 'active'
                                    CHECK (status IN ('active', 'archived', 'deleted')),
    message_count   INT             NOT NULL DEFAULT 0,
    last_context    JSONB           NOT NULL DEFAULT '{}',
    last_message_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_vciso_conv_user
    ON vciso_conversations (tenant_id, user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_vciso_conv_active
    ON vciso_conversations (tenant_id, status)
    WHERE status = 'active';

CREATE TABLE IF NOT EXISTS vciso_messages (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id     UUID            NOT NULL REFERENCES vciso_conversations(id) ON DELETE CASCADE,
    tenant_id           UUID            NOT NULL,
    role                TEXT            NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content             TEXT            NOT NULL,
    intent              TEXT,
    intent_confidence   DECIMAL(5,4),
    match_method        TEXT,
    matched_pattern     TEXT,
    extracted_entities  JSONB           NOT NULL DEFAULT '{}',
    tool_name           TEXT,
    tool_params         JSONB,
    tool_result         JSONB,
    tool_latency_ms     INT,
    tool_error          TEXT,
    response_type       TEXT,
    suggested_actions   JSONB           NOT NULL DEFAULT '[]',
    entity_references   JSONB           NOT NULL DEFAULT '[]',
    prediction_log_id   UUID,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_vciso_msg_conv
    ON vciso_messages (conversation_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_vciso_msg_tenant
    ON vciso_messages (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_vciso_msg_intent
    ON vciso_messages (tenant_id, intent, created_at DESC)
    WHERE intent IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_vciso_msg_tool
    ON vciso_messages (tenant_id, tool_name)
    WHERE tool_name IS NOT NULL;

ALTER TABLE vciso_conversations ENABLE ROW LEVEL SECURITY;
ALTER TABLE vciso_conversations FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON vciso_conversations;
DROP POLICY IF EXISTS tenant_insert ON vciso_conversations;
DROP POLICY IF EXISTS tenant_update ON vciso_conversations;
DROP POLICY IF EXISTS tenant_delete ON vciso_conversations;
CREATE POLICY tenant_isolation ON vciso_conversations
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON vciso_conversations
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON vciso_conversations
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON vciso_conversations
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE vciso_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE vciso_messages FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON vciso_messages;
DROP POLICY IF EXISTS tenant_insert ON vciso_messages;
DROP POLICY IF EXISTS tenant_update ON vciso_messages;
DROP POLICY IF EXISTS tenant_delete ON vciso_messages;
CREATE POLICY tenant_isolation ON vciso_messages
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON vciso_messages
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON vciso_messages
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON vciso_messages
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
