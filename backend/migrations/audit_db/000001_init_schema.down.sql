DROP TABLE IF EXISTS audit_chain_state;
DROP TRIGGER IF EXISTS audit_immutability_guard ON audit_logs;
DROP FUNCTION IF EXISTS prevent_audit_mutation();
DROP TABLE IF EXISTS audit_logs CASCADE;
