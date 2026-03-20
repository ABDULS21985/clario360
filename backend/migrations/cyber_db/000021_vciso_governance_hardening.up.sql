-- =============================================================================
-- Migration 000021: vCISO Governance Hardening
-- Adds: vciso_benchmarks table, vciso_control_dependencies table,
--        verified_by column on vciso_evidence
-- =============================================================================

-- 1. Add verified_by to vciso_evidence for audit trail
ALTER TABLE vciso_evidence ADD COLUMN IF NOT EXISTS verified_by UUID;

-- 2. vciso_benchmarks — Industry benchmark data (replaces hardcoded mock)
CREATE TABLE IF NOT EXISTS vciso_benchmarks (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    dimension            VARCHAR(255) NOT NULL,
    category             VARCHAR(100) NOT NULL,
    organization_score   DOUBLE PRECISION DEFAULT 0,
    industry_average     DOUBLE PRECISION DEFAULT 0,
    industry_top_quartile DOUBLE PRECISION DEFAULT 0,
    peer_average         DOUBLE PRECISION DEFAULT 0,
    gap                  DOUBLE PRECISION DEFAULT 0,
    framework            VARCHAR(255) DEFAULT '',
    created_at           TIMESTAMPTZ DEFAULT NOW(),
    updated_at           TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_vciso_benchmarks_tenant ON vciso_benchmarks(tenant_id);
CREATE INDEX IF NOT EXISTS idx_vciso_benchmarks_category ON vciso_benchmarks(tenant_id, category);

-- 3. vciso_control_dependencies — Control dependency graph (replaces hardcoded mock)
CREATE TABLE IF NOT EXISTS vciso_control_dependencies (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL,
    control_id        VARCHAR(100) NOT NULL,
    control_name      VARCHAR(500) NOT NULL,
    framework         VARCHAR(255) NOT NULL,
    depends_on        TEXT[] DEFAULT '{}',
    depended_by       TEXT[] DEFAULT '{}',
    risk_domains      TEXT[] DEFAULT '{}',
    compliance_domains TEXT[] DEFAULT '{}',
    failure_impact    VARCHAR(50) DEFAULT 'medium',
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_vciso_control_deps_tenant ON vciso_control_dependencies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_vciso_control_deps_framework ON vciso_control_dependencies(tenant_id, framework);

-- =============================================================================
-- Seed benchmarks for dev tenant
-- =============================================================================
INSERT INTO vciso_benchmarks (id, tenant_id, dimension, category, organization_score, industry_average, industry_top_quartile, peer_average, gap, framework)
VALUES
('aaaaaaaa-0021-0101-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'Identity & Access Management', 'security', 72, 65, 82, 68, 7, 'NIST CSF'),
('aaaaaaaa-0021-0101-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'Data Protection', 'security', 68, 70, 85, 71, -2, 'NIST CSF'),
('aaaaaaaa-0021-0101-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'Incident Response', 'operations', 75, 60, 80, 63, 15, 'NIST CSF'),
('aaaaaaaa-0021-0101-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'Vulnerability Management', 'security', 60, 62, 78, 64, -2, 'NIST CSF'),
('aaaaaaaa-0021-0101-0000-000000000005', 'aaaaaaaa-0000-0000-0000-000000000001', 'Security Awareness', 'people', 55, 58, 75, 60, -3, 'NIST CSF'),
('aaaaaaaa-0021-0101-0000-000000000006', 'aaaaaaaa-0000-0000-0000-000000000001', 'Cloud Security', 'security', 65, 63, 80, 66, 2, 'NIST CSF'),
('aaaaaaaa-0021-0101-0000-000000000007', 'aaaaaaaa-0000-0000-0000-000000000001', 'Compliance', 'governance', 78, 72, 88, 74, 6, 'NIST CSF'),
('aaaaaaaa-0021-0101-0000-000000000008', 'aaaaaaaa-0000-0000-0000-000000000001', 'Risk Management', 'governance', 70, 66, 82, 69, 4, 'NIST CSF');

-- =============================================================================
-- Seed control dependencies for dev tenant
-- =============================================================================
INSERT INTO vciso_control_dependencies (id, tenant_id, control_id, control_name, framework, depends_on, depended_by, risk_domains, compliance_domains, failure_impact)
VALUES
('aaaaaaaa-0021-0201-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001', 'AC-1', 'Access Control Policy', 'NIST', '{}', '{"AC-2","AC-3"}', '{"identity"}', '{"access_control"}', 'high'),
('aaaaaaaa-0021-0201-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001', 'AC-2', 'Account Management', 'NIST', '{"AC-1"}', '{"AC-6"}', '{"identity"}', '{"access_control"}', 'high'),
('aaaaaaaa-0021-0201-0000-000000000003', 'aaaaaaaa-0000-0000-0000-000000000001', 'AC-3', 'Access Enforcement', 'NIST', '{"AC-1"}', '{"AC-6"}', '{"identity"}', '{"access_control"}', 'critical'),
('aaaaaaaa-0021-0201-0000-000000000004', 'aaaaaaaa-0000-0000-0000-000000000001', 'IR-1', 'Incident Response Policy', 'NIST', '{}', '{"IR-4","IR-5"}', '{"operations"}', '{"incident_response"}', 'high'),
('aaaaaaaa-0021-0201-0000-000000000005', 'aaaaaaaa-0000-0000-0000-000000000001', 'RA-1', 'Risk Assessment Policy', 'NIST', '{}', '{"RA-3","RA-5"}', '{"governance"}', '{"risk_assessment"}', 'medium');
