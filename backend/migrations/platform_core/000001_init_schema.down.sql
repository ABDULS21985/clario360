-- =============================================================================
-- Clario 360 — Platform Core Database Schema Rollback
-- Database: platform_core
-- =============================================================================

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS system_settings;
DROP TABLE IF EXISTS notifications;

-- Drop partitioned table (cascades to all partitions)
DROP TABLE IF EXISTS audit_logs CASCADE;

DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop enum types
DROP TYPE IF EXISTS notification_type;
DROP TYPE IF EXISTS user_status;
DROP TYPE IF EXISTS subscription_tier;
DROP TYPE IF EXISTS tenant_status;
