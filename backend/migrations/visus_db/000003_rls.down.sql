-- Removes Row-Level Security from all tenant-scoped tables in visus_db.

-- TABLE: visus_suite_cache
ALTER TABLE visus_suite_cache DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON visus_suite_cache;
DROP POLICY IF EXISTS tenant_insert ON visus_suite_cache;
DROP POLICY IF EXISTS tenant_update ON visus_suite_cache;
DROP POLICY IF EXISTS tenant_delete ON visus_suite_cache;

-- TABLE: visus_report_snapshots
ALTER TABLE visus_report_snapshots DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON visus_report_snapshots;
DROP POLICY IF EXISTS tenant_insert ON visus_report_snapshots;
DROP POLICY IF EXISTS tenant_update ON visus_report_snapshots;
DROP POLICY IF EXISTS tenant_delete ON visus_report_snapshots;

-- TABLE: visus_report_definitions
ALTER TABLE visus_report_definitions DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON visus_report_definitions;
DROP POLICY IF EXISTS tenant_insert ON visus_report_definitions;
DROP POLICY IF EXISTS tenant_update ON visus_report_definitions;
DROP POLICY IF EXISTS tenant_delete ON visus_report_definitions;

-- TABLE: visus_executive_alerts
ALTER TABLE visus_executive_alerts DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON visus_executive_alerts;
DROP POLICY IF EXISTS tenant_insert ON visus_executive_alerts;
DROP POLICY IF EXISTS tenant_update ON visus_executive_alerts;
DROP POLICY IF EXISTS tenant_delete ON visus_executive_alerts;

-- TABLE: visus_kpi_snapshots
ALTER TABLE visus_kpi_snapshots DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON visus_kpi_snapshots;
DROP POLICY IF EXISTS tenant_insert ON visus_kpi_snapshots;
DROP POLICY IF EXISTS tenant_update ON visus_kpi_snapshots;
DROP POLICY IF EXISTS tenant_delete ON visus_kpi_snapshots;

-- TABLE: visus_kpi_definitions
ALTER TABLE visus_kpi_definitions DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON visus_kpi_definitions;
DROP POLICY IF EXISTS tenant_insert ON visus_kpi_definitions;
DROP POLICY IF EXISTS tenant_update ON visus_kpi_definitions;
DROP POLICY IF EXISTS tenant_delete ON visus_kpi_definitions;

-- TABLE: visus_widgets
ALTER TABLE visus_widgets DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON visus_widgets;
DROP POLICY IF EXISTS tenant_insert ON visus_widgets;
DROP POLICY IF EXISTS tenant_update ON visus_widgets;
DROP POLICY IF EXISTS tenant_delete ON visus_widgets;

-- TABLE: visus_dashboards
ALTER TABLE visus_dashboards DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON visus_dashboards;
DROP POLICY IF EXISTS tenant_insert ON visus_dashboards;
DROP POLICY IF EXISTS tenant_update ON visus_dashboards;
DROP POLICY IF EXISTS tenant_delete ON visus_dashboards;
