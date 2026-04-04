CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS contracts (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID        NOT NULL,
    title                TEXT        NOT NULL,
    contract_number      TEXT,
    type                 TEXT        NOT NULL CHECK (type IN (
        'service_agreement', 'nda', 'employment', 'vendor', 'license',
        'lease', 'partnership', 'consulting', 'procurement', 'sla',
        'mou', 'amendment', 'renewal', 'other'
    )),
    description          TEXT        NOT NULL DEFAULT '',
    party_a_name         TEXT        NOT NULL,
    party_a_entity       TEXT,
    party_b_name         TEXT        NOT NULL,
    party_b_entity       TEXT,
    party_b_contact      TEXT,
    total_value          DECIMAL(18,2),
    currency             TEXT        NOT NULL DEFAULT 'SAR',
    payment_terms        TEXT,
    effective_date       DATE,
    expiry_date          DATE,
    renewal_date         DATE,
    auto_renew           BOOLEAN     NOT NULL DEFAULT false,
    renewal_notice_days  INT         NOT NULL DEFAULT 30,
    signed_date          DATE,
    status               TEXT        NOT NULL DEFAULT 'draft' CHECK (status IN (
        'draft', 'internal_review', 'legal_review', 'negotiation',
        'pending_signature', 'active', 'suspended', 'expired',
        'terminated', 'renewed', 'cancelled'
    )),
    previous_status      TEXT,
    status_changed_at    TIMESTAMPTZ,
    status_changed_by    UUID,
    owner_user_id        UUID        NOT NULL,
    owner_name           TEXT        NOT NULL,
    legal_reviewer_id    UUID,
    legal_reviewer_name  TEXT,
    risk_score           DECIMAL(5,2),
    risk_level           TEXT        CHECK (risk_level IN ('critical', 'high', 'medium', 'low', 'none')),
    analysis_status      TEXT        DEFAULT 'pending' CHECK (analysis_status IN ('pending', 'analyzing', 'completed', 'failed')),
    last_analyzed_at     TIMESTAMPTZ,
    document_file_id     UUID,
    document_text        TEXT,
    current_version      INT         NOT NULL DEFAULT 1,
    parent_contract_id   UUID        REFERENCES contracts(id),
    workflow_instance_id UUID,
    department           TEXT,
    tags                 TEXT[]      NOT NULL DEFAULT '{}',
    metadata             JSONB       NOT NULL DEFAULT '{}',
    created_by           UUID        NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at           TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_contracts_contract_number_unique
    ON contracts (tenant_id, contract_number)
    WHERE contract_number IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_contracts_tenant_status
    ON contracts (tenant_id, status, created_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_contracts_tenant_type
    ON contracts (tenant_id, type)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_contracts_expiry
    ON contracts (tenant_id, expiry_date)
    WHERE status = 'active' AND expiry_date IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_contracts_owner
    ON contracts (owner_user_id)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_contracts_risk
    ON contracts (tenant_id, risk_level)
    WHERE risk_level IN ('critical', 'high') AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_contracts_parent
    ON contracts (parent_contract_id)
    WHERE parent_contract_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_contracts_fts
    ON contracts USING GIN (
        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(party_b_name, '') || ' ' || coalesce(description, ''))
    )
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS contract_versions (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    contract_id      UUID        NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    version          INT         NOT NULL,
    file_id          UUID        NOT NULL,
    file_name        TEXT        NOT NULL,
    file_size_bytes  BIGINT      NOT NULL,
    content_hash     TEXT        NOT NULL,
    extracted_text   TEXT,
    change_summary   TEXT,
    uploaded_by      UUID        NOT NULL,
    uploaded_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (contract_id, version)
);

CREATE INDEX IF NOT EXISTS idx_contract_versions
    ON contract_versions (contract_id, version DESC);

CREATE TABLE IF NOT EXISTS contract_clauses (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID        NOT NULL,
    contract_id           UUID        NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    clause_type           TEXT        NOT NULL CHECK (clause_type IN (
        'indemnification', 'termination', 'limitation_of_liability', 'confidentiality',
        'ip_ownership', 'non_compete', 'payment_terms', 'warranty', 'force_majeure',
        'dispute_resolution', 'data_protection', 'governing_law', 'assignment',
        'insurance', 'audit_rights', 'sla', 'auto_renewal', 'non_solicitation',
        'representations', 'other'
    )),
    title                 TEXT        NOT NULL,
    content               TEXT        NOT NULL,
    section_reference     TEXT,
    page_number           INT,
    risk_level            TEXT        NOT NULL DEFAULT 'none' CHECK (risk_level IN ('critical', 'high', 'medium', 'low', 'none')),
    risk_score            DECIMAL(5,2) NOT NULL DEFAULT 0,
    risk_keywords         TEXT[]      NOT NULL DEFAULT '{}',
    analysis_summary      TEXT,
    recommendations       TEXT[]      NOT NULL DEFAULT '{}',
    compliance_flags      TEXT[]      NOT NULL DEFAULT '{}',
    review_status         TEXT        NOT NULL DEFAULT 'pending' CHECK (review_status IN ('pending', 'reviewed', 'flagged', 'accepted', 'rejected')),
    reviewed_by           UUID,
    reviewed_at           TIMESTAMPTZ,
    review_notes          TEXT,
    extraction_confidence DECIMAL(3,2) NOT NULL DEFAULT 0.80,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_clauses_contract
    ON contract_clauses (contract_id, clause_type);
CREATE INDEX IF NOT EXISTS idx_clauses_risk
    ON contract_clauses (tenant_id, risk_level)
    WHERE risk_level IN ('critical', 'high');

CREATE TABLE IF NOT EXISTS contract_analyses (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID        NOT NULL,
    contract_id            UUID        NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    contract_version       INT         NOT NULL,
    overall_risk           TEXT        NOT NULL CHECK (overall_risk IN ('critical', 'high', 'medium', 'low', 'none')),
    risk_score             DECIMAL(5,2) NOT NULL,
    clause_count           INT         NOT NULL DEFAULT 0,
    high_risk_clause_count INT         NOT NULL DEFAULT 0,
    missing_clauses        TEXT[]      NOT NULL DEFAULT '{}',
    key_findings           JSONB       NOT NULL DEFAULT '[]',
    recommendations        TEXT[]      NOT NULL DEFAULT '{}',
    compliance_flags       JSONB       NOT NULL DEFAULT '[]',
    extracted_parties      JSONB       NOT NULL DEFAULT '{}',
    extracted_dates        JSONB       NOT NULL DEFAULT '{}',
    extracted_amounts      JSONB       NOT NULL DEFAULT '{}',
    analysis_duration_ms   BIGINT,
    analyzed_by            TEXT        NOT NULL DEFAULT 'system',
    analyzed_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_analyses_contract
    ON contract_analyses (contract_id, analyzed_at DESC);

CREATE TABLE IF NOT EXISTS legal_documents (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    title            TEXT        NOT NULL,
    type             TEXT        NOT NULL CHECK (type IN (
        'policy', 'regulation', 'template', 'memo', 'opinion',
        'filing', 'correspondence', 'resolution', 'power_of_attorney', 'other'
    )),
    description      TEXT        NOT NULL DEFAULT '',
    file_id          UUID,
    file_name        TEXT,
    file_size_bytes  BIGINT,
    category         TEXT,
    confidentiality  TEXT        NOT NULL DEFAULT 'internal' CHECK (confidentiality IN ('public', 'internal', 'confidential', 'privileged')),
    contract_id      UUID        REFERENCES contracts(id),
    current_version  INT         NOT NULL DEFAULT 1,
    status           TEXT        NOT NULL DEFAULT 'active' CHECK (status IN ('draft', 'active', 'archived', 'superseded')),
    tags             TEXT[]      NOT NULL DEFAULT '{}',
    metadata         JSONB       NOT NULL DEFAULT '{}',
    created_by       UUID        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_legal_docs_tenant
    ON legal_documents (tenant_id, type, status)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_legal_docs_contract
    ON legal_documents (contract_id)
    WHERE contract_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS document_versions (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    document_id      UUID        NOT NULL REFERENCES legal_documents(id) ON DELETE CASCADE,
    version          INT         NOT NULL,
    file_id          UUID        NOT NULL,
    file_name        TEXT        NOT NULL,
    file_size_bytes  BIGINT      NOT NULL,
    content_hash     TEXT        NOT NULL,
    change_summary   TEXT,
    uploaded_by      UUID        NOT NULL,
    uploaded_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (document_id, version)
);

CREATE TABLE IF NOT EXISTS compliance_rules (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL,
    name            TEXT        NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    rule_type       TEXT        NOT NULL CHECK (rule_type IN (
        'expiry_warning', 'missing_clause', 'risk_threshold',
        'review_overdue', 'unsigned_contract', 'value_threshold',
        'jurisdiction_check', 'data_protection_required', 'custom'
    )),
    severity        TEXT        NOT NULL DEFAULT 'medium' CHECK (severity IN ('critical', 'high', 'medium', 'low')),
    config          JSONB       NOT NULL,
    contract_types  TEXT[]      NOT NULL DEFAULT '{}',
    enabled         BOOLEAN     NOT NULL DEFAULT true,
    created_by      UUID        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_compliance_rules_tenant
    ON compliance_rules (tenant_id, enabled)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS compliance_alerts (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    rule_id          UUID        REFERENCES compliance_rules(id),
    contract_id      UUID        REFERENCES contracts(id),
    title            TEXT        NOT NULL,
    description      TEXT        NOT NULL,
    severity         TEXT        NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low')),
    status           TEXT        NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'acknowledged', 'investigating', 'resolved', 'dismissed')),
    resolved_by      UUID,
    resolved_at      TIMESTAMPTZ,
    resolution_notes TEXT,
    dedup_key        TEXT,
    evidence         JSONB       NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_compliance_alerts_tenant
    ON compliance_alerts (tenant_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_compliance_alerts_contract
    ON compliance_alerts (contract_id)
    WHERE contract_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_compliance_alerts_dedup
    ON compliance_alerts (tenant_id, dedup_key)
    WHERE dedup_key IS NOT NULL AND status NOT IN ('resolved', 'dismissed');

CREATE TABLE IF NOT EXISTS expiry_notifications (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL,
    contract_id  UUID        NOT NULL REFERENCES contracts(id),
    horizon_days INT         NOT NULL,
    sent_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (contract_id, horizon_days)
);
