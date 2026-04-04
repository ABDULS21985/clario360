-- Revert fixes for RLS policies
-- Migration: 000012_file_storage_fixes

DROP POLICY IF EXISTS tenant_isolation ON file_quarantine_log;
DROP POLICY IF EXISTS tenant_insert ON file_quarantine_log;
DROP POLICY IF EXISTS tenant_update ON file_quarantine_log;
DROP POLICY IF EXISTS tenant_delete ON file_quarantine_log;

ALTER TABLE file_quarantine_log DISABLE ROW LEVEL SECURITY;
ALTER TABLE file_quarantine_log DROP COLUMN IF EXISTS tenant_id;

-- Revert files policies
DROP POLICY IF EXISTS tenant_isolation ON files;
CREATE POLICY tenant_isolation ON files
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_insert ON files;
CREATE POLICY tenant_insert ON files
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_update ON files;
CREATE POLICY tenant_update ON files
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_delete ON files;
CREATE POLICY tenant_delete ON files
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- Revert file_access_log policies
DROP POLICY IF EXISTS tenant_isolation ON file_access_log;
CREATE POLICY tenant_isolation ON file_access_log
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_insert ON file_access_log;
CREATE POLICY tenant_insert ON file_access_log
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_update ON file_access_log;
CREATE POLICY tenant_update ON file_access_log
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_delete ON file_access_log;
CREATE POLICY tenant_delete ON file_access_log
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
