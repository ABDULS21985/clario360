-- =============================================================================
-- IMMUTABLE AUDIT LOG — Partitioned by month on created_at
-- =============================================================================

CREATE TABLE IF NOT EXISTS audit_logs (
    id              UUID            NOT NULL DEFAULT gen_random_uuid(),
    tenant_id       UUID            NOT NULL,
    user_id         UUID,
    user_email      TEXT            NOT NULL DEFAULT '',
    service         TEXT            NOT NULL,
    action          TEXT            NOT NULL,
    severity        TEXT            NOT NULL CHECK (severity IN ('info', 'warning', 'high', 'critical')),
    resource_type   TEXT            NOT NULL,
    resource_id     TEXT            NOT NULL DEFAULT '',
    old_value       JSONB,
    new_value       JSONB,
    ip_address      TEXT            NOT NULL DEFAULT '',
    user_agent      TEXT            NOT NULL DEFAULT '',
    metadata        JSONB           NOT NULL DEFAULT '{}',
    event_id        TEXT            NOT NULL,
    correlation_id  TEXT            NOT NULL DEFAULT '',
    previous_hash   TEXT            NOT NULL,
    entry_hash      TEXT            NOT NULL,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),

    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- =============================================================================
-- IMMUTABILITY ENFORCEMENT
-- =============================================================================

-- Prevent UPDATE and DELETE on audit_logs via trigger
CREATE OR REPLACE FUNCTION prevent_audit_mutation() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs is immutable: % operations are prohibited', TG_OP;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_immutability_guard
    BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_mutation();

-- =============================================================================
-- INDEXES (applied to each partition automatically)
-- =============================================================================

CREATE INDEX IF NOT EXISTS idx_audit_tenant_created
    ON audit_logs (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_tenant_action
    ON audit_logs (tenant_id, action);

CREATE INDEX IF NOT EXISTS idx_audit_tenant_resource
    ON audit_logs (tenant_id, resource_type, resource_id);

CREATE INDEX IF NOT EXISTS idx_audit_tenant_user
    ON audit_logs (tenant_id, user_id) WHERE user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_audit_tenant_severity
    ON audit_logs (tenant_id, severity, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_event_id
    ON audit_logs (event_id, created_at);

-- Full-text search index (GIN on tsvector)
CREATE INDEX IF NOT EXISTS idx_audit_fts
    ON audit_logs USING GIN (
        to_tsvector('english', coalesce(action,'') || ' ' || coalesce(resource_type,'') || ' ' || coalesce(user_email,''))
    );

-- Deduplication: unique constraint on event_id per partition
CREATE UNIQUE INDEX IF NOT EXISTS idx_audit_event_unique
    ON audit_logs (event_id, created_at);

-- =============================================================================
-- HASH CHAIN STATE TRACKING
-- =============================================================================

CREATE TABLE IF NOT EXISTS audit_chain_state (
    tenant_id       UUID        PRIMARY KEY,
    last_entry_id   UUID        NOT NULL,
    last_hash       TEXT        NOT NULL,
    last_created_at TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
