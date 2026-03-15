-- =============================================================================
-- Webhook Enhancements: add headers, retry_policy, stats, delivery tracking
-- =============================================================================

-- Add new columns to notification_webhooks
ALTER TABLE notification_webhooks
    ADD COLUMN IF NOT EXISTS headers       JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS retry_policy  JSONB NOT NULL DEFAULT '{"max_retries": 3, "backoff_type": "exponential", "initial_delay_seconds": 10}',
    ADD COLUMN IF NOT EXISTS last_triggered_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS success_count BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS failure_count BIGINT NOT NULL DEFAULT 0;

-- Add webhook_id to delivery log for direct webhook delivery tracking
ALTER TABLE notification_delivery_log
    ADD COLUMN IF NOT EXISTS webhook_id UUID REFERENCES notification_webhooks(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS event_type TEXT,
    ADD COLUMN IF NOT EXISTS request_url TEXT,
    ADD COLUMN IF NOT EXISTS request_body JSONB,
    ADD COLUMN IF NOT EXISTS response_status INT,
    ADD COLUMN IF NOT EXISTS response_body TEXT,
    ADD COLUMN IF NOT EXISTS duration_ms INT,
    ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_delivery_webhook ON notification_delivery_log (webhook_id, created_at DESC)
    WHERE webhook_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_delivery_tenant_date ON notification_delivery_log (created_at)
    USING btree (created_at);
