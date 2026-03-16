-- =============================================================================
-- Clario 360 — Platform Core Database Schema
-- Database: platform_core
-- Contains: tenants, users, roles, sessions, api_keys, audit_logs,
--           notifications, system_settings
-- =============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- ENUM TYPES
-- =============================================================================

CREATE TYPE tenant_status AS ENUM ('active', 'inactive', 'suspended', 'trial');
COMMENT ON TYPE tenant_status IS 'Possible statuses for a tenant account';

CREATE TYPE subscription_tier AS ENUM ('free', 'starter', 'professional', 'enterprise');
COMMENT ON TYPE subscription_tier IS 'Subscription tiers determining feature access';

CREATE TYPE user_status AS ENUM ('active', 'inactive', 'suspended');
COMMENT ON TYPE user_status IS 'Possible statuses for a user account';

CREATE TYPE notification_type AS ENUM ('info', 'warning', 'error', 'success', 'action_required');
COMMENT ON TYPE notification_type IS 'Types of notifications sent to users';

-- =============================================================================
-- TRIGGER FUNCTION: auto-update updated_at
-- =============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION update_updated_at_column() IS 'Automatically sets updated_at to current timestamp on row update';

-- =============================================================================
-- TABLE: tenants
-- =============================================================================

CREATE TABLE tenants (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name              VARCHAR(255) NOT NULL,
    slug              VARCHAR(100) NOT NULL UNIQUE,
    domain            VARCHAR(255),
    settings          JSONB       NOT NULL DEFAULT '{}',
    status            tenant_status NOT NULL DEFAULT 'active',
    subscription_tier subscription_tier NOT NULL DEFAULT 'free',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE tenants IS 'Organizations using the Clario 360 platform';
COMMENT ON COLUMN tenants.id IS 'Unique tenant identifier';
COMMENT ON COLUMN tenants.name IS 'Display name of the organization';
COMMENT ON COLUMN tenants.slug IS 'URL-safe unique identifier for the tenant';
COMMENT ON COLUMN tenants.domain IS 'Custom domain associated with the tenant';
COMMENT ON COLUMN tenants.settings IS 'Tenant-specific configuration (JSON)';
COMMENT ON COLUMN tenants.status IS 'Current status of the tenant account';
COMMENT ON COLUMN tenants.subscription_tier IS 'Active subscription tier';

CREATE INDEX idx_tenants_slug ON tenants (slug);
CREATE INDEX idx_tenants_status ON tenants (status);
CREATE INDEX idx_tenants_domain ON tenants (domain) WHERE domain IS NOT NULL;
CREATE INDEX idx_tenants_settings ON tenants USING GIN (settings);

CREATE TRIGGER trg_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: users
-- =============================================================================

CREATE TABLE users (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email           VARCHAR(255) NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    first_name      VARCHAR(100) NOT NULL,
    last_name       VARCHAR(100) NOT NULL,
    avatar_url      TEXT,
    status          user_status NOT NULL DEFAULT 'active',
    mfa_enabled     BOOLEAN     NOT NULL DEFAULT false,
    mfa_secret      VARCHAR(255),
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID,
    updated_by      UUID,
    deleted_at      TIMESTAMPTZ,
    CONSTRAINT uq_users_tenant_email UNIQUE (tenant_id, email)
);

COMMENT ON TABLE users IS 'Platform users scoped to a tenant';
COMMENT ON COLUMN users.id IS 'Unique user identifier';
COMMENT ON COLUMN users.tenant_id IS 'Tenant this user belongs to';
COMMENT ON COLUMN users.email IS 'User email address, unique within tenant';
COMMENT ON COLUMN users.password_hash IS 'Bcrypt hashed password';
COMMENT ON COLUMN users.first_name IS 'User first name';
COMMENT ON COLUMN users.last_name IS 'User last name';
COMMENT ON COLUMN users.avatar_url IS 'URL to user avatar image';
COMMENT ON COLUMN users.status IS 'Current user account status';
COMMENT ON COLUMN users.mfa_enabled IS 'Whether multi-factor authentication is enabled';
COMMENT ON COLUMN users.mfa_secret IS 'TOTP secret for MFA (encrypted)';
COMMENT ON COLUMN users.last_login_at IS 'Timestamp of the last successful login';
COMMENT ON COLUMN users.deleted_at IS 'Soft delete timestamp';

CREATE INDEX idx_users_tenant_id ON users (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_email ON users (email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_tenant_status ON users (tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_last_login ON users (last_login_at DESC NULLS LAST) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: roles
-- =============================================================================

CREATE TABLE roles (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    slug            VARCHAR(100) NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    is_system_role  BOOLEAN     NOT NULL DEFAULT false,
    permissions     JSONB       NOT NULL DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_roles_tenant_slug UNIQUE (tenant_id, slug)
);

COMMENT ON TABLE roles IS 'Authorization roles with embedded permissions';
COMMENT ON COLUMN roles.id IS 'Unique role identifier';
COMMENT ON COLUMN roles.tenant_id IS 'Tenant this role belongs to';
COMMENT ON COLUMN roles.name IS 'Human-readable role name';
COMMENT ON COLUMN roles.slug IS 'URL-safe role identifier, unique within tenant';
COMMENT ON COLUMN roles.is_system_role IS 'System roles cannot be modified by tenants';
COMMENT ON COLUMN roles.permissions IS 'JSON array of permission strings (e.g., ["user:read", "cyber:write"])';

CREATE INDEX idx_roles_tenant_id ON roles (tenant_id);
CREATE INDEX idx_roles_tenant_slug ON roles (tenant_id, slug);
CREATE INDEX idx_roles_permissions ON roles USING GIN (permissions);

CREATE TRIGGER trg_roles_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- TABLE: user_roles (many-to-many)
-- =============================================================================

CREATE TABLE user_roles (
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id     UUID        NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    tenant_id   UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_by UUID        REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (user_id, role_id)
);

COMMENT ON TABLE user_roles IS 'Associates users with their assigned roles';
COMMENT ON COLUMN user_roles.assigned_at IS 'When the role was assigned';
COMMENT ON COLUMN user_roles.assigned_by IS 'User who assigned this role';

CREATE INDEX idx_user_roles_tenant ON user_roles (tenant_id);
CREATE INDEX idx_user_roles_role ON user_roles (role_id);

-- =============================================================================
-- TABLE: sessions
-- =============================================================================

CREATE TABLE sessions (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id          UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255) NOT NULL,
    ip_address         INET,
    user_agent         TEXT,
    expires_at         TIMESTAMPTZ NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE sessions IS 'Active user sessions with refresh tokens';
COMMENT ON COLUMN sessions.refresh_token_hash IS 'SHA-256 hash of the refresh token';
COMMENT ON COLUMN sessions.ip_address IS 'IP address from which the session was created';
COMMENT ON COLUMN sessions.expires_at IS 'When the refresh token expires';

CREATE INDEX idx_sessions_user ON sessions (user_id);
CREATE INDEX idx_sessions_tenant ON sessions (tenant_id);
CREATE INDEX idx_sessions_expires ON sessions (expires_at);
CREATE INDEX idx_sessions_token_hash ON sessions (refresh_token_hash);

-- =============================================================================
-- TABLE: api_keys
-- =============================================================================

CREATE TABLE api_keys (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name         VARCHAR(255) NOT NULL,
    key_hash     VARCHAR(255) NOT NULL UNIQUE,
    key_prefix   VARCHAR(20) NOT NULL,
    permissions  JSONB       NOT NULL DEFAULT '[]',
    last_used_at TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by   UUID        REFERENCES users(id) ON DELETE SET NULL,
    revoked_at   TIMESTAMPTZ
);

COMMENT ON TABLE api_keys IS 'API keys for programmatic access';
COMMENT ON COLUMN api_keys.key_hash IS 'SHA-256 hash of the API key';
COMMENT ON COLUMN api_keys.key_prefix IS 'First few characters of the key for identification (e.g., "clr_abc")';
COMMENT ON COLUMN api_keys.permissions IS 'JSON array of permission strings granted to this key';
COMMENT ON COLUMN api_keys.revoked_at IS 'When the key was revoked (NULL if active)';

CREATE INDEX idx_api_keys_tenant ON api_keys (tenant_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_api_keys_hash ON api_keys (key_hash) WHERE revoked_at IS NULL;
CREATE INDEX idx_api_keys_prefix ON api_keys (key_prefix);
CREATE INDEX idx_api_keys_permissions ON api_keys USING GIN (permissions);

-- =============================================================================
-- TABLE: password_reset_tokens
-- =============================================================================

CREATE TABLE password_reset_tokens (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   TEXT        NOT NULL,
    expires_at   TIMESTAMPTZ NOT NULL,
    used         BOOLEAN     NOT NULL DEFAULT false,
    used_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE password_reset_tokens IS 'Password reset tokens (hashed with SHA-256)';
COMMENT ON COLUMN password_reset_tokens.token_hash IS 'SHA-256 hash of the reset token';

CREATE INDEX idx_reset_tokens_hash ON password_reset_tokens (token_hash) WHERE used = false;
CREATE INDEX idx_reset_tokens_user ON password_reset_tokens (user_id);

-- =============================================================================
-- TABLE: audit_logs (PARTITIONED by created_at — monthly)
-- =============================================================================

CREATE TABLE audit_logs (
    id            UUID        NOT NULL DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL,
    user_id       UUID,
    service       VARCHAR(100) NOT NULL,
    action        VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id   UUID,
    old_value     JSONB,
    new_value     JSONB,
    ip_address    INET,
    user_agent    TEXT,
    metadata      JSONB       DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

COMMENT ON TABLE audit_logs IS 'Immutable audit trail of all platform actions (partitioned monthly)';
COMMENT ON COLUMN audit_logs.service IS 'Originating service (e.g., iam-service, cyber-service)';
COMMENT ON COLUMN audit_logs.action IS 'Action performed (e.g., user.login, role.assign)';
COMMENT ON COLUMN audit_logs.resource_type IS 'Type of resource affected';
COMMENT ON COLUMN audit_logs.resource_id IS 'UUID of the affected resource';
COMMENT ON COLUMN audit_logs.old_value IS 'Previous state of the resource (for updates)';
COMMENT ON COLUMN audit_logs.new_value IS 'New state of the resource (for creates/updates)';
COMMENT ON COLUMN audit_logs.metadata IS 'Additional contextual data';

-- Create monthly partitions for the next 12 months
CREATE TABLE audit_logs_2025_01 PARTITION OF audit_logs FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
CREATE TABLE audit_logs_2025_02 PARTITION OF audit_logs FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
CREATE TABLE audit_logs_2025_03 PARTITION OF audit_logs FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');
CREATE TABLE audit_logs_2025_04 PARTITION OF audit_logs FOR VALUES FROM ('2025-04-01') TO ('2025-05-01');
CREATE TABLE audit_logs_2025_05 PARTITION OF audit_logs FOR VALUES FROM ('2025-05-01') TO ('2025-06-01');
CREATE TABLE audit_logs_2025_06 PARTITION OF audit_logs FOR VALUES FROM ('2025-06-01') TO ('2025-07-01');
CREATE TABLE audit_logs_2025_07 PARTITION OF audit_logs FOR VALUES FROM ('2025-07-01') TO ('2025-08-01');
CREATE TABLE audit_logs_2025_08 PARTITION OF audit_logs FOR VALUES FROM ('2025-08-01') TO ('2025-09-01');
CREATE TABLE audit_logs_2025_09 PARTITION OF audit_logs FOR VALUES FROM ('2025-09-01') TO ('2025-10-01');
CREATE TABLE audit_logs_2025_10 PARTITION OF audit_logs FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');
CREATE TABLE audit_logs_2025_11 PARTITION OF audit_logs FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
CREATE TABLE audit_logs_2025_12 PARTITION OF audit_logs FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');
CREATE TABLE audit_logs_2026_01 PARTITION OF audit_logs FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE audit_logs_2026_02 PARTITION OF audit_logs FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE audit_logs_2026_03 PARTITION OF audit_logs FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE audit_logs_2026_04 PARTITION OF audit_logs FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE audit_logs_2026_05 PARTITION OF audit_logs FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE audit_logs_2026_06 PARTITION OF audit_logs FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE audit_logs_2026_07 PARTITION OF audit_logs FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
CREATE TABLE audit_logs_2026_08 PARTITION OF audit_logs FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');
CREATE TABLE audit_logs_2026_09 PARTITION OF audit_logs FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');
CREATE TABLE audit_logs_2026_10 PARTITION OF audit_logs FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');
CREATE TABLE audit_logs_2026_11 PARTITION OF audit_logs FOR VALUES FROM ('2026-11-01') TO ('2026-12-01');
CREATE TABLE audit_logs_2026_12 PARTITION OF audit_logs FOR VALUES FROM ('2026-12-01') TO ('2027-01-01');

-- Default partition for any dates outside defined ranges
CREATE TABLE audit_logs_default PARTITION OF audit_logs DEFAULT;

CREATE INDEX idx_audit_logs_tenant_created ON audit_logs (tenant_id, created_at DESC);
CREATE INDEX idx_audit_logs_user_created ON audit_logs (user_id, created_at DESC);
CREATE INDEX idx_audit_logs_resource ON audit_logs (resource_type, resource_id);
CREATE INDEX idx_audit_logs_service_action ON audit_logs (service, action);
CREATE INDEX idx_audit_logs_metadata ON audit_logs USING GIN (metadata);

-- =============================================================================
-- TABLE: notifications
-- =============================================================================

CREATE TABLE notifications (
    id         UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID              NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id    UUID              NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       notification_type NOT NULL DEFAULT 'info',
    title      VARCHAR(255)      NOT NULL,
    body       TEXT              NOT NULL DEFAULT '',
    data       JSONB             DEFAULT '{}',
    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ       NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE notifications IS 'User notifications for platform events';
COMMENT ON COLUMN notifications.type IS 'Category of notification';
COMMENT ON COLUMN notifications.data IS 'Additional structured data (e.g., link, entity reference)';
COMMENT ON COLUMN notifications.read_at IS 'When the user read/dismissed the notification';

CREATE INDEX idx_notifications_user_unread ON notifications (user_id, created_at DESC) WHERE read_at IS NULL;
CREATE INDEX idx_notifications_tenant ON notifications (tenant_id, created_at DESC);
CREATE INDEX idx_notifications_type ON notifications (type);

-- =============================================================================
-- TABLE: system_settings
-- =============================================================================

CREATE TABLE system_settings (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        REFERENCES tenants(id) ON DELETE CASCADE,
    key         VARCHAR(255) NOT NULL,
    value       JSONB       NOT NULL DEFAULT '{}',
    description TEXT        NOT NULL DEFAULT '',
    updated_by  UUID        REFERENCES users(id) ON DELETE SET NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_system_settings_tenant_key UNIQUE (tenant_id, key)
);

COMMENT ON TABLE system_settings IS 'Key-value configuration per tenant (NULL tenant_id = global)';
COMMENT ON COLUMN system_settings.key IS 'Setting key (e.g., "notification.email_enabled")';
COMMENT ON COLUMN system_settings.value IS 'Setting value as JSON';
COMMENT ON COLUMN system_settings.tenant_id IS 'Owning tenant; NULL for global/platform settings';

CREATE INDEX idx_system_settings_tenant ON system_settings (tenant_id);
CREATE INDEX idx_system_settings_key ON system_settings (key);
CREATE INDEX idx_system_settings_value ON system_settings USING GIN (value);

CREATE TRIGGER trg_system_settings_updated_at
    BEFORE UPDATE ON system_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- SEED: Default system permissions in roles
-- =============================================================================

-- Seed a default "super_admin" system role for first-time setup
-- Actual tenant-specific roles should be created via the IAM service
INSERT INTO tenants (id, name, slug, status, subscription_tier)
VALUES ('00000000-0000-0000-0000-000000000001', 'System', 'system', 'active', 'enterprise')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO roles (id, tenant_id, name, slug, description, is_system_role, permissions)
VALUES (
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    'Super Admin',
    'super-admin',
    'Full platform access across all suites and services',
    true,
    '["admin:*"]'
),
(
    '00000000-0000-0000-0000-000000000011',
    '00000000-0000-0000-0000-000000000001',
    'Tenant Admin',
    'tenant-admin',
    'Full access within a tenant',
    true,
    '["user:read","user:write","user:delete","role:read","role:write","tenant:read","tenant:write","audit:read","cyber:read","cyber:write","data:read","data:write","acta:read","acta:write","lex:read","lex:write","visus:read","visus:write"]'
),
(
    '00000000-0000-0000-0000-000000000012',
    '00000000-0000-0000-0000-000000000001',
    'Analyst',
    'analyst',
    'Read access to all suites with write access to assigned areas',
    true,
    '["cyber:read","data:read","acta:read","lex:read","visus:read","audit:read"]'
),
(
    '00000000-0000-0000-0000-000000000013',
    '00000000-0000-0000-0000-000000000001',
    'Viewer',
    'viewer',
    'Read-only access to dashboards and reports',
    true,
    '["visus:read","audit:read"]'
)
ON CONFLICT ON CONSTRAINT uq_roles_tenant_slug DO NOTHING;
