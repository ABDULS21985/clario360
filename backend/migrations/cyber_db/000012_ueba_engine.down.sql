DROP FUNCTION IF EXISTS manage_ueba_event_partitions(INTEGER);

ALTER TABLE IF EXISTS ueba_alerts DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON ueba_alerts;
DROP POLICY IF EXISTS tenant_insert ON ueba_alerts;
DROP POLICY IF EXISTS tenant_update ON ueba_alerts;
DROP POLICY IF EXISTS tenant_delete ON ueba_alerts;

ALTER TABLE IF EXISTS ueba_access_events DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON ueba_access_events;
DROP POLICY IF EXISTS tenant_insert ON ueba_access_events;
DROP POLICY IF EXISTS tenant_update ON ueba_access_events;
DROP POLICY IF EXISTS tenant_delete ON ueba_access_events;

ALTER TABLE IF EXISTS ueba_profiles DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON ueba_profiles;
DROP POLICY IF EXISTS tenant_insert ON ueba_profiles;
DROP POLICY IF EXISTS tenant_update ON ueba_profiles;
DROP POLICY IF EXISTS tenant_delete ON ueba_profiles;

DROP TABLE IF EXISTS ueba_alerts;
DROP TABLE IF EXISTS ueba_access_events CASCADE;
DROP TABLE IF EXISTS ueba_profiles;
