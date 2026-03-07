-- =============================================================================
-- CTEM assessment engine schema upgrade
-- =============================================================================

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'ctem_assessments'
          AND column_name = 'status'
          AND udt_name = 'ctem_status'
    ) THEN
        ALTER TABLE ctem_assessments
            ALTER COLUMN status DROP DEFAULT;

        ALTER TABLE ctem_assessments
            ALTER COLUMN status TYPE TEXT
            USING CASE status::text
                WHEN 'scheduled' THEN 'created'
                WHEN 'running' THEN 'discovery'
                WHEN 'completed' THEN 'completed'
                WHEN 'failed' THEN 'failed'
                ELSE 'created'
            END;
    END IF;
END $$;

ALTER TABLE IF EXISTS ctem_assessments
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS resolved_asset_ids UUID[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS resolved_asset_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS phases JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS current_phase TEXT,
    ADD COLUMN IF NOT EXISTS exposure_score DECIMAL(5,2),
    ADD COLUMN IF NOT EXISTS score_breakdown JSONB,
    ADD COLUMN IF NOT EXISTS findings_summary JSONB,
    ADD COLUMN IF NOT EXISTS duration_ms BIGINT,
    ADD COLUMN IF NOT EXISTS error_message TEXT,
    ADD COLUMN IF NOT EXISTS error_phase TEXT,
    ADD COLUMN IF NOT EXISTS scheduled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS schedule_cron TEXT,
    ADD COLUMN IF NOT EXISTS parent_assessment_id UUID REFERENCES ctem_assessments(id),
    ADD COLUMN IF NOT EXISTS tags TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE IF EXISTS ctem_assessments
    ALTER COLUMN scope SET DEFAULT '{}',
    ALTER COLUMN status SET DEFAULT 'created';

ALTER TABLE IF EXISTS ctem_assessments
    DROP CONSTRAINT IF EXISTS ctem_assessments_status_check;

ALTER TABLE IF EXISTS ctem_assessments
    ADD CONSTRAINT ctem_assessments_status_check
    CHECK (status IN (
        'created', 'scoping', 'discovery', 'prioritizing',
        'validating', 'mobilizing', 'completed', 'failed', 'cancelled'
    ));

CREATE INDEX IF NOT EXISTS idx_ctem_assessment_tenant
    ON ctem_assessments (tenant_id, status, created_at DESC) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ctem_assessment_scheduled
    ON ctem_assessments (tenant_id, scheduled, schedule_cron)
    WHERE scheduled = true AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ctem_assessment_tags
    ON ctem_assessments USING GIN (tags) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ctem_assessment_scope
    ON ctem_assessments USING GIN (scope) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS ctem_findings (
    id                      UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID            NOT NULL,
    assessment_id           UUID            NOT NULL REFERENCES ctem_assessments(id) ON DELETE CASCADE,
    type                    TEXT            NOT NULL CHECK (type IN (
        'vulnerability', 'misconfiguration', 'attack_path', 'exposure',
        'weak_credential', 'missing_patch', 'expired_certificate', 'insecure_protocol'
    )),
    category                TEXT            NOT NULL DEFAULT 'technical'
                                            CHECK (category IN ('technical', 'configuration', 'architectural', 'operational')),
    severity                TEXT            NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info')),
    title                   TEXT            NOT NULL,
    description             TEXT            NOT NULL,
    evidence                JSONB           NOT NULL DEFAULT '{}',
    affected_asset_ids      UUID[]          NOT NULL DEFAULT '{}',
    affected_asset_count    INT             NOT NULL DEFAULT 0,
    primary_asset_id        UUID            REFERENCES assets(id),
    vulnerability_ids       UUID[]          NOT NULL DEFAULT '{}',
    cve_ids                 TEXT[]          NOT NULL DEFAULT '{}',
    business_impact_score   DECIMAL(5,2)    NOT NULL DEFAULT 0,
    business_impact_factors JSONB           NOT NULL DEFAULT '[]',
    exploitability_score    DECIMAL(5,2)    NOT NULL DEFAULT 0,
    exploitability_factors  JSONB           NOT NULL DEFAULT '[]',
    priority_score          DECIMAL(5,2)    NOT NULL DEFAULT 0,
    priority_group          INT             NOT NULL DEFAULT 4,
    priority_rank           INT,
    validation_status       TEXT            NOT NULL DEFAULT 'pending'
                                            CHECK (validation_status IN ('pending', 'validated', 'compensated',
                                                                         'not_exploitable', 'requires_manual')),
    compensating_controls   TEXT[]          NOT NULL DEFAULT '{}',
    validation_notes        TEXT,
    validated_at            TIMESTAMPTZ,
    remediation_type        TEXT            CHECK (remediation_type IN ('patch', 'configuration', 'architecture',
                                                                        'upgrade', 'decommission', 'accept_risk')),
    remediation_description TEXT,
    remediation_effort      TEXT            CHECK (remediation_effort IN ('low', 'medium', 'high')),
    remediation_group_id    UUID,
    estimated_days          INT,
    status                  TEXT            NOT NULL DEFAULT 'open'
                                            CHECK (status IN ('open', 'in_remediation', 'remediated', 'accepted_risk',
                                                              'false_positive', 'deferred')),
    status_changed_by       UUID,
    status_changed_at       TIMESTAMPTZ,
    status_notes            TEXT,
    attack_path             JSONB,
    attack_path_length      INT,
    metadata                JSONB           NOT NULL DEFAULT '{}',
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ctem_finding_assessment
    ON ctem_findings (assessment_id, priority_score DESC);

CREATE INDEX IF NOT EXISTS idx_ctem_finding_tenant_sev
    ON ctem_findings (tenant_id, severity, status);

CREATE INDEX IF NOT EXISTS idx_ctem_finding_type
    ON ctem_findings (assessment_id, type);

CREATE INDEX IF NOT EXISTS idx_ctem_finding_priority
    ON ctem_findings (assessment_id, priority_group, priority_rank);

CREATE INDEX IF NOT EXISTS idx_ctem_finding_asset
    ON ctem_findings USING GIN (affected_asset_ids);

CREATE INDEX IF NOT EXISTS idx_ctem_finding_cve
    ON ctem_findings USING GIN (cve_ids) WHERE cve_ids != '{}';

CREATE INDEX IF NOT EXISTS idx_ctem_finding_status
    ON ctem_findings (assessment_id, status);

CREATE TABLE IF NOT EXISTS ctem_remediation_groups (
    id                   UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID            NOT NULL,
    assessment_id        UUID            NOT NULL REFERENCES ctem_assessments(id) ON DELETE CASCADE,
    title                TEXT            NOT NULL,
    description          TEXT            NOT NULL DEFAULT '',
    type                 TEXT            NOT NULL CHECK (type IN ('patch', 'configuration', 'architecture',
                                                                  'upgrade', 'decommission', 'accept_risk')),
    finding_count        INT             NOT NULL DEFAULT 0,
    affected_asset_count INT             NOT NULL DEFAULT 0,
    cve_ids              TEXT[]          NOT NULL DEFAULT '{}',
    max_priority_score   DECIMAL(5,2)    NOT NULL DEFAULT 0,
    priority_group       INT             NOT NULL DEFAULT 4,
    effort               TEXT            NOT NULL DEFAULT 'medium'
                                         CHECK (effort IN ('low', 'medium', 'high')),
    estimated_days       INT,
    score_reduction      DECIMAL(5,2),
    status               TEXT            NOT NULL DEFAULT 'planned'
                                         CHECK (status IN ('planned', 'in_progress', 'completed', 'deferred', 'accepted')),
    workflow_instance_id UUID,
    target_date          DATE,
    started_at           TIMESTAMPTZ,
    completed_at         TIMESTAMPTZ,
    created_at           TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_remediation_group_assessment
    ON ctem_remediation_groups (assessment_id, priority_group);

CREATE INDEX IF NOT EXISTS idx_remediation_group_status
    ON ctem_remediation_groups (tenant_id, status);

CREATE TABLE IF NOT EXISTS exposure_score_snapshots (
    id             UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID            NOT NULL,
    score          DECIMAL(5,2)    NOT NULL,
    breakdown      JSONB           NOT NULL,
    asset_count    INT             NOT NULL,
    vuln_count     INT             NOT NULL,
    finding_count  INT             NOT NULL,
    assessment_id  UUID            REFERENCES ctem_assessments(id),
    snapshot_type  TEXT            NOT NULL DEFAULT 'assessment'
                                   CHECK (snapshot_type IN ('assessment', 'daily', 'manual')),
    created_at     TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_exposure_snapshot_tenant
    ON exposure_score_snapshots (tenant_id, created_at DESC);
