DROP TABLE IF EXISTS exposure_score_snapshots;
DROP TABLE IF EXISTS ctem_remediation_groups;
DROP TABLE IF EXISTS ctem_findings;

DROP INDEX IF EXISTS idx_ctem_assessment_tenant;
DROP INDEX IF EXISTS idx_ctem_assessment_scheduled;
DROP INDEX IF EXISTS idx_ctem_assessment_tags;
DROP INDEX IF EXISTS idx_ctem_assessment_scope;

ALTER TABLE IF EXISTS ctem_assessments
    DROP CONSTRAINT IF EXISTS ctem_assessments_status_check;

ALTER TABLE IF EXISTS ctem_assessments
    ADD CONSTRAINT ctem_assessments_status_check
    CHECK (status IN ('created', 'scoping', 'discovery', 'prioritizing',
                      'validating', 'mobilizing', 'completed', 'failed', 'cancelled'));
