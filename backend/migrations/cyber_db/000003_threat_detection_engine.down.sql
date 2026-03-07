-- =============================================================================
-- Clario 360 — Cyber Suite Threat Detection Engine rollback
-- =============================================================================

DROP FUNCTION IF EXISTS create_security_events_partition(DATE);
DROP TABLE IF EXISTS security_events CASCADE;
DROP TABLE IF EXISTS alert_timeline;
DROP TABLE IF EXISTS alert_comments;

DROP INDEX IF EXISTS idx_rules_template_unique;
DROP INDEX IF EXISTS idx_rules_tenant_name_unique;
DROP INDEX IF EXISTS idx_rules_tenant_type;

ALTER TABLE IF EXISTS detection_rules
    DROP COLUMN IF EXISTS mitre_tactic_ids,
    DROP COLUMN IF EXISTS base_confidence,
    DROP COLUMN IF EXISTS false_positive_count,
    DROP COLUMN IF EXISTS true_positive_count,
    DROP COLUMN IF EXISTS tags,
    DROP COLUMN IF EXISTS is_template,
    DROP COLUMN IF EXISTS template_id,
    DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE IF EXISTS alerts
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS asset_id,
    DROP COLUMN IF EXISTS assigned_at,
    DROP COLUMN IF EXISTS escalated_to,
    DROP COLUMN IF EXISTS escalated_at,
    DROP COLUMN IF EXISTS mitre_tactic_id,
    DROP COLUMN IF EXISTS mitre_tactic_name,
    DROP COLUMN IF EXISTS mitre_technique_id,
    DROP COLUMN IF EXISTS mitre_technique_name,
    DROP COLUMN IF EXISTS event_count,
    DROP COLUMN IF EXISTS first_event_at,
    DROP COLUMN IF EXISTS last_event_at,
    DROP COLUMN IF EXISTS false_positive_reason,
    DROP COLUMN IF EXISTS tags,
    DROP COLUMN IF EXISTS metadata,
    DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE IF EXISTS threats
    DROP COLUMN IF EXISTS threat_actor,
    DROP COLUMN IF EXISTS campaign,
    DROP COLUMN IF EXISTS mitre_tactic_ids,
    DROP COLUMN IF EXISTS mitre_technique_ids,
    DROP COLUMN IF EXISTS affected_asset_count,
    DROP COLUMN IF EXISTS alert_count,
    DROP COLUMN IF EXISTS first_seen_at,
    DROP COLUMN IF EXISTS last_seen_at,
    DROP COLUMN IF EXISTS contained_at,
    DROP COLUMN IF EXISTS tags,
    DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE IF EXISTS threat_indicators
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS severity,
    DROP COLUMN IF EXISTS active,
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS tags,
    DROP COLUMN IF EXISTS metadata,
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS updated_at;

