DROP INDEX IF EXISTS idx_assets_tenant_owner;

ALTER TABLE assets
    ALTER COLUMN owner TYPE UUID USING NULLIF(owner, '')::uuid;

COMMENT ON COLUMN assets.owner IS 'User responsible for this asset (references platform_core.users)';
