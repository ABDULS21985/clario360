-- Enables Row-Level Security on all tenant-scoped tables.
-- The application must SET LOCAL app.current_tenant_id = '<uuid>' within each transaction.
-- The database role used for migrations must have BYPASSRLS privilege.
-- Use: ALTER ROLE migrator_role BYPASSRLS;

-- =============================================================================
-- TABLE: audit_logs — PARTITIONED, IMMUTABLE
-- RLS is applied to the parent table and automatically inherited by all partitions.
--
-- IMPORTANT: Only SELECT and INSERT policies are created here.
-- The audit_logs table is immutable by design — a database trigger (prevent_audit_mutation)
-- raises an exception on any UPDATE or DELETE attempt. Adding UPDATE/DELETE RLS policies
-- would be misleading; the immutability trigger is the authoritative enforcement mechanism.
-- RLS tenant_isolation ensures each tenant can only read their own entries.
-- =============================================================================

ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs FORCE ROW LEVEL SECURITY;

-- SELECT: each tenant sees only their own audit log entries.
CREATE POLICY tenant_isolation ON audit_logs
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- INSERT: audit entries can only be written under the correct tenant context.
CREATE POLICY tenant_insert ON audit_logs
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- Note: No tenant_update or tenant_delete policies — audit_logs is immutable.
-- The prevent_audit_mutation trigger enforces this at the database level.

-- =============================================================================
-- TABLE: audit_chain_state
-- The tenant_id column IS the primary key (UUID NOT NULL). All 4 policies apply.
-- Each tenant can only read and manage their own chain state.
-- =============================================================================

ALTER TABLE audit_chain_state ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_chain_state FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON audit_chain_state
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON audit_chain_state
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON audit_chain_state
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON audit_chain_state
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
