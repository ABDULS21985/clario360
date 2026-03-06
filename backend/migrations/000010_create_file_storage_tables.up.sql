-- File Storage Service tables
-- Migration: 000010_create_file_storage_tables

-- FILES — Metadata for every stored file
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

CREATE INDEX IF NOT EXISTS idx_files_tenant_suite       ON files (tenant_id, suite, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_files_entity             ON files (entity_type, entity_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_files_tenant_uploader    ON files (tenant_id, uploaded_by, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_files_virus_pending      ON files (virus_scan_status) WHERE virus_scan_status IN ('pending','scanning');
CREATE INDEX IF NOT EXISTS idx_files_lifecycle_expiry   ON files (expires_at) WHERE expires_at IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_files_tags               ON files USING GIN (tags);
CREATE INDEX IF NOT EXISTS idx_files_checksum           ON files (checksum_sha256);

-- FILE ACCESS LOG — Compliance audit trail for every access
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

-- QUARANTINE LOG — Infected files moved to quarantine
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
    resolution_action   TEXT CHECK (resolution_action IN ('deleted','restored','false_positive'))
);

CREATE INDEX IF NOT EXISTS idx_quarantine_unresolved ON file_quarantine_log (resolved) WHERE resolved = false;
