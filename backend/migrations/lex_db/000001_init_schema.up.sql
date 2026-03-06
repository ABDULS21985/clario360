-- =============================================================================
-- Clario 360 — Lex Suite Database Schema
-- Database: lex_db
-- Contains: contracts, clauses, legal documents, compliance rules,
--           compliance alerts, legal workflows
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- ENUM TYPES
-- =============================================================================

CREATE TYPE contract_type AS ENUM (
    'nda', 'service_agreement', 'employment', 'vendor', 'license', 'other'
);
COMMENT ON TYPE contract_type IS 'Types of legal contracts';

CREATE TYPE contract_status AS ENUM (
    'draft', 'review', 'negotiation', 'active', 'expired', 'terminated'
);
COMMENT ON TYPE contract_status IS 'Lifecycle status of a contract';

CREATE TYPE clause_risk_level AS ENUM ('high', 'medium', 'low', 'none');
COMMENT ON TYPE clause_risk_level IS 'Risk level of a contract clause';

CREATE TYPE clause_status AS ENUM ('draft', 'reviewed', 'approved', 'flagged');
COMMENT ON TYPE clause_status IS 'Review status of a contract clause';

CREATE TYPE legal_doc_status AS ENUM ('draft', 'review', 'approved', 'archived');
COMMENT ON TYPE legal_doc_status IS 'Lifecycle status of a legal document';

CREATE TYPE compliance_severity AS ENUM ('critical', 'high', 'medium', 'low');
COMMENT ON TYPE compliance_severity IS 'Severity of compliance rule violations';

CREATE TYPE compliance_alert_status AS ENUM ('new', 'acknowledged', 'resolved', 'dismissed');
COMMENT ON TYPE compliance_alert_status IS 'Status of a compliance alert';

CREATE TYPE legal_workflow_type AS ENUM (
    'contract_review', 'document_approval', 'compliance_check', 'dispute_resolution'
);
COMMENT ON TYPE legal_workflow_type IS 'Types of legal workflows';

CREATE TYPE legal_workflow_status AS ENUM ('active', 'inactive', 'archived');
COMMENT ON TYPE legal_workflow_status IS 'Status of a legal workflow definition';

CREATE TYPE legal_instance_status AS ENUM ('active', 'completed', 'cancelled', 'suspended');
COMMENT ON TYPE legal_instance_status IS 'Status of a running legal workflow instance';

-- =============================================================================
-- TRIGGER FUNCTION
-- =============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- TABLE: contracts
-- =============================================================================

CREATE TABLE contracts (
    id             UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID            NOT NULL,
    title          VARCHAR(500)    NOT NULL,
    type           contract_type   NOT NULL,
    status         contract_status NOT NULL DEFAULT 'draft',
    parties        JSONB           NOT NULL DEFAULT '[]',
    effective_date DATE,
    expiry_date    DATE,
    value          DECIMAL(15,2),
    currency       VARCHAR(3)      DEFAULT 'SAR',
    file_url       TEXT,
    metadata       JSONB           DEFAULT '{}',
    created_by     UUID,
    created_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_by     UUID
);

COMMENT ON TABLE contracts IS 'Legal contracts managed by the Lex suite';
COMMENT ON COLUMN contracts.parties IS 'JSON array of party objects (name, role, contact)';
COMMENT ON COLUMN contracts.value IS 'Monetary value of the contract';
COMMENT ON COLUMN contracts.currency IS 'ISO 4217 currency code (default: SAR)';
COMMENT ON COLUMN contracts.file_url IS 'URL to the contract document in object storage';

CREATE INDEX idx_contracts_tenant_status ON contracts (tenant_id, status);
CREATE INDEX idx_contracts_tenant_type ON contracts (tenant_id, type);
CREATE INDEX idx_contracts_expiry ON contracts (expiry_date) WHERE status = 'active';
CREATE INDEX idx_contracts_tenant_created ON contracts (tenant_id, created_at DESC);
CREATE INDEX idx_contracts_parties ON contracts USING GIN (parties);
CREATE INDEX idx_contracts_metadata ON contracts USING GIN (metadata);

CREATE TRIGGER trg_contracts_updated_at
    BEFORE UPDATE ON contracts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: contract_clauses
-- =============================================================================

CREATE TABLE contract_clauses (
    id             UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID             NOT NULL,
    contract_id    UUID             NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    clause_number  VARCHAR(20)      NOT NULL,
    title          VARCHAR(500)     NOT NULL,
    content        TEXT             NOT NULL,
    risk_level     clause_risk_level NOT NULL DEFAULT 'none',
    ai_analysis    JSONB            DEFAULT '{}',
    ai_risk_flags  JSONB            DEFAULT '[]',
    status         clause_status    NOT NULL DEFAULT 'draft',
    reviewed_by    UUID,
    created_at     TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    created_by     UUID,
    updated_by     UUID
);

COMMENT ON TABLE contract_clauses IS 'Individual clauses within a contract with AI risk analysis';
COMMENT ON COLUMN contract_clauses.clause_number IS 'Clause numbering (e.g., 1.1, 2.3.a)';
COMMENT ON COLUMN contract_clauses.ai_analysis IS 'AI-generated analysis of the clause content';
COMMENT ON COLUMN contract_clauses.ai_risk_flags IS 'AI-identified risk flags as JSON array';
COMMENT ON COLUMN contract_clauses.reviewed_by IS 'User who reviewed the clause (references platform_core.users)';

CREATE INDEX idx_clauses_contract ON contract_clauses (contract_id, clause_number);
CREATE INDEX idx_clauses_tenant_risk ON contract_clauses (tenant_id, risk_level);
CREATE INDEX idx_clauses_tenant_status ON contract_clauses (tenant_id, status);
CREATE INDEX idx_clauses_ai_analysis ON contract_clauses USING GIN (ai_analysis);
CREATE INDEX idx_clauses_ai_risk ON contract_clauses USING GIN (ai_risk_flags);

CREATE TRIGGER trg_clauses_updated_at
    BEFORE UPDATE ON contract_clauses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: legal_documents
-- =============================================================================

CREATE TABLE legal_documents (
    id          UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID             NOT NULL,
    title       VARCHAR(500)     NOT NULL,
    type        VARCHAR(100)     NOT NULL,
    content     TEXT             NOT NULL DEFAULT '',
    file_url    TEXT,
    status      legal_doc_status NOT NULL DEFAULT 'draft',
    version     INTEGER          NOT NULL DEFAULT 1,
    parent_id   UUID             REFERENCES legal_documents(id) ON DELETE SET NULL,
    tags        TEXT[]           DEFAULT '{}',
    created_by  UUID,
    created_at  TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_by  UUID
);

COMMENT ON TABLE legal_documents IS 'Legal documents, memos, opinions, and other legal content';
COMMENT ON COLUMN legal_documents.version IS 'Document version number';
COMMENT ON COLUMN legal_documents.parent_id IS 'Parent document for versioning chains';
COMMENT ON COLUMN legal_documents.tags IS 'Searchable tags for categorization';

CREATE INDEX idx_legal_docs_tenant_status ON legal_documents (tenant_id, status);
CREATE INDEX idx_legal_docs_tenant_type ON legal_documents (tenant_id, type);
CREATE INDEX idx_legal_docs_parent ON legal_documents (parent_id);
CREATE INDEX idx_legal_docs_tags ON legal_documents USING GIN (tags);
CREATE INDEX idx_legal_docs_tenant_created ON legal_documents (tenant_id, created_at DESC);

CREATE TRIGGER trg_legal_docs_updated_at
    BEFORE UPDATE ON legal_documents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: compliance_rules
-- =============================================================================

CREATE TABLE compliance_rules (
    id                    UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID              NOT NULL,
    name                  VARCHAR(255)      NOT NULL,
    description           TEXT              NOT NULL DEFAULT '',
    jurisdiction          VARCHAR(100),
    regulation_reference  VARCHAR(255),
    rule_logic            JSONB             NOT NULL DEFAULT '{}',
    severity              compliance_severity NOT NULL DEFAULT 'medium',
    enabled               BOOLEAN           NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    created_by            UUID,
    updated_by            UUID
);

COMMENT ON TABLE compliance_rules IS 'Compliance rules based on regulations and internal policies';
COMMENT ON COLUMN compliance_rules.jurisdiction IS 'Applicable legal jurisdiction (e.g., SA, GCC, International)';
COMMENT ON COLUMN compliance_rules.regulation_reference IS 'Reference to specific regulation (e.g., NCA ECC-1:2018)';
COMMENT ON COLUMN compliance_rules.rule_logic IS 'Executable rule definition (conditions, thresholds)';

CREATE INDEX idx_comp_rules_tenant ON compliance_rules (tenant_id);
CREATE INDEX idx_comp_rules_enabled ON compliance_rules (tenant_id, enabled) WHERE enabled = true;
CREATE INDEX idx_comp_rules_jurisdiction ON compliance_rules (jurisdiction);
CREATE INDEX idx_comp_rules_logic ON compliance_rules USING GIN (rule_logic);

CREATE TRIGGER trg_comp_rules_updated_at
    BEFORE UPDATE ON compliance_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: compliance_alerts
-- =============================================================================

CREATE TABLE compliance_alerts (
    id          UUID                     PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID                     NOT NULL,
    rule_id     UUID                     NOT NULL REFERENCES compliance_rules(id) ON DELETE CASCADE,
    entity_type VARCHAR(100)             NOT NULL,
    entity_id   UUID                     NOT NULL,
    title       VARCHAR(500)             NOT NULL,
    description TEXT                     NOT NULL DEFAULT '',
    severity    compliance_severity      NOT NULL DEFAULT 'medium',
    status      compliance_alert_status  NOT NULL DEFAULT 'new',
    resolved_by UUID,
    resolved_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ              NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ              NOT NULL DEFAULT NOW(),
    created_by  UUID,
    updated_by  UUID
);

COMMENT ON TABLE compliance_alerts IS 'Alerts triggered by compliance rule violations';
COMMENT ON COLUMN compliance_alerts.entity_type IS 'Type of entity that violated the rule (contract, document, etc.)';
COMMENT ON COLUMN compliance_alerts.entity_id IS 'UUID of the violating entity';

CREATE INDEX idx_comp_alerts_tenant_status ON compliance_alerts (tenant_id, status);
CREATE INDEX idx_comp_alerts_tenant_severity ON compliance_alerts (tenant_id, severity);
CREATE INDEX idx_comp_alerts_rule ON compliance_alerts (rule_id);
CREATE INDEX idx_comp_alerts_entity ON compliance_alerts (entity_type, entity_id);
CREATE INDEX idx_comp_alerts_tenant_created ON compliance_alerts (tenant_id, created_at DESC);

CREATE TRIGGER trg_comp_alerts_updated_at
    BEFORE UPDATE ON compliance_alerts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: legal_workflows
-- =============================================================================

CREATE TABLE legal_workflows (
    id          UUID                  PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID                  NOT NULL,
    name        VARCHAR(255)          NOT NULL,
    type        legal_workflow_type   NOT NULL,
    definition  JSONB                 NOT NULL DEFAULT '{}',
    status      legal_workflow_status NOT NULL DEFAULT 'active',
    created_by  UUID,
    created_at  TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    updated_by  UUID
);

COMMENT ON TABLE legal_workflows IS 'Legal process workflow definitions';
COMMENT ON COLUMN legal_workflows.definition IS 'Workflow steps, transitions, and conditions (JSON)';

CREATE INDEX idx_legal_wf_tenant_status ON legal_workflows (tenant_id, status);
CREATE INDEX idx_legal_wf_type ON legal_workflows (tenant_id, type);
CREATE INDEX idx_legal_wf_definition ON legal_workflows USING GIN (definition);

CREATE TRIGGER trg_legal_wf_updated_at
    BEFORE UPDATE ON legal_workflows
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: legal_workflow_instances
-- =============================================================================

CREATE TABLE legal_workflow_instances (
    id           UUID                  PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID                  NOT NULL,
    workflow_id  UUID                  NOT NULL REFERENCES legal_workflows(id) ON DELETE CASCADE,
    entity_type  VARCHAR(100)          NOT NULL,
    entity_id    UUID                  NOT NULL,
    current_step VARCHAR(100)          NOT NULL DEFAULT '',
    data         JSONB                 NOT NULL DEFAULT '{}',
    status       legal_instance_status NOT NULL DEFAULT 'active',
    started_at   TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    created_by   UUID,
    updated_by   UUID
);

COMMENT ON TABLE legal_workflow_instances IS 'Running instances of legal workflows';
COMMENT ON COLUMN legal_workflow_instances.entity_type IS 'Type of entity being processed (contract, document, etc.)';
COMMENT ON COLUMN legal_workflow_instances.entity_id IS 'UUID of the entity being processed';

CREATE INDEX idx_legal_inst_tenant_status ON legal_workflow_instances (tenant_id, status);
CREATE INDEX idx_legal_inst_workflow ON legal_workflow_instances (workflow_id);
CREATE INDEX idx_legal_inst_entity ON legal_workflow_instances (entity_type, entity_id);
CREATE INDEX idx_legal_inst_data ON legal_workflow_instances USING GIN (data);

CREATE TRIGGER trg_legal_inst_updated_at
    BEFORE UPDATE ON legal_workflow_instances
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
