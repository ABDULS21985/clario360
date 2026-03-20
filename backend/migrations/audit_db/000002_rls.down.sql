-- Removes Row-Level Security from all tenant-scoped tables in audit_db.

-- TABLE: audit_chain_state
ALTER TABLE audit_chain_state DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON audit_chain_state;
DROP POLICY IF EXISTS tenant_insert ON audit_chain_state;
DROP POLICY IF EXISTS tenant_update ON audit_chain_state;
DROP POLICY IF EXISTS tenant_delete ON audit_chain_state;

-- TABLE: audit_logs
-- Note: No tenant_update or tenant_delete policies were created (table is immutable).
ALTER TABLE audit_logs DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON audit_logs;
DROP POLICY IF EXISTS tenant_insert ON audit_logs;
DROP POLICY IF EXISTS tenant_update ON audit_logs;
DROP POLICY IF EXISTS tenant_delete ON audit_logs;
