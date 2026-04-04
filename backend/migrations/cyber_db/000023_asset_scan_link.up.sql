-- Add last_scan_id to assets so scan detail pages can filter by scan.
ALTER TABLE assets ADD COLUMN IF NOT EXISTS last_scan_id UUID;

CREATE INDEX IF NOT EXISTS idx_assets_last_scan_id
    ON assets (last_scan_id)
    WHERE last_scan_id IS NOT NULL AND deleted_at IS NULL;
