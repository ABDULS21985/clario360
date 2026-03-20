-- Removes Row-Level Security from all tenant-scoped tables in notification_db.

-- TABLE: notification_templates
ALTER TABLE notification_templates DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON notification_templates;
DROP POLICY IF EXISTS tenant_insert ON notification_templates;
DROP POLICY IF EXISTS tenant_update ON notification_templates;
DROP POLICY IF EXISTS tenant_delete ON notification_templates;

-- TABLE: notification_webhooks
ALTER TABLE notification_webhooks DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON notification_webhooks;
DROP POLICY IF EXISTS tenant_insert ON notification_webhooks;
DROP POLICY IF EXISTS tenant_update ON notification_webhooks;
DROP POLICY IF EXISTS tenant_delete ON notification_webhooks;

-- TABLE: notification_preferences
ALTER TABLE notification_preferences DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON notification_preferences;
DROP POLICY IF EXISTS tenant_insert ON notification_preferences;
DROP POLICY IF EXISTS tenant_update ON notification_preferences;
DROP POLICY IF EXISTS tenant_delete ON notification_preferences;

-- TABLE: notifications
ALTER TABLE notifications DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON notifications;
DROP POLICY IF EXISTS tenant_insert ON notifications;
DROP POLICY IF EXISTS tenant_update ON notifications;
DROP POLICY IF EXISTS tenant_delete ON notifications;
