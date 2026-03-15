-- =============================================================================
-- Migration 000022: RLS for all cyber_db tables added after the initial RLS
-- migration (000011). Covers:
--   - 17 vCISO governance tables (000016)
--   -  2 vCISO governance hardening tables (000021)
--   -  4 DSPM access intelligence tables (000017)
--   -  4 DSPM remediation engine tables (000018)
--   -  5 DSPM advanced intelligence tables (000019)
--   -  2 threat feed config tables (000020)
-- Total: 34 tables
-- =============================================================================

-- Helper: generates ENABLE + FORCE + 4 policies per table.
DO $$
DECLARE
    tbl TEXT;
    tables TEXT[] := ARRAY[
        -- vCISO governance (000016)
        'vciso_risks',
        'vciso_policies',
        'vciso_policy_exceptions',
        'vciso_vendors',
        'vciso_questionnaires',
        'vciso_evidence',
        'vciso_maturity_assessments',
        'vciso_budget_items',
        'vciso_awareness_programs',
        'vciso_iam_findings',
        'vciso_escalation_rules',
        'vciso_playbooks',
        'vciso_obligations',
        'vciso_control_tests',
        'vciso_integrations',
        'vciso_control_ownership',
        'vciso_approvals',
        -- vCISO governance hardening (000021)
        'vciso_benchmarks',
        'vciso_control_dependencies',
        -- DSPM access intelligence (000017)
        'dspm_access_mappings',
        'dspm_identity_profiles',
        'dspm_access_audit',
        'dspm_access_policies',
        -- DSPM remediation engine (000018)
        'dspm_remediations',
        'dspm_remediation_history',
        'dspm_data_policies',
        'dspm_risk_exceptions',
        -- DSPM advanced intelligence (000019)
        'dspm_data_lineage',
        'dspm_ai_data_usage',
        'dspm_classification_history',
        'dspm_compliance_posture',
        'dspm_financial_impact',
        -- Threat feed configs (000020)
        'threat_feed_configs',
        'threat_feed_sync_history'
    ];
BEGIN
    FOREACH tbl IN ARRAY tables
    LOOP
        -- Enable + force RLS
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', tbl);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', tbl);

        -- SELECT policy
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON %I USING (tenant_id = current_setting(''app.current_tenant_id'', true)::uuid)',
            tbl
        );

        -- INSERT policy
        EXECUTE format(
            'CREATE POLICY tenant_insert ON %I FOR INSERT WITH CHECK (tenant_id = current_setting(''app.current_tenant_id'', true)::uuid)',
            tbl
        );

        -- UPDATE policy
        EXECUTE format(
            'CREATE POLICY tenant_update ON %I FOR UPDATE USING (tenant_id = current_setting(''app.current_tenant_id'', true)::uuid) WITH CHECK (tenant_id = current_setting(''app.current_tenant_id'', true)::uuid)',
            tbl
        );

        -- DELETE policy
        EXECUTE format(
            'CREATE POLICY tenant_delete ON %I FOR DELETE USING (tenant_id = current_setting(''app.current_tenant_id'', true)::uuid)',
            tbl
        );
    END LOOP;
END
$$;
