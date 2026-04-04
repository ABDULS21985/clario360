CREATE TABLE IF NOT EXISTS asset_activity (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    asset_id    UUID NOT NULL,
    action      VARCHAR(100) NOT NULL,
    actor_id    UUID,
    actor_name  VARCHAR(255),
    description TEXT NOT NULL DEFAULT '',
    old_value   TEXT,
    new_value   TEXT,
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_asset_activity_tenant_asset ON asset_activity (tenant_id, asset_id, created_at DESC);
