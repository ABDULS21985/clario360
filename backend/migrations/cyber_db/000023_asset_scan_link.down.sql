DROP INDEX IF EXISTS idx_assets_last_scan_id;
ALTER TABLE assets DROP COLUMN IF EXISTS last_scan_id;
