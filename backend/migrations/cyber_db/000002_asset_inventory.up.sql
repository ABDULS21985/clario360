-- =============================================================================
-- Clario 360 — Cyber Suite Asset Inventory Extensions (PROMPT 16)
-- Adds missing columns, tables, indexes, and functions needed for the
-- full Asset Discovery & Inventory Engine.
-- =============================================================================

-- ---------------------------------------------------------------------------
-- Extend asset_status enum to include 'unknown'
-- ---------------------------------------------------------------------------
ALTER TYPE asset_status ADD VALUE IF NOT EXISTS 'unknown';

-- ---------------------------------------------------------------------------
-- Add missing columns to assets table
-- ---------------------------------------------------------------------------
ALTER TABLE assets
    ADD COLUMN IF NOT EXISTS discovery_source TEXT NOT NULL DEFAULT 'manual'
        CHECK (discovery_source IN ('manual', 'network_scan', 'cloud_scan', 'agent', 'import')),
    ADD COLUMN IF NOT EXISTS location TEXT;

-- ---------------------------------------------------------------------------
-- Add unique index on (tenant_id, ip_address) for upsert-on-scan support
-- ---------------------------------------------------------------------------
CREATE UNIQUE INDEX IF NOT EXISTS idx_assets_tenant_ip_unique
    ON assets (tenant_id, ip_address)
    WHERE ip_address IS NOT NULL AND deleted_at IS NULL;

-- Full-text search index on assets
CREATE INDEX IF NOT EXISTS idx_assets_fts ON assets USING GIN (
    to_tsvector('english',
        coalesce(name, '') || ' ' ||
        coalesce(hostname, '') || ' ' ||
        coalesce(host(ip_address), '') || ' ' ||
        coalesce(os, '') || ' ' ||
        coalesce(department, '') || ' ' ||
        coalesce(location, '')
    )
) WHERE deleted_at IS NULL;

-- Additional index for owner/department filtering
CREATE INDEX IF NOT EXISTS idx_assets_tenant_department
    ON assets (tenant_id, department) WHERE deleted_at IS NULL;

-- ---------------------------------------------------------------------------
-- Add missing columns to vulnerabilities table
-- ---------------------------------------------------------------------------
ALTER TABLE vulnerabilities
    ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'cve_enrichment'
        CHECK (source IN ('cve_enrichment', 'manual', 'scan_tool', 'penetration_test')),
    ADD COLUMN IF NOT EXISTS remediation TEXT,
    ADD COLUMN IF NOT EXISTS proof TEXT,
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Add unique constraint: one CVE per asset per tenant
ALTER TABLE vulnerabilities
    DROP CONSTRAINT IF EXISTS uq_vuln_tenant_asset_cve;

ALTER TABLE vulnerabilities
    ADD CONSTRAINT uq_vuln_tenant_asset_cve
        UNIQUE (tenant_id, asset_id, cve_id);

-- ---------------------------------------------------------------------------
-- Add unique constraint to asset_relationships
-- ---------------------------------------------------------------------------
ALTER TABLE asset_relationships
    DROP CONSTRAINT IF EXISTS uq_asset_rel_unique;

ALTER TABLE asset_relationships
    ADD CONSTRAINT uq_asset_rel_unique
        UNIQUE (tenant_id, source_asset_id, target_asset_id, relationship_type);

-- Add valid relationship_type check if not present
ALTER TABLE asset_relationships
    DROP CONSTRAINT IF EXISTS chk_relationship_type;

ALTER TABLE asset_relationships
    ADD CONSTRAINT chk_relationship_type
        CHECK (relationship_type IN (
            'hosts', 'runs_on', 'connects_to', 'depends_on',
            'managed_by', 'backs_up', 'load_balances'
        ));

-- ---------------------------------------------------------------------------
-- Severity ordering helper function
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION severity_order(sev TEXT) RETURNS INT AS $$
BEGIN
    RETURN CASE sev
        WHEN 'critical' THEN 5
        WHEN 'high' THEN 4
        WHEN 'medium' THEN 3
        WHEN 'low' THEN 2
        WHEN 'info' THEN 1
        ELSE 0
    END;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- ---------------------------------------------------------------------------
-- TABLE: scan_history
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS scan_history (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    scan_type         TEXT        NOT NULL CHECK (scan_type IN ('network', 'cloud', 'agent', 'import')),
    config            JSONB       NOT NULL DEFAULT '{}',
    status            TEXT        NOT NULL DEFAULT 'running'
                                  CHECK (status IN ('running', 'completed', 'failed', 'cancelled')),
    assets_discovered INT         NOT NULL DEFAULT 0,
    assets_new        INT         NOT NULL DEFAULT 0,
    assets_updated    INT         NOT NULL DEFAULT 0,
    error_count       INT         NOT NULL DEFAULT 0,
    errors            JSONB,
    started_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at      TIMESTAMPTZ,
    duration_ms       BIGINT,
    created_by        UUID        NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_scan_tenant_created ON scan_history (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_scan_tenant_status  ON scan_history (tenant_id, status);

-- ---------------------------------------------------------------------------
-- TABLE: cve_database
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cve_database (
    cve_id            TEXT        PRIMARY KEY,
    description       TEXT        NOT NULL DEFAULT '',
    severity          TEXT        NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info')),
    cvss_v3_score     DECIMAL(3,1),
    cvss_v3_vector    TEXT,
    cpe_matches       TEXT[]      NOT NULL DEFAULT '{}',
    affected_products JSONB       NOT NULL DEFAULT '[]',
    published_at      TIMESTAMPTZ NOT NULL,
    modified_at       TIMESTAMPTZ NOT NULL,
    references        JSONB       NOT NULL DEFAULT '[]',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_cve_cpe       ON cve_database USING GIN (cpe_matches);
CREATE INDEX IF NOT EXISTS idx_cve_severity  ON cve_database (severity);
CREATE INDEX IF NOT EXISTS idx_cve_published ON cve_database (published_at DESC);
