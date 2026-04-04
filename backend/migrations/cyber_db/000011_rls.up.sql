-- Enables Row-Level Security on all tenant-scoped tables.
-- The application must SET LOCAL app.current_tenant_id = '<uuid>' within each transaction.
-- The database role used for migrations must have BYPASSRLS privilege.
-- Use: ALTER ROLE migrator_role BYPASSRLS;

-- =============================================================================
-- TABLE: assets
-- =============================================================================

ALTER TABLE assets ENABLE ROW LEVEL SECURITY;
ALTER TABLE assets FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON assets
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON assets
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON assets
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON assets
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: asset_relationships
-- =============================================================================

ALTER TABLE asset_relationships ENABLE ROW LEVEL SECURITY;
ALTER TABLE asset_relationships FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON asset_relationships
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON asset_relationships
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON asset_relationships
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON asset_relationships
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: vulnerabilities
-- =============================================================================

ALTER TABLE vulnerabilities ENABLE ROW LEVEL SECURITY;
ALTER TABLE vulnerabilities FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON vulnerabilities
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON vulnerabilities
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON vulnerabilities
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON vulnerabilities
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: threats
-- =============================================================================

ALTER TABLE threats ENABLE ROW LEVEL SECURITY;
ALTER TABLE threats FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON threats
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON threats
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON threats
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON threats
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: threat_indicators
-- =============================================================================

ALTER TABLE threat_indicators ENABLE ROW LEVEL SECURITY;
ALTER TABLE threat_indicators FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON threat_indicators
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON threat_indicators
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON threat_indicators
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON threat_indicators
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: detection_rules
-- =============================================================================

ALTER TABLE detection_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE detection_rules FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON detection_rules
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON detection_rules
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON detection_rules
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON detection_rules
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY template_select ON detection_rules
    FOR SELECT
    USING (is_template = true);
CREATE POLICY template_insert ON detection_rules
    FOR INSERT
    WITH CHECK (
        current_setting('app.current_tenant_id', true) IS NULL
        AND tenant_id IS NULL
        AND is_template = true
    );
CREATE POLICY template_update ON detection_rules
    FOR UPDATE
    USING (
        current_setting('app.current_tenant_id', true) IS NULL
        AND tenant_id IS NULL
        AND is_template = true
    )
    WITH CHECK (
        current_setting('app.current_tenant_id', true) IS NULL
        AND tenant_id IS NULL
        AND is_template = true
    );

-- =============================================================================
-- TABLE: alerts
-- =============================================================================

ALTER TABLE alerts ENABLE ROW LEVEL SECURITY;
ALTER TABLE alerts FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON alerts
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON alerts
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON alerts
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON alerts
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: alert_comments
-- =============================================================================

ALTER TABLE alert_comments ENABLE ROW LEVEL SECURITY;
ALTER TABLE alert_comments FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON alert_comments
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON alert_comments
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON alert_comments
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON alert_comments
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: alert_timeline
-- =============================================================================

ALTER TABLE alert_timeline ENABLE ROW LEVEL SECURITY;
ALTER TABLE alert_timeline FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON alert_timeline
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON alert_timeline
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON alert_timeline
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON alert_timeline
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: security_events
-- =============================================================================

ALTER TABLE security_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE security_events FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON security_events
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON security_events
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON security_events
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON security_events
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: remediation_actions
-- =============================================================================

ALTER TABLE remediation_actions ENABLE ROW LEVEL SECURITY;
ALTER TABLE remediation_actions FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON remediation_actions
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON remediation_actions
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON remediation_actions
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON remediation_actions
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: remediation_audit_trail
-- =============================================================================

ALTER TABLE remediation_audit_trail ENABLE ROW LEVEL SECURITY;
ALTER TABLE remediation_audit_trail FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON remediation_audit_trail
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON remediation_audit_trail
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON remediation_audit_trail
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON remediation_audit_trail
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: ctem_assessments
-- =============================================================================

ALTER TABLE ctem_assessments ENABLE ROW LEVEL SECURITY;
ALTER TABLE ctem_assessments FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON ctem_assessments
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON ctem_assessments
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON ctem_assessments
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON ctem_assessments
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: dspm_data_assets
-- =============================================================================

ALTER TABLE dspm_data_assets ENABLE ROW LEVEL SECURITY;
ALTER TABLE dspm_data_assets FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON dspm_data_assets
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON dspm_data_assets
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON dspm_data_assets
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON dspm_data_assets
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: dspm_scans
-- =============================================================================

ALTER TABLE dspm_scans ENABLE ROW LEVEL SECURITY;
ALTER TABLE dspm_scans FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON dspm_scans
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON dspm_scans
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON dspm_scans
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON dspm_scans
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: scan_history
-- =============================================================================

ALTER TABLE scan_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE scan_history FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON scan_history
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON scan_history
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON scan_history
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON scan_history
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: vciso_briefings
-- =============================================================================

ALTER TABLE vciso_briefings ENABLE ROW LEVEL SECURITY;
ALTER TABLE vciso_briefings FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON vciso_briefings
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON vciso_briefings
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON vciso_briefings
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON vciso_briefings
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
