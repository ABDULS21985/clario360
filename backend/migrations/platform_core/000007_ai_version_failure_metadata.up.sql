ALTER TABLE ai_model_versions
    ADD COLUMN IF NOT EXISTS failed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS failed_by UUID,
    ADD COLUMN IF NOT EXISTS failed_from_status TEXT CHECK (
        failed_from_status IS NULL OR failed_from_status IN (
            'development', 'staging', 'shadow', 'production', 'retired', 'failed', 'rolled_back'
        )
    ),
    ADD COLUMN IF NOT EXISTS failure_reason TEXT;

CREATE INDEX IF NOT EXISTS idx_ai_versions_failed
    ON ai_model_versions (tenant_id, model_id)
    WHERE status = 'failed';
