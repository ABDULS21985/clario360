-- =============================================================================
-- Clario 360 — Rollback Initial Schema
-- =============================================================================

DROP TRIGGER IF EXISTS trg_roles_updated_at ON roles;
DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
DROP TRIGGER IF EXISTS trg_tenants_updated_at ON tenants;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;
