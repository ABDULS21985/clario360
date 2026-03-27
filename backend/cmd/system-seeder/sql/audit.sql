WITH demo_users AS (
    SELECT *
    FROM (
        VALUES
            (1, '{{ .MainAdminUserID }}'::uuid, 'admin@apexbank.demo'),
            (2, '{{ .SecurityManagerUserID }}'::uuid, 'security@apexbank.demo'),
            (3, '{{ .DataStewardUserID }}'::uuid, 'data@apexbank.demo'),
            (4, '{{ .LegalManagerUserID }}'::uuid, 'legal@apexbank.demo'),
            (5, '{{ .BoardSecretaryUserID }}'::uuid, 'board@apexbank.demo'),
            (6, '{{ .ExecutiveUserID }}'::uuid, 'executive@apexbank.demo'),
            (7, '{{ .AuditorUserID }}'::uuid, 'audit@apexbank.demo')
    ) AS t(slot, user_id, user_email)
)
INSERT INTO audit_logs (
    id, tenant_id, user_id, user_email, service, action, severity, resource_type,
    resource_id, old_value, new_value, ip_address, user_agent, metadata, event_id,
    correlation_id, previous_hash, entry_hash, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'audit-log-' || gs),
    '{{ .MainTenantID }}'::uuid,
    du.user_id,
    du.user_email,
    CASE gs % 6
        WHEN 0 THEN 'cyber-service'
        WHEN 1 THEN 'data-service'
        WHEN 2 THEN 'workflow-engine'
        WHEN 3 THEN 'notification-service'
        WHEN 4 THEN 'visus-service'
        ELSE 'api-gateway'
    END,
    CASE gs % 8
        WHEN 0 THEN 'alert.created'
        WHEN 1 THEN 'workflow.task.completed'
        WHEN 2 THEN 'quality.rule.failed'
        WHEN 3 THEN 'briefing.generated'
        WHEN 4 THEN 'contract.reviewed'
        WHEN 5 THEN 'meeting.minutes.published'
        WHEN 6 THEN 'notification.delivered'
        ELSE 'user.login'
    END,
    CASE gs % 5
        WHEN 0 THEN 'critical'
        WHEN 1 THEN 'high'
        WHEN 2 THEN 'warning'
        ELSE 'info'
    END,
    CASE gs % 6
        WHEN 0 THEN 'alert'
        WHEN 1 THEN 'workflow_task'
        WHEN 2 THEN 'data_model'
        WHEN 3 THEN 'briefing'
        WHEN 4 THEN 'contract'
        ELSE 'session'
    END,
    format('resource-%s', gs),
    CASE WHEN gs % 7 = 0 THEN jsonb_build_object('status', 'previous', 'sequence', gs) ELSE NULL END,
    jsonb_build_object('status', 'current', 'sequence', gs, 'seed_key', '{{ .SeedKey }}'),
    format('172.16.%s.%s', ((gs - 1) % 255), (gs % 255)),
    format('Clario Audit Client/%s', 1 + (gs % 4)),
    jsonb_build_object('seeded', true, 'sequence', gs, 'module', CASE WHEN gs % 2 = 0 THEN 'cyber' ELSE 'data' END),
    format('seeded-audit-event-%s', gs),
    format('seeded-correlation-%s', ((gs - 1) / 12) + 1),
    md5('audit-prev-' || gs),
    md5('audit-entry-' || gs),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320))
FROM generate_series(1, {{ .Scale.AuditLogCount }}) gs
JOIN demo_users du ON du.slot = ((gs - 1) % 7) + 1
ON CONFLICT DO NOTHING;

WITH last_event AS (
    SELECT
        uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'audit-log-' || {{ .Scale.AuditLogCount }}) AS last_entry_id,
        md5('audit-entry-' || {{ .Scale.AuditLogCount }}) AS last_hash,
        date_trunc('month', now()) + make_interval(mins => (({{ .Scale.AuditLogCount }} - 1) % 40320)) AS last_created_at
)
INSERT INTO audit_chain_state (
    tenant_id, last_entry_id, last_hash, last_created_at, updated_at
)
SELECT
    '{{ .MainTenantID }}'::uuid,
    last_entry_id,
    last_hash,
    last_created_at,
    now()
FROM last_event
ON CONFLICT (tenant_id) DO UPDATE SET
    last_entry_id = EXCLUDED.last_entry_id,
    last_hash = EXCLUDED.last_hash,
    last_created_at = EXCLUDED.last_created_at,
    updated_at = EXCLUDED.updated_at;
