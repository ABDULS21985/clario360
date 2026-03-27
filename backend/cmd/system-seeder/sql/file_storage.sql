CREATE TABLE IF NOT EXISTS files (
    id                    UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID            NOT NULL,
    bucket                TEXT            NOT NULL,
    storage_key           TEXT            NOT NULL,
    original_name         TEXT            NOT NULL,
    sanitized_name        TEXT            NOT NULL,
    content_type          TEXT            NOT NULL,
    detected_content_type TEXT,
    size_bytes            BIGINT          NOT NULL CHECK (size_bytes >= 0),
    checksum_sha256       TEXT            NOT NULL,
    encrypted             BOOLEAN         NOT NULL DEFAULT false,
    encryption_metadata   JSONB,
    virus_scan_status     TEXT            NOT NULL DEFAULT 'pending'
                                          CHECK (virus_scan_status IN ('pending','scanning','clean','infected','error','skipped')),
    virus_scan_result     TEXT,
    virus_scanned_at      TIMESTAMPTZ,
    uploaded_by           UUID            NOT NULL,
    suite                 TEXT            NOT NULL CHECK (suite IN ('cyber','data','acta','lex','visus','platform','models')),
    entity_type           TEXT,
    entity_id             UUID,
    tags                  TEXT[]          NOT NULL DEFAULT '{}',
    version_id            TEXT,
    version_number        INT             NOT NULL DEFAULT 1,
    is_public             BOOLEAN         NOT NULL DEFAULT false,
    lifecycle_policy      TEXT            NOT NULL DEFAULT 'standard'
                                          CHECK (lifecycle_policy IN ('standard','temporary','archive','audit_retention')),
    expires_at            TIMESTAMPTZ,
    created_at            TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ     NOT NULL DEFAULT now(),
    deleted_at            TIMESTAMPTZ,
    UNIQUE (bucket, storage_key)
);

CREATE INDEX IF NOT EXISTS idx_files_tenant_suite ON files (tenant_id, suite, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_files_entity ON files (entity_type, entity_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_files_tenant_uploader ON files (tenant_id, uploaded_by, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_files_virus_pending ON files (virus_scan_status) WHERE virus_scan_status IN ('pending','scanning');
CREATE INDEX IF NOT EXISTS idx_files_lifecycle_expiry ON files (expires_at) WHERE expires_at IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_files_tags ON files USING GIN (tags);
CREATE INDEX IF NOT EXISTS idx_files_checksum ON files (checksum_sha256);

CREATE TABLE IF NOT EXISTS file_access_log (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id         UUID            NOT NULL REFERENCES files(id),
    tenant_id       UUID            NOT NULL,
    user_id         UUID            NOT NULL,
    action          TEXT            NOT NULL CHECK (action IN ('upload','download','presigned_download','presigned_upload','view_metadata','delete')),
    ip_address      TEXT            NOT NULL DEFAULT '',
    user_agent      TEXT            NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_file_access_file ON file_access_log (file_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_file_access_user ON file_access_log (tenant_id, user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS file_quarantine_log (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id             UUID            NOT NULL REFERENCES files(id),
    original_bucket     TEXT            NOT NULL,
    original_key        TEXT            NOT NULL,
    quarantine_bucket   TEXT            NOT NULL,
    quarantine_key      TEXT            NOT NULL,
    virus_name          TEXT            NOT NULL,
    scanned_at          TIMESTAMPTZ     NOT NULL,
    quarantined_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    resolved            BOOLEAN         NOT NULL DEFAULT false,
    resolved_by         UUID,
    resolved_at         TIMESTAMPTZ,
    resolution_action   TEXT CHECK (resolution_action IN ('deleted','restored','false_positive')),
    tenant_id           UUID            NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_quarantine_unresolved ON file_quarantine_log (resolved) WHERE resolved = false;

ALTER TABLE file_quarantine_log ADD COLUMN IF NOT EXISTS tenant_id UUID;

UPDATE file_quarantine_log q
SET tenant_id = f.tenant_id
FROM files f
WHERE q.file_id = f.id
  AND q.tenant_id IS NULL;

ALTER TABLE file_quarantine_log ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE files ENABLE ROW LEVEL SECURITY;
ALTER TABLE files FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON files;
DROP POLICY IF EXISTS tenant_insert ON files;
DROP POLICY IF EXISTS tenant_update ON files;
DROP POLICY IF EXISTS tenant_delete ON files;
CREATE POLICY tenant_isolation ON files
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    );
CREATE POLICY tenant_insert ON files
    FOR INSERT
    WITH CHECK (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    );
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
CREATE POLICY tenant_delete ON files
    FOR DELETE
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    );

ALTER TABLE file_access_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE file_access_log FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON file_access_log;
DROP POLICY IF EXISTS tenant_insert ON file_access_log;
DROP POLICY IF EXISTS tenant_update ON file_access_log;
DROP POLICY IF EXISTS tenant_delete ON file_access_log;
CREATE POLICY tenant_isolation ON file_access_log
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    );
CREATE POLICY tenant_insert ON file_access_log
    FOR INSERT
    WITH CHECK (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    );
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
CREATE POLICY tenant_delete ON file_access_log
    FOR DELETE
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        OR current_setting('app.bypass_rls', true) = 'on'
    );

ALTER TABLE file_quarantine_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE file_quarantine_log FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON file_quarantine_log;
DROP POLICY IF EXISTS tenant_insert ON file_quarantine_log;
DROP POLICY IF EXISTS tenant_update ON file_quarantine_log;
DROP POLICY IF EXISTS tenant_delete ON file_quarantine_log;
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

WITH demo_users AS (
    SELECT *
    FROM (
        VALUES
            (1, '{{ .MainAdminUserID }}'::uuid),
            (2, '{{ .SecurityManagerUserID }}'::uuid),
            (3, '{{ .DataStewardUserID }}'::uuid),
            (4, '{{ .LegalManagerUserID }}'::uuid),
            (5, '{{ .BoardSecretaryUserID }}'::uuid),
            (6, '{{ .ExecutiveUserID }}'::uuid),
            (7, '{{ .AuditorUserID }}'::uuid)
    ) AS t(slot, user_id)
)
INSERT INTO files (
    id, tenant_id, bucket, storage_key, original_name, sanitized_name, content_type, detected_content_type,
    size_bytes, checksum_sha256, encrypted, encryption_metadata, virus_scan_status, virus_scan_result,
    virus_scanned_at, uploaded_by, suite, entity_type, entity_id, tags, version_id, version_number,
    is_public, lifecycle_policy, expires_at, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'file-' || gs),
    '{{ .MainTenantID }}'::uuid,
    'clario360-demo',
    format('seeded/%s/%s.dat',
        CASE gs % 6
            WHEN 0 THEN 'cyber'
            WHEN 1 THEN 'data'
            WHEN 2 THEN 'acta'
            WHEN 3 THEN 'lex'
            WHEN 4 THEN 'visus'
            ELSE 'platform'
        END,
        lpad(gs::text, 6, '0')
    ),
    format('seeded-file-%s.txt', lpad(gs::text, 6, '0')),
    format('seeded-file-%s.txt', lpad(gs::text, 6, '0')),
    CASE WHEN gs % 4 = 0 THEN 'application/pdf' ELSE 'text/plain' END,
    CASE WHEN gs % 4 = 0 THEN 'application/pdf' ELSE 'text/plain' END,
    2048 + (gs * 37),
    md5('file-checksum-' || gs),
    gs % 3 = 0,
    CASE WHEN gs % 3 = 0 THEN '{"algorithm":"aes256","key_id":"demo"}'::jsonb ELSE NULL END,
    CASE WHEN gs % 17 = 0 THEN 'infected' WHEN gs % 5 = 0 THEN 'clean' ELSE 'clean' END,
    CASE WHEN gs % 17 = 0 THEN 'EICAR-Test-File' ELSE NULL END,
    now() - interval '1 day',
    du.user_id,
    CASE gs % 6
        WHEN 0 THEN 'cyber'
        WHEN 1 THEN 'data'
        WHEN 2 THEN 'acta'
        WHEN 3 THEN 'lex'
        WHEN 4 THEN 'visus'
        ELSE 'platform'
    END,
    CASE gs % 5
        WHEN 0 THEN 'report'
        WHEN 1 THEN 'contract'
        WHEN 2 THEN 'meeting'
        WHEN 3 THEN 'dashboard'
        ELSE 'evidence'
    END,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'file-entity-' || gs),
    ARRAY['seeded','demo', CASE WHEN gs % 2 = 0 THEN 'retained' ELSE 'working' END],
    format('v%s', 1 + (gs % 3)),
    1 + (gs % 3),
    gs % 11 = 0,
    CASE WHEN gs % 7 = 0 THEN 'archive' WHEN gs % 3 = 0 THEN 'audit_retention' ELSE 'standard' END,
    CASE WHEN gs % 9 = 0 THEN now() + interval '30 days' ELSE NULL END,
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320)),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 5)
FROM generate_series(1, {{ .Scale.FileCount }}) gs
JOIN demo_users du ON du.slot = ((gs - 1) % 7) + 1
ON CONFLICT (id) DO UPDATE SET
    detected_content_type = EXCLUDED.detected_content_type,
    size_bytes = EXCLUDED.size_bytes,
    checksum_sha256 = EXCLUDED.checksum_sha256,
    encrypted = EXCLUDED.encrypted,
    encryption_metadata = EXCLUDED.encryption_metadata,
    virus_scan_status = EXCLUDED.virus_scan_status,
    virus_scan_result = EXCLUDED.virus_scan_result,
    virus_scanned_at = EXCLUDED.virus_scanned_at,
    uploaded_by = EXCLUDED.uploaded_by,
    suite = EXCLUDED.suite,
    entity_type = EXCLUDED.entity_type,
    entity_id = EXCLUDED.entity_id,
    tags = EXCLUDED.tags,
    version_id = EXCLUDED.version_id,
    version_number = EXCLUDED.version_number,
    is_public = EXCLUDED.is_public,
    lifecycle_policy = EXCLUDED.lifecycle_policy,
    expires_at = EXCLUDED.expires_at,
    updated_at = EXCLUDED.updated_at,
    deleted_at = NULL;

WITH demo_users AS (
    SELECT *
    FROM (
        VALUES
            (1, '{{ .MainAdminUserID }}'::uuid),
            (2, '{{ .SecurityManagerUserID }}'::uuid),
            (3, '{{ .DataStewardUserID }}'::uuid),
            (4, '{{ .LegalManagerUserID }}'::uuid),
            (5, '{{ .BoardSecretaryUserID }}'::uuid),
            (6, '{{ .ExecutiveUserID }}'::uuid),
            (7, '{{ .AuditorUserID }}'::uuid)
    ) AS t(slot, user_id)
)
INSERT INTO file_access_log (
    id, file_id, tenant_id, user_id, action, ip_address, user_agent, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'file-access-' || gs),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'file-' || (((gs - 1) % {{ .Scale.FileCount }}) + 1)),
    '{{ .MainTenantID }}'::uuid,
    du.user_id,
    CASE gs % 6
        WHEN 0 THEN 'upload'
        WHEN 1 THEN 'download'
        WHEN 2 THEN 'presigned_download'
        WHEN 3 THEN 'presigned_upload'
        WHEN 4 THEN 'view_metadata'
        ELSE 'delete'
    END,
    format('192.168.%s.%s', ((gs - 1) % 255), (gs % 255)),
    format('Clario File Client/%s', 1 + (gs % 4)),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320))
FROM generate_series(1, {{ .Scale.FileAccessLogCount }}) gs
JOIN demo_users du ON du.slot = ((gs - 1) % 7) + 1
ON CONFLICT (id) DO NOTHING;

INSERT INTO file_quarantine_log (
    id, file_id, original_bucket, original_key, quarantine_bucket, quarantine_key, virus_name,
    scanned_at, quarantined_at, resolved, resolved_by, resolved_at, resolution_action, tenant_id
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'file-quarantine-' || gs),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'file-' || gs),
    'clario360-demo',
    format('seeded/platform/%s.dat', lpad(gs::text, 6, '0')),
    'clario360-quarantine',
    format('quarantine/%s.dat', lpad(gs::text, 6, '0')),
    'EICAR-Test-File',
    now() - interval '3 days',
    now() - interval '3 days' + make_interval(hours => gs),
    gs % 3 = 0,
    CASE WHEN gs % 3 = 0 THEN '{{ .SecurityManagerUserID }}'::uuid ELSE NULL END,
    CASE WHEN gs % 3 = 0 THEN now() - interval '1 day' ELSE NULL END,
    CASE WHEN gs % 3 = 0 THEN 'deleted' ELSE NULL END,
    '{{ .MainTenantID }}'::uuid
FROM generate_series(1, LEAST({{ .Scale.FileCount }}, 18)) gs
ON CONFLICT (id) DO NOTHING;
