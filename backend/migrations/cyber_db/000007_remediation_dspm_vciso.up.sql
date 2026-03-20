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
    type                    TEXT            NOT NULL DEFAULT 'custom',
    severity                TEXT            NOT NULL DEFAULT 'medium',
    title                   TEXT            NOT NULL DEFAULT '',
    description             TEXT            NOT NULL DEFAULT '',
    plan                    JSONB           NOT NULL DEFAULT '{"steps":[{"number":1,"action":"manual_review","description":"Legacy remediation migrated to governed workflow","expected":"Review completed"}],"reversible":true,"requires_reboot":false,"estimated_downtime":"0","risk_level":"medium"}',
    affected_asset_ids      UUID[]          NOT NULL DEFAULT '{}',
    affected_asset_count    INT             NOT NULL DEFAULT 0,
    execution_mode          TEXT            NOT NULL DEFAULT 'manual',
    status                  TEXT            NOT NULL DEFAULT 'pending_approval',
    submitted_by            UUID,
    submitted_at            TIMESTAMPTZ,
    approved_by             UUID,
    approved_at             TIMESTAMPTZ,
    rejected_by             UUID,
    rejected_at             TIMESTAMPTZ,
    rejection_reason        TEXT,
    approval_notes          TEXT,
    requires_approval_from  TEXT            NOT NULL DEFAULT 'security_manager',
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
    created_by              UUID            NOT NULL DEFAULT gen_random_uuid(),
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT now(),
    deleted_at              TIMESTAMPTZ
);

ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS assessment_id UUID REFERENCES ctem_assessments(id);
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS ctem_finding_id UUID REFERENCES ctem_findings(id);
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS remediation_group_id UUID REFERENCES ctem_remediation_groups(id);
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS severity TEXT DEFAULT 'medium';
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS title TEXT DEFAULT '';
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS description TEXT DEFAULT '';
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS plan JSONB DEFAULT '{"steps":[{"number":1,"action":"manual_review","description":"Legacy remediation migrated to governed workflow","expected":"Review completed"}],"reversible":true,"requires_reboot":false,"estimated_downtime":"0","risk_level":"medium"}';
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS affected_asset_ids UUID[] DEFAULT '{}';
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS affected_asset_count INT DEFAULT 0;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS submitted_by UUID;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS submitted_at TIMESTAMPTZ;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS approved_at TIMESTAMPTZ;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS rejected_by UUID;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS rejected_at TIMESTAMPTZ;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS rejection_reason TEXT;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS approval_notes TEXT;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS requires_approval_from TEXT DEFAULT 'security_manager';
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS dry_run_at TIMESTAMPTZ;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS dry_run_duration_ms BIGINT;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS pre_execution_state JSONB;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS execution_started_at TIMESTAMPTZ;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS execution_completed_at TIMESTAMPTZ;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS execution_duration_ms BIGINT;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS verification_result JSONB;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS verified_by UUID;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS verified_at TIMESTAMPTZ;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS rollback_result JSONB;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS rollback_reason TEXT;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS rollback_approved_by UUID;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS rolled_back_at TIMESTAMPTZ;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS rollback_deadline TIMESTAMPTZ;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS workflow_instance_id UUID;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS tags TEXT[] DEFAULT '{}';
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS rollback_data JSONB;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS executed_at TIMESTAMPTZ;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS updated_by UUID;

ALTER TABLE remediation_actions
    ALTER COLUMN type TYPE TEXT USING (
        CASE type::text
            WHEN 'block' THEN 'block_ip'
            WHEN 'isolate' THEN 'isolate_asset'
            ELSE type::text
        END
    ),
    ALTER COLUMN status TYPE TEXT USING (
        CASE status::text
            WHEN 'dry_run' THEN 'dry_run_completed'
            WHEN 'completed' THEN 'executed'
            WHEN 'failed' THEN 'execution_failed'
            ELSE status::text
        END
    ),
    ALTER COLUMN execution_mode TYPE TEXT USING (
        CASE execution_mode::text
            WHEN 'semi_auto' THEN 'semi_automated'
            WHEN 'auto' THEN 'automated'
            ELSE execution_mode::text
        END
    );

UPDATE remediation_actions
SET title = CASE
        WHEN COALESCE(title, '') <> '' THEN title
        ELSE initcap(replace(type, '_', ' ')) || ' remediation'
    END,
    description = COALESCE(NULLIF(description, ''), 'Migrated remediation action awaiting governed review'),
    plan = COALESCE(plan, '{"steps":[{"number":1,"action":"manual_review","description":"Legacy remediation migrated to governed workflow","expected":"Review completed"}],"reversible":true,"requires_reboot":false,"estimated_downtime":"0","risk_level":"medium"}'::jsonb),
    severity = COALESCE(NULLIF(severity, ''),
        (SELECT severity::text FROM alerts WHERE id = remediation_actions.alert_id),
        (SELECT severity::text FROM vulnerabilities WHERE id = remediation_actions.vulnerability_id),
        'medium'
    ),
    affected_asset_ids = COALESCE(
        NULLIF(affected_asset_ids, '{}'::uuid[]),
        ARRAY_REMOVE(ARRAY[
            (SELECT asset_id FROM alerts WHERE id = remediation_actions.alert_id),
            (SELECT asset_id FROM vulnerabilities WHERE id = remediation_actions.vulnerability_id)
        ], NULL),
        '{}'::uuid[]
    ),
    requires_approval_from = COALESCE(NULLIF(requires_approval_from, ''), 'security_manager'),
    tags = COALESCE(tags, '{}'::text[]),
    metadata = COALESCE(metadata, '{}'::jsonb),
    pre_execution_state = COALESCE(pre_execution_state, rollback_data),
    submitted_by = COALESCE(submitted_by, created_by),
    submitted_at = COALESCE(submitted_at, created_at),
    approved_at = COALESCE(approved_at, CASE WHEN approved_by IS NOT NULL THEN updated_at END),
    dry_run_at = COALESCE(dry_run_at, CASE WHEN dry_run_result IS NOT NULL THEN updated_at END),
    execution_started_at = COALESCE(execution_started_at, executed_at),
    execution_completed_at = COALESCE(execution_completed_at, completed_at, executed_at),
    execution_duration_ms = COALESCE(
        execution_duration_ms,
        CASE
            WHEN execution_started_at IS NOT NULL AND COALESCE(execution_completed_at, completed_at, executed_at) IS NOT NULL
            THEN GREATEST(0, (EXTRACT(EPOCH FROM (COALESCE(execution_completed_at, completed_at, executed_at) - execution_started_at)) * 1000)::bigint)
        END
    ),
    rollback_result = COALESCE(
        rollback_result,
        CASE
            WHEN status = 'rolled_back' THEN '{"success":true,"duration_ms":0,"steps_reverted":0}'::jsonb
        END
    ),
    rolled_back_at = COALESCE(rolled_back_at, CASE WHEN status = 'rolled_back' THEN completed_at END),
    rollback_deadline = COALESCE(
        rollback_deadline,
        CASE
            WHEN COALESCE(execution_completed_at, completed_at, executed_at) IS NOT NULL
            THEN COALESCE(execution_completed_at, completed_at, executed_at) + interval '72 hours'
        END
    ),
    created_by = COALESCE(created_by, gen_random_uuid());

UPDATE remediation_actions
SET affected_asset_count = COALESCE(cardinality(affected_asset_ids), 0);

ALTER TABLE remediation_actions ALTER COLUMN severity SET DEFAULT 'medium';
ALTER TABLE remediation_actions ALTER COLUMN title SET DEFAULT '';
ALTER TABLE remediation_actions ALTER COLUMN description SET DEFAULT '';
ALTER TABLE remediation_actions ALTER COLUMN plan SET DEFAULT '{"steps":[{"number":1,"action":"manual_review","description":"Legacy remediation migrated to governed workflow","expected":"Review completed"}],"reversible":true,"requires_reboot":false,"estimated_downtime":"0","risk_level":"medium"}';
ALTER TABLE remediation_actions ALTER COLUMN affected_asset_ids SET DEFAULT '{}';
ALTER TABLE remediation_actions ALTER COLUMN affected_asset_count SET DEFAULT 0;
ALTER TABLE remediation_actions ALTER COLUMN execution_mode SET DEFAULT 'manual';
ALTER TABLE remediation_actions ALTER COLUMN status SET DEFAULT 'pending_approval';
ALTER TABLE remediation_actions ALTER COLUMN requires_approval_from SET DEFAULT 'security_manager';
ALTER TABLE remediation_actions ALTER COLUMN tags SET DEFAULT '{}';
ALTER TABLE remediation_actions ALTER COLUMN metadata SET DEFAULT '{}';

ALTER TABLE remediation_actions ALTER COLUMN severity SET NOT NULL;
ALTER TABLE remediation_actions ALTER COLUMN title SET NOT NULL;
ALTER TABLE remediation_actions ALTER COLUMN description SET NOT NULL;
ALTER TABLE remediation_actions ALTER COLUMN plan SET NOT NULL;
ALTER TABLE remediation_actions ALTER COLUMN affected_asset_ids SET NOT NULL;
ALTER TABLE remediation_actions ALTER COLUMN affected_asset_count SET NOT NULL;
ALTER TABLE remediation_actions ALTER COLUMN execution_mode SET NOT NULL;
ALTER TABLE remediation_actions ALTER COLUMN status SET NOT NULL;
ALTER TABLE remediation_actions ALTER COLUMN requires_approval_from SET NOT NULL;
ALTER TABLE remediation_actions ALTER COLUMN tags SET NOT NULL;
ALTER TABLE remediation_actions ALTER COLUMN metadata SET NOT NULL;
ALTER TABLE remediation_actions ALTER COLUMN created_by SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_remediation_actions_type_prompt20') THEN
        ALTER TABLE remediation_actions
            ADD CONSTRAINT chk_remediation_actions_type_prompt20 CHECK (type IN (
                'patch', 'config_change', 'block_ip', 'isolate_asset',
                'firewall_rule', 'access_revoke', 'certificate_renew', 'custom'
            ));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_remediation_actions_severity_prompt20') THEN
        ALTER TABLE remediation_actions
            ADD CONSTRAINT chk_remediation_actions_severity_prompt20 CHECK (severity IN ('critical', 'high', 'medium', 'low'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_remediation_actions_execution_mode_prompt20') THEN
        ALTER TABLE remediation_actions
            ADD CONSTRAINT chk_remediation_actions_execution_mode_prompt20 CHECK (execution_mode IN ('manual', 'semi_automated', 'automated'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_remediation_actions_status_prompt20') THEN
        ALTER TABLE remediation_actions
            ADD CONSTRAINT chk_remediation_actions_status_prompt20 CHECK (status IN (
                'draft', 'pending_approval', 'approved', 'rejected', 'revision_requested',
                'dry_run_running', 'dry_run_completed', 'dry_run_failed',
                'execution_pending', 'executing', 'executed', 'execution_failed',
                'verification_pending', 'verified', 'verification_failed',
                'rollback_pending', 'rolling_back', 'rolled_back', 'rollback_failed', 'closed'
            ));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_remediation_actions_requires_approval_prompt20') THEN
        ALTER TABLE remediation_actions
            ADD CONSTRAINT chk_remediation_actions_requires_approval_prompt20 CHECK (requires_approval_from IN (
                'security_manager', 'ciso', 'tenant_admin'
            ));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_remediation_tenant_status_active ON remediation_actions (tenant_id, status, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_remediation_tenant_type_active ON remediation_actions (tenant_id, type) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_remediation_alert_active ON remediation_actions (alert_id) WHERE alert_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_remediation_vuln_active ON remediation_actions (vulnerability_id) WHERE vulnerability_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_remediation_assessment_active ON remediation_actions (assessment_id) WHERE assessment_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_remediation_workflow_active ON remediation_actions (workflow_instance_id) WHERE workflow_instance_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_remediation_assets_gin ON remediation_actions USING GIN (affected_asset_ids) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_remediation_rollback_deadline_active ON remediation_actions (tenant_id, rollback_deadline)
    WHERE status = 'executed' AND rollback_deadline IS NOT NULL AND deleted_at IS NULL;

-- =============================================================================
-- REMEDIATION AUDIT TRAIL — Immutable log of every action in the remediation lifecycle
-- =============================================================================

CREATE TABLE IF NOT EXISTS remediation_audit_trail (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    remediation_id      UUID            NOT NULL REFERENCES remediation_actions(id) ON DELETE CASCADE,
    action              TEXT            NOT NULL,
    actor_id            UUID,
    actor_name          TEXT,
    old_status          TEXT,
    new_status          TEXT,
    step_number         INT,
    step_action         TEXT,
    step_result         TEXT            CHECK (step_result IN ('success', 'failure', 'skipped', 'warning')),
    details             JSONB           NOT NULL DEFAULT '{}',
    error_message       TEXT,
    duration_ms         BIGINT,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_remediation_audit_trail ON remediation_audit_trail (remediation_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_remediation_audit_tenant ON remediation_audit_trail (tenant_id, created_at DESC);

-- =============================================================================
-- DSPM DATA ASSETS — Data-centric asset view for security posture
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

ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS asset_id UUID REFERENCES assets(id);
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS scan_id UUID;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS data_classification TEXT DEFAULT 'internal';
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS name TEXT;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS type TEXT;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS location TEXT;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS classification TEXT;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS owner UUID;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS data_types TEXT[] DEFAULT '{}';
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS contains_pii BOOLEAN DEFAULT false;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS pii_types TEXT[] DEFAULT '{}';
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS pii_column_count INT DEFAULT 0;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS estimated_record_count BIGINT;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS encrypted_at_rest BOOLEAN;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS encrypted_in_transit BOOLEAN;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS access_control_type TEXT;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS network_exposure TEXT;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS backup_configured BOOLEAN;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS audit_logging BOOLEAN;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS last_access_review TIMESTAMPTZ;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS risk_factors JSONB DEFAULT '[]';
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS posture_score DECIMAL(5,2) DEFAULT 0;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS posture_findings JSONB DEFAULT '[]';
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS consumer_count INT DEFAULT 0;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS producer_count INT DEFAULT 0;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS database_type TEXT;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS schema_info JSONB;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS last_scanned_at TIMESTAMPTZ;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS created_by UUID;
ALTER TABLE dspm_data_assets ADD COLUMN IF NOT EXISTS updated_by UUID;

UPDATE dspm_data_assets
SET data_classification = COALESCE(data_classification, classification::text, 'internal'),
    pii_types = COALESCE(NULLIF(pii_types, '{}'::text[]), data_types, '{}'::text[]),
    pii_column_count = COALESCE(pii_column_count, COALESCE(cardinality(data_types), 0)),
    contains_pii = COALESCE(contains_pii, COALESCE(cardinality(data_types), 0) > 0),
    sensitivity_score = CASE
        WHEN sensitivity_score IS NULL THEN 0
        WHEN sensitivity_score <= 1 THEN ROUND((sensitivity_score * 100)::numeric, 2)
        ELSE sensitivity_score
    END,
    risk_score = CASE
        WHEN risk_score IS NULL THEN 0
        WHEN risk_score <= 1 THEN ROUND((risk_score * 100)::numeric, 2)
        ELSE risk_score
    END,
    posture_score = COALESCE(posture_score, 0),
    risk_factors = COALESCE(risk_factors, '[]'::jsonb),
    posture_findings = COALESCE(posture_findings, '[]'::jsonb),
    metadata = COALESCE(metadata, '{}'::jsonb),
    database_type = COALESCE(database_type, NULLIF(type, '')),
    last_scanned_at = COALESCE(last_scanned_at, updated_at, created_at);

UPDATE dspm_data_assets da
SET asset_id = COALESCE(
    da.asset_id,
    (
        SELECT a.id
        FROM assets a
        WHERE a.tenant_id = da.tenant_id
          AND lower(a.name) = lower(da.name)
          AND a.deleted_at IS NULL
        ORDER BY a.created_at ASC
        LIMIT 1
    )
)
WHERE da.asset_id IS NULL;

ALTER TABLE dspm_data_assets ALTER COLUMN data_classification SET DEFAULT 'internal';
ALTER TABLE dspm_data_assets ALTER COLUMN contains_pii SET DEFAULT false;
ALTER TABLE dspm_data_assets ALTER COLUMN pii_types SET DEFAULT '{}';
ALTER TABLE dspm_data_assets ALTER COLUMN pii_column_count SET DEFAULT 0;
ALTER TABLE dspm_data_assets ALTER COLUMN risk_factors SET DEFAULT '[]';
ALTER TABLE dspm_data_assets ALTER COLUMN posture_score SET DEFAULT 0;
ALTER TABLE dspm_data_assets ALTER COLUMN posture_findings SET DEFAULT '[]';
ALTER TABLE dspm_data_assets ALTER COLUMN consumer_count SET DEFAULT 0;
ALTER TABLE dspm_data_assets ALTER COLUMN producer_count SET DEFAULT 0;
ALTER TABLE dspm_data_assets ALTER COLUMN metadata SET DEFAULT '{}';

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_dspm_data_assets_classification_prompt20') THEN
        ALTER TABLE dspm_data_assets
            ADD CONSTRAINT chk_dspm_data_assets_classification_prompt20 CHECK (data_classification IN ('public', 'internal', 'confidential', 'restricted'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_dspm_data_assets_access_control_prompt20') THEN
        ALTER TABLE dspm_data_assets
            ADD CONSTRAINT chk_dspm_data_assets_access_control_prompt20 CHECK (access_control_type IS NULL OR access_control_type IN ('none', 'basic', 'rbac', 'abac'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_dspm_data_assets_network_exposure_prompt20') THEN
        ALTER TABLE dspm_data_assets
            ADD CONSTRAINT chk_dspm_data_assets_network_exposure_prompt20 CHECK (network_exposure IS NULL OR network_exposure IN ('internal_only', 'vpn_accessible', 'internet_facing'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_dspm_tenant_risk_desc ON dspm_data_assets (tenant_id, risk_score DESC);
CREATE INDEX IF NOT EXISTS idx_dspm_data_classification_v2 ON dspm_data_assets (tenant_id, data_classification);
CREATE INDEX IF NOT EXISTS idx_dspm_contains_pii_v2 ON dspm_data_assets (tenant_id, contains_pii) WHERE contains_pii = true;
CREATE INDEX IF NOT EXISTS idx_dspm_asset_v2 ON dspm_data_assets (asset_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_dspm_tenant_asset_unique ON dspm_data_assets (tenant_id, asset_id);

-- =============================================================================
-- DSPM SCAN HISTORY
-- =============================================================================

CREATE TABLE IF NOT EXISTS dspm_scans (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    status              TEXT            NOT NULL DEFAULT 'running'
                                        CHECK (status IN ('running', 'completed', 'failed')),
    assets_scanned      INT             NOT NULL DEFAULT 0,
    pii_assets_found    INT             NOT NULL DEFAULT 0,
    high_risk_found     INT             NOT NULL DEFAULT 0,
    findings_count      INT             NOT NULL DEFAULT 0,
    started_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    completed_at        TIMESTAMPTZ,
    duration_ms         BIGINT,
    created_by          UUID            NOT NULL,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_dspm_scan_tenant ON dspm_scans (tenant_id, created_at DESC);

-- =============================================================================
-- VCISO BRIEFING HISTORY — Archived executive briefings
-- =============================================================================

CREATE TABLE IF NOT EXISTS vciso_briefings (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID            NOT NULL,
    type                TEXT            NOT NULL CHECK (type IN ('executive', 'technical', 'compliance', 'custom')),
    period_start        DATE            NOT NULL,
    period_end          DATE            NOT NULL,
    content             JSONB           NOT NULL,
    risk_score_at_time  DECIMAL(5,2),
    generated_by        UUID            NOT NULL,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_vciso_briefing_tenant ON vciso_briefings (tenant_id, created_at DESC);
