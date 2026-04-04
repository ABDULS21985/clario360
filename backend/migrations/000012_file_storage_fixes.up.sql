-- Fixes for RLS policies missing and missing tenant_id in quarantine table
-- Migration: 000012_file_storage_fixes

-- Add tenant_id to file_quarantine_log and backfill from files table
ALTER TABLE file_quarantine_log ADD COLUMN IF NOT EXISTS tenant_id UUID;

UPDATE file_quarantine_log q
SET tenant_id = f.tenant_id
FROM files f
WHERE q.file_id = f.id AND q.tenant_id IS NULL;

ALTER TABLE file_quarantine_log ALTER COLUMN tenant_id SET NOT NULL;

-- Enable RLS on file_quarantine_log
ALTER TABLE file_quarantine_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE file_quarantine_log FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON file_quarantine_log
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    );
CREATE POLICY tenant_insert ON file_quarantine_log
    FOR INSERT
    WITH CHECK (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    );
CREATE POLICY tenant_update ON file_quarantine_log
    FOR UPDATE
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    )
    WITH CHECK (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    );
CREATE POLICY tenant_delete ON file_quarantine_log
    FOR DELETE
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    );

-- Update existing policies to support bypass_rls for background workers
DROP POLICY IF EXISTS tenant_isolation ON files;
CREATE POLICY tenant_isolation ON files
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    );

DROP POLICY IF EXISTS tenant_insert ON files;
CREATE POLICY tenant_insert ON files
    FOR INSERT
    WITH CHECK (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    );

DROP POLICY IF EXISTS tenant_update ON files;
CREATE POLICY tenant_update ON files
    FOR UPDATE
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    )
    WITH CHECK (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    );

DROP POLICY IF EXISTS tenant_delete ON files;
CREATE POLICY tenant_delete ON files
    FOR DELETE
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    );

DROP POLICY IF EXISTS tenant_isolation ON file_access_log;
CREATE POLICY tenant_isolation ON file_access_log
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    );

DROP POLICY IF EXISTS tenant_insert ON file_access_log;
CREATE POLICY tenant_insert ON file_access_log
    FOR INSERT
    WITH CHECK (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    );

DROP POLICY IF EXISTS tenant_update ON file_access_log;
CREATE POLICY tenant_update ON file_access_log
    FOR UPDATE
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    )
    WITH CHECK (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    );

DROP POLICY IF EXISTS tenant_delete ON file_access_log;
CREATE POLICY tenant_delete ON file_access_log
    FOR DELETE
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    );
