-- =============================================================================
-- DSPM ACCESS MAPPINGS — Identity-to-data-asset permission mappings
-- =============================================================================

CREATE TABLE IF NOT EXISTS dspm_access_mappings (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    -- Identity
    identity_type       TEXT            NOT NULL CHECK (identity_type IN (
        'user', 'service_account', 'role', 'group', 'api_key', 'application'
    )),
    identity_id         TEXT            NOT NULL,
    identity_name       TEXT,
    identity_source     TEXT            NOT NULL,
    -- Target data asset
    data_asset_id       UUID            NOT NULL REFERENCES dspm_data_assets(id),
    data_asset_name     TEXT,
    data_classification TEXT,
    -- Permission details
    permission_type     TEXT            NOT NULL CHECK (permission_type IN (
        'read', 'write', 'admin', 'delete', 'create', 'alter', 'execute', 'full_control'
    )),
    permission_source   TEXT            NOT NULL,
    permission_path     TEXT[],
    is_wildcard         BOOLEAN         NOT NULL DEFAULT false,
    -- Usage tracking
    last_used_at        TIMESTAMPTZ,
    usage_count_30d     INT             NOT NULL DEFAULT 0,
    usage_count_90d     INT             NOT NULL DEFAULT 0,
    is_stale            BOOLEAN         NOT NULL DEFAULT false,
    -- Risk scoring
    sensitivity_weight  FLOAT           NOT NULL DEFAULT 1.0,
    access_risk_score   FLOAT           NOT NULL DEFAULT 0.0,
    -- Status
    status              TEXT            NOT NULL DEFAULT 'active'
                                        CHECK (status IN ('active', 'revoked', 'expired', 'pending_review')),
    expires_at          TIMESTAMPTZ,
    -- Metadata
    discovered_at       TIMESTAMPTZ     NOT NULL DEFAULT now(),
    last_verified_at    TIMESTAMPTZ     NOT NULL DEFAULT now(),
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),

    UNIQUE (tenant_id, identity_type, identity_id, data_asset_id, permission_type)
);

CREATE INDEX IF NOT EXISTS idx_dspm_access_identity
    ON dspm_access_mappings (tenant_id, identity_type, identity_id);
CREATE INDEX IF NOT EXISTS idx_dspm_access_asset
    ON dspm_access_mappings (tenant_id, data_asset_id);
CREATE INDEX IF NOT EXISTS idx_dspm_access_stale
    ON dspm_access_mappings (tenant_id, is_stale, last_used_at)
    WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_dspm_access_risk
    ON dspm_access_mappings (tenant_id, access_risk_score DESC)
    WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_dspm_access_classification
    ON dspm_access_mappings (tenant_id, data_classification)
    WHERE status = 'active';

-- =============================================================================
-- DSPM IDENTITY PROFILES — Aggregated identity risk profile
-- =============================================================================

CREATE TABLE IF NOT EXISTS dspm_identity_profiles (
    id                      UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID            NOT NULL,
    identity_type           TEXT            NOT NULL,
    identity_id             TEXT            NOT NULL,
    identity_name           TEXT,
    identity_email          TEXT,
    identity_source         TEXT            NOT NULL,
    -- Access summary
    total_assets_accessible INT             NOT NULL DEFAULT 0,
    sensitive_assets_count  INT             NOT NULL DEFAULT 0,
    permission_count        INT             NOT NULL DEFAULT 0,
    overprivileged_count    INT             NOT NULL DEFAULT 0,
    stale_permission_count  INT             NOT NULL DEFAULT 0,
    -- Blast radius
    blast_radius_score      FLOAT           NOT NULL DEFAULT 0.0,
    blast_radius_level      TEXT            NOT NULL DEFAULT 'low'
                                            CHECK (blast_radius_level IN ('low', 'medium', 'high', 'critical')),
    -- Access risk (composite)
    access_risk_score       FLOAT           NOT NULL DEFAULT 0.0,
    access_risk_level       TEXT            NOT NULL DEFAULT 'low'
                                            CHECK (access_risk_level IN ('low', 'medium', 'high', 'critical')),
    risk_factors            JSONB           NOT NULL DEFAULT '[]',
    -- Access patterns
    last_activity_at        TIMESTAMPTZ,
    avg_daily_access_count  FLOAT           NOT NULL DEFAULT 0.0,
    access_pattern_summary  JSONB           NOT NULL DEFAULT '{}',
    -- Recommendations
    recommendations         JSONB           NOT NULL DEFAULT '[]',
    -- Status
    status                  TEXT            NOT NULL DEFAULT 'active'
                                            CHECK (status IN ('active', 'inactive', 'under_review', 'remediated')),
    last_review_at          TIMESTAMPTZ,
    next_review_due         TIMESTAMPTZ,
    -- Metadata
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT now(),

    UNIQUE (tenant_id, identity_type, identity_id)
);

CREATE INDEX IF NOT EXISTS idx_dspm_identity_risk
    ON dspm_identity_profiles (tenant_id, access_risk_score DESC)
    WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_dspm_identity_blast
    ON dspm_identity_profiles (tenant_id, blast_radius_score DESC)
    WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_dspm_identity_overpriv
    ON dspm_identity_profiles (tenant_id, overprivileged_count DESC)
    WHERE status = 'active' AND overprivileged_count > 0;
CREATE INDEX IF NOT EXISTS idx_dspm_identity_review
    ON dspm_identity_profiles (tenant_id, next_review_due)
    WHERE status = 'active';

-- =============================================================================
-- DSPM ACCESS AUDIT — Historical access events for usage tracking
-- =============================================================================

CREATE TABLE IF NOT EXISTS dspm_access_audit (
    id                  UUID            NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    identity_type       TEXT            NOT NULL,
    identity_id         TEXT            NOT NULL,
    data_asset_id       UUID            NOT NULL,
    -- Event
    action              TEXT            NOT NULL,
    source_ip           TEXT,
    query_hash          TEXT,
    rows_affected       BIGINT,
    duration_ms         INT,
    success             BOOLEAN         NOT NULL DEFAULT true,
    -- Context
    access_mapping_id   UUID,
    table_name          TEXT,
    database_name       TEXT,
    -- Timestamp
    event_timestamp     TIMESTAMPTZ     NOT NULL,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),

    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create 4 months of partitions
DO $$
DECLARE
    start_date DATE := date_trunc('month', CURRENT_DATE);
    partition_date DATE;
    partition_name TEXT;
BEGIN
    FOR i IN 0..3 LOOP
        partition_date := start_date + (i || ' months')::interval;
        partition_name := 'dspm_access_audit_' || to_char(partition_date, 'YYYY_MM');
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS %I PARTITION OF dspm_access_audit
             FOR VALUES FROM (%L) TO (%L)',
            partition_name,
            partition_date,
            partition_date + interval '1 month'
        );
    END LOOP;
END $$;

CREATE INDEX IF NOT EXISTS idx_dspm_audit_identity
    ON dspm_access_audit (tenant_id, identity_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_dspm_audit_asset
    ON dspm_access_audit (tenant_id, data_asset_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_dspm_audit_mapping
    ON dspm_access_audit (tenant_id, access_mapping_id, created_at DESC);

-- =============================================================================
-- DSPM ACCESS POLICIES — Governance policy definitions
-- =============================================================================

CREATE TABLE IF NOT EXISTS dspm_access_policies (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    -- Policy definition
    name                TEXT            NOT NULL,
    description         TEXT,
    policy_type         TEXT            NOT NULL CHECK (policy_type IN (
        'max_idle_days',
        'classification_restrict',
        'separation_of_duties',
        'time_bound_access',
        'blast_radius_limit',
        'periodic_review'
    )),
    -- Rule configuration
    rule_config         JSONB           NOT NULL,
    -- Enforcement
    enforcement         TEXT            NOT NULL DEFAULT 'alert'
                                        CHECK (enforcement IN ('alert', 'block', 'auto_remediate')),
    severity            TEXT            NOT NULL DEFAULT 'medium'
                                        CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    -- Status
    enabled             BOOLEAN         NOT NULL DEFAULT true,
    -- Metadata
    created_by          UUID,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),

    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_dspm_policies_tenant
    ON dspm_access_policies (tenant_id, enabled) WHERE enabled = true;
