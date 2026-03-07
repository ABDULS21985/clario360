ALTER TABLE assets
    ALTER COLUMN owner TYPE TEXT USING owner::text;

COMMENT ON COLUMN assets.owner IS 'Business owner or team responsible for this asset';

CREATE INDEX IF NOT EXISTS idx_assets_tenant_owner
    ON assets (tenant_id, owner)
    WHERE owner IS NOT NULL AND deleted_at IS NULL;
