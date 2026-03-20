-- =============================================================================
-- Migration 000016 DOWN: Drop all vCISO Governance tables
-- Uses the correct table names matching the Go repository code.
-- =============================================================================

DROP TABLE IF EXISTS vciso_approvals CASCADE;
DROP TABLE IF EXISTS vciso_control_ownership CASCADE;
DROP TABLE IF EXISTS vciso_integrations CASCADE;
DROP TABLE IF EXISTS vciso_control_tests CASCADE;
DROP TABLE IF EXISTS vciso_obligations CASCADE;
DROP TABLE IF EXISTS vciso_playbooks CASCADE;
DROP TABLE IF EXISTS vciso_escalation_rules CASCADE;
DROP TABLE IF EXISTS vciso_iam_findings CASCADE;
DROP TABLE IF EXISTS vciso_awareness_programs CASCADE;
DROP TABLE IF EXISTS vciso_budget_items CASCADE;
DROP TABLE IF EXISTS vciso_maturity_assessments CASCADE;
DROP TABLE IF EXISTS vciso_evidence CASCADE;
DROP TABLE IF EXISTS vciso_questionnaires CASCADE;
DROP TABLE IF EXISTS vciso_vendors CASCADE;
DROP TABLE IF EXISTS vciso_policy_exceptions CASCADE;
DROP TABLE IF EXISTS vciso_policies CASCADE;
DROP TABLE IF EXISTS vciso_risks CASCADE;
