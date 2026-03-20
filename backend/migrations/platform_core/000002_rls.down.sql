-- Removes Row-Level Security from all tenant-scoped tables in platform_core.

-- TABLE: audit_logs
ALTER TABLE audit_logs DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON audit_logs;
DROP POLICY IF EXISTS tenant_insert ON audit_logs;
DROP POLICY IF EXISTS tenant_update ON audit_logs;
DROP POLICY IF EXISTS tenant_delete ON audit_logs;

-- TABLE: system_settings
ALTER TABLE system_settings DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON system_settings;
DROP POLICY IF EXISTS tenant_insert ON system_settings;
DROP POLICY IF EXISTS tenant_update ON system_settings;
DROP POLICY IF EXISTS tenant_delete ON system_settings;

-- TABLE: notifications
ALTER TABLE notifications DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON notifications;
DROP POLICY IF EXISTS tenant_insert ON notifications;
DROP POLICY IF EXISTS tenant_update ON notifications;
DROP POLICY IF EXISTS tenant_delete ON notifications;

-- TABLE: api_keys
ALTER TABLE api_keys DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON api_keys;
DROP POLICY IF EXISTS tenant_insert ON api_keys;
DROP POLICY IF EXISTS tenant_update ON api_keys;
DROP POLICY IF EXISTS tenant_delete ON api_keys;

-- TABLE: sessions
ALTER TABLE sessions DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON sessions;
DROP POLICY IF EXISTS tenant_insert ON sessions;
DROP POLICY IF EXISTS tenant_update ON sessions;
DROP POLICY IF EXISTS tenant_delete ON sessions;

-- TABLE: user_roles
ALTER TABLE user_roles DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON user_roles;
DROP POLICY IF EXISTS tenant_insert ON user_roles;
DROP POLICY IF EXISTS tenant_update ON user_roles;
DROP POLICY IF EXISTS tenant_delete ON user_roles;

-- TABLE: roles
ALTER TABLE roles DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON roles;
DROP POLICY IF EXISTS tenant_insert ON roles;
DROP POLICY IF EXISTS tenant_update ON roles;
DROP POLICY IF EXISTS tenant_delete ON roles;

-- TABLE: users
ALTER TABLE users DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON users;
DROP POLICY IF EXISTS tenant_insert ON users;
DROP POLICY IF EXISTS tenant_update ON users;
DROP POLICY IF EXISTS tenant_delete ON users;
