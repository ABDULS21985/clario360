-- Enables Row-Level Security on all tenant-scoped tables.
-- The application must SET LOCAL app.current_tenant_id = '<uuid>' within each transaction.
-- The database role used for migrations must have BYPASSRLS privilege.
-- Use: ALTER ROLE migrator_role BYPASSRLS;

-- =============================================================================
-- TABLE: visus_dashboards
-- =============================================================================

ALTER TABLE visus_dashboards ENABLE ROW LEVEL SECURITY;
ALTER TABLE visus_dashboards FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON visus_dashboards
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON visus_dashboards
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON visus_dashboards
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON visus_dashboards
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: visus_widgets
-- =============================================================================

ALTER TABLE visus_widgets ENABLE ROW LEVEL SECURITY;
ALTER TABLE visus_widgets FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON visus_widgets
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON visus_widgets
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON visus_widgets
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON visus_widgets
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: visus_kpi_definitions
-- =============================================================================

ALTER TABLE visus_kpi_definitions ENABLE ROW LEVEL SECURITY;
ALTER TABLE visus_kpi_definitions FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON visus_kpi_definitions
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON visus_kpi_definitions
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON visus_kpi_definitions
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON visus_kpi_definitions
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: visus_kpi_snapshots
-- =============================================================================

ALTER TABLE visus_kpi_snapshots ENABLE ROW LEVEL SECURITY;
ALTER TABLE visus_kpi_snapshots FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON visus_kpi_snapshots
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON visus_kpi_snapshots
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON visus_kpi_snapshots
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON visus_kpi_snapshots
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: visus_executive_alerts
-- =============================================================================

ALTER TABLE visus_executive_alerts ENABLE ROW LEVEL SECURITY;
ALTER TABLE visus_executive_alerts FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON visus_executive_alerts
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON visus_executive_alerts
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON visus_executive_alerts
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON visus_executive_alerts
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: visus_report_definitions
-- =============================================================================

ALTER TABLE visus_report_definitions ENABLE ROW LEVEL SECURITY;
ALTER TABLE visus_report_definitions FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON visus_report_definitions
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON visus_report_definitions
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON visus_report_definitions
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON visus_report_definitions
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: visus_report_snapshots
-- =============================================================================

ALTER TABLE visus_report_snapshots ENABLE ROW LEVEL SECURITY;
ALTER TABLE visus_report_snapshots FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON visus_report_snapshots
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON visus_report_snapshots
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON visus_report_snapshots
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON visus_report_snapshots
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: visus_suite_cache
-- =============================================================================

ALTER TABLE visus_suite_cache ENABLE ROW LEVEL SECURITY;
ALTER TABLE visus_suite_cache FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON visus_suite_cache
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON visus_suite_cache
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON visus_suite_cache
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON visus_suite_cache
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
