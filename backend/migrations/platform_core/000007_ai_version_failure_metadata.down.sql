DROP INDEX IF EXISTS idx_ai_versions_failed;

ALTER TABLE ai_model_versions
    DROP COLUMN IF EXISTS failure_reason,
    DROP COLUMN IF EXISTS failed_from_status,
    DROP COLUMN IF EXISTS failed_by,
    DROP COLUMN IF EXISTS failed_at;
