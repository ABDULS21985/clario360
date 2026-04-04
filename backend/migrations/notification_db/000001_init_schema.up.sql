-- =============================================================================
-- NOTIFICATIONS — Per-user notification records
-- =============================================================================

CREATE TABLE IF NOT EXISTS notifications (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID            NOT NULL,
    user_id         UUID            NOT NULL,
    type            TEXT            NOT NULL,
    category        TEXT            NOT NULL CHECK (category IN ('security', 'data', 'governance', 'legal', 'system', 'workflow')),
    priority        TEXT            NOT NULL CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    title           TEXT            NOT NULL,
    body            TEXT            NOT NULL,
    data            JSONB           NOT NULL DEFAULT '{}',
    action_url      TEXT            NOT NULL DEFAULT '',
    source_event_id TEXT,
    read_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX idx_notif_user_unread ON notifications (tenant_id, user_id, created_at DESC)
    WHERE read_at IS NULL;
CREATE INDEX idx_notif_user_all ON notifications (tenant_id, user_id, created_at DESC);
CREATE INDEX idx_notif_tenant_type ON notifications (tenant_id, type, created_at DESC);
CREATE INDEX idx_notif_source_event ON notifications (source_event_id)
    WHERE source_event_id IS NOT NULL;

-- Deduplication: prevent duplicate notifications from Kafka redelivery
CREATE UNIQUE INDEX idx_notif_dedup ON notifications (tenant_id, user_id, source_event_id)
    WHERE source_event_id IS NOT NULL;

-- =============================================================================
-- NOTIFICATION PREFERENCES — Per-user per-tenant channel preferences
-- =============================================================================

CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id         UUID            NOT NULL,
    tenant_id       UUID            NOT NULL,
    global_prefs    JSONB           NOT NULL DEFAULT '{"in_app": true, "email": true, "websocket": true, "webhook": false}',
    per_type_prefs  JSONB           NOT NULL DEFAULT '{}',
    quiet_hours     JSONB,
    digest_config   JSONB           NOT NULL DEFAULT '{"daily": false, "weekly": true}',
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, tenant_id)
);

-- =============================================================================
-- DELIVERY LOG — Tracks delivery attempts per channel for every notification
-- =============================================================================

CREATE TABLE IF NOT EXISTS notification_delivery_log (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID            NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    channel         TEXT            NOT NULL CHECK (channel IN ('in_app', 'email', 'websocket', 'webhook')),
    status          TEXT            NOT NULL CHECK (status IN ('pending', 'delivered', 'failed', 'skipped')),
    attempt         INT             NOT NULL DEFAULT 1,
    error_message   TEXT,
    metadata        JSONB           NOT NULL DEFAULT '{}',
    delivered_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX idx_delivery_notification ON notification_delivery_log (notification_id);
CREATE INDEX idx_delivery_status ON notification_delivery_log (status, created_at)
    WHERE status = 'failed';

-- =============================================================================
-- WEBHOOK REGISTRATIONS — Per-tenant external webhook endpoints
-- =============================================================================

CREATE TABLE IF NOT EXISTS notification_webhooks (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID            NOT NULL,
    name            TEXT            NOT NULL,
    url             TEXT            NOT NULL,
    secret          TEXT,
    event_types     TEXT[]          NOT NULL DEFAULT '{}',
    active          BOOLEAN         NOT NULL DEFAULT true,
    created_by      UUID            NOT NULL,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX idx_webhook_tenant ON notification_webhooks (tenant_id, active)
    WHERE active = true;

-- =============================================================================
-- NOTIFICATION TEMPLATES — Stored templates (overridable per tenant)
-- =============================================================================

CREATE TABLE IF NOT EXISTS notification_templates (
    id              TEXT            NOT NULL,
    tenant_id       UUID            NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    channel         TEXT            NOT NULL CHECK (channel IN ('email', 'in_app', 'websocket')),
    subject_tmpl    TEXT            NOT NULL DEFAULT '',
    body_tmpl       TEXT            NOT NULL,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    PRIMARY KEY (id, channel, tenant_id)
);
