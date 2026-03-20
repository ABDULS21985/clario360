-- Row-Level Security for file storage tables.
-- Ensures tenant isolation at the database level (defense-in-depth).

-- ═══════════════════════════════════════════════════════════════════════
-- files table
-- ═══════════════════════════════════════════════════════════════════════

ALTER TABLE files ENABLE ROW LEVEL SECURITY;
ALTER TABLE files FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON files
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON files
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON files
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON files
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ═══════════════════════════════════════════════════════════════════════
-- file_access_log table
-- ═══════════════════════════════════════════════════════════════════════

ALTER TABLE file_access_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE file_access_log FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON file_access_log
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON file_access_log
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON file_access_log
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON file_access_log
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
