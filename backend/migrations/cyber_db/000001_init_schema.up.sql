-- =============================================================================
-- Clario 360 — Cyber Suite Database Schema
-- Database: cyber_db
-- Contains: assets, vulnerabilities, threats, alerts, detection rules,
--           remediation actions, CTEM assessments, DSPM data assets
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- ENUM TYPES
-- =============================================================================

CREATE TYPE asset_type AS ENUM (
    'server', 'endpoint', 'network_device', 'cloud_resource',
    'iot_device', 'application', 'database', 'container'
);
COMMENT ON TYPE asset_type IS 'Classification of IT/OT assets';

CREATE TYPE asset_criticality AS ENUM ('critical', 'high', 'medium', 'low');
COMMENT ON TYPE asset_criticality IS 'Business criticality level of an asset';

CREATE TYPE asset_status AS ENUM ('active', 'inactive', 'decommissioned');
COMMENT ON TYPE asset_status IS 'Lifecycle status of an asset';

CREATE TYPE severity_level AS ENUM ('critical', 'high', 'medium', 'low', 'info');
COMMENT ON TYPE severity_level IS 'Standard severity levels used across cyber entities';

CREATE TYPE vulnerability_status AS ENUM (
    'open', 'in_progress', 'mitigated', 'resolved', 'accepted', 'false_positive'
);
COMMENT ON TYPE vulnerability_status IS 'Lifecycle status of a vulnerability';

CREATE TYPE threat_status AS ENUM (
    'detected', 'investigating', 'confirmed', 'mitigated', 'false_positive'
);
COMMENT ON TYPE threat_status IS 'Lifecycle status of a threat';

CREATE TYPE indicator_type AS ENUM (
    'ip', 'domain', 'hash_md5', 'hash_sha1', 'hash_sha256',
    'url', 'email', 'filename', 'registry_key'
);
COMMENT ON TYPE indicator_type IS 'Types of threat indicators (IOCs)';

CREATE TYPE detection_rule_type AS ENUM ('sigma', 'yara', 'custom', 'ml_model');
COMMENT ON TYPE detection_rule_type IS 'Types of detection rules';

CREATE TYPE alert_status AS ENUM (
    'new', 'acknowledged', 'investigating', 'resolved', 'false_positive', 'escalated'
);
COMMENT ON TYPE alert_status IS 'Lifecycle status of a security alert';

CREATE TYPE remediation_type AS ENUM (
    'patch', 'config_change', 'block', 'isolate', 'custom'
);
COMMENT ON TYPE remediation_type IS 'Types of remediation actions';

CREATE TYPE remediation_status AS ENUM (
    'pending_approval', 'approved', 'dry_run', 'executing',
    'completed', 'failed', 'rolled_back'
);
COMMENT ON TYPE remediation_status IS 'Lifecycle status of a remediation action';

CREATE TYPE execution_mode AS ENUM ('manual', 'semi_auto', 'auto');
COMMENT ON TYPE execution_mode IS 'How a remediation action is executed';

CREATE TYPE ctem_status AS ENUM ('scheduled', 'running', 'completed', 'failed');
COMMENT ON TYPE ctem_status IS 'Status of a CTEM assessment';

CREATE TYPE data_classification AS ENUM ('public', 'internal', 'confidential', 'restricted');
COMMENT ON TYPE data_classification IS 'Data sensitivity classification levels';

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
-- TABLE: assets
-- =============================================================================

CREATE TABLE assets (
    id            UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID             NOT NULL,
    name          VARCHAR(255)     NOT NULL,
    type          asset_type       NOT NULL,
    ip_address    INET,
    hostname      VARCHAR(255),
    mac_address   VARCHAR(17),
    os            VARCHAR(100),
    os_version    VARCHAR(50),
    owner         UUID,
    department    VARCHAR(100),
    criticality   asset_criticality NOT NULL DEFAULT 'medium',
    status        asset_status     NOT NULL DEFAULT 'active',
    discovered_at TIMESTAMPTZ,
    last_seen_at  TIMESTAMPTZ,
    metadata      JSONB            NOT NULL DEFAULT '{}',
    tags          TEXT[]           DEFAULT '{}',
    created_at    TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    created_by    UUID,
    updated_by    UUID,
    deleted_at    TIMESTAMPTZ
);

COMMENT ON TABLE assets IS 'IT and OT assets tracked by the Cyber suite';
COMMENT ON COLUMN assets.tenant_id IS 'Owning tenant (references platform_core.tenants)';
COMMENT ON COLUMN assets.type IS 'Category of asset';
COMMENT ON COLUMN assets.owner IS 'User responsible for this asset (references platform_core.users)';
COMMENT ON COLUMN assets.criticality IS 'Business criticality ranking';
COMMENT ON COLUMN assets.discovered_at IS 'When the asset was first discovered by scanning';
COMMENT ON COLUMN assets.last_seen_at IS 'Most recent observation of the asset on the network';
COMMENT ON COLUMN assets.metadata IS 'Additional asset properties (JSON)';
COMMENT ON COLUMN assets.tags IS 'Freeform tags for categorization';

CREATE INDEX idx_assets_tenant_status ON assets (tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_assets_tenant_type ON assets (tenant_id, type) WHERE deleted_at IS NULL;
CREATE INDEX idx_assets_tenant_criticality ON assets (tenant_id, criticality) WHERE deleted_at IS NULL;
CREATE INDEX idx_assets_hostname ON assets (hostname) WHERE deleted_at IS NULL;
CREATE INDEX idx_assets_ip ON assets (ip_address) WHERE deleted_at IS NULL;
CREATE INDEX idx_assets_tags ON assets USING GIN (tags) WHERE deleted_at IS NULL;
CREATE INDEX idx_assets_metadata ON assets USING GIN (metadata);
CREATE INDEX idx_assets_tenant_created ON assets (tenant_id, created_at DESC);

CREATE TRIGGER trg_assets_updated_at
    BEFORE UPDATE ON assets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: asset_relationships
-- =============================================================================

CREATE TABLE asset_relationships (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    source_asset_id   UUID        NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    target_asset_id   UUID        NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    relationship_type VARCHAR(50) NOT NULL,
    metadata          JSONB       DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by        UUID,
    CONSTRAINT chk_no_self_relationship CHECK (source_asset_id != target_asset_id)
);

COMMENT ON TABLE asset_relationships IS 'Directed relationships between assets (e.g., hosts, connects_to)';
COMMENT ON COLUMN asset_relationships.relationship_type IS 'Type of relationship (e.g., hosts, depends_on, connects_to)';

CREATE INDEX idx_asset_rel_tenant ON asset_relationships (tenant_id);
CREATE INDEX idx_asset_rel_source ON asset_relationships (source_asset_id);
CREATE INDEX idx_asset_rel_target ON asset_relationships (target_asset_id);

-- =============================================================================
-- TABLE: vulnerabilities
-- =============================================================================

CREATE TABLE vulnerabilities (
    id            UUID                 PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID                 NOT NULL,
    asset_id      UUID                 NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    cve_id        VARCHAR(20),
    title         VARCHAR(500)         NOT NULL,
    description   TEXT                 NOT NULL DEFAULT '',
    severity      severity_level       NOT NULL DEFAULT 'medium',
    cvss_score    DECIMAL(3,1)         CHECK (cvss_score >= 0.0 AND cvss_score <= 10.0),
    cvss_vector   VARCHAR(100),
    status        vulnerability_status NOT NULL DEFAULT 'open',
    discovered_at TIMESTAMPTZ          NOT NULL DEFAULT NOW(),
    resolved_at   TIMESTAMPTZ,
    due_date      DATE,
    assigned_to   UUID,
    metadata      JSONB                DEFAULT '{}',
    created_at    TIMESTAMPTZ          NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ          NOT NULL DEFAULT NOW(),
    created_by    UUID,
    updated_by    UUID
);

COMMENT ON TABLE vulnerabilities IS 'Discovered vulnerabilities associated with assets';
COMMENT ON COLUMN vulnerabilities.cve_id IS 'Common Vulnerabilities and Exposures identifier';
COMMENT ON COLUMN vulnerabilities.cvss_score IS 'CVSS v3.1 base score (0.0 to 10.0)';
COMMENT ON COLUMN vulnerabilities.cvss_vector IS 'CVSS vector string';
COMMENT ON COLUMN vulnerabilities.assigned_to IS 'User assigned to remediate (references platform_core.users)';

CREATE INDEX idx_vulns_tenant_status ON vulnerabilities (tenant_id, status);
CREATE INDEX idx_vulns_tenant_severity ON vulnerabilities (tenant_id, severity);
CREATE INDEX idx_vulns_asset ON vulnerabilities (asset_id);
CREATE INDEX idx_vulns_cve ON vulnerabilities (cve_id) WHERE cve_id IS NOT NULL;
CREATE INDEX idx_vulns_due_date ON vulnerabilities (due_date) WHERE status NOT IN ('resolved', 'accepted', 'false_positive');
CREATE INDEX idx_vulns_tenant_created ON vulnerabilities (tenant_id, created_at DESC);
CREATE INDEX idx_vulns_metadata ON vulnerabilities USING GIN (metadata);

CREATE TRIGGER trg_vulnerabilities_updated_at
    BEFORE UPDATE ON vulnerabilities
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: threats
-- =============================================================================

CREATE TABLE threats (
    id                  UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID           NOT NULL,
    type                VARCHAR(100)   NOT NULL,
    title               VARCHAR(500)   NOT NULL,
    description         TEXT           NOT NULL DEFAULT '',
    severity            severity_level NOT NULL DEFAULT 'medium',
    confidence_score    DECIMAL(5,4)   CHECK (confidence_score >= 0.0 AND confidence_score <= 1.0),
    source              VARCHAR(255),
    indicators          JSONB          DEFAULT '[]',
    mitre_technique_id  VARCHAR(20),
    mitre_tactic        VARCHAR(100),
    status              threat_status  NOT NULL DEFAULT 'detected',
    detected_at         TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    resolved_at         TIMESTAMPTZ,
    metadata            JSONB          DEFAULT '{}',
    created_at          TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    created_by          UUID,
    updated_by          UUID
);

COMMENT ON TABLE threats IS 'Detected threats and threat intelligence entries';
COMMENT ON COLUMN threats.confidence_score IS 'ML confidence score between 0.0 and 1.0';
COMMENT ON COLUMN threats.source IS 'Source of threat intelligence (e.g., internal, OSINT, vendor)';
COMMENT ON COLUMN threats.indicators IS 'JSON array of associated indicators of compromise';
COMMENT ON COLUMN threats.mitre_technique_id IS 'MITRE ATT&CK technique ID (e.g., T1059)';
COMMENT ON COLUMN threats.mitre_tactic IS 'MITRE ATT&CK tactic (e.g., Initial Access)';

CREATE INDEX idx_threats_tenant_status ON threats (tenant_id, status);
CREATE INDEX idx_threats_tenant_severity ON threats (tenant_id, severity);
CREATE INDEX idx_threats_mitre ON threats (mitre_technique_id) WHERE mitre_technique_id IS NOT NULL;
CREATE INDEX idx_threats_detected ON threats (detected_at DESC);
CREATE INDEX idx_threats_tenant_created ON threats (tenant_id, created_at DESC);
CREATE INDEX idx_threats_indicators ON threats USING GIN (indicators);
CREATE INDEX idx_threats_metadata ON threats USING GIN (metadata);

CREATE TRIGGER trg_threats_updated_at
    BEFORE UPDATE ON threats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: threat_indicators
-- =============================================================================

CREATE TABLE threat_indicators (
    id          UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID           NOT NULL,
    threat_id   UUID           NOT NULL REFERENCES threats(id) ON DELETE CASCADE,
    type        indicator_type NOT NULL,
    value       TEXT           NOT NULL,
    confidence  DECIMAL(5,4)   CHECK (confidence >= 0.0 AND confidence <= 1.0),
    source      VARCHAR(255),
    first_seen  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    last_seen   TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE threat_indicators IS 'Individual indicators of compromise (IOCs) linked to threats';
COMMENT ON COLUMN threat_indicators.type IS 'IOC type (IP, domain, hash, etc.)';
COMMENT ON COLUMN threat_indicators.value IS 'The actual indicator value';
COMMENT ON COLUMN threat_indicators.confidence IS 'Confidence that this IOC is malicious (0.0 to 1.0)';

CREATE INDEX idx_indicators_tenant ON threat_indicators (tenant_id);
CREATE INDEX idx_indicators_threat ON threat_indicators (threat_id);
CREATE INDEX idx_indicators_type_value ON threat_indicators (type, value);
CREATE INDEX idx_indicators_value ON threat_indicators (value);

-- =============================================================================
-- TABLE: detection_rules
-- =============================================================================

CREATE TABLE detection_rules (
    id                UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID               NOT NULL,
    name              VARCHAR(255)       NOT NULL,
    description       TEXT               NOT NULL DEFAULT '',
    rule_type         detection_rule_type NOT NULL,
    rule_content      TEXT               NOT NULL,
    severity          severity_level     NOT NULL DEFAULT 'medium',
    mitre_techniques  TEXT[]             DEFAULT '{}',
    enabled           BOOLEAN            NOT NULL DEFAULT true,
    last_triggered_at TIMESTAMPTZ,
    trigger_count     BIGINT             NOT NULL DEFAULT 0,
    created_by        UUID,
    created_at        TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    updated_by        UUID
);

COMMENT ON TABLE detection_rules IS 'Security detection rules (Sigma, YARA, custom, ML models)';
COMMENT ON COLUMN detection_rules.rule_content IS 'The actual rule definition (Sigma YAML, YARA rule, etc.)';
COMMENT ON COLUMN detection_rules.mitre_techniques IS 'MITRE ATT&CK technique IDs this rule detects';
COMMENT ON COLUMN detection_rules.trigger_count IS 'Number of times this rule has triggered';

CREATE INDEX idx_rules_tenant ON detection_rules (tenant_id);
CREATE INDEX idx_rules_tenant_enabled ON detection_rules (tenant_id, enabled) WHERE enabled = true;
CREATE INDEX idx_rules_type ON detection_rules (rule_type);
CREATE INDEX idx_rules_severity ON detection_rules (severity);
CREATE INDEX idx_rules_mitre ON detection_rules USING GIN (mitre_techniques);

CREATE TRIGGER trg_detection_rules_updated_at
    BEFORE UPDATE ON detection_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: alerts
-- =============================================================================

CREATE TABLE alerts (
    id                    UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID           NOT NULL,
    rule_id               UUID           REFERENCES detection_rules(id) ON DELETE SET NULL,
    title                 VARCHAR(500)   NOT NULL,
    description           TEXT           NOT NULL DEFAULT '',
    severity              severity_level NOT NULL DEFAULT 'medium',
    status                alert_status   NOT NULL DEFAULT 'new',
    confidence_score      DECIMAL(5,4)   CHECK (confidence_score >= 0.0 AND confidence_score <= 1.0),
    explanation           JSONB          DEFAULT '{}',
    contributing_factors  JSONB          DEFAULT '[]',
    affected_assets       UUID[]         DEFAULT '{}',
    assigned_to           UUID,
    acknowledged_at       TIMESTAMPTZ,
    resolved_at           TIMESTAMPTZ,
    resolution_notes      TEXT,
    created_at            TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    created_by            UUID,
    updated_by            UUID
);

COMMENT ON TABLE alerts IS 'Security alerts generated by detection rules or ML models';
COMMENT ON COLUMN alerts.explanation IS 'AI explainability output (SHAP/LIME values)';
COMMENT ON COLUMN alerts.contributing_factors IS 'JSON array of factors that contributed to the alert';
COMMENT ON COLUMN alerts.affected_assets IS 'Array of asset UUIDs involved in this alert';
COMMENT ON COLUMN alerts.resolution_notes IS 'Notes on how the alert was resolved';

CREATE INDEX idx_alerts_tenant_status ON alerts (tenant_id, status);
CREATE INDEX idx_alerts_tenant_severity ON alerts (tenant_id, severity);
CREATE INDEX idx_alerts_tenant_created ON alerts (tenant_id, created_at DESC);
CREATE INDEX idx_alerts_rule ON alerts (rule_id);
CREATE INDEX idx_alerts_assigned ON alerts (assigned_to) WHERE status NOT IN ('resolved', 'false_positive');
CREATE INDEX idx_alerts_affected_assets ON alerts USING GIN (affected_assets);
CREATE INDEX idx_alerts_explanation ON alerts USING GIN (explanation);
CREATE INDEX idx_alerts_factors ON alerts USING GIN (contributing_factors);

CREATE TRIGGER trg_alerts_updated_at
    BEFORE UPDATE ON alerts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: remediation_actions
-- =============================================================================

CREATE TABLE remediation_actions (
    id                UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID               NOT NULL,
    alert_id          UUID               REFERENCES alerts(id) ON DELETE SET NULL,
    vulnerability_id  UUID               REFERENCES vulnerabilities(id) ON DELETE SET NULL,
    type              remediation_type   NOT NULL,
    status            remediation_status NOT NULL DEFAULT 'pending_approval',
    execution_mode    execution_mode     NOT NULL DEFAULT 'manual',
    dry_run_result    JSONB,
    execution_result  JSONB,
    rollback_data     JSONB,
    approved_by       UUID,
    executed_by       UUID,
    executed_at       TIMESTAMPTZ,
    completed_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    created_by        UUID,
    updated_by        UUID
);

COMMENT ON TABLE remediation_actions IS 'Automated or manual remediation actions for alerts and vulnerabilities';
COMMENT ON COLUMN remediation_actions.dry_run_result IS 'Result of dry run execution (if performed)';
COMMENT ON COLUMN remediation_actions.execution_result IS 'Outcome of actual execution';
COMMENT ON COLUMN remediation_actions.rollback_data IS 'Data needed to reverse the action';

CREATE INDEX idx_remediation_tenant_status ON remediation_actions (tenant_id, status);
CREATE INDEX idx_remediation_alert ON remediation_actions (alert_id);
CREATE INDEX idx_remediation_vuln ON remediation_actions (vulnerability_id);
CREATE INDEX idx_remediation_tenant_created ON remediation_actions (tenant_id, created_at DESC);

CREATE TRIGGER trg_remediation_updated_at
    BEFORE UPDATE ON remediation_actions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: ctem_assessments
-- =============================================================================

CREATE TABLE ctem_assessments (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL,
    name            VARCHAR(255) NOT NULL,
    scope           JSONB       NOT NULL DEFAULT '{}',
    status          ctem_status NOT NULL DEFAULT 'scheduled',
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    findings_count  INTEGER     NOT NULL DEFAULT 0,
    critical_count  INTEGER     NOT NULL DEFAULT 0,
    high_count      INTEGER     NOT NULL DEFAULT 0,
    medium_count    INTEGER     NOT NULL DEFAULT 0,
    low_count       INTEGER     NOT NULL DEFAULT 0,
    report          JSONB       DEFAULT '{}',
    created_by      UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by      UUID
);

COMMENT ON TABLE ctem_assessments IS 'Continuous Threat Exposure Management assessments';
COMMENT ON COLUMN ctem_assessments.scope IS 'Assessment scope definition (asset ranges, types, etc.)';
COMMENT ON COLUMN ctem_assessments.report IS 'Full assessment report as structured JSON';

CREATE INDEX idx_ctem_tenant_status ON ctem_assessments (tenant_id, status);
CREATE INDEX idx_ctem_tenant_created ON ctem_assessments (tenant_id, created_at DESC);
CREATE INDEX idx_ctem_scope ON ctem_assessments USING GIN (scope);

CREATE TRIGGER trg_ctem_updated_at
    BEFORE UPDATE ON ctem_assessments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: dspm_data_assets
-- =============================================================================

CREATE TABLE dspm_data_assets (
    id                UUID                PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID                NOT NULL,
    name              VARCHAR(255)        NOT NULL,
    type              VARCHAR(100)        NOT NULL,
    location          TEXT                NOT NULL,
    classification    data_classification NOT NULL DEFAULT 'internal',
    sensitivity_score DECIMAL(5,4)        CHECK (sensitivity_score >= 0.0 AND sensitivity_score <= 1.0),
    owner             UUID,
    data_types        TEXT[]              DEFAULT '{}',
    risk_score        DECIMAL(5,4)        CHECK (risk_score >= 0.0 AND risk_score <= 1.0),
    last_scanned_at   TIMESTAMPTZ,
    metadata          JSONB               DEFAULT '{}',
    created_at        TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    created_by        UUID,
    updated_by        UUID
);

COMMENT ON TABLE dspm_data_assets IS 'Data Security Posture Management — tracked data stores and assets';
COMMENT ON COLUMN dspm_data_assets.classification IS 'Data classification level';
COMMENT ON COLUMN dspm_data_assets.sensitivity_score IS 'Calculated sensitivity score (0.0 to 1.0)';
COMMENT ON COLUMN dspm_data_assets.data_types IS 'Types of data found (e.g., PII, PHI, financial)';
COMMENT ON COLUMN dspm_data_assets.risk_score IS 'Calculated risk score (0.0 to 1.0)';

CREATE INDEX idx_dspm_tenant ON dspm_data_assets (tenant_id);
CREATE INDEX idx_dspm_classification ON dspm_data_assets (tenant_id, classification);
CREATE INDEX idx_dspm_risk ON dspm_data_assets (tenant_id, risk_score DESC NULLS LAST);
CREATE INDEX idx_dspm_data_types ON dspm_data_assets USING GIN (data_types);
CREATE INDEX idx_dspm_metadata ON dspm_data_assets USING GIN (metadata);

CREATE TRIGGER trg_dspm_updated_at
    BEFORE UPDATE ON dspm_data_assets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
