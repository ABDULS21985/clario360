WITH demo_users AS (
    SELECT *
    FROM (
        VALUES
            (1, '{{ .MainAdminUserID }}'::uuid),
            (2, '{{ .SecurityManagerUserID }}'::uuid),
            (3, '{{ .DataStewardUserID }}'::uuid),
            (4, '{{ .LegalManagerUserID }}'::uuid),
            (5, '{{ .BoardSecretaryUserID }}'::uuid),
            (6, '{{ .ExecutiveUserID }}'::uuid),
            (7, '{{ .AuditorUserID }}'::uuid)
    ) AS t(slot, user_id)
)
INSERT INTO notification_preferences (
    user_id, tenant_id, global_prefs, per_type_prefs, quiet_hours, digest_config, updated_at
)
SELECT
    user_id,
    '{{ .MainTenantID }}'::uuid,
    jsonb_build_object('in_app', true, 'email', true, 'websocket', true, 'webhook', slot IN (1, 2, 6)),
    jsonb_build_object(
        'critical_alert', jsonb_build_object('email', true, 'in_app', true),
        'governance_digest', jsonb_build_object('email', slot IN (1, 6), 'in_app', true)
    ),
    jsonb_build_object('start', '22:00', 'end', '06:00', 'timezone', 'Africa/Lagos'),
    jsonb_build_object('daily', slot IN (1, 2, 3), 'weekly', true),
    now()
FROM demo_users
ON CONFLICT (user_id, tenant_id) DO UPDATE SET
    global_prefs = EXCLUDED.global_prefs,
    per_type_prefs = EXCLUDED.per_type_prefs,
    quiet_hours = EXCLUDED.quiet_hours,
    digest_config = EXCLUDED.digest_config,
    updated_at = EXCLUDED.updated_at;

INSERT INTO notification_webhooks (
    id, tenant_id, name, url, secret, event_types, active, created_by, created_at, updated_at
)
VALUES
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'notif-webhook-slack'), '{{ .MainTenantID }}'::uuid, 'Slack Ops Webhook', 'https://hooks.demo.local/slack/security', 'seeded-slack-secret', ARRAY['alert.created','workflow.task.pending','risk.score.changed'], true, '{{ .MainAdminUserID }}'::uuid, now() - interval '20 days', now()),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'notif-webhook-board'), '{{ .MainTenantID }}'::uuid, 'Board Digest Webhook', 'https://hooks.demo.local/webhook/board', 'seeded-board-secret', ARRAY['briefing.generated','report.ready'], true, '{{ .ExecutiveUserID }}'::uuid, now() - interval '18 days', now())
ON CONFLICT (id) DO UPDATE SET
    url = EXCLUDED.url,
    secret = EXCLUDED.secret,
    event_types = EXCLUDED.event_types,
    active = EXCLUDED.active,
    updated_at = EXCLUDED.updated_at;

INSERT INTO notification_templates (
    id, tenant_id, channel, subject_tmpl, body_tmpl, created_at, updated_at
)
VALUES
    ('critical-alert', '{{ .MainTenantID }}'::uuid, 'email', 'Critical alert notification', 'A seeded critical alert requires immediate action.', now() - interval '15 days', now()),
    ('critical-alert', '{{ .MainTenantID }}'::uuid, 'in_app', '', 'Critical alert notification', now() - interval '15 days', now()),
    ('workflow-task', '{{ .MainTenantID }}'::uuid, 'email', 'Workflow task assigned', 'A seeded workflow task is awaiting review.', now() - interval '15 days', now()),
    ('workflow-task', '{{ .MainTenantID }}'::uuid, 'websocket', '', 'Workflow task pending', now() - interval '15 days', now()),
    ('briefing-ready', '{{ .MainTenantID }}'::uuid, 'email', 'Executive briefing ready', 'Your seeded executive briefing is available.', now() - interval '15 days', now()),
    ('briefing-ready', '{{ .MainTenantID }}'::uuid, 'in_app', '', 'Executive briefing ready', now() - interval '15 days', now())
ON CONFLICT (id, channel, tenant_id) DO UPDATE SET
    subject_tmpl = EXCLUDED.subject_tmpl,
    body_tmpl = EXCLUDED.body_tmpl,
    updated_at = EXCLUDED.updated_at;

INSERT INTO integrations (
    id, tenant_id, type, name, description, config_encrypted, config_nonce, config_key_id,
    status, error_message, error_count, last_error_at, event_filters, last_used_at,
    delivery_count, created_by, created_at, updated_at, deleted_at
)
VALUES
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'integration-slack'), '{{ .MainTenantID }}'::uuid, 'slack', 'Slack SOC Bridge', 'Seeded Slack integration for SOC workflows.', convert_to('{"channel":"#soc","workspace":"apex-demo"}', 'UTF8'), convert_to('nonce-slack', 'UTF8'), 'demo-key-1', 'active', NULL, 0, NULL, '["alert.created","alert.escalated","ueba.alert.created"]'::jsonb, now() - interval '1 day', 0, '{{ .MainAdminUserID }}'::uuid, now() - interval '25 days', now(), NULL),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'integration-jira'), '{{ .MainTenantID }}'::uuid, 'jira', 'Jira Service Desk', 'Seeded Jira integration for remediation tasks.', convert_to('{"project":"SEC","issueType":"Incident"}', 'UTF8'), convert_to('nonce-jira', 'UTF8'), 'demo-key-2', 'active', NULL, 0, NULL, '["remediation.created","workflow.task.pending"]'::jsonb, now() - interval '1 day', 0, '{{ .SecurityManagerUserID }}'::uuid, now() - interval '22 days', now(), NULL),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'integration-servicenow'), '{{ .MainTenantID }}'::uuid, 'servicenow', 'ServiceNow Ops', 'Seeded ServiceNow integration for incident coordination.', convert_to('{"table":"incident","assignmentGroup":"security"}', 'UTF8'), convert_to('nonce-snow', 'UTF8'), 'demo-key-3', 'active', NULL, 0, NULL, '["alert.created","remediation.approved"]'::jsonb, now() - interval '2 days', 0, '{{ .SecurityManagerUserID }}'::uuid, now() - interval '22 days', now(), NULL),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'integration-teams'), '{{ .MainTenantID }}'::uuid, 'teams', 'Executive Teams Channel', 'Seeded Teams integration for executive notifications.', convert_to('{"team":"Exec","channel":"Risk Updates"}', 'UTF8'), convert_to('nonce-teams', 'UTF8'), 'demo-key-4', 'active', NULL, 0, NULL, '["briefing.generated","report.ready"]'::jsonb, now() - interval '2 days', 0, '{{ .ExecutiveUserID }}'::uuid, now() - interval '22 days', now(), NULL)
ON CONFLICT (id) DO UPDATE SET
    description = EXCLUDED.description,
    config_encrypted = EXCLUDED.config_encrypted,
    config_nonce = EXCLUDED.config_nonce,
    config_key_id = EXCLUDED.config_key_id,
    status = EXCLUDED.status,
    event_filters = EXCLUDED.event_filters,
    updated_at = EXCLUDED.updated_at,
    deleted_at = NULL;

WITH demo_users AS (
    SELECT *
    FROM (
        VALUES
            (1, '{{ .MainAdminUserID }}'::uuid),
            (2, '{{ .SecurityManagerUserID }}'::uuid),
            (3, '{{ .DataStewardUserID }}'::uuid),
            (4, '{{ .LegalManagerUserID }}'::uuid),
            (5, '{{ .BoardSecretaryUserID }}'::uuid),
            (6, '{{ .ExecutiveUserID }}'::uuid),
            (7, '{{ .AuditorUserID }}'::uuid)
    ) AS t(slot, user_id)
)
INSERT INTO notifications (
    id, tenant_id, user_id, type, category, priority, title, body, data, action_url, source_event_id, read_at, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'notif-db-notification-' || gs),
    '{{ .MainTenantID }}'::uuid,
    du.user_id,
    CASE gs % 6
        WHEN 0 THEN 'critical_alert'
        WHEN 1 THEN 'workflow_task'
        WHEN 2 THEN 'data_quality'
        WHEN 3 THEN 'briefing_ready'
        WHEN 4 THEN 'contract_expiry'
        ELSE 'system_health'
    END,
    CASE gs % 6
        WHEN 0 THEN 'security'
        WHEN 1 THEN 'workflow'
        WHEN 2 THEN 'data'
        WHEN 3 THEN 'governance'
        WHEN 4 THEN 'legal'
        ELSE 'system'
    END,
    CASE gs % 5
        WHEN 0 THEN 'critical'
        WHEN 1 THEN 'high'
        WHEN 2 THEN 'medium'
        WHEN 3 THEN 'medium'
        ELSE 'low'
    END,
    format('Seeded notification %s', lpad(gs::text, 6, '0')),
    format('Notification %s supports the demonstration datasets for dashboards, triage queues, and workflow inboxes.', gs),
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs, 'module', CASE WHEN gs % 2 = 0 THEN 'cyber' ELSE 'data' END),
    CASE
        WHEN gs % 6 = 0 THEN format('/cyber/alerts/%s', gs)
        WHEN gs % 6 = 1 THEN format('/workflows/tasks/%s', gs)
        WHEN gs % 6 = 2 THEN format('/data/quality/%s', gs)
        WHEN gs % 6 = 3 THEN format('/visus/briefings/%s', gs)
        WHEN gs % 6 = 4 THEN format('/lex/contracts/%s', gs)
        ELSE '/settings/notifications'
    END,
    format('seeded-notification-event-%s', gs),
    CASE WHEN gs % 5 = 0 THEN now() - interval '3 hours' ELSE NULL END,
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320))
FROM generate_series(1, {{ .Scale.NotificationCount }}) gs
JOIN demo_users du ON du.slot = ((gs - 1) % 7) + 1
ON CONFLICT DO NOTHING;

INSERT INTO notification_delivery_log (
    id, notification_id, channel, status, attempt, error_message, metadata, webhook_id,
    event_type, request_url, request_body, response_status, response_body, duration_ms,
    next_retry_at, delivered_at, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'notification-delivery-' || gs),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'notif-db-notification-' || (((gs - 1) % {{ .Scale.NotificationCount }}) + 1)),
    CASE gs % 4
        WHEN 0 THEN 'in_app'
        WHEN 1 THEN 'email'
        WHEN 2 THEN 'websocket'
        ELSE 'webhook'
    END,
    CASE
        WHEN gs % 17 = 0 THEN 'failed'
        WHEN gs % 13 = 0 THEN 'pending'
        ELSE 'delivered'
    END,
    1 + (gs % 3),
    CASE WHEN gs % 17 = 0 THEN 'Simulated downstream timeout' ELSE NULL END,
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs),
    CASE WHEN gs % 4 = 3 THEN uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'notif-webhook-slack') ELSE NULL END,
    CASE
        WHEN gs % 6 = 0 THEN 'alert.created'
        WHEN gs % 6 = 1 THEN 'workflow.task.pending'
        WHEN gs % 6 = 2 THEN 'briefing.generated'
        WHEN gs % 6 = 3 THEN 'report.ready'
        WHEN gs % 6 = 4 THEN 'quality.rule.failed'
        ELSE 'system.health.warn'
    END,
    CASE WHEN gs % 4 = 3 THEN 'https://hooks.demo.local/slack/security' ELSE NULL END,
    CASE WHEN gs % 4 = 3 THEN jsonb_build_object('message', format('seeded delivery %s', gs)) ELSE NULL END,
    CASE
        WHEN gs % 17 = 0 THEN 504
        WHEN gs % 13 = 0 THEN 202
        ELSE 200
    END,
    CASE
        WHEN gs % 17 = 0 THEN 'Gateway timeout'
        WHEN gs % 13 = 0 THEN 'Pending retry'
        ELSE 'ok'
    END,
    45 + (gs % 700),
    CASE WHEN gs % 13 = 0 THEN now() + interval '10 minutes' ELSE NULL END,
    CASE WHEN gs % 17 = 0 OR gs % 13 = 0 THEN NULL ELSE date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 2) END,
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320))
FROM generate_series(1, {{ .Scale.NotificationDeliveryCount }}) gs
ON CONFLICT (id) DO NOTHING;

INSERT INTO external_ticket_links (
    id, tenant_id, integration_id, entity_type, entity_id, external_system, external_id, external_key,
    external_url, external_status, external_priority, sync_direction, last_synced_at,
    last_sync_direction, sync_error, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'external-ticket-link-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE (gs - 1) % 3
        WHEN 0 THEN uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'integration-jira')
        WHEN 1 THEN uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'integration-servicenow')
        ELSE uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'integration-teams')
    END,
    CASE (gs - 1) % 3
        WHEN 0 THEN 'alert'
        WHEN 1 THEN 'remediation'
        ELSE 'workflow_task'
    END,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ticket-entity-' || gs),
    CASE (gs - 1) % 2 WHEN 0 THEN 'jira' ELSE 'servicenow' END,
    format('EXT-%s', lpad(gs::text, 5, '0')),
    format('KEY-%s', lpad(gs::text, 5, '0')),
    format('https://tickets.demo.local/%s', lpad(gs::text, 5, '0')),
    CASE gs % 4 WHEN 0 THEN 'open' WHEN 1 THEN 'in_progress' WHEN 2 THEN 'resolved' ELSE 'backlog' END,
    CASE gs % 4 WHEN 0 THEN 'critical' WHEN 1 THEN 'high' WHEN 2 THEN 'medium' ELSE 'low' END,
    'bidirectional',
    now() - make_interval(days => (gs % 14)),
    'outbound',
    NULL,
    now() - make_interval(days => (gs % 21)),
    now()
FROM generate_series(1, 120) gs
ON CONFLICT (tenant_id, entity_type, entity_id, external_system, external_id) DO NOTHING;

INSERT INTO integration_deliveries (
    id, tenant_id, integration_id, event_type, event_id, event_data, status, attempts,
    max_attempts, response_code, response_body, last_error, error_category, next_retry_at,
    latency_ms, delivered_at, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'integration-delivery-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE (gs - 1) % 4
        WHEN 0 THEN uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'integration-slack')
        WHEN 1 THEN uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'integration-jira')
        WHEN 2 THEN uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'integration-servicenow')
        ELSE uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'integration-teams')
    END,
    CASE gs % 6
        WHEN 0 THEN 'alert.created'
        WHEN 1 THEN 'alert.escalated'
        WHEN 2 THEN 'remediation.created'
        WHEN 3 THEN 'workflow.task.pending'
        WHEN 4 THEN 'briefing.generated'
        ELSE 'report.ready'
    END,
    format('seeded-integration-event-%s', gs),
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs, 'event', CASE WHEN gs % 2 = 0 THEN 'cyber' ELSE 'governance' END),
    CASE
        WHEN gs % 23 = 0 THEN 'failed'
        WHEN gs % 19 = 0 THEN 'retrying'
        ELSE 'delivered'
    END,
    1 + (gs % 3),
    4,
    CASE
        WHEN gs % 23 = 0 THEN 500
        WHEN gs % 19 = 0 THEN 429
        ELSE 200
    END,
    CASE
        WHEN gs % 23 = 0 THEN 'Delivery failed'
        WHEN gs % 19 = 0 THEN 'Retry pending'
        ELSE 'ok'
    END,
    CASE WHEN gs % 23 = 0 THEN 'Seeded outbound delivery failure' ELSE NULL END,
    CASE WHEN gs % 23 = 0 THEN 'remote_error' WHEN gs % 19 = 0 THEN 'rate_limit' ELSE NULL END,
    CASE WHEN gs % 19 = 0 THEN now() + interval '15 minutes' ELSE NULL END,
    70 + (gs % 900),
    CASE WHEN gs % 23 = 0 OR gs % 19 = 0 THEN NULL ELSE date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 3) END,
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320))
FROM generate_series(1, {{ .Scale.IntegrationDeliveryCount }}) gs
ON CONFLICT DO NOTHING;
