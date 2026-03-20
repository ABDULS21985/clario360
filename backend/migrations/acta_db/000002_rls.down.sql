-- Removes Row-Level Security from all tenant-scoped tables in acta_db.

-- TABLE: compliance_checks
ALTER TABLE compliance_checks DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON compliance_checks;
DROP POLICY IF EXISTS tenant_insert ON compliance_checks;
DROP POLICY IF EXISTS tenant_update ON compliance_checks;
DROP POLICY IF EXISTS tenant_delete ON compliance_checks;

-- TABLE: action_items
ALTER TABLE action_items DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON action_items;
DROP POLICY IF EXISTS tenant_insert ON action_items;
DROP POLICY IF EXISTS tenant_update ON action_items;
DROP POLICY IF EXISTS tenant_delete ON action_items;

-- TABLE: meeting_minutes
ALTER TABLE meeting_minutes DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON meeting_minutes;
DROP POLICY IF EXISTS tenant_insert ON meeting_minutes;
DROP POLICY IF EXISTS tenant_update ON meeting_minutes;
DROP POLICY IF EXISTS tenant_delete ON meeting_minutes;

-- TABLE: agenda_items
ALTER TABLE agenda_items DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON agenda_items;
DROP POLICY IF EXISTS tenant_insert ON agenda_items;
DROP POLICY IF EXISTS tenant_update ON agenda_items;
DROP POLICY IF EXISTS tenant_delete ON agenda_items;

-- TABLE: meeting_attendance
ALTER TABLE meeting_attendance DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON meeting_attendance;
DROP POLICY IF EXISTS tenant_insert ON meeting_attendance;
DROP POLICY IF EXISTS tenant_update ON meeting_attendance;
DROP POLICY IF EXISTS tenant_delete ON meeting_attendance;

-- TABLE: meetings
ALTER TABLE meetings DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON meetings;
DROP POLICY IF EXISTS tenant_insert ON meetings;
DROP POLICY IF EXISTS tenant_update ON meetings;
DROP POLICY IF EXISTS tenant_delete ON meetings;

-- TABLE: committee_members
ALTER TABLE committee_members DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON committee_members;
DROP POLICY IF EXISTS tenant_insert ON committee_members;
DROP POLICY IF EXISTS tenant_update ON committee_members;
DROP POLICY IF EXISTS tenant_delete ON committee_members;

-- TABLE: committees
ALTER TABLE committees DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON committees;
DROP POLICY IF EXISTS tenant_insert ON committees;
DROP POLICY IF EXISTS tenant_update ON committees;
DROP POLICY IF EXISTS tenant_delete ON committees;
