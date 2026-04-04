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
