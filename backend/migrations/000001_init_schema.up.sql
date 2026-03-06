-- =============================================================================
-- Clario 360 — Initial Schema Migration
-- Creates core tables for multi-tenant IAM, audit logging, and shared types
-- =============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =============================================================================
-- TENANTS
-- =============================================================================
CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(100) NOT NULL UNIQUE,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    settings    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_tenants_slug ON tenants (slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_active ON tenants (is_active) WHERE deleted_at IS NULL;

-- =============================================================================
-- USERS
-- =============================================================================
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email           VARCHAR(255) NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    first_name      VARCHAR(100) NOT NULL,
    last_name       VARCHAR(100) NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    email_verified  BOOLEAN NOT NULL DEFAULT false,
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    UNIQUE(tenant_id, email)
);

CREATE INDEX idx_users_tenant ON users (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_email ON users (email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_active ON users (tenant_id, is_active) WHERE deleted_at IS NULL;

-- =============================================================================
-- ROLES
-- =============================================================================
CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_system   BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_roles_tenant ON roles (tenant_id);

-- =============================================================================
-- PERMISSIONS
-- =============================================================================
CREATE TABLE permissions (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    resource    VARCHAR(50) NOT NULL,
    action      VARCHAR(50) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_permissions_resource ON permissions (resource);

-- =============================================================================
-- ROLE ↔ PERMISSION (many-to-many)
-- =============================================================================
CREATE TABLE role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- =============================================================================
-- USER ↔ ROLE (many-to-many)
-- =============================================================================
CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- =============================================================================
-- REFRESH TOKENS
-- =============================================================================
CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(255) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked     BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens (expires_at) WHERE NOT revoked;

-- =============================================================================
-- AUDIT LOG
-- =============================================================================
CREATE TABLE audit_logs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id         UUID REFERENCES users(id) ON DELETE SET NULL,
    action          VARCHAR(50) NOT NULL,
    resource_type   VARCHAR(100) NOT NULL,
    resource_id     UUID,
    description     TEXT NOT NULL DEFAULT '',
    old_value       JSONB,
    new_value       JSONB,
    ip_address      INET,
    user_agent      TEXT,
    request_id      VARCHAR(64),
    service_name    VARCHAR(100) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_tenant ON audit_logs (tenant_id, created_at DESC);
CREATE INDEX idx_audit_logs_user ON audit_logs (user_id, created_at DESC);
CREATE INDEX idx_audit_logs_resource ON audit_logs (resource_type, resource_id);
CREATE INDEX idx_audit_logs_action ON audit_logs (action, created_at DESC);

-- =============================================================================
-- UPDATED_AT TRIGGER FUNCTION
-- =============================================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at triggers
CREATE TRIGGER trg_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_roles_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- SEED SYSTEM PERMISSIONS
-- =============================================================================
INSERT INTO permissions (code, description, resource, action) VALUES
    ('user:read',    'View users',               'user',    'read'),
    ('user:write',   'Create and update users',   'user',    'write'),
    ('user:delete',  'Delete users',              'user',    'delete'),
    ('role:read',    'View roles',                'role',    'read'),
    ('role:write',   'Create and update roles',   'role',    'write'),
    ('tenant:read',  'View tenant settings',      'tenant',  'read'),
    ('tenant:write', 'Update tenant settings',    'tenant',  'write'),
    ('audit:read',   'View audit logs',           'audit',   'read'),
    ('cyber:read',   'View cyber suite',          'cyber',   'read'),
    ('cyber:write',  'Manage cyber suite',        'cyber',   'write'),
    ('data:read',    'View data suite',           'data',    'read'),
    ('data:write',   'Manage data suite',         'data',    'write'),
    ('acta:read',    'View Acta documents',       'acta',    'read'),
    ('acta:write',   'Manage Acta documents',     'acta',    'write'),
    ('lex:read',     'View Lex cases',            'lex',     'read'),
    ('lex:write',    'Manage Lex cases',          'lex',     'write'),
    ('visus:read',   'View Visus dashboards',     'visus',   'read'),
    ('visus:write',  'Manage Visus dashboards',   'visus',   'write'),
    ('admin:*',      'Full administrative access', 'admin',  '*')
ON CONFLICT (code) DO NOTHING;
