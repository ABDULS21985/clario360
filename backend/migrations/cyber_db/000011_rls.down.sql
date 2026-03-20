-- Removes Row-Level Security from all tenant-scoped tables in cyber_db.

-- TABLE: vciso_briefings
ALTER TABLE vciso_briefings DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON vciso_briefings;
DROP POLICY IF EXISTS tenant_insert ON vciso_briefings;
DROP POLICY IF EXISTS tenant_update ON vciso_briefings;
DROP POLICY IF EXISTS tenant_delete ON vciso_briefings;

-- TABLE: scan_history
ALTER TABLE scan_history DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON scan_history;
DROP POLICY IF EXISTS tenant_insert ON scan_history;
DROP POLICY IF EXISTS tenant_update ON scan_history;
DROP POLICY IF EXISTS tenant_delete ON scan_history;

-- TABLE: dspm_scans
ALTER TABLE dspm_scans DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON dspm_scans;
DROP POLICY IF EXISTS tenant_insert ON dspm_scans;
DROP POLICY IF EXISTS tenant_update ON dspm_scans;
DROP POLICY IF EXISTS tenant_delete ON dspm_scans;

-- TABLE: dspm_data_assets
ALTER TABLE dspm_data_assets DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON dspm_data_assets;
DROP POLICY IF EXISTS tenant_insert ON dspm_data_assets;
DROP POLICY IF EXISTS tenant_update ON dspm_data_assets;
DROP POLICY IF EXISTS tenant_delete ON dspm_data_assets;

-- TABLE: ctem_assessments
ALTER TABLE ctem_assessments DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON ctem_assessments;
DROP POLICY IF EXISTS tenant_insert ON ctem_assessments;
DROP POLICY IF EXISTS tenant_update ON ctem_assessments;
DROP POLICY IF EXISTS tenant_delete ON ctem_assessments;

-- TABLE: remediation_audit_trail
ALTER TABLE remediation_audit_trail DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON remediation_audit_trail;
DROP POLICY IF EXISTS tenant_insert ON remediation_audit_trail;
DROP POLICY IF EXISTS tenant_update ON remediation_audit_trail;
DROP POLICY IF EXISTS tenant_delete ON remediation_audit_trail;

-- TABLE: remediation_actions
ALTER TABLE remediation_actions DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON remediation_actions;
DROP POLICY IF EXISTS tenant_insert ON remediation_actions;
DROP POLICY IF EXISTS tenant_update ON remediation_actions;
DROP POLICY IF EXISTS tenant_delete ON remediation_actions;

-- TABLE: alerts
ALTER TABLE alerts DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON alerts;
DROP POLICY IF EXISTS tenant_insert ON alerts;
DROP POLICY IF EXISTS tenant_update ON alerts;
DROP POLICY IF EXISTS tenant_delete ON alerts;

-- TABLE: security_events
ALTER TABLE security_events DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON security_events;
DROP POLICY IF EXISTS tenant_insert ON security_events;
DROP POLICY IF EXISTS tenant_update ON security_events;
DROP POLICY IF EXISTS tenant_delete ON security_events;

-- TABLE: alert_timeline
ALTER TABLE alert_timeline DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON alert_timeline;
DROP POLICY IF EXISTS tenant_insert ON alert_timeline;
DROP POLICY IF EXISTS tenant_update ON alert_timeline;
DROP POLICY IF EXISTS tenant_delete ON alert_timeline;

-- TABLE: alert_comments
ALTER TABLE alert_comments DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON alert_comments;
DROP POLICY IF EXISTS tenant_insert ON alert_comments;
DROP POLICY IF EXISTS tenant_update ON alert_comments;
DROP POLICY IF EXISTS tenant_delete ON alert_comments;

-- TABLE: detection_rules
ALTER TABLE detection_rules DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON detection_rules;
DROP POLICY IF EXISTS tenant_insert ON detection_rules;
DROP POLICY IF EXISTS tenant_update ON detection_rules;
DROP POLICY IF EXISTS tenant_delete ON detection_rules;
DROP POLICY IF EXISTS template_select ON detection_rules;
DROP POLICY IF EXISTS template_insert ON detection_rules;
DROP POLICY IF EXISTS template_update ON detection_rules;

-- TABLE: threat_indicators
ALTER TABLE threat_indicators DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON threat_indicators;
DROP POLICY IF EXISTS tenant_insert ON threat_indicators;
DROP POLICY IF EXISTS tenant_update ON threat_indicators;
DROP POLICY IF EXISTS tenant_delete ON threat_indicators;

-- TABLE: threats
ALTER TABLE threats DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON threats;
DROP POLICY IF EXISTS tenant_insert ON threats;
DROP POLICY IF EXISTS tenant_update ON threats;
DROP POLICY IF EXISTS tenant_delete ON threats;

-- TABLE: vulnerabilities
ALTER TABLE vulnerabilities DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON vulnerabilities;
DROP POLICY IF EXISTS tenant_insert ON vulnerabilities;
DROP POLICY IF EXISTS tenant_update ON vulnerabilities;
DROP POLICY IF EXISTS tenant_delete ON vulnerabilities;

-- TABLE: asset_relationships
ALTER TABLE asset_relationships DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON asset_relationships;
DROP POLICY IF EXISTS tenant_insert ON asset_relationships;
DROP POLICY IF EXISTS tenant_update ON asset_relationships;
DROP POLICY IF EXISTS tenant_delete ON asset_relationships;

-- TABLE: assets
ALTER TABLE assets DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON assets;
DROP POLICY IF EXISTS tenant_insert ON assets;
DROP POLICY IF EXISTS tenant_update ON assets;
DROP POLICY IF EXISTS tenant_delete ON assets;
