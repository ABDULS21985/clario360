-- =============================================================================
-- Clario 360 — Tenant Onboarding, Provisioning, Invitations, Deprovisioning
-- Database: platform_core
-- =============================================================================

ALTER TYPE tenant_status ADD VALUE IF NOT EXISTS 'onboarding';
ALTER TYPE tenant_status ADD VALUE IF NOT EXISTS 'deprovisioned';

ALTER TYPE user_status ADD VALUE IF NOT EXISTS 'pending_verification';

ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS deprovisioned_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS deprovisioned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS retain_until TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS tenant_onboarding (
    id                        UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID            NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    admin_user_id             UUID            NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    admin_email               TEXT            NOT NULL,
    email_verified            BOOLEAN         NOT NULL DEFAULT false,
    email_verified_at         TIMESTAMPTZ,
    current_step              INT             NOT NULL DEFAULT 0,
    steps_completed           INT[]           NOT NULL DEFAULT '{}',
    wizard_completed          BOOLEAN         NOT NULL DEFAULT false,
    wizard_completed_at       TIMESTAMPTZ,
    org_name                  TEXT,
    org_industry              TEXT            CHECK (org_industry IN (
        'financial', 'government', 'healthcare', 'technology', 'energy',
        'telecom', 'education', 'retail', 'manufacturing', 'other'
    )),
    org_country               TEXT            NOT NULL DEFAULT 'SA',
    org_city                  TEXT,
    org_size                  TEXT            CHECK (org_size IN ('1-50', '51-200', '201-1000', '1000+')),
    logo_file_id              UUID,
    primary_color             TEXT,
    accent_color              TEXT,
    active_suites             TEXT[]          NOT NULL DEFAULT '{cyber,data,visus}',
    provisioning_status       TEXT            NOT NULL DEFAULT 'pending'
                                              CHECK (provisioning_status IN ('pending', 'provisioning', 'completed', 'failed')),
    provisioning_started_at   TIMESTAMPTZ,
    provisioning_completed_at TIMESTAMPTZ,
    provisioning_error        TEXT,
    referral_source           TEXT,
    created_at                TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_onboarding_tenant ON tenant_onboarding (tenant_id);
CREATE INDEX IF NOT EXISTS idx_onboarding_admin_email ON tenant_onboarding (admin_email);

CREATE TRIGGER trg_tenant_onboarding_updated_at
    BEFORE UPDATE ON tenant_onboarding
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS provisioning_steps (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID            NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    onboarding_id   UUID            NOT NULL REFERENCES tenant_onboarding(id) ON DELETE CASCADE,
    step_number     INT             NOT NULL,
    step_name       TEXT            NOT NULL,
    status          TEXT            NOT NULL DEFAULT 'pending'
                                      CHECK (status IN ('pending', 'running', 'completed', 'failed', 'skipped')),
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    duration_ms     BIGINT,
    error_message   TEXT,
    retry_count     INT             NOT NULL DEFAULT 0,
    idempotency_key TEXT,
    metadata        JSONB           NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    UNIQUE (onboarding_id, step_number)
);

CREATE INDEX IF NOT EXISTS idx_provisioning_steps_onboarding
    ON provisioning_steps (onboarding_id, step_number);

CREATE INDEX IF NOT EXISTS idx_provisioning_steps_tenant_status
    ON provisioning_steps (tenant_id, status, step_number);

CREATE TABLE IF NOT EXISTS email_verifications (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT            NOT NULL,
    otp_hash        TEXT            NOT NULL,
    purpose         TEXT            NOT NULL DEFAULT 'registration'
                                      CHECK (purpose IN ('registration', 'email_change', 'password_reset')),
    verified        BOOLEAN         NOT NULL DEFAULT false,
    attempts        INT             NOT NULL DEFAULT 0,
    max_attempts    INT             NOT NULL DEFAULT 5,
    locked_at       TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ     NOT NULL,
    verified_at     TIMESTAMPTZ,
    ip_address      TEXT,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_email_verifications_email
    ON email_verifications (email, purpose, created_at DESC);

CREATE TABLE IF NOT EXISTS invitations (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID            NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email           TEXT            NOT NULL,
    role_slug       TEXT            NOT NULL,
    token_hash      TEXT            NOT NULL,
    token_prefix    TEXT            NOT NULL,
    status          TEXT            NOT NULL DEFAULT 'pending'
                                      CHECK (status IN ('pending', 'accepted', 'expired', 'cancelled', 'revoked')),
    invited_by      UUID            NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    invited_by_name TEXT            NOT NULL,
    accepted_at     TIMESTAMPTZ,
    accepted_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    expires_at      TIMESTAMPTZ     NOT NULL,
    message         TEXT,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_invitations_pending_unique
    ON invitations (tenant_id, lower(email))
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_invitations_tenant
    ON invitations (tenant_id, status);

CREATE INDEX IF NOT EXISTS idx_invitations_token
    ON invitations (token_prefix)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_invitations_email
    ON invitations (email, status);

CREATE TRIGGER trg_invitations_updated_at
    BEFORE UPDATE ON invitations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
