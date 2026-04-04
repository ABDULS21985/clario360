-- Rollback RLS for file storage tables.

DROP POLICY IF EXISTS tenant_delete ON file_access_log;
DROP POLICY IF EXISTS tenant_update ON file_access_log;
DROP POLICY IF EXISTS tenant_insert ON file_access_log;
DROP POLICY IF EXISTS tenant_isolation ON file_access_log;
ALTER TABLE file_access_log DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_delete ON files;
DROP POLICY IF EXISTS tenant_update ON files;
DROP POLICY IF EXISTS tenant_insert ON files;
DROP POLICY IF EXISTS tenant_isolation ON files;
ALTER TABLE files DISABLE ROW LEVEL SECURITY;
