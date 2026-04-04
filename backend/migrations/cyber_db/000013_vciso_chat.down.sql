ALTER TABLE IF EXISTS vciso_messages DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON vciso_messages;
DROP POLICY IF EXISTS tenant_insert ON vciso_messages;
DROP POLICY IF EXISTS tenant_update ON vciso_messages;
DROP POLICY IF EXISTS tenant_delete ON vciso_messages;

ALTER TABLE IF EXISTS vciso_conversations DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON vciso_conversations;
DROP POLICY IF EXISTS tenant_insert ON vciso_conversations;
DROP POLICY IF EXISTS tenant_update ON vciso_conversations;
DROP POLICY IF EXISTS tenant_delete ON vciso_conversations;

DROP TABLE IF EXISTS vciso_messages;
DROP TABLE IF EXISTS vciso_conversations;
