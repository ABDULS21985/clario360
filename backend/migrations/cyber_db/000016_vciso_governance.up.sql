-- =============================================================================
-- Migration 000016: vCISO Governance Tables (RECREATED)
-- Drops and recreates all governance tables to match the Go repository layer.
-- Tables: vciso_risks, vciso_policies, vciso_policy_exceptions, vciso_vendors,
--   vciso_questionnaires, vciso_evidence, vciso_maturity_assessments,
--   vciso_budget_items, vciso_awareness_programs, vciso_iam_findings,
--   vciso_escalation_rules, vciso_playbooks, vciso_obligations,
--   vciso_control_tests, vciso_integrations, vciso_control_ownership,
--   vciso_approvals
-- =============================================================================

-- ─── DROP ALL (including legacy names) ──────────────────────────────────────

DROP TABLE IF EXISTS vciso_approvals CASCADE;
DROP TABLE IF EXISTS vciso_approval_requests CASCADE;
DROP TABLE IF EXISTS vciso_control_ownership CASCADE;
DROP TABLE IF EXISTS vciso_integrations CASCADE;
DROP TABLE IF EXISTS vciso_control_tests CASCADE;
DROP TABLE IF EXISTS vciso_obligations CASCADE;
DROP TABLE IF EXISTS vciso_regulatory_obligations CASCADE;
DROP TABLE IF EXISTS vciso_playbooks CASCADE;
DROP TABLE IF EXISTS vciso_escalation_rules CASCADE;
DROP TABLE IF EXISTS vciso_iam_findings CASCADE;
DROP TABLE IF EXISTS vciso_awareness_programs CASCADE;
DROP TABLE IF EXISTS vciso_budget_items CASCADE;
DROP TABLE IF EXISTS vciso_benchmarks CASCADE;
DROP TABLE IF EXISTS vciso_maturity_assessments CASCADE;
DROP TABLE IF EXISTS vciso_evidence CASCADE;
DROP TABLE IF EXISTS vciso_questionnaires CASCADE;
DROP TABLE IF EXISTS vciso_vendors CASCADE;
DROP TABLE IF EXISTS vciso_policy_exceptions CASCADE;
DROP TABLE IF EXISTS vciso_policies CASCADE;
DROP TABLE IF EXISTS vciso_risks CASCADE;
DROP TABLE IF EXISTS vciso_control_dependencies CASCADE;

-- =============================================================================
-- 1. vciso_risks — Risk register entries
-- Columns from repo: id, tenant_id, title, description, category, department,
--   inherent_score, residual_score, likelihood, impact, status, treatment,
--   owner_id, owner_name, review_date, business_services, controls, tags,
--   treatment_plan, acceptance_rationale, acceptance_approved_by,
--   acceptance_approved_by_name, acceptance_expiry, created_at, updated_at
-- =============================================================================

CREATE TABLE vciso_risks (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                   UUID NOT NULL,
    title                       VARCHAR(500) NOT NULL,
    description                 TEXT DEFAULT '',
    category                    VARCHAR(100) DEFAULT '',
    department                  VARCHAR(100) DEFAULT '',
    inherent_score              INTEGER DEFAULT 0,
    residual_score              INTEGER DEFAULT 0,
    likelihood                  VARCHAR(50) DEFAULT 'medium',
    impact                      VARCHAR(50) DEFAULT 'medium',
    status                      VARCHAR(50) DEFAULT 'open',
    treatment                   VARCHAR(100) DEFAULT 'mitigate',
    owner_id                    UUID,
    owner_name                  VARCHAR(255) DEFAULT '',
    review_date                 VARCHAR(50),
    business_services           TEXT[] DEFAULT '{}',
    controls                    TEXT[] DEFAULT '{}',
    tags                        TEXT[] DEFAULT '{}',
    treatment_plan              TEXT DEFAULT '',
    acceptance_rationale        TEXT,
    acceptance_approved_by      UUID,
    acceptance_approved_by_name VARCHAR(255),
    acceptance_expiry           VARCHAR(50),
    created_at                  TIMESTAMPTZ DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_vciso_risks_tenant ON vciso_risks(tenant_id);

-- =============================================================================
-- 2. vciso_policies — Security governance policies
-- Columns from repo: id, tenant_id, title, domain, version, status, content,
--   owner_id, owner_name, reviewer_id, reviewer_name, approved_by,
--   approved_by_name, approved_at, review_due, last_reviewed_at, tags,
--   exceptions_count, created_at, updated_at
-- =============================================================================

CREATE TABLE vciso_policies (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL,
    title             VARCHAR(500) NOT NULL,
    domain            VARCHAR(100) DEFAULT '',
    version           VARCHAR(50) DEFAULT '1.0',
    status            VARCHAR(50) DEFAULT 'draft',
    content           TEXT DEFAULT '',
    owner_id          UUID NOT NULL,
    owner_name        VARCHAR(255) DEFAULT '',
    reviewer_id       UUID,
    reviewer_name     VARCHAR(255),
    approved_by       UUID,
    approved_by_name  VARCHAR(255),
    approved_at       TIMESTAMPTZ,
    review_due        VARCHAR(50) DEFAULT '',
    last_reviewed_at  TIMESTAMPTZ,
    tags              TEXT[] DEFAULT '{}',
    exceptions_count  INTEGER DEFAULT 0,
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_vciso_policies_tenant ON vciso_policies(tenant_id);

-- =============================================================================
-- 3. vciso_policy_exceptions — Policy exception requests
-- Columns from repo: id, tenant_id, policy_id, policy_title, title,
--   description, justification, compensating_controls, status, requested_by,
--   requested_by_name, approved_by, approved_by_name, decision_notes,
--   expires_at, created_at, updated_at
-- =============================================================================

CREATE TABLE vciso_policy_exceptions (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID NOT NULL,
    policy_id              UUID NOT NULL,
    policy_title           VARCHAR(500) DEFAULT '',
    title                  VARCHAR(500) NOT NULL,
    description            TEXT DEFAULT '',
    justification          TEXT DEFAULT '',
    compensating_controls  TEXT DEFAULT '',
    status                 VARCHAR(50) DEFAULT 'pending',
    requested_by           UUID NOT NULL,
    requested_by_name      VARCHAR(255) DEFAULT '',
    approved_by            UUID,
    approved_by_name       VARCHAR(255),
    decision_notes         TEXT,
    expires_at             VARCHAR(50) DEFAULT '',
    created_at             TIMESTAMPTZ DEFAULT NOW(),
    updated_at             TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_vciso_policy_exceptions_tenant ON vciso_policy_exceptions(tenant_id);

-- =============================================================================
-- 4. vciso_vendors — Third-party vendor records
-- Columns from repo: id, tenant_id, name, category, risk_tier, status,
--   risk_score, last_assessment_date, next_review_date, contact_name,
--   contact_email, services_provided, data_shared, compliance_frameworks,
--   controls_met, controls_total, open_findings, created_at, updated_at
-- =============================================================================

CREATE TABLE vciso_vendors (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID NOT NULL,
    name                  VARCHAR(500) NOT NULL,
    category              VARCHAR(100) DEFAULT '',
    risk_tier             VARCHAR(50) DEFAULT 'medium',
    status                VARCHAR(50) DEFAULT 'active',
    risk_score            INTEGER DEFAULT 0,
    last_assessment_date  TIMESTAMPTZ,
    next_review_date      VARCHAR(50) DEFAULT '',
    contact_name          VARCHAR(255),
    contact_email         VARCHAR(255),
    services_provided     TEXT[] DEFAULT '{}',
    data_shared           TEXT[] DEFAULT '{}',
    compliance_frameworks TEXT[] DEFAULT '{}',
    controls_met          INTEGER DEFAULT 0,
    controls_total        INTEGER DEFAULT 0,
    open_findings         INTEGER DEFAULT 0,
    created_at            TIMESTAMPTZ DEFAULT NOW(),
    updated_at            TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_vciso_vendors_tenant ON vciso_vendors(tenant_id);

-- =============================================================================
-- 5. vciso_questionnaires — Vendor / audit questionnaires
-- Columns from repo: id, tenant_id, title, type, status, vendor_id,
--   vendor_name, total_questions, answered_questions, due_date, completed_at,
--   score, assigned_to, assigned_to_name, created_at, updated_at
-- =============================================================================

CREATE TABLE vciso_questionnaires (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL,
    title              VARCHAR(500) NOT NULL,
    type               VARCHAR(100) DEFAULT '',
    status             VARCHAR(50) DEFAULT 'draft',
    vendor_id          UUID,
    vendor_name        VARCHAR(255),
    total_questions    INTEGER DEFAULT 0,
    answered_questions INTEGER DEFAULT 0,
    due_date           VARCHAR(50) DEFAULT '',
    completed_at       TIMESTAMPTZ,
    score              INTEGER,
    assigned_to        UUID,
    assigned_to_name   VARCHAR(255),
    created_at         TIMESTAMPTZ DEFAULT NOW(),
    updated_at         TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_vciso_questionnaires_tenant ON vciso_questionnaires(tenant_id);

-- =============================================================================
-- 6. vciso_evidence — Compliance evidence records
-- Columns from repo: id, tenant_id, title, description, type, source, status,
--   frameworks, control_ids, file_name, file_size, file_url, collected_at,
--   expires_at, collector_name, last_verified_at, created_at, updated_at
-- =============================================================================

CREATE TABLE vciso_evidence (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL,
    title            VARCHAR(500) NOT NULL,
    description      TEXT DEFAULT '',
    type             VARCHAR(100) DEFAULT '',
    source           VARCHAR(100) DEFAULT '',
    status           VARCHAR(50) DEFAULT 'active',
    frameworks       TEXT[] DEFAULT '{}',
    control_ids      TEXT[] DEFAULT '{}',
    file_name        VARCHAR(500),
    file_size        INTEGER,
    file_url         TEXT,
    collected_at     TIMESTAMPTZ DEFAULT NOW(),
    expires_at       TIMESTAMPTZ,
    collector_name   VARCHAR(255),
    last_verified_at TIMESTAMPTZ,
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_vciso_evidence_tenant ON vciso_evidence(tenant_id);

-- =============================================================================
-- 7. vciso_maturity_assessments — Maturity assessment records
-- Columns from repo: id, tenant_id, framework, status, overall_score,
--   overall_level, dimensions (JSONB), assessor_name, assessed_at,
--   created_at, updated_at
-- =============================================================================

CREATE TABLE vciso_maturity_assessments (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    framework      VARCHAR(100) DEFAULT '',
    status         VARCHAR(50) DEFAULT 'in_progress',
    overall_score  NUMERIC(5,2) DEFAULT 0,
    overall_level  INTEGER DEFAULT 0,
    dimensions     JSONB DEFAULT '[]',
    assessor_name  VARCHAR(255),
    assessed_at    TIMESTAMPTZ DEFAULT NOW(),
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at     TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_vciso_maturity_assessments_tenant ON vciso_maturity_assessments(tenant_id);

-- =============================================================================
-- 8. vciso_budget_items — Security budget line items
-- Columns from repo: id, tenant_id, title, category, type, amount, currency,
--   status, risk_reduction_estimate, priority, justification, linked_risk_ids,
--   linked_recommendation_ids, fiscal_year, quarter, owner_name,
--   created_at, updated_at
-- =============================================================================

CREATE TABLE vciso_budget_items (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID NOT NULL,
    title                     VARCHAR(500) NOT NULL,
    category                  VARCHAR(100) DEFAULT '',
    type                      VARCHAR(100) DEFAULT '',
    amount                    NUMERIC(12,2) DEFAULT 0,
    currency                  VARCHAR(10) DEFAULT 'USD',
    status                    VARCHAR(50) DEFAULT 'proposed',
    risk_reduction_estimate   NUMERIC(5,2) DEFAULT 0,
    priority                  INTEGER DEFAULT 3,
    justification             TEXT DEFAULT '',
    linked_risk_ids           TEXT[] DEFAULT '{}',
    linked_recommendation_ids TEXT[] DEFAULT '{}',
    fiscal_year               VARCHAR(20) DEFAULT '',
    quarter                   VARCHAR(10),
    owner_name                VARCHAR(255),
    created_at                TIMESTAMPTZ DEFAULT NOW(),
    updated_at                TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_vciso_budget_items_tenant ON vciso_budget_items(tenant_id);

-- =============================================================================
-- 9. vciso_awareness_programs — Security awareness training programs
-- Columns from repo: id, tenant_id, name, type, status, total_users,
--   completed_users, passed_users, failed_users, completion_rate, pass_rate,
--   start_date, end_date, created_at, updated_at
-- =============================================================================

CREATE TABLE vciso_awareness_programs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    name            VARCHAR(500) NOT NULL,
    type            VARCHAR(100) DEFAULT '',
    status          VARCHAR(50) DEFAULT 'planned',
    total_users     INTEGER DEFAULT 0,
    completed_users INTEGER DEFAULT 0,
    passed_users    INTEGER DEFAULT 0,
    failed_users    INTEGER DEFAULT 0,
    completion_rate NUMERIC(5,2) DEFAULT 0,
    pass_rate       NUMERIC(5,2) DEFAULT 0,
    start_date      VARCHAR(50) DEFAULT '',
    end_date        VARCHAR(50) DEFAULT '',
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_vciso_awareness_programs_tenant ON vciso_awareness_programs(tenant_id);

-- =============================================================================
-- 10. vciso_iam_findings — IAM-related security findings
-- Columns from repo: id, tenant_id, type, severity, title, description,
--   affected_users, status, remediation, discovered_at, resolved_at,
--   created_at, updated_at
-- =============================================================================

CREATE TABLE vciso_iam_findings (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    type           VARCHAR(100) DEFAULT '',
    severity       VARCHAR(50) DEFAULT 'medium',
    title          VARCHAR(500) NOT NULL,
    description    TEXT DEFAULT '',
    affected_users INTEGER DEFAULT 0,
    status         VARCHAR(50) DEFAULT 'open',
    remediation    TEXT,
    discovered_at  TIMESTAMPTZ DEFAULT NOW(),
    resolved_at    TIMESTAMPTZ,
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at     TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_vciso_iam_findings_tenant ON vciso_iam_findings(tenant_id);

-- =============================================================================
-- 11. vciso_escalation_rules — Incident escalation rules
-- Columns from repo: id, tenant_id, name, description, trigger_type,
--   trigger_condition, escalation_target, target_contacts,
--   notification_channels, enabled, last_triggered_at, trigger_count,
--   created_at, updated_at
-- =============================================================================

CREATE TABLE vciso_escalation_rules (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID NOT NULL,
    name                   VARCHAR(500) NOT NULL,
    description            TEXT DEFAULT '',
    trigger_type           VARCHAR(100) DEFAULT '',
    trigger_condition      TEXT DEFAULT '',
    escalation_target      VARCHAR(255) DEFAULT '',
    target_contacts        TEXT[] DEFAULT '{}',
    notification_channels  TEXT[] DEFAULT '{}',
    enabled                BOOLEAN DEFAULT true,
    last_triggered_at      TIMESTAMPTZ,
    trigger_count          INTEGER DEFAULT 0,
    created_at             TIMESTAMPTZ DEFAULT NOW(),
    updated_at             TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_vciso_escalation_rules_tenant ON vciso_escalation_rules(tenant_id);

-- =============================================================================
-- 12. vciso_playbooks — Incident response / BCP playbooks
-- Columns from repo: id, tenant_id, name, scenario, status, last_tested_at,
--   next_test_date, owner_id, owner_name, steps_count, dependencies,
--   rto_hours, rpo_hours, last_simulation_result, created_at, updated_at
-- =============================================================================

CREATE TABLE vciso_playbooks (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID NOT NULL,
    name                   VARCHAR(500) NOT NULL,
    scenario               VARCHAR(255) DEFAULT '',
    status                 VARCHAR(50) DEFAULT 'draft',
    last_tested_at         TIMESTAMPTZ,
    next_test_date         VARCHAR(50) DEFAULT '',
    owner_id               UUID NOT NULL,
    owner_name             VARCHAR(255) DEFAULT '',
    steps_count            INTEGER DEFAULT 0,
    dependencies           TEXT[] DEFAULT '{}',
    rto_hours              NUMERIC(8,2),
    rpo_hours              NUMERIC(8,2),
    last_simulation_result VARCHAR(100),
    created_at             TIMESTAMPTZ DEFAULT NOW(),
    updated_at             TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_vciso_playbooks_tenant ON vciso_playbooks(tenant_id);

-- =============================================================================
-- 13. vciso_obligations — Regulatory / contractual obligations
-- Table name in Go code: vciso_obligations (NOT vciso_regulatory_obligations)
-- Columns from repo: id, tenant_id, name, type, jurisdiction, description,
--   requirements, status, mapped_controls, total_requirements,
--   met_requirements, owner_id, owner_name, effective_date, review_date,
--   created_at, updated_at
-- =============================================================================

CREATE TABLE vciso_obligations (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL,
    name               VARCHAR(500) NOT NULL,
    type               VARCHAR(100) DEFAULT '',
    jurisdiction       VARCHAR(100) DEFAULT '',
    description        TEXT DEFAULT '',
    requirements       TEXT[] DEFAULT '{}',
    status             VARCHAR(50) DEFAULT 'active',
    mapped_controls    INTEGER DEFAULT 0,
    total_requirements INTEGER DEFAULT 0,
    met_requirements   INTEGER DEFAULT 0,
    owner_id           UUID,
    owner_name         VARCHAR(255),
    effective_date     VARCHAR(50) DEFAULT '',
    review_date        VARCHAR(50) DEFAULT '',
    created_at         TIMESTAMPTZ DEFAULT NOW(),
    updated_at         TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_vciso_obligations_tenant ON vciso_obligations(tenant_id);

-- =============================================================================
-- 14. vciso_control_tests — Control effectiveness test records
-- Columns from repo: id, tenant_id, control_id, control_name, framework,
--   test_type, result, tester_name, test_date, next_test_date, findings,
--   evidence_ids, created_at, updated_at
-- Note: search also filters on test_name alias (mapped to control_name in search)
-- =============================================================================

CREATE TABLE vciso_control_tests (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    control_id     VARCHAR(100) DEFAULT '',
    control_name   VARCHAR(500) DEFAULT '',
    framework      VARCHAR(100) DEFAULT '',
    test_type      VARCHAR(100) DEFAULT '',
    result         VARCHAR(50) DEFAULT '',
    tester_name    VARCHAR(255) DEFAULT '',
    test_date      VARCHAR(50) DEFAULT '',
    next_test_date VARCHAR(50) DEFAULT '',
    findings       TEXT DEFAULT '',
    evidence_ids   TEXT[] DEFAULT '{}',
    test_name      VARCHAR(500) DEFAULT '',
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at     TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_vciso_control_tests_tenant ON vciso_control_tests(tenant_id);

-- =============================================================================
-- 15. vciso_integrations — External tool integrations
-- Columns from repo: id, tenant_id, name, type, provider, status,
--   last_sync_at, sync_frequency, items_synced, config (JSONB),
--   health_status, error_message, created_at, updated_at
-- =============================================================================

CREATE TABLE vciso_integrations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    name            VARCHAR(500) NOT NULL,
    type            VARCHAR(100) DEFAULT '',
    provider        VARCHAR(255) DEFAULT '',
    status          VARCHAR(50) DEFAULT 'inactive',
    last_sync_at    TIMESTAMPTZ,
    sync_frequency  VARCHAR(50) DEFAULT 'daily',
    items_synced    INTEGER DEFAULT 0,
    config          JSONB DEFAULT '{}',
    health_status   VARCHAR(50) DEFAULT 'healthy',
    error_message   TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_vciso_integrations_tenant ON vciso_integrations(tenant_id);

-- =============================================================================
-- 16. vciso_control_ownership — Control ownership assignments
-- Columns from repo: id, tenant_id, control_id, control_name, framework,
--   owner_id, owner_name, delegate_id, delegate_name, status,
--   last_reviewed_at, next_review_date, created_at, updated_at
-- =============================================================================

CREATE TABLE vciso_control_ownership (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL,
    control_id       VARCHAR(100) DEFAULT '',
    control_name     VARCHAR(500) DEFAULT '',
    framework        VARCHAR(100) DEFAULT '',
    owner_id         UUID NOT NULL,
    owner_name       VARCHAR(255) DEFAULT '',
    delegate_id      UUID,
    delegate_name    VARCHAR(255),
    status           VARCHAR(50) DEFAULT 'assigned',
    last_reviewed_at TIMESTAMPTZ,
    next_review_date VARCHAR(50) DEFAULT '',
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_vciso_control_ownership_tenant ON vciso_control_ownership(tenant_id);

-- =============================================================================
-- 17. vciso_approvals — Governance approval requests
-- Table name in Go code: vciso_approvals (NOT vciso_approval_requests)
-- Columns from repo: id, tenant_id, type, title, description, status,
--   requested_by, requested_by_name, approver_id, approver_name, priority,
--   decision_notes, decided_at, deadline, linked_entity_type,
--   linked_entity_id, created_at, updated_at
-- =============================================================================

CREATE TABLE vciso_approvals (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL,
    type               VARCHAR(100) DEFAULT '',
    title              VARCHAR(500) NOT NULL,
    description        TEXT DEFAULT '',
    status             VARCHAR(50) DEFAULT 'pending',
    requested_by       UUID NOT NULL,
    requested_by_name  VARCHAR(255) DEFAULT '',
    approver_id        UUID NOT NULL,
    approver_name      VARCHAR(255) DEFAULT '',
    priority           VARCHAR(50) DEFAULT 'medium',
    decision_notes     TEXT,
    decided_at         TIMESTAMPTZ,
    deadline           VARCHAR(50) DEFAULT '',
    linked_entity_type VARCHAR(100) DEFAULT '',
    linked_entity_id   VARCHAR(100) DEFAULT '',
    created_at         TIMESTAMPTZ DEFAULT NOW(),
    updated_at         TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_vciso_approvals_tenant ON vciso_approvals(tenant_id);


-- =============================================================================
-- SEED DATA for dev tenant aaaaaaaa-0000-0000-0000-000000000001
-- created_by: bbbbbbbb-0000-0000-0000-000000000001
-- UUID pattern: aaaaaaaa-0016-XXXX-0000-YYYYYYYYYYYY
-- =============================================================================

-- ─── 1. vciso_risks seed ────────────────────────────────────────────────────

INSERT INTO vciso_risks (id, tenant_id, title, description, category, department, inherent_score, residual_score, likelihood, impact, status, treatment, owner_id, owner_name, review_date, business_services, controls, tags, treatment_plan)
VALUES
('aaaaaaaa-0016-0101-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'Ransomware Attack on Core Infrastructure', 'Risk of ransomware encrypting critical servers and demanding payment.', 'cybersecurity', 'IT Operations', 85, 40, 'high', 'critical', 'open', 'mitigate', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', '2026-06-15', '{"ERP","Email","File Storage"}', '{"AC-3","IR-4","CP-9"}', '{"ransomware","infrastructure","backup"}', 'Deploy EDR solution, implement immutable backups, conduct tabletop exercises.'),
('aaaaaaaa-0016-0101-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'Third-Party Data Breach via Cloud SaaS Vendor', 'Sensitive customer data exposed through compromised SaaS provider.', 'third_party', 'Legal', 70, 50, 'medium', 'high', 'open', 'transfer', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', '2026-05-01', '{"CRM","Analytics"}', '{"SA-9","CA-7"}', '{"vendor","cloud","data_breach"}', 'Require SOC 2 attestation, add contractual breach notification clauses.'),
('aaaaaaaa-0016-0101-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'Insider Threat from Privileged Accounts', 'Malicious or accidental misuse by users with elevated privileges.', 'insider_threat', 'Security', 60, 35, 'medium', 'high', 'mitigated', 'mitigate', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', '2026-07-30', '{"IAM","Database"}', '{"AC-2","AC-6","AU-6"}', '{"insider","privileged_access","monitoring"}', 'Implement PAM solution, enforce MFA on all admin accounts, monthly access reviews.'),
('aaaaaaaa-0016-0101-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'Regulatory Non-Compliance with GDPR', 'Failure to meet GDPR data processing requirements leading to fines.', 'compliance', 'Legal', 55, 30, 'low', 'critical', 'accepted', 'accept', NULL, '', '2026-04-01', '{"Customer Portal","Marketing"}', '{"PM-1","SI-12"}', '{"gdpr","compliance","privacy"}', ''),
('aaaaaaaa-0016-0101-0000-000000000005', 'aaaaaaaa-0000-0000-0000-000000000001', 'Phishing Campaign Targeting Executives', 'Spear-phishing campaign aimed at C-suite to gain access credentials.', 'cybersecurity', 'Executive Office', 75, 45, 'high', 'high', 'open', 'mitigate', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', '2026-08-15', '{"Email","VPN"}', '{"AT-2","SI-8","IR-4"}', '{"phishing","executive","social_engineering"}', 'Mandatory anti-phishing training, deploy email gateway with AI detection.');

-- ─── 2. vciso_policies seed ─────────────────────────────────────────────────

INSERT INTO vciso_policies (id, tenant_id, title, domain, version, status, content, owner_id, owner_name, review_due, tags)
VALUES
('aaaaaaaa-0016-0201-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'Information Security Policy', 'information_security', '2.1', 'active', '# Information Security Policy\n\nThis policy establishes the framework for protecting organizational information assets.', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', '2026-12-31', '{"core","mandatory","iso27001"}'),
('aaaaaaaa-0016-0201-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'Acceptable Use Policy', 'acceptable_use', '1.3', 'active', '# Acceptable Use Policy\n\nDefines acceptable use of IT resources by all employees and contractors.', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', '2026-09-30', '{"hr","onboarding","compliance"}'),
('aaaaaaaa-0016-0201-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'Incident Response Policy', 'incident_response', '1.0', 'draft', '# Incident Response Policy\n\nProcedures for detecting, responding to, and recovering from security incidents.', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', '2026-06-30', '{"incident","soc","nist"}'),
('aaaaaaaa-0016-0201-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'Data Classification Policy', 'data_protection', '1.5', 'active', '# Data Classification Policy\n\nDefines classification levels: Public, Internal, Confidential, Restricted.', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', '2027-01-15', '{"data","classification","dlp"}');

-- ─── 3. vciso_policy_exceptions seed ────────────────────────────────────────

INSERT INTO vciso_policy_exceptions (id, tenant_id, policy_id, policy_title, title, description, justification, compensating_controls, status, requested_by, requested_by_name, expires_at)
VALUES
('aaaaaaaa-0016-0301-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'aaaaaaaa-0016-0201-0000-000000000001', 'Information Security Policy', 'Legacy System MFA Exemption', 'Legacy ERP system does not support modern MFA protocols.', 'System is scheduled for replacement in Q3 2026; MFA integration not cost-effective.', 'VPN-only access, enhanced logging, IP whitelisting.', 'approved', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', '2026-09-30'),
('aaaaaaaa-0016-0301-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'aaaaaaaa-0016-0201-0000-000000000002', 'Acceptable Use Policy', 'Developer BYOD Exception', 'Development team needs personal devices for on-call support.', 'Critical business need for 24/7 incident response capability.', 'MDM enrollment required, corporate VPN mandatory, remote wipe capability.', 'pending', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', '2026-12-31'),
('aaaaaaaa-0016-0301-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'aaaaaaaa-0016-0201-0000-000000000004', 'Data Classification Policy', 'Temporary Unencrypted Transfer', 'Short-term exception for bulk data migration to new storage platform.', 'Encryption overhead causes unacceptable transfer speeds for the 50TB migration.', 'Isolated network segment, transfer monitoring, data verification checksums.', 'pending', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', '2026-05-15');

-- ─── 4. vciso_vendors seed ──────────────────────────────────────────────────

INSERT INTO vciso_vendors (id, tenant_id, name, category, risk_tier, status, risk_score, next_review_date, contact_name, contact_email, services_provided, data_shared, compliance_frameworks, controls_met, controls_total, open_findings)
VALUES
('aaaaaaaa-0016-0401-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'CloudGuard Security', 'cloud_security', 'critical', 'active', 25, '2026-06-30', 'Jane Smith', 'jane@cloudguard.example', '{"SIEM","EDR","Threat Intelligence"}', '{"security_logs","endpoint_telemetry"}', '{"SOC2","ISO27001","FedRAMP"}', 42, 45, 2),
('aaaaaaaa-0016-0401-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'DataVault Inc.', 'data_management', 'high', 'active', 45, '2026-05-15', 'Bob Johnson', 'bob@datavault.example', '{"Backup","Disaster Recovery","Archival"}', '{"customer_data","financial_records"}', '{"SOC2","HIPAA"}', 30, 38, 5),
('aaaaaaaa-0016-0401-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'NetConnect Solutions', 'networking', 'medium', 'under_review', 55, '2026-04-30', 'Alice Chen', 'alice@netconnect.example', '{"SD-WAN","Firewall Management"}', '{"network_configs","traffic_logs"}', '{"ISO27001"}', 18, 25, 3),
('aaaaaaaa-0016-0401-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'TalentHub HR', 'hr_technology', 'low', 'active', 20, '2027-01-15', 'Mark Lee', 'mark@talenthub.example', '{"HRIS","Payroll Processing"}', '{"employee_pii","compensation_data"}', '{"SOC2","GDPR"}', 28, 30, 0);

-- ─── 5. vciso_questionnaires seed ───────────────────────────────────────────

INSERT INTO vciso_questionnaires (id, tenant_id, title, type, status, vendor_id, vendor_name, total_questions, answered_questions, due_date, assigned_to, assigned_to_name)
VALUES
('aaaaaaaa-0016-0501-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'CloudGuard Annual Security Assessment', 'vendor_assessment', 'in_progress', 'aaaaaaaa-0016-0401-0000-000000000001', 'CloudGuard Security', 85, 62, '2026-05-15', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User'),
('aaaaaaaa-0016-0501-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'SOC 2 Readiness Self-Assessment', 'internal_audit', 'draft', NULL, NULL, 120, 0, '2026-06-30', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User'),
('aaaaaaaa-0016-0501-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'DataVault HIPAA Compliance Check', 'vendor_assessment', 'completed', 'aaaaaaaa-0016-0401-0000-000000000002', 'DataVault Inc.', 60, 60, '2026-03-01', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User');

-- ─── 6. vciso_evidence seed ─────────────────────────────────────────────────

INSERT INTO vciso_evidence (id, tenant_id, title, description, type, source, status, frameworks, control_ids, file_name, file_size, collected_at)
VALUES
('aaaaaaaa-0016-0601-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'Penetration Test Report Q1 2026', 'External penetration test conducted by CyberAudit Partners.', 'report', 'external_assessment', 'verified', '{"NIST CSF","SOC2"}', '{"CA-8","RA-5"}', 'pentest_q1_2026.pdf', 2450000, '2026-02-28'),
('aaaaaaaa-0016-0601-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'Firewall Configuration Backup', 'Automated backup of production firewall rulesets.', 'configuration', 'automated_scan', 'active', '{"NIST CSF"}', '{"SC-7","AC-4"}', 'fw_config_backup_2026-03.tar.gz', 850000, '2026-03-10'),
('aaaaaaaa-0016-0601-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'Security Awareness Training Completion', 'Q1 training completion certificates and scores.', 'certificate', 'internal', 'active', '{"ISO27001","SOC2"}', '{"AT-2","AT-3"}', 'training_completion_q1.xlsx', 125000, '2026-03-15'),
('aaaaaaaa-0016-0601-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'Access Review Evidence March 2026', 'Quarterly access review documentation for privileged accounts.', 'screenshot', 'internal', 'pending', '{"SOC2","HIPAA"}', '{"AC-2","AC-6"}', NULL, NULL, '2026-03-12');

-- ─── 7. vciso_maturity_assessments seed ─────────────────────────────────────

INSERT INTO vciso_maturity_assessments (id, tenant_id, framework, status, overall_score, overall_level, dimensions, assessor_name, assessed_at)
VALUES
('aaaaaaaa-0016-0701-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'NIST CSF', 'completed', 3.2, 3,
 '[{"name":"Identify","category":"governance","current_level":3,"target_level":4,"score":3.5,"findings":["Asset inventory incomplete"],"recommendations":["Deploy automated discovery"]},{"name":"Protect","category":"security","current_level":3,"target_level":4,"score":3.0,"findings":["MFA not universal"],"recommendations":["Enforce MFA everywhere"]},{"name":"Detect","category":"operations","current_level":3,"target_level":5,"score":3.2,"findings":["Limited correlation"],"recommendations":["Implement SOAR"]},{"name":"Respond","category":"operations","current_level":4,"target_level":5,"score":3.8,"findings":["Playbooks need updating"],"recommendations":["Quarterly tabletop exercises"]},{"name":"Recover","category":"resilience","current_level":2,"target_level":4,"score":2.5,"findings":["DR testing infrequent"],"recommendations":["Monthly DR drills"]}]',
 'Security Assessment Team', '2026-02-15'),
('aaaaaaaa-0016-0701-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'ISO 27001', 'in_progress', 2.8, 3,
 '[{"name":"Information Security Policies","category":"governance","current_level":3,"target_level":4,"score":3.0,"findings":["Policies need update"],"recommendations":["Annual policy review cycle"]},{"name":"Asset Management","category":"governance","current_level":2,"target_level":4,"score":2.5,"findings":["No CMDB"],"recommendations":["Implement CMDB"]},{"name":"Access Control","category":"security","current_level":3,"target_level":4,"score":3.0,"findings":["Weak password policy"],"recommendations":["Deploy passwordless auth"]}]',
 'External Auditor', '2026-03-01'),
('aaaaaaaa-0016-0701-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'CIS Controls v8', 'completed', 3.5, 4,
 '[{"name":"Inventory and Control of Assets","category":"governance","current_level":4,"target_level":5,"score":3.8,"findings":["Shadow IT present"],"recommendations":["Network access control"]},{"name":"Data Protection","category":"security","current_level":3,"target_level":4,"score":3.2,"findings":["DLP gaps"],"recommendations":["Deploy DLP solution"]}]',
 'Security Assessment Team', '2026-01-20');

-- ─── 8. vciso_budget_items seed ─────────────────────────────────────────────

INSERT INTO vciso_budget_items (id, tenant_id, title, category, type, amount, currency, status, risk_reduction_estimate, priority, justification, linked_risk_ids, linked_recommendation_ids, fiscal_year, quarter, owner_name)
VALUES
('aaaaaaaa-0016-0801-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'EDR Platform Deployment', 'endpoint_security', 'capex', 185000.00, 'USD', 'approved', 25.0, 1, 'Critical for ransomware protection and threat detection across all endpoints.', '{"aaaaaaaa-0016-0101-0000-000000000001"}', '{}', 'FY2026', 'Q2', 'Admin User'),
('aaaaaaaa-0016-0801-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'Security Awareness Platform License', 'training', 'opex', 45000.00, 'USD', 'approved', 15.0, 2, 'Annual subscription for phishing simulation and security awareness training.', '{"aaaaaaaa-0016-0101-0000-000000000005"}', '{}', 'FY2026', 'Q1', 'Admin User'),
('aaaaaaaa-0016-0801-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'PAM Solution Implementation', 'identity_security', 'capex', 120000.00, 'USD', 'proposed', 20.0, 1, 'Privileged Access Management to address insider threat and compliance gaps.', '{"aaaaaaaa-0016-0101-0000-000000000003"}', '{}', 'FY2026', 'Q3', 'Admin User'),
('aaaaaaaa-0016-0801-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'Vulnerability Scanner Upgrade', 'vulnerability_management', 'opex', 35000.00, 'USD', 'spent', 10.0, 3, 'Upgrade from legacy scanner to support cloud-native workloads.', '{}', '{}', 'FY2026', 'Q1', 'Admin User'),
('aaaaaaaa-0016-0801-0000-000000000005', 'aaaaaaaa-0000-0000-0000-000000000001', 'Third-Party Risk Management Platform', 'third_party_risk', 'capex', 75000.00, 'USD', 'proposed', 12.0, 2, 'Automate vendor risk assessments and continuous monitoring.', '{"aaaaaaaa-0016-0101-0000-000000000002"}', '{}', 'FY2026', 'Q4', 'Admin User');

-- ─── 9. vciso_awareness_programs seed ───────────────────────────────────────

INSERT INTO vciso_awareness_programs (id, tenant_id, name, type, status, total_users, completed_users, passed_users, failed_users, completion_rate, pass_rate, start_date, end_date)
VALUES
('aaaaaaaa-0016-0901-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'Annual Security Awareness Training 2026', 'training', 'active', 500, 380, 365, 15, 76.0, 73.0, '2026-01-15', '2026-03-31'),
('aaaaaaaa-0016-0901-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'Phishing Simulation Q1', 'phishing_simulation', 'completed', 500, 500, 420, 80, 100.0, 84.0, '2026-02-01', '2026-02-28'),
('aaaaaaaa-0016-0901-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'Secure Coding Workshop', 'workshop', 'planned', 50, 0, 0, 0, 0.0, 0.0, '2026-04-15', '2026-04-17'),
('aaaaaaaa-0016-0901-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'Executive Cyber Briefing', 'briefing', 'active', 15, 10, 10, 0, 66.67, 66.67, '2026-03-01', '2026-03-30');

-- ─── 10. vciso_iam_findings seed ────────────────────────────────────────────

INSERT INTO vciso_iam_findings (id, tenant_id, type, severity, title, description, affected_users, status, remediation, discovered_at)
VALUES
('aaaaaaaa-0016-1001-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'over_privileged', 'high', 'Excessive Admin Rights in Production', '15 users have full admin access to production systems beyond their role requirements.', 15, 'open', 'Revoke unnecessary admin privileges and implement role-based access.', '2026-03-01'),
('aaaaaaaa-0016-1001-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'orphaned_account', 'medium', 'Orphaned Service Accounts Detected', '8 service accounts have no associated owner or last login over 180 days.', 8, 'open', 'Disable orphaned accounts and establish service account lifecycle policy.', '2026-02-20'),
('aaaaaaaa-0016-1001-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'stale_access', 'medium', 'Stale VPN Access for Former Contractors', '12 contractor accounts still have active VPN credentials after contract end.', 12, 'remediated', 'Revoked all contractor VPN access and implemented automated offboarding.', '2026-01-15'),
('aaaaaaaa-0016-1001-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'mfa_gap', 'critical', 'MFA Not Enforced for Cloud Admin Console', 'Cloud platform admin console accessible without MFA for 3 admin accounts.', 3, 'open', NULL, '2026-03-10'),
('aaaaaaaa-0016-1001-0000-000000000005', 'aaaaaaaa-0000-0000-0000-000000000001', 'shared_credentials', 'high', 'Shared Credentials for Database Admin', 'Multiple team members using a single shared database admin credential.', 5, 'open', 'Deploy individual accounts with PAM and rotate shared credential.', '2026-02-28');

-- ─── 11. vciso_escalation_rules seed ────────────────────────────────────────

INSERT INTO vciso_escalation_rules (id, tenant_id, name, description, trigger_type, trigger_condition, escalation_target, target_contacts, notification_channels, enabled, trigger_count)
VALUES
('aaaaaaaa-0016-1101-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'Critical Incident Auto-Escalation', 'Automatically escalate critical severity incidents to CISO and CTO.', 'severity', 'severity == critical AND status == open AND age_hours > 1', 'executive', '{"ciso@example.com","cto@example.com"}', '{"email","sms","slack"}', true, 3),
('aaaaaaaa-0016-1101-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'Unacknowledged High-Risk Alert', 'Escalate high-risk alerts not acknowledged within 30 minutes.', 'acknowledgement_timeout', 'severity >= high AND acknowledged == false AND age_minutes > 30', 'soc_manager', '{"soc-lead@example.com"}', '{"email","slack"}', true, 7),
('aaaaaaaa-0016-1101-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'Vendor Breach Notification', 'Escalate when a vendor reports a data breach affecting our data.', 'vendor_notification', 'event_type == vendor_breach AND data_impact == true', 'legal_and_compliance', '{"legal@example.com","dpo@example.com","ciso@example.com"}', '{"email","phone","slack"}', true, 0);

-- ─── 12. vciso_playbooks seed ───────────────────────────────────────────────

INSERT INTO vciso_playbooks (id, tenant_id, name, scenario, status, next_test_date, owner_id, owner_name, steps_count, dependencies, rto_hours, rpo_hours)
VALUES
('aaaaaaaa-0016-1201-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'Ransomware Response Playbook', 'ransomware_attack', 'active', '2026-06-15', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', 12, '{"EDR","Backup System","Incident Response Team"}', 4.0, 1.0),
('aaaaaaaa-0016-1201-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'Data Breach Response Playbook', 'data_breach', 'active', '2026-05-30', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', 15, '{"Legal Team","PR Team","Forensics Vendor"}', 2.0, 0.5),
('aaaaaaaa-0016-1201-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'Business Continuity Plan', 'natural_disaster', 'draft', '2026-09-01', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', 20, '{"DR Site","Cloud Failover","Communication System"}', 8.0, 4.0),
('aaaaaaaa-0016-1201-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'DDoS Mitigation Playbook', 'ddos_attack', 'active', '2026-07-15', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', 8, '{"CDN Provider","ISP","WAF"}', 0.5, 0.0);

-- ─── 13. vciso_obligations seed ─────────────────────────────────────────────

INSERT INTO vciso_obligations (id, tenant_id, name, type, jurisdiction, description, requirements, status, mapped_controls, total_requirements, met_requirements, owner_id, owner_name, effective_date, review_date)
VALUES
('aaaaaaaa-0016-1301-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'GDPR Compliance', 'regulatory', 'EU', 'General Data Protection Regulation compliance for EU customer data.', '{"Data minimization","Right to erasure","Breach notification","DPO appointment","Privacy impact assessments"}', 'partially_compliant', 35, 50, 38, 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', '2018-05-25', '2026-06-01'),
('aaaaaaaa-0016-1301-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'SOC 2 Type II', 'framework', 'US', 'Service Organization Control 2 Type II audit requirements.', '{"Security controls","Availability","Processing integrity","Confidentiality","Privacy"}', 'compliant', 42, 45, 42, 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', '2025-01-01', '2026-12-31'),
('aaaaaaaa-0016-1301-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'HIPAA Security Rule', 'regulatory', 'US', 'Health Insurance Portability and Accountability Act security requirements.', '{"Administrative safeguards","Physical safeguards","Technical safeguards","Risk analysis","Contingency planning"}', 'partially_compliant', 28, 40, 30, NULL, NULL, '2003-04-14', '2026-09-15'),
('aaaaaaaa-0016-1301-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'PCI DSS v4.0', 'contractual', 'Global', 'Payment Card Industry Data Security Standard for payment processing.', '{"Network security","Cardholder data protection","Vulnerability management","Access control","Monitoring","Security policy"}', 'non_compliant', 15, 60, 20, 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', '2024-03-31', '2026-04-30');

-- ─── 14. vciso_control_tests seed ───────────────────────────────────────────

INSERT INTO vciso_control_tests (id, tenant_id, control_id, control_name, framework, test_type, result, tester_name, test_date, next_test_date, findings, evidence_ids, test_name)
VALUES
('aaaaaaaa-0016-1401-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'AC-2', 'Account Management', 'NIST SP 800-53', 'operational', 'pass', 'Security Analyst', '2026-02-15', '2026-05-15', 'All accounts reviewed; 3 stale accounts identified and disabled.', '{"aaaaaaaa-0016-0601-0000-000000000004"}', 'Account Management Quarterly Review'),
('aaaaaaaa-0016-1401-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'SC-7', 'Boundary Protection', 'NIST SP 800-53', 'technical', 'pass_with_findings', 'Network Engineer', '2026-03-01', '2026-06-01', 'Firewall rules effective but 5 overly permissive rules found.', '{"aaaaaaaa-0016-0601-0000-000000000002"}', 'Firewall Rule Effectiveness Test'),
('aaaaaaaa-0016-1401-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'CP-9', 'System Backup', 'NIST SP 800-53', 'operational', 'fail', 'Backup Administrator', '2026-02-28', '2026-03-28', 'Backup restoration test failed for 2 of 10 critical systems.', '{}', 'Backup Restoration Test'),
('aaaaaaaa-0016-1401-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'AT-2', 'Security Awareness Training', 'NIST SP 800-53', 'operational', 'pass', 'Training Coordinator', '2026-03-10', '2026-06-10', 'Training completion rate at 95%; phishing click rate below 5%.', '{"aaaaaaaa-0016-0601-0000-000000000003"}', 'Security Awareness Effectiveness Assessment');

-- ─── 15. vciso_integrations seed ────────────────────────────────────────────

INSERT INTO vciso_integrations (id, tenant_id, name, type, provider, status, sync_frequency, items_synced, config, health_status)
VALUES
('aaaaaaaa-0016-1501-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'Splunk SIEM Integration', 'siem', 'Splunk', 'connected', 'every_5m', 15420, '{"endpoint":"https://splunk.example.com:8089","index":"security_events","token":"[REDACTED]"}', 'healthy'),
('aaaaaaaa-0016-1501-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'CrowdStrike EDR Feed', 'cloud_security', 'CrowdStrike', 'connected', 'every_5m', 8930, '{"api_url":"https://api.crowdstrike.com","client_id":"[REDACTED]"}', 'healthy'),
('aaaaaaaa-0016-1501-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'ServiceNow ITSM', 'ticketing', 'ServiceNow', 'connected', 'every_hour', 3200, '{"instance":"dev12345.service-now.com","table":"incident"}', 'healthy'),
('aaaaaaaa-0016-1501-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'Qualys Vulnerability Scanner', 'asset_management', 'Qualys', 'disconnected', 'daily', 0, '{"api_url":"https://qualysapi.qualys.com","subscription":"[REDACTED]"}', 'degraded');

-- ─── 16. vciso_control_ownership seed ───────────────────────────────────────

INSERT INTO vciso_control_ownership (id, tenant_id, control_id, control_name, framework, owner_id, owner_name, delegate_id, delegate_name, status, next_review_date)
VALUES
('aaaaaaaa-0016-1601-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'AC-1', 'Access Control Policy', 'NIST SP 800-53', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', NULL, NULL, 'assigned', '2026-06-30'),
('aaaaaaaa-0016-1601-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'IR-1', 'Incident Response Policy', 'NIST SP 800-53', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', NULL, NULL, 'assigned', '2026-06-30'),
('aaaaaaaa-0016-1601-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'RA-1', 'Risk Assessment Policy', 'NIST SP 800-53', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', NULL, NULL, 'needs_review', '2026-04-15'),
('aaaaaaaa-0016-1601-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'CP-1', 'Contingency Planning Policy', 'NIST SP 800-53', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', NULL, NULL, 'assigned', '2026-09-30');

-- ─── 17. vciso_approvals seed ───────────────────────────────────────────────

INSERT INTO vciso_approvals (id, tenant_id, type, title, description, status, requested_by, requested_by_name, approver_id, approver_name, priority, deadline, linked_entity_type, linked_entity_id)
VALUES
('aaaaaaaa-0016-1701-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'policy_approval', 'Approve Incident Response Policy v1.0', 'New incident response policy requires CISO approval before activation.', 'pending', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', 'high', '2026-04-15', 'policy', 'aaaaaaaa-0016-0201-0000-000000000003'),
('aaaaaaaa-0016-1701-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'budget_approval', 'PAM Solution Budget Request', 'Budget request for $120,000 PAM implementation needs finance approval.', 'pending', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', 'medium', '2026-05-01', 'budget', 'aaaaaaaa-0016-0801-0000-000000000003'),
('aaaaaaaa-0016-1701-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'risk_acceptance', 'Accept GDPR Non-Compliance Risk', 'Request to formally accept residual GDPR compliance risk until remediation.', 'approved', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', 'critical', '2026-04-01', 'risk', 'aaaaaaaa-0016-0101-0000-000000000004'),
('aaaaaaaa-0016-1701-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'exception_approval', 'Approve Legacy System MFA Exception', 'Policy exception for legacy ERP MFA bypass needs security committee approval.', 'approved', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', 'bbbbbbbb-0000-0000-0000-000000000001', 'Admin User', 'high', '2026-03-31', 'policy_exception', 'aaaaaaaa-0016-0301-0000-000000000001');
