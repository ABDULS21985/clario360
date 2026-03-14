-- =============================================================================
-- DSPM REMEDIATIONS — Tracked remediation work items
-- =============================================================================

CREATE TABLE IF NOT EXISTS dspm_remediations (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    -- Source finding
    finding_type        TEXT            NOT NULL CHECK (finding_type IN (
        'posture_gap',
        'overprivileged_access',
        'stale_access',
        'classification_drift',
        'shadow_copy',
        'policy_violation',
        'encryption_missing',
        'exposure_risk',
        'pii_unprotected',
        'retention_expired',
        'blast_radius_excessive'
    )),
    finding_id          UUID,
    -- Target
    data_asset_id       UUID            REFERENCES dspm_data_assets(id),
    data_asset_name     TEXT,
    identity_id         TEXT,
    -- Remediation plan
    playbook_id         TEXT            NOT NULL,
    title               TEXT            NOT NULL,
    description         TEXT            NOT NULL,
    severity            TEXT            NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    -- Steps
    steps               JSONB           NOT NULL DEFAULT '[]',
    current_step        INT             NOT NULL DEFAULT 0,
    total_steps         INT             NOT NULL DEFAULT 0,
    -- Assignment
    assigned_to         UUID,
    assigned_team       TEXT,
    -- SLA
    sla_due_at          TIMESTAMPTZ,
    sla_breached        BOOLEAN         NOT NULL DEFAULT false,
    -- Risk context
    risk_score_before   FLOAT,
    risk_score_after    FLOAT,
    risk_reduction      FLOAT,
    -- Rollback
    pre_action_state    JSONB,
    rollback_available  BOOLEAN         NOT NULL DEFAULT false,
    rolled_back         BOOLEAN         NOT NULL DEFAULT false,
    -- Status
    status              TEXT            NOT NULL DEFAULT 'open'
                                        CHECK (status IN (
                                            'open', 'in_progress', 'awaiting_approval',
                                            'completed', 'failed', 'cancelled',
                                            'rolled_back', 'exception_granted'
                                        )),
    -- Linked alerts
    cyber_alert_id      UUID,
    -- Metadata
    created_by          UUID,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    completed_at        TIMESTAMPTZ,
    compliance_tags     JSONB           NOT NULL DEFAULT '[]'
);

CREATE INDEX idx_dspm_remediation_tenant   ON dspm_remediations (tenant_id, status, created_at DESC);
CREATE INDEX idx_dspm_remediation_asset    ON dspm_remediations (tenant_id, data_asset_id);
CREATE INDEX idx_dspm_remediation_assignee ON dspm_remediations (tenant_id, assigned_to) WHERE status IN ('open', 'in_progress');
CREATE INDEX idx_dspm_remediation_sla      ON dspm_remediations (tenant_id, sla_due_at) WHERE status NOT IN ('completed', 'cancelled') AND sla_breached = false;
CREATE INDEX idx_dspm_remediation_severity ON dspm_remediations (tenant_id, severity, status);

-- =============================================================================
-- DSPM REMEDIATION HISTORY — Tamper-evident audit trail
-- =============================================================================

CREATE TABLE IF NOT EXISTS dspm_remediation_history (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    remediation_id      UUID            NOT NULL REFERENCES dspm_remediations(id),
    -- Action
    action              TEXT            NOT NULL,
    actor_id            UUID,
    actor_type          TEXT            NOT NULL DEFAULT 'system'
                                        CHECK (actor_type IN ('user', 'system', 'policy_engine', 'scheduler')),
    -- Details
    details             JSONB           NOT NULL DEFAULT '{}',
    -- Integrity
    entry_hash          TEXT            NOT NULL,
    prev_hash           TEXT,
    -- Timestamp
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX idx_dspm_history_remediation  ON dspm_remediation_history (tenant_id, remediation_id, created_at);

-- =============================================================================
-- DSPM DATA POLICIES — Policy-as-code definitions
-- =============================================================================

CREATE TABLE IF NOT EXISTS dspm_data_policies (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    -- Policy
    name                TEXT            NOT NULL,
    description         TEXT,
    category            TEXT            NOT NULL CHECK (category IN (
        'encryption',
        'classification',
        'retention',
        'exposure',
        'pii_protection',
        'access_review',
        'backup',
        'audit_logging'
    )),
    -- Rule (JSONB — varies by category)
    rule                JSONB           NOT NULL,
    -- Enforcement
    enforcement         TEXT            NOT NULL DEFAULT 'alert'
                                        CHECK (enforcement IN ('alert', 'auto_remediate', 'block')),
    auto_playbook_id    TEXT,
    severity            TEXT            NOT NULL DEFAULT 'medium'
                                        CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    -- Scope
    scope_classification TEXT[],
    scope_asset_types   TEXT[],
    -- Status
    enabled             BOOLEAN         NOT NULL DEFAULT true,
    last_evaluated_at   TIMESTAMPTZ,
    violation_count     INT             NOT NULL DEFAULT 0,
    -- Compliance
    compliance_frameworks TEXT[],
    -- Metadata
    created_by          UUID,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),

    UNIQUE (tenant_id, name)
);

CREATE INDEX idx_dspm_data_policies_tenant ON dspm_data_policies (tenant_id, enabled) WHERE enabled = true;

-- =============================================================================
-- DSPM RISK EXCEPTIONS — Accepted risks with governance
-- =============================================================================

CREATE TABLE IF NOT EXISTS dspm_risk_exceptions (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    -- What is being excepted
    exception_type      TEXT            NOT NULL CHECK (exception_type IN (
        'posture_finding',
        'policy_violation',
        'overprivileged_access',
        'exposure_risk',
        'encryption_gap'
    )),
    -- Reference
    remediation_id      UUID            REFERENCES dspm_remediations(id),
    data_asset_id       UUID            REFERENCES dspm_data_assets(id),
    policy_id           UUID            REFERENCES dspm_data_policies(id),
    -- Justification
    justification       TEXT            NOT NULL,
    business_reason     TEXT,
    compensating_controls TEXT,
    -- Risk context
    risk_score          FLOAT           NOT NULL,
    risk_level          TEXT            NOT NULL CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    -- Approval
    requested_by        UUID            NOT NULL,
    approved_by         UUID,
    approval_status     TEXT            NOT NULL DEFAULT 'pending'
                                        CHECK (approval_status IN ('pending', 'approved', 'rejected', 'expired')),
    approved_at         TIMESTAMPTZ,
    rejection_reason    TEXT,
    -- Lifecycle
    expires_at          TIMESTAMPTZ     NOT NULL,
    review_interval_days INT            NOT NULL DEFAULT 90,
    next_review_at      TIMESTAMPTZ,
    last_reviewed_at    TIMESTAMPTZ,
    review_count        INT             NOT NULL DEFAULT 0,
    -- Status
    status              TEXT            NOT NULL DEFAULT 'active'
                                        CHECK (status IN ('active', 'expired', 'revoked', 'superseded')),
    -- Metadata
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX idx_dspm_exceptions_tenant    ON dspm_risk_exceptions (tenant_id, status, expires_at);
CREATE INDEX idx_dspm_exceptions_asset     ON dspm_risk_exceptions (tenant_id, data_asset_id) WHERE status = 'active';
CREATE INDEX idx_dspm_exceptions_review    ON dspm_risk_exceptions (tenant_id, next_review_at) WHERE status = 'active';
