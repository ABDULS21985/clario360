DROP INDEX IF EXISTS idx_delivery_tenant_date;
DROP INDEX IF EXISTS idx_delivery_webhook;

ALTER TABLE notification_delivery_log
    DROP COLUMN IF EXISTS webhook_id,
    DROP COLUMN IF EXISTS event_type,
    DROP COLUMN IF EXISTS request_url,
    DROP COLUMN IF EXISTS request_body,
    DROP COLUMN IF EXISTS response_status,
    DROP COLUMN IF EXISTS response_body,
    DROP COLUMN IF EXISTS duration_ms,
    DROP COLUMN IF EXISTS next_retry_at;

ALTER TABLE notification_webhooks
    DROP COLUMN IF EXISTS headers,
    DROP COLUMN IF EXISTS retry_policy,
    DROP COLUMN IF EXISTS last_triggered_at,
    DROP COLUMN IF EXISTS success_count,
    DROP COLUMN IF EXISTS failure_count;
