-- Removes Row-Level Security from all tenant-scoped tables in lex_db.

-- TABLE: expiry_notifications
ALTER TABLE expiry_notifications DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON expiry_notifications;
DROP POLICY IF EXISTS tenant_insert ON expiry_notifications;
DROP POLICY IF EXISTS tenant_update ON expiry_notifications;
DROP POLICY IF EXISTS tenant_delete ON expiry_notifications;

-- TABLE: compliance_alerts
ALTER TABLE compliance_alerts DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON compliance_alerts;
DROP POLICY IF EXISTS tenant_insert ON compliance_alerts;
DROP POLICY IF EXISTS tenant_update ON compliance_alerts;
DROP POLICY IF EXISTS tenant_delete ON compliance_alerts;

-- TABLE: compliance_rules
ALTER TABLE compliance_rules DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON compliance_rules;
DROP POLICY IF EXISTS tenant_insert ON compliance_rules;
DROP POLICY IF EXISTS tenant_update ON compliance_rules;
DROP POLICY IF EXISTS tenant_delete ON compliance_rules;

-- TABLE: document_versions
ALTER TABLE document_versions DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON document_versions;
DROP POLICY IF EXISTS tenant_insert ON document_versions;
DROP POLICY IF EXISTS tenant_update ON document_versions;
DROP POLICY IF EXISTS tenant_delete ON document_versions;

-- TABLE: legal_documents
ALTER TABLE legal_documents DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON legal_documents;
DROP POLICY IF EXISTS tenant_insert ON legal_documents;
DROP POLICY IF EXISTS tenant_update ON legal_documents;
DROP POLICY IF EXISTS tenant_delete ON legal_documents;

-- TABLE: contract_analyses
ALTER TABLE contract_analyses DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON contract_analyses;
DROP POLICY IF EXISTS tenant_insert ON contract_analyses;
DROP POLICY IF EXISTS tenant_update ON contract_analyses;
DROP POLICY IF EXISTS tenant_delete ON contract_analyses;

-- TABLE: contract_clauses
ALTER TABLE contract_clauses DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON contract_clauses;
DROP POLICY IF EXISTS tenant_insert ON contract_clauses;
DROP POLICY IF EXISTS tenant_update ON contract_clauses;
DROP POLICY IF EXISTS tenant_delete ON contract_clauses;

-- TABLE: contract_versions
ALTER TABLE contract_versions DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON contract_versions;
DROP POLICY IF EXISTS tenant_insert ON contract_versions;
DROP POLICY IF EXISTS tenant_update ON contract_versions;
DROP POLICY IF EXISTS tenant_delete ON contract_versions;

-- TABLE: contracts
ALTER TABLE contracts DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON contracts;
DROP POLICY IF EXISTS tenant_insert ON contracts;
DROP POLICY IF EXISTS tenant_update ON contracts;
DROP POLICY IF EXISTS tenant_delete ON contracts;
