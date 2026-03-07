-- Rollback: Cyber Suite Asset Inventory Extensions
DROP INDEX IF EXISTS idx_assets_tenant_ip_unique;
DROP INDEX IF EXISTS idx_assets_fts;
DROP INDEX IF EXISTS idx_assets_tenant_department;
DROP INDEX IF EXISTS idx_scan_tenant_created;
DROP INDEX IF EXISTS idx_scan_tenant_status;
DROP INDEX IF EXISTS idx_cve_cpe;
DROP INDEX IF EXISTS idx_cve_severity;
DROP INDEX IF EXISTS idx_cve_published;

ALTER TABLE asset_relationships DROP CONSTRAINT IF EXISTS uq_asset_rel_unique;
ALTER TABLE asset_relationships DROP CONSTRAINT IF EXISTS chk_relationship_type;
ALTER TABLE vulnerabilities DROP CONSTRAINT IF EXISTS uq_vuln_tenant_asset_cve;

ALTER TABLE vulnerabilities
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS remediation,
    DROP COLUMN IF EXISTS proof,
    DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE assets
    DROP COLUMN IF EXISTS discovery_source,
    DROP COLUMN IF EXISTS location;

DROP TABLE IF EXISTS scan_history;
DROP TABLE IF EXISTS cve_database;
DROP FUNCTION IF EXISTS severity_order(TEXT);
