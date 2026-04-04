package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// schemaSQL is the DDL for all notification service tables.
const schemaSQL = `
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

CREATE INDEX IF NOT EXISTS idx_notif_user_unread ON notifications (tenant_id, user_id, created_at DESC)
    WHERE read_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_notif_user_all ON notifications (tenant_id, user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notif_tenant_type ON notifications (tenant_id, type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notif_source_event ON notifications (source_event_id)
    WHERE source_event_id IS NOT NULL;

-- Deduplication: prevent duplicate notifications from Kafka redelivery
CREATE UNIQUE INDEX IF NOT EXISTS idx_notif_dedup ON notifications (tenant_id, user_id, source_event_id)
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
    status          TEXT            NOT NULL CHECK (status IN ('pending', 'delivered', 'failed', 'skipped', 'retrying')),
    attempt         INT             NOT NULL DEFAULT 1,
    error_message   TEXT,
    metadata        JSONB           NOT NULL DEFAULT '{}',
    webhook_id      UUID,
    event_type      TEXT,
    request_url     TEXT,
    request_body    JSONB,
    response_status INT,
    response_body   TEXT,
    duration_ms     INT,
    next_retry_at   TIMESTAMPTZ,
    delivered_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_delivery_notification ON notification_delivery_log (notification_id);
CREATE INDEX IF NOT EXISTS idx_delivery_status ON notification_delivery_log (status, created_at)
    WHERE status = 'failed';
CREATE INDEX IF NOT EXISTS idx_delivery_webhook ON notification_delivery_log (webhook_id)
    WHERE webhook_id IS NOT NULL;

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

CREATE INDEX IF NOT EXISTS idx_webhook_tenant ON notification_webhooks (tenant_id, active)
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

-- =============================================================================
-- INTEGRATIONS — External tool connection configurations
-- =============================================================================

CREATE TABLE IF NOT EXISTS integrations (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    type                TEXT            NOT NULL CHECK (type IN ('slack', 'teams', 'jira', 'servicenow', 'webhook')),
    name                TEXT            NOT NULL,
    description         TEXT            NOT NULL DEFAULT '',
    config_encrypted    BYTEA           NOT NULL,
    config_nonce        BYTEA           NOT NULL,
    config_key_id       TEXT            NOT NULL,
    status              TEXT            NOT NULL DEFAULT 'active'
                                        CHECK (status IN ('active', 'inactive', 'error', 'setup_pending')),
    error_message       TEXT,
    error_count         INT             NOT NULL DEFAULT 0,
    last_error_at       TIMESTAMPTZ,
    event_filters       JSONB           NOT NULL DEFAULT '[]',
    last_used_at        TIMESTAMPTZ,
    delivery_count      BIGINT          NOT NULL DEFAULT 0,
    created_by          UUID            NOT NULL,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_integrations_tenant ON integrations (tenant_id, type, status)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_integrations_active ON integrations (tenant_id, status)
    WHERE status = 'active' AND deleted_at IS NULL;

-- =============================================================================
-- DELIVERY LOG — Outbound delivery tracking with retry status
-- =============================================================================

CREATE TABLE IF NOT EXISTS integration_deliveries (
    id              UUID            NOT NULL DEFAULT gen_random_uuid(),
    tenant_id       UUID            NOT NULL,
    integration_id  UUID            NOT NULL REFERENCES integrations(id),
    event_type      TEXT            NOT NULL,
    event_id        TEXT            NOT NULL,
    event_data      JSONB,
    status          TEXT            NOT NULL DEFAULT 'pending'
                                    CHECK (status IN ('pending', 'delivered', 'failed', 'retrying')),
    attempts        INT             NOT NULL DEFAULT 0,
    max_attempts    INT             NOT NULL DEFAULT 4,
    response_code   INT,
    response_body   TEXT,
    last_error      TEXT,
    error_category  TEXT,
    next_retry_at   TIMESTAMPTZ,
    latency_ms      INT,
    delivered_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

DO $$
DECLARE
    start_month DATE := date_trunc('month', current_date)::date;
    next_month DATE := (start_month + INTERVAL '1 month')::date;
    after_next_month DATE := (start_month + INTERVAL '2 month')::date;
    current_partition TEXT := format('integration_deliveries_%s', to_char(start_month, 'YYYY_MM'));
    next_partition TEXT := format('integration_deliveries_%s', to_char(next_month, 'YYYY_MM'));
BEGIN
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF integration_deliveries FOR VALUES FROM (%L) TO (%L)',
        current_partition, start_month, next_month
    );
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF integration_deliveries FOR VALUES FROM (%L) TO (%L)',
        next_partition, next_month, after_next_month
    );
END $$;

CREATE INDEX IF NOT EXISTS idx_deliveries_integration ON integration_deliveries (integration_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_deliveries_retry ON integration_deliveries (next_retry_at ASC)
    WHERE status = 'retrying';
CREATE INDEX IF NOT EXISTS idx_deliveries_event ON integration_deliveries (event_id);

-- =============================================================================
-- EXTERNAL TICKET LINKS — Bidirectional link: Clario entity ↔ external ticket
-- =============================================================================

CREATE TABLE IF NOT EXISTS external_ticket_links (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    integration_id      UUID            NOT NULL REFERENCES integrations(id),
    entity_type         TEXT            NOT NULL,
    entity_id           UUID            NOT NULL,
    external_system     TEXT            NOT NULL,
    external_id         TEXT            NOT NULL,
    external_key        TEXT            NOT NULL,
    external_url        TEXT            NOT NULL,
    external_status     TEXT,
    external_priority   TEXT,
    sync_direction      TEXT            NOT NULL DEFAULT 'bidirectional'
                                        CHECK (sync_direction IN ('outbound', 'inbound', 'bidirectional')),
    last_synced_at      TIMESTAMPTZ,
    last_sync_direction TEXT,
    sync_error          TEXT,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, entity_type, entity_id, external_system, external_id)
);

CREATE INDEX IF NOT EXISTS idx_ticket_links_entity ON external_ticket_links (tenant_id, entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_ticket_links_external ON external_ticket_links (external_system, external_id);
CREATE INDEX IF NOT EXISTS idx_ticket_links_integration ON external_ticket_links (integration_id);
`

// RunMigration executes the notification schema DDL against the database.
func RunMigration(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("notification schema migration: %w", err)
	}
	return nil
}
