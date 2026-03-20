-- Enables Row-Level Security on all tenant-scoped tables.
-- The application must SET LOCAL app.current_tenant_id = '<uuid>' within each transaction.
-- The database role used for migrations must have BYPASSRLS privilege.
-- Use: ALTER ROLE migrator_role BYPASSRLS;

-- Note: notification_delivery_log is intentionally excluded — it has no tenant_id column
-- and references notifications via FK only. Tenant isolation for delivery logs is enforced
-- by the FK cascade from the notifications table (RLS on notifications prevents cross-tenant
-- access to the notification rows that delivery logs reference).

-- =============================================================================
-- TABLE: notifications
-- =============================================================================

ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON notifications
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON notifications
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON notifications
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON notifications
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: notification_preferences
-- =============================================================================

ALTER TABLE notification_preferences ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_preferences FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON notification_preferences
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON notification_preferences
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON notification_preferences
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON notification_preferences
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: notification_webhooks
-- =============================================================================

ALTER TABLE notification_webhooks ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_webhooks FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON notification_webhooks
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON notification_webhooks
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON notification_webhooks
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_delete ON notification_webhooks
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- TABLE: notification_templates — NULLABLE tenant_id
-- Global templates (tenant_id IS NULL) are system-provided and visible to all tenants.
-- Tenant-specific templates (tenant_id IS NOT NULL) override globals for that tenant only.
-- DELETE is restricted to tenant-specific rows only (global templates are protected).
-- =============================================================================

ALTER TABLE notification_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_templates FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON notification_templates
    USING (tenant_id IS NULL OR tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_insert ON notification_templates
    FOR INSERT
    WITH CHECK (tenant_id IS NULL OR tenant_id = current_setting('app.current_tenant_id', true)::uuid);
CREATE POLICY tenant_update ON notification_templates
    FOR UPDATE
    USING (tenant_id IS NULL OR tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id IS NULL OR tenant_id = current_setting('app.current_tenant_id', true)::uuid);
-- Only tenant-specific templates can be deleted; global system templates are protected.
CREATE POLICY tenant_delete ON notification_templates
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
