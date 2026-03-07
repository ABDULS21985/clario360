-- =============================================================================
-- REMEDIATION ACTIONS — Governed remediation lifecycle records
-- =============================================================================

CREATE TABLE IF NOT EXISTS remediation_actions (
    id                      UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID            NOT NULL,
    alert_id                UUID            REFERENCES alerts(id),
    vulnerability_id        UUID            REFERENCES vulnerabilities(id),
    assessment_id           UUID            REFERENCES ctem_assessments(id),
    ctem_finding_id         UUID            REFERENCES ctem_findings(id),
    remediation_group_id    UUID            REFERENCES ctem_remediation_groups(id),
    type                    TEXT            NOT NULL CHECK (type IN (
        'patch', 'config_change', 'block_ip', 'isolate_asset',
        'firewall_rule', 'access_revoke', 'certificate_renew', 'custom'
    )),
    severity                TEXT            NOT NULL DEFAULT 'medium'
                                            CHECK (severity IN ('critical', 'high', 'medium', 'low')),
    title                   TEXT            NOT NULL,
    description             TEXT            NOT NULL DEFAULT '',
    plan                    JSONB           NOT NULL,
    affected_asset_ids      UUID[]          NOT NULL DEFAULT '{}',
    affected_asset_count    INT             NOT NULL DEFAULT 0,
    execution_mode          TEXT            NOT NULL DEFAULT 'manual'
                                            CHECK (execution_mode IN ('manual', 'semi_automated', 'automated')),
    status                  TEXT            NOT NULL DEFAULT 'pending_approval'
                                            CHECK (status IN (
        'draft', 'pending_approval', 'approved', 'rejected', 'revision_requested',
        'dry_run_running', 'dry_run_completed', 'dry_run_failed',
        'execution_pending', 'executing', 'executed', 'execution_failed',
        'verification_pending', 'verified', 'verification_failed',
        'rollback_pending', 'rolling_back', 'rolled_back', 'rollback_failed', 'closed'
    )),
    submitted_by            UUID,
    submitted_at            TIMESTAMPTZ,
    approved_by             UUID,
    approved_at             TIMESTAMPTZ,
    rejected_by             UUID,
    rejected_at             TIMESTAMPTZ,
    rejection_reason        TEXT,
    approval_notes          TEXT,
    requires_approval_from  TEXT            NOT NULL DEFAULT 'security_manager'
                                            CHECK (requires_approval_from IN (
        'security_manager', 'ciso', 'tenant_admin'
    )),
    dry_run_result          JSONB,
    dry_run_at              TIMESTAMPTZ,
    dry_run_duration_ms     BIGINT,
    pre_execution_state     JSONB,
    execution_result        JSONB,
    executed_by             UUID,
    execution_started_at    TIMESTAMPTZ,
    execution_completed_at  TIMESTAMPTZ,
    execution_duration_ms   BIGINT,
    verification_result     JSONB,
    verified_by             UUID,
    verified_at             TIMESTAMPTZ,
    rollback_result         JSONB,
    rollback_reason         TEXT,
    rollback_approved_by    UUID,
    rolled_back_at          TIMESTAMPTZ,
    rollback_deadline       TIMESTAMPTZ,
    workflow_instance_id    UUID,
    tags                    TEXT[]          NOT NULL DEFAULT '{}',
    metadata                JSONB           NOT NULL DEFAULT '{}',
    created_by              UUID            NOT NULL,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT now(),
    deleted_at              TIMESTAMPTZ
);

CREATE INDEX idx_remediation_tenant_status ON remediation_actions (tenant_id, status, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_remediation_tenant_type   ON remediation_actions (tenant_id, type) WHERE deleted_at IS NULL;
CREATE INDEX idx_remediation_alert         ON remediation_actions (alert_id) WHERE alert_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_remediation_vuln          ON remediation_actions (vulnerability_id) WHERE vulnerability_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_remediation_assessment    ON remediation_actions (assessment_id) WHERE assessment_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_remediation_workflow      ON remediation_actions (workflow_instance_id) WHERE workflow_instance_id IS NOT NULL;
CREATE INDEX idx_remediation_assets        ON remediation_actions USING GIN (affected_asset_ids) WHERE deleted_at IS NULL;
CREATE INDEX idx_remediation_rollback_deadline ON remediation_actions (tenant_id, rollback_deadline)
    WHERE status = 'executed' AND rollback_deadline > now() AND deleted_at IS NULL;

-- =============================================================================
-- REMEDIATION AUDIT TRAIL — Immutable log of every action
-- =============================================================================

CREATE TABLE IF NOT EXISTS remediation_audit_trail (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID            NOT NULL,
    remediation_id  UUID            NOT NULL REFERENCES remediation_actions(id) ON DELETE CASCADE,
    action          TEXT            NOT NULL,
    actor_id        UUID,
    actor_name      TEXT,
    old_status      TEXT,
    new_status      TEXT,
    step_number     INT,
    step_action     TEXT,
    step_result     TEXT            CHECK (step_result IN ('success', 'failure', 'skipped', 'warning')),
    details         JSONB           NOT NULL DEFAULT '{}',
    error_message   TEXT,
    duration_ms     BIGINT,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX idx_remediation_audit_trail  ON remediation_audit_trail (remediation_id, created_at ASC);
CREATE INDEX idx_remediation_audit_tenant ON remediation_audit_trail (tenant_id, created_at DESC);

-- =============================================================================
-- DSPM DATA ASSETS
-- =============================================================================

CREATE TABLE IF NOT EXISTS dspm_data_assets (
    id                      UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID            NOT NULL,
    asset_id                UUID            NOT NULL REFERENCES assets(id),
    scan_id                 UUID,
    data_classification     TEXT            NOT NULL DEFAULT 'internal'
                                            CHECK (data_classification IN ('public', 'internal', 'confidential', 'restricted')),
    sensitivity_score       DECIMAL(5,2)    NOT NULL DEFAULT 0,
    contains_pii            BOOLEAN         NOT NULL DEFAULT false,
    pii_types               TEXT[]          NOT NULL DEFAULT '{}',
    pii_column_count        INT             NOT NULL DEFAULT 0,
    estimated_record_count  BIGINT,
    encrypted_at_rest       BOOLEAN,
    encrypted_in_transit    BOOLEAN,
    access_control_type     TEXT            CHECK (access_control_type IN ('none', 'basic', 'rbac', 'abac')),
    network_exposure        TEXT            CHECK (network_exposure IN ('internal_only', 'vpn_accessible', 'internet_facing')),
    backup_configured       BOOLEAN,
    audit_logging           BOOLEAN,
    last_access_review      TIMESTAMPTZ,
    risk_score              DECIMAL(5,2)    NOT NULL DEFAULT 0,
    risk_factors            JSONB           NOT NULL DEFAULT '[]',
    posture_score           DECIMAL(5,2)    NOT NULL DEFAULT 0,
    posture_findings        JSONB           NOT NULL DEFAULT '[]',
    consumer_count          INT             NOT NULL DEFAULT 0,
    producer_count          INT             NOT NULL DEFAULT 0,
    database_type           TEXT,
    schema_info             JSONB,
    metadata                JSONB           NOT NULL DEFAULT '{}',
    last_scanned_at         TIMESTAMPTZ,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX idx_dspm_tenant         ON dspm_data_assets (tenant_id, risk_score DESC);
CREATE INDEX idx_dspm_classification ON dspm_data_assets (tenant_id, data_classification);
CREATE INDEX idx_dspm_pii            ON dspm_data_assets (tenant_id, contains_pii) WHERE contains_pii = true;
CREATE INDEX idx_dspm_asset          ON dspm_data_assets (asset_id);

-- =============================================================================
-- DSPM SCAN HISTORY
-- =============================================================================

CREATE TABLE IF NOT EXISTS dspm_scans (
    id               UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID            NOT NULL,
    status           TEXT            NOT NULL DEFAULT 'running'
                                     CHECK (status IN ('running', 'completed', 'failed')),
    assets_scanned   INT             NOT NULL DEFAULT 0,
    pii_assets_found INT             NOT NULL DEFAULT 0,
    high_risk_found  INT             NOT NULL DEFAULT 0,
    findings_count   INT             NOT NULL DEFAULT 0,
    started_at       TIMESTAMPTZ     NOT NULL DEFAULT now(),
    completed_at     TIMESTAMPTZ,
    duration_ms      BIGINT,
    created_by       UUID            NOT NULL,
    created_at       TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX idx_dspm_scan_tenant ON dspm_scans (tenant_id, created_at DESC);

-- =============================================================================
-- VCISO BRIEFING HISTORY
-- =============================================================================

CREATE TABLE IF NOT EXISTS vciso_briefings (
    id                 UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID            NOT NULL,
    type               TEXT            NOT NULL CHECK (type IN ('executive', 'technical', 'compliance', 'custom')),
    period_start       DATE            NOT NULL,
    period_end         DATE            NOT NULL,
    content            JSONB           NOT NULL,
    risk_score_at_time DECIMAL(5,2),
    generated_by       UUID            NOT NULL,
    created_at         TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX idx_vciso_briefing_tenant ON vciso_briefings (tenant_id, created_at DESC);
