-- Rollback RLS for all 34 tables added by migration 000022.
DO $$
DECLARE
    tbl TEXT;
    tables TEXT[] := ARRAY[
        'vciso_risks', 'vciso_policies', 'vciso_policy_exceptions',
        'vciso_vendors', 'vciso_questionnaires', 'vciso_evidence',
        'vciso_maturity_assessments', 'vciso_budget_items', 'vciso_awareness_programs',
        'vciso_iam_findings', 'vciso_escalation_rules', 'vciso_playbooks',
        'vciso_obligations', 'vciso_control_tests', 'vciso_integrations',
        'vciso_control_ownership', 'vciso_approvals',
        'vciso_benchmarks', 'vciso_control_dependencies',
        'dspm_access_mappings', 'dspm_identity_profiles', 'dspm_access_audit',
        'dspm_access_policies',
        'dspm_remediations', 'dspm_remediation_history', 'dspm_data_policies',
        'dspm_risk_exceptions',
        'dspm_data_lineage', 'dspm_ai_data_usage', 'dspm_classification_history',
        'dspm_compliance_posture', 'dspm_financial_impact',
        'threat_feed_configs', 'threat_feed_sync_history'
    ];
BEGIN
    FOREACH tbl IN ARRAY tables
    LOOP
        EXECUTE format('DROP POLICY IF EXISTS tenant_delete ON %I', tbl);
        EXECUTE format('DROP POLICY IF EXISTS tenant_update ON %I', tbl);
        EXECUTE format('DROP POLICY IF EXISTS tenant_insert ON %I', tbl);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', tbl);
        EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY', tbl);
    END LOOP;
END
$$;
