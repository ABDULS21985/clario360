DELETE FROM ai_validation_results WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM ai_drift_reports WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM ai_shadow_comparisons WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM ai_prediction_logs WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM ai_benchmark_runs WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM ai_benchmark_suites WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM ai_inference_servers WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM ai_compute_cost_models WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM ai_model_versions WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM ai_models WHERE tenant_id = '{{ .MainTenantID }}'::uuid;

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
INSERT INTO sessions (
    id, user_id, tenant_id, refresh_token_hash, ip_address, user_agent, expires_at, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'platform-session-' || u.slot || '-' || gs),
    u.user_id,
    '{{ .MainTenantID }}'::uuid,
    md5('refresh-token-' || u.slot || '-' || gs),
    format('10.20.%s.%s', u.slot, gs)::inet,
    format('Clario Demo Browser/%s', gs),
    now() + interval '30 days',
    now() - make_interval(days => gs, hours => u.slot)
FROM demo_users u
CROSS JOIN generate_series(1, 2) gs
ON CONFLICT (id) DO UPDATE SET
    refresh_token_hash = EXCLUDED.refresh_token_hash,
    ip_address = EXCLUDED.ip_address,
    user_agent = EXCLUDED.user_agent,
    expires_at = EXCLUDED.expires_at;

WITH api_keys AS (
    SELECT *
    FROM (
        VALUES
            ('cyber-demo-key', '{{ .MainAdminUserID }}'::uuid, 'Cyber Operations Key', '["cyber:*","alerts:*"]'::jsonb),
            ('data-demo-key', '{{ .DataStewardUserID }}'::uuid, 'Data Operations Key', '["data:*","quality:*"]'::jsonb),
            ('visus-demo-key', '{{ .ExecutiveUserID }}'::uuid, 'Executive Reporting Key', '["visus:*","reports:read"]'::jsonb)
    ) AS t(seed_name, created_by, name, permissions)
)
INSERT INTO api_keys (
    id, tenant_id, name, key_hash, key_prefix, permissions, last_used_at, expires_at, created_by, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, seed_name),
    '{{ .MainTenantID }}'::uuid,
    name,
    md5(seed_name || ':hash'),
    'clr_demo',
    permissions,
    now() - interval '1 day',
    now() + interval '365 days',
    created_by,
    now() - interval '14 days'
FROM api_keys
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    permissions = EXCLUDED.permissions,
    last_used_at = EXCLUDED.last_used_at,
    expires_at = EXCLUDED.expires_at,
    revoked_at = NULL;

WITH reset_tokens AS (
    SELECT *
    FROM (
        VALUES
            ('reset-admin', '{{ .MainAdminUserID }}'::uuid),
            ('reset-security', '{{ .SecurityManagerUserID }}'::uuid),
            ('reset-data', '{{ .DataStewardUserID }}'::uuid),
            ('reset-legal', '{{ .LegalManagerUserID }}'::uuid)
    ) AS t(seed_name, user_id)
)
INSERT INTO password_reset_tokens (
    id, user_id, token_hash, expires_at, used, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, seed_name),
    user_id,
    md5(seed_name || ':token'),
    now() + interval '7 days',
    false,
    now() - interval '1 day'
FROM reset_tokens
ON CONFLICT (id) DO UPDATE SET
    token_hash = EXCLUDED.token_hash,
    expires_at = EXCLUDED.expires_at,
    used = false,
    used_at = NULL;

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
    id, tenant_id, user_id, type, title, body, data, read_at, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'platform-notification-' || gs),
    '{{ .MainTenantID }}'::uuid,
    du.user_id,
    CASE gs % 5
        WHEN 0 THEN 'action_required'::notification_type
        WHEN 1 THEN 'info'::notification_type
        WHEN 2 THEN 'warning'::notification_type
        WHEN 3 THEN 'success'::notification_type
        ELSE 'error'::notification_type
    END,
    format('Platform demo notification %s', lpad(gs::text, 5, '0')),
    format('Seeded platform event %s for walkthroughs and onboarding demos.', gs),
    jsonb_build_object(
        'seed_key', '{{ .SeedKey }}',
        'module', CASE
            WHEN gs % 5 = 0 THEN 'workflow'
            WHEN gs % 5 = 1 THEN 'cyber'
            WHEN gs % 5 = 2 THEN 'data'
            WHEN gs % 5 = 3 THEN 'visus'
            ELSE 'platform'
        END,
        'sequence', gs
    ),
    CASE WHEN gs % 4 = 0 THEN now() - interval '2 hours' ELSE NULL END,
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320))
FROM generate_series(1, {{ .Scale.PlatformNotificationCount }}) gs
JOIN demo_users du ON du.slot = ((gs - 1) % 7) + 1
ON CONFLICT (id) DO NOTHING;

INSERT INTO ai_models (
    id, tenant_id, name, slug, description, model_type, suite, owner_user_id, owner_team,
    risk_tier, status, tags, metadata, created_by
)
VALUES
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-cyber-vciso-llm'), '{{ .MainTenantID }}'::uuid, 'Virtual CISO LLM Engine', 'cyber-vciso-llm', 'LLM-powered conversational governance engine for complex security investigations.', 'llm_agentic', 'cyber', '{{ .SecurityManagerUserID }}'::uuid, 'security-operations', 'high', 'active', ARRAY['cyber','vciso','llm','agentic'], '{"seeded":true,"module":"cyber"}', '{{ .MainAdminUserID }}'::uuid),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-cyber-vciso-predictive'), '{{ .MainTenantID }}'::uuid, 'vCISO Predictive Threat Engine', 'cyber-vciso-predictive', 'Forecasting model for security demand, threat pressure, and risk trajectories.', 'ml_classifier', 'cyber', '{{ .SecurityManagerUserID }}'::uuid, 'security-operations', 'high', 'active', ARRAY['cyber','forecasting','predictive'], '{"seeded":true,"module":"cyber"}', '{{ .MainAdminUserID }}'::uuid),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-cyber-risk-scorer'), '{{ .MainTenantID }}'::uuid, 'Risk Scoring Composite', 'cyber-risk-scorer', 'Transparent weighted risk scorer for cyber posture.', 'scorer', 'cyber', '{{ .SecurityManagerUserID }}'::uuid, 'security-risk', 'high', 'active', ARRAY['cyber','risk','scorer'], '{"seeded":true,"module":"cyber"}', '{{ .MainAdminUserID }}'::uuid),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-data-quality-scorer'), '{{ .MainTenantID }}'::uuid, 'Data Quality Scorer', 'data-quality-scorer', 'Weighted quality score model for enterprise data domains.', 'scorer', 'data', '{{ .DataStewardUserID }}'::uuid, 'data-quality', 'medium', 'active', ARRAY['data','quality','scorer'], '{"seeded":true,"module":"data"}', '{{ .MainAdminUserID }}'::uuid),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-data-contradiction-detector'), '{{ .MainTenantID }}'::uuid, 'Contradiction Detector', 'data-contradiction-detector', 'Transparent contradiction detector across overlapping data sources.', 'rule_based', 'data', '{{ .DataStewardUserID }}'::uuid, 'data-governance', 'medium', 'active', ARRAY['data','contradiction','governance'], '{"seeded":true,"module":"data"}', '{{ .MainAdminUserID }}'::uuid),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-acta-minutes-generator'), '{{ .MainTenantID }}'::uuid, 'Meeting Minutes Generator', 'acta-minutes-generator', 'Template-driven meeting minutes generation model.', 'nlp_extractor', 'acta', '{{ .BoardSecretaryUserID }}'::uuid, 'governance-operations', 'medium', 'active', ARRAY['acta','minutes','template'], '{"seeded":true,"module":"acta"}', '{{ .MainAdminUserID }}'::uuid),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-lex-risk-analyzer'), '{{ .MainTenantID }}'::uuid, 'Contract Risk Analyzer', 'lex-risk-analyzer', 'Explainable legal contract risk analysis model.', 'scorer', 'lex', '{{ .LegalManagerUserID }}'::uuid, 'legal-operations', 'high', 'active', ARRAY['lex','risk','analysis'], '{"seeded":true,"module":"lex"}', '{{ .MainAdminUserID }}'::uuid),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-visus-kpi-monitor'), '{{ .MainTenantID }}'::uuid, 'KPI Threshold Monitor', 'visus-kpi-monitor', 'Threshold evaluator for executive KPI state changes.', 'statistical', 'visus', '{{ .ExecutiveUserID }}'::uuid, 'executive-reporting', 'medium', 'active', ARRAY['visus','kpi','monitor'], '{"seeded":true,"module":"visus"}', '{{ .MainAdminUserID }}'::uuid)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    model_type = EXCLUDED.model_type,
    suite = EXCLUDED.suite,
    owner_user_id = EXCLUDED.owner_user_id,
    owner_team = EXCLUDED.owner_team,
    risk_tier = EXCLUDED.risk_tier,
    status = EXCLUDED.status,
    tags = EXCLUDED.tags,
    metadata = EXCLUDED.metadata,
    updated_at = now(),
    deleted_at = NULL;

INSERT INTO ai_model_versions (
    id, tenant_id, model_id, version_number, status, description, artifact_type,
    artifact_config, artifact_hash, explainability_type, explanation_template,
    training_metrics, promoted_to_production_at, promoted_by, created_by
)
VALUES
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-cyber-vciso-llm-v1'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-cyber-vciso-llm'), 1, 'production', 'Seeded production vCISO LLM release.', 'template_config', '{"providers":["openai","anthropic","local"],"tool_count":24,"guardrails":["grounding","rate_limits","pii_filter"]}', 'seed-cyber-vciso-llm-v1', 'reasoning_trace', 'Seeded vCISO LLM explanation template.', '{"latency_p95_ms":930,"grounding_pass_rate":0.98}', now() - interval '30 days', '{{ .MainAdminUserID }}'::uuid, '{{ .MainAdminUserID }}'::uuid),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-cyber-vciso-predictive-v1'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-cyber-vciso-predictive'), 1, 'production', 'Seeded production predictive release.', 'serialized_model', '{"engine":"gradient_boosting","targets":["alert_volume","risk_trajectory","campaign_detection"]}', 'seed-cyber-vciso-predictive-v1', 'feature_importance', 'Seeded predictive explanation template.', '{"mae":0.07,"precision":0.91}', now() - interval '45 days', '{{ .MainAdminUserID }}'::uuid, '{{ .MainAdminUserID }}'::uuid),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-cyber-risk-scorer-v1'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-cyber-risk-scorer'), 1, 'production', 'Seeded production risk scorer.', 'statistical_config', '{"weights":{"vulnerability":0.32,"threat":0.28,"config":0.14,"surface":0.14,"compliance":0.12}}', 'seed-cyber-risk-scorer-v1', 'feature_importance', 'Seeded risk scorer explanation template.', '{"r2":0.88}', now() - interval '45 days', '{{ .MainAdminUserID }}'::uuid, '{{ .MainAdminUserID }}'::uuid),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-data-quality-scorer-v1'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-data-quality-scorer'), 1, 'production', 'Seeded data quality production version.', 'statistical_config', '{"weights":{"critical":4,"high":3,"medium":2,"low":1}}', 'seed-data-quality-scorer-v1', 'feature_importance', 'Seeded data quality explanation template.', '{"accuracy":0.93}', now() - interval '25 days', '{{ .MainAdminUserID }}'::uuid, '{{ .MainAdminUserID }}'::uuid),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-data-contradiction-detector-v1'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-data-contradiction-detector'), 1, 'production', 'Seeded contradiction detector.', 'rule_set', '{"strategies":["logical","semantic","temporal","analytical"]}', 'seed-data-contradiction-detector-v1', 'rule_trace', 'Seeded contradiction explanation template.', '{"precision":0.89}', now() - interval '20 days', '{{ .MainAdminUserID }}'::uuid, '{{ .MainAdminUserID }}'::uuid),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-acta-minutes-generator-v1'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-acta-minutes-generator'), 1, 'production', 'Seeded ACTA minutes generator.', 'template_config', '{"template":"board_minutes_markdown","summary_style":"deterministic"}', 'seed-acta-minutes-generator-v1', 'template_based', 'Seeded minutes generator explanation template.', '{"human_acceptance_rate":0.94}', now() - interval '35 days', '{{ .MainAdminUserID }}'::uuid, '{{ .MainAdminUserID }}'::uuid),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-lex-risk-analyzer-v1'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-lex-risk-analyzer'), 1, 'production', 'Seeded LEX risk analyzer.', 'statistical_config', '{"factors":["missing_clause","termination","liability","jurisdiction","pii"]}', 'seed-lex-risk-analyzer-v1', 'feature_importance', 'Seeded legal risk explanation template.', '{"accuracy":0.9}', now() - interval '28 days', '{{ .MainAdminUserID }}'::uuid, '{{ .MainAdminUserID }}'::uuid),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-visus-kpi-monitor-v1'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-visus-kpi-monitor'), 1, 'production', 'Seeded Visus KPI monitor.', 'statistical_config', '{"logic":"directional_threshold"}', 'seed-visus-kpi-monitor-v1', 'statistical_deviation', 'Seeded KPI monitor explanation template.', '{"stability":0.96}', now() - interval '18 days', '{{ .MainAdminUserID }}'::uuid, '{{ .MainAdminUserID }}'::uuid),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-cyber-vciso-llm-v2'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-cyber-vciso-llm'), 2, 'shadow', 'Seeded shadow vCISO LLM release.', 'template_config', '{"providers":["openai","anthropic","local"],"tool_count":27,"guardrails":["grounding","rate_limits","pii_filter","prompt_injection"]}', 'seed-cyber-vciso-llm-v2', 'reasoning_trace', 'Seeded shadow vCISO LLM explanation template.', '{"latency_p95_ms":810,"grounding_pass_rate":0.985}', NULL, NULL, '{{ .MainAdminUserID }}'::uuid),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-cyber-vciso-predictive-v2'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-cyber-vciso-predictive'), 2, 'shadow', 'Seeded shadow predictive release.', 'serialized_model', '{"engine":"xgboost","targets":["alert_volume","risk_trajectory","campaign_detection"]}', 'seed-cyber-vciso-predictive-v2', 'feature_importance', 'Seeded shadow predictive explanation template.', '{"mae":0.05,"precision":0.93}', NULL, NULL, '{{ .MainAdminUserID }}'::uuid),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-data-quality-scorer-v2'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-data-quality-scorer'), 2, 'shadow', 'Seeded shadow quality scorer.', 'statistical_config', '{"weights":{"critical":5,"high":3,"medium":2,"low":1}}', 'seed-data-quality-scorer-v2', 'feature_importance', 'Seeded shadow quality explanation template.', '{"accuracy":0.95}', NULL, NULL, '{{ .MainAdminUserID }}'::uuid)
ON CONFLICT (id) DO UPDATE SET
    status = EXCLUDED.status,
    description = EXCLUDED.description,
    artifact_config = EXCLUDED.artifact_config,
    artifact_hash = EXCLUDED.artifact_hash,
    explainability_type = EXCLUDED.explainability_type,
    explanation_template = EXCLUDED.explanation_template,
    training_metrics = EXCLUDED.training_metrics,
    promoted_to_production_at = EXCLUDED.promoted_to_production_at,
    promoted_by = EXCLUDED.promoted_by,
    updated_at = now();

WITH models AS (
    SELECT *
    FROM (
        VALUES
            (1, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-cyber-vciso-llm'), uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-cyber-vciso-llm-v1'), uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-cyber-vciso-llm-v2'), 'cyber', 'vciso_chat', 'alert'),
            (2, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-cyber-vciso-predictive'), uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-cyber-vciso-predictive-v1'), uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-cyber-vciso-predictive-v2'), 'cyber', 'forecasting', 'asset'),
            (3, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-data-quality-scorer'), uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-data-quality-scorer-v1'), uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-data-quality-scorer-v2'), 'data', 'quality_scoring', 'data_model')
    ) AS t(slot, model_id, production_version_id, shadow_version_id, suite, use_case, entity_type)
)
INSERT INTO ai_prediction_logs (
    id, tenant_id, model_id, model_version_id, input_hash, input_summary, prediction, confidence,
    explanation_structured, explanation_text, explanation_factors, suite, use_case, entity_type,
    entity_id, is_shadow, shadow_production_version_id, shadow_divergence, latency_ms, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-prediction-log-' || gs),
    '{{ .MainTenantID }}'::uuid,
    m.model_id,
    CASE WHEN gs % 10 = 0 THEN m.shadow_version_id ELSE m.production_version_id END,
    md5('ai-input-' || gs),
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs, 'source', m.use_case),
    jsonb_build_object('score', round(((50 + (gs % 50))::numeric / 100), 4), 'label', CASE WHEN gs % 5 = 0 THEN 'elevated' ELSE 'normal' END),
    round(((60 + (gs % 40))::numeric / 100), 4),
    jsonb_build_object('route', m.use_case, 'agreement', CASE WHEN gs % 10 = 0 THEN false ELSE true END),
    format('Seeded %s prediction %s for %s.', m.use_case, gs, m.suite),
    jsonb_build_array(
        jsonb_build_object('factor', 'volume_change', 'weight', 0.42),
        jsonb_build_object('factor', 'historical_baseline', 'weight', 0.31),
        jsonb_build_object('factor', 'seasonality', 'weight', 0.27)
    ),
    m.suite,
    m.use_case,
    m.entity_type,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'entity-' || gs),
    gs % 10 = 0,
    CASE WHEN gs % 10 = 0 THEN m.production_version_id ELSE NULL END,
    CASE WHEN gs % 10 = 0 THEN jsonb_build_object('delta', round(((gs % 7)::numeric / 100), 4), 'reason', 'shadow disagreement sample') ELSE NULL END,
    80 + (gs % 220),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320))
FROM generate_series(1, {{ .Scale.AIPredictionLogCount }}) gs
JOIN models m ON m.slot = ((gs - 1) % 3) + 1
ON CONFLICT DO NOTHING;

INSERT INTO ai_shadow_comparisons (
    id, tenant_id, model_id, production_version_id, shadow_version_id, period_start, period_end,
    total_predictions, agreement_count, disagreement_count, agreement_rate,
    production_metrics, shadow_metrics, metrics_delta, divergence_samples, divergence_by_use_case,
    recommendation, recommendation_reason, recommendation_factors, created_at
)
VALUES
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'shadow-comparison-vciso-llm'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-cyber-vciso-llm'), uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-cyber-vciso-llm-v1'), uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-cyber-vciso-llm-v2'), now() - interval '14 days', now() - interval '1 day', 4200, 3847, 353, 0.9160, '{"latency_p95_ms":930,"grounding_pass_rate":0.98}', '{"latency_p95_ms":810,"grounding_pass_rate":0.985}', '{"latency_delta_ms":-120,"grounding_delta":0.005}', '[{"case":"insider-risk-summary","delta":"tone shift"}]', '{"vciso_chat":{"agreement_rate":0.916}}', 'keep_shadow', 'Shadow model is faster but still requires more governance review.', '["grounding better","response framing changed"]', now() - interval '1 day'),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'shadow-comparison-predictive'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-cyber-vciso-predictive'), uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-cyber-vciso-predictive-v1'), uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-cyber-vciso-predictive-v2'), now() - interval '14 days', now() - interval '1 day', 2800, 2506, 294, 0.8950, '{"mae":0.07}', '{"mae":0.05}', '{"mae_delta":-0.02}', '[{"case":"campaign-detection","delta":"higher recall"}]', '{"forecasting":{"agreement_rate":0.895}}', 'promote', 'Shadow predictive model materially improves forecast accuracy.', '["lower mae","higher precision"]', now() - interval '1 day')
ON CONFLICT (id) DO NOTHING;

INSERT INTO ai_drift_reports (
    id, tenant_id, model_id, model_version_id, period, period_start, period_end, output_psi,
    output_drift_level, confidence_psi, confidence_drift_level, current_volume, reference_volume,
    volume_change_pct, current_p95_latency_ms, reference_p95_latency_ms, latency_change_pct,
    current_accuracy, reference_accuracy, accuracy_change, alerts, alert_count, created_at
)
VALUES
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'drift-report-vciso-llm'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-cyber-vciso-llm'), uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-cyber-vciso-llm-v1'), '14d', now() - interval '14 days', now(), 0.0810, 'low', 0.0520, 'low', 18240, 17410, 4.77, 930.00, 905.00, 2.76, 0.9810, 0.9780, 0.0030, '["token cost stable","grounding quality healthy"]', 0, now()),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'drift-report-quality-scorer'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-data-quality-scorer'), uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-data-quality-scorer-v1'), '14d', now() - interval '14 days', now(), 0.2310, 'moderate', 0.1420, 'low', 9440, 8800, 7.27, 120.00, 118.00, 1.69, 0.9350, 0.9410, -0.0060, '["quality failures increased in customer golden record domain"]', 1, now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO ai_validation_results (
    id, tenant_id, model_id, version_id, dataset_type, dataset_size, positive_count, negative_count,
    true_positives, false_positives, true_negatives, false_negatives, precision, recall, f1_score,
    false_positive_rate, accuracy, auc, roc_curve, production_metrics, deltas, by_severity,
    by_rule_type, false_positive_samples, false_negative_samples, recommendation,
    recommendation_reason, warnings, validated_at, duration_ms
)
VALUES
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'validation-vciso-llm-v2'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-cyber-vciso-llm'), uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-cyber-vciso-llm-v2'), 'live_replay', 12000, 5300, 6700, 4920, 210, 6490, 380, 0.9591, 0.9283, 0.9435, 0.0313, 0.9508, 0.9721, '[{"fpr":0.0,"tpr":0.0},{"fpr":0.03,"tpr":0.89},{"fpr":1.0,"tpr":1.0}]', '{"grounding_pass_rate":0.985}', '{"precision_delta":0.012}', '{"critical":0.97,"high":0.95}', '{"tool_grounding":0.98}', '[{"sample":"alert summary wording mismatch"}]', '[{"sample":"missed mitigation hint"}]', 'keep_testing', 'Shadow model is promising but still has phrasing drift on critical summaries.', '["shadow disagreement remains above threshold"]', now() - interval '2 days', 14200),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'validation-predictive-v2'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-model-cyber-vciso-predictive'), uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-version-cyber-vciso-predictive-v2'), 'historical', 18000, 8100, 9900, 7560, 320, 9580, 540, 0.9594, 0.9333, 0.9462, 0.0323, 0.9522, 0.9788, '[{"fpr":0.0,"tpr":0.0},{"fpr":0.04,"tpr":0.91},{"fpr":1.0,"tpr":1.0}]', '{"mae":0.07}', '{"mae_delta":-0.02}', '{"high":0.95,"medium":0.94}', '{"forecasting":0.96}', '[]', '[]', 'promote', 'Predictive shadow version outperforms the production baseline on historical replay.', '[]', now() - interval '2 days', 18700)
ON CONFLICT (id) DO NOTHING;

INSERT INTO ai_inference_servers (
    id, tenant_id, name, backend_type, base_url, health_endpoint, model_name,
    quantization, status, cpu_cores, memory_mb, gpu_type, gpu_count, max_concurrent,
    stream_capable, metadata
)
VALUES
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'inference-server-llm-gpu'), '{{ .MainTenantID }}'::uuid, 'Lagos GPU LLM Inference', 'vllm_gpu', 'http://demo-llm-gpu.local:8000', '/health', 'vciso-llm-shadow', 'int4', 'healthy', 16, 65536, 'L4', 1, 128, true, '{"seeded":true,"zone":"lagos-a"}'),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'inference-server-onnx-cpu'), '{{ .MainTenantID }}'::uuid, 'Lagos ONNX CPU Bench', 'onnx_cpu', 'http://demo-onnx.local:8100', '/healthz', 'predictive-cyber-v2', 'fp16', 'healthy', 32, 98304, NULL, 0, 64, false, '{"seeded":true,"zone":"lagos-b"}')
ON CONFLICT (id) DO UPDATE SET
    status = EXCLUDED.status,
    metadata = EXCLUDED.metadata,
    updated_at = now();

INSERT INTO ai_benchmark_suites (
    id, tenant_id, name, description, model_slug, prompt_dataset, dataset_size,
    warmup_count, iteration_count, concurrency, timeout_seconds, stream_enabled,
    max_retries, created_by, created_at, updated_at
)
VALUES
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'benchmark-suite-llm'), '{{ .MainTenantID }}'::uuid, 'Executive vCISO LLM Benchmark', 'Latency and grounding benchmark for executive chat flows.', 'cyber-vciso-llm', '[{"prompt":"summarize executive risk posture"},{"prompt":"explain top unresolved alerts"}]', 2, 5, 120, 8, 60, true, 2, '{{ .MainAdminUserID }}'::uuid, now() - interval '3 days', now()),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'benchmark-suite-predictive'), '{{ .MainTenantID }}'::uuid, 'Predictive Threat Forecast Benchmark', 'Throughput and quality benchmark for forecasting workloads.', 'cyber-vciso-predictive', '[{"prompt":"forecast alert volume 7d"},{"prompt":"forecast campaign likelihood"}]', 2, 5, 100, 4, 60, false, 2, '{{ .MainAdminUserID }}'::uuid, now() - interval '3 days', now())
ON CONFLICT (id) DO UPDATE SET
    description = EXCLUDED.description,
    prompt_dataset = EXCLUDED.prompt_dataset,
    dataset_size = EXCLUDED.dataset_size,
    updated_at = now();

INSERT INTO ai_benchmark_runs (
    id, tenant_id, suite_id, server_id, backend_type, model_name, quantization, status, stream_used,
    p50_latency_ms, p95_latency_ms, p99_latency_ms, avg_latency_ms, min_latency_ms, max_latency_ms,
    tokens_per_second, requests_per_second, total_tokens, total_requests, failed_requests, retried_requests,
    p50_ttft_ms, p95_ttft_ms, avg_ttft_ms, avg_perplexity, semantic_similarity, factual_accuracy,
    peak_cpu_percent, peak_memory_mb, avg_cpu_percent, avg_memory_mb, estimated_hourly_cost_usd,
    cost_per_1k_tokens_usd, started_at, completed_at, duration_seconds, raw_results, created_by, created_at
)
VALUES
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'benchmark-run-llm'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'benchmark-suite-llm'), uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'inference-server-llm-gpu'), 'vllm_gpu', 'vciso-llm-shadow', 'int4', 'completed', true, 640, 910, 1200, 702, 410, 1280, 162.2, 18.6, 542000, 2400, 2, 14, 120, 230, 148, 0.0, 0.96, 0.98, 78.0, 48120, 63.0, 41000, 1.95, 0.013, now() - interval '3 days', now() - interval '3 days' + interval '35 minutes', 2100, '[{"seeded":true}]', '{{ .MainAdminUserID }}'::uuid, now() - interval '3 days'),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'benchmark-run-predictive'), '{{ .MainTenantID }}'::uuid, uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'benchmark-suite-predictive'), uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'inference-server-onnx-cpu'), 'onnx_cpu', 'predictive-cyber-v2', 'fp16', 'completed', false, 85, 122, 148, 94, 64, 169, 318.7, 42.2, 126000, 1600, 0, 5, NULL, NULL, NULL, 2.4, 0.93, 0.95, 61.0, 24500, 48.0, 21200, 0.84, 0.007, now() - interval '2 days', now() - interval '2 days' + interval '18 minutes', 1080, '[{"seeded":true}]', '{{ .MainAdminUserID }}'::uuid, now() - interval '2 days')
ON CONFLICT (id) DO NOTHING;

INSERT INTO ai_compute_cost_models (
    id, tenant_id, name, backend_type, instance_type, hourly_cost_usd,
    cpu_cores, memory_gb, gpu_type, gpu_count, max_tokens_per_second, notes
)
VALUES
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cost-model-vllm-gpu'), '{{ .MainTenantID }}'::uuid, 'GPU LLM Runtime', 'vllm_gpu', 'l4-24gb', 1.95, 16, 64, 'L4', 1, 165.0, 'Seeded benchmark cost model for executive LLM workloads.'),
    (uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cost-model-onnx-cpu'), '{{ .MainTenantID }}'::uuid, 'CPU Forecast Runtime', 'onnx_cpu', 'c32-standard', 0.84, 32, 96, NULL, 0, 320.0, 'Seeded benchmark cost model for predictive scoring.')
ON CONFLICT (id) DO NOTHING;

INSERT INTO workflow_definitions (
    id, tenant_id, name, description, category, version, status, trigger_config,
    variables, steps, created_by, updated_by, created_at, updated_at, published_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'workflow-definition-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('Seeded Workflow %s', lpad(gs::text, 2, '0')),
    format('Demonstration workflow %s spanning approvals, escalations, and evidence capture.', gs),
    CASE WHEN gs % 3 = 0 THEN 'governance' WHEN gs % 3 = 1 THEN 'security' ELSE 'data' END,
    1,
    'active',
    jsonb_build_object('topic', format('demo.workflow.%s', gs), 'type', 'manual'),
    jsonb_build_object('priority', 'normal', 'seeded', true),
    jsonb_build_array(
        jsonb_build_object('id', 'review', 'type', 'human_task', 'name', 'Review request'),
        jsonb_build_object('id', 'approve', 'type', 'decision', 'name', 'Approve change'),
        jsonb_build_object('id', 'notify', 'type', 'notification', 'name', 'Notify stakeholders')
    ),
    '{{ .MainAdminUserID }}'::uuid,
    '{{ .MainAdminUserID }}'::uuid,
    now() - make_interval(days => gs),
    now() - make_interval(days => gs - 1),
    now() - make_interval(days => gs)
FROM generate_series(1, {{ .Scale.WorkflowDefinitionCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    status = EXCLUDED.status,
    trigger_config = EXCLUDED.trigger_config,
    variables = EXCLUDED.variables,
    steps = EXCLUDED.steps,
    updated_by = EXCLUDED.updated_by,
    updated_at = EXCLUDED.updated_at,
    published_at = EXCLUDED.published_at,
    deleted_at = NULL;

INSERT INTO workflow_templates (
    id, name, description, category, definition_json, icon, created_at
)
SELECT
    format('seeded-template-%s', lpad(gs::text, 2, '0')),
    format('Seeded Template %s', lpad(gs::text, 2, '0')),
    format('Reusable seeded workflow template %s.', gs),
    CASE WHEN gs % 2 = 0 THEN 'operations' ELSE 'governance' END,
    jsonb_build_object(
        'name', format('Seeded Template %s', gs),
        'steps', jsonb_build_array(
            jsonb_build_object('id', 'collect', 'type', 'form'),
            jsonb_build_object('id', 'review', 'type', 'human_task'),
            jsonb_build_object('id', 'close', 'type', 'notification')
        )
    ),
    'workflow',
    now() - make_interval(days => gs)
FROM generate_series(1, {{ .Scale.WorkflowTemplateCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    definition_json = EXCLUDED.definition_json,
    icon = EXCLUDED.icon;

INSERT INTO workflow_instances (
    id, tenant_id, definition_id, definition_ver, status, current_step_id,
    variables, step_outputs, trigger_data, error_message, started_by, started_at, completed_at, updated_at, lock_version
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'workflow-instance-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'workflow-definition-' || (((gs - 1) % {{ .Scale.WorkflowDefinitionCount }}) + 1)),
    1,
    CASE
        WHEN gs % 11 = 0 THEN 'failed'
        WHEN gs % 5 = 0 THEN 'completed'
        ELSE 'running'
    END,
    CASE WHEN gs % 5 = 0 THEN 'notify' ELSE 'review' END,
    jsonb_build_object('request_id', gs, 'source', 'demo_seed'),
    jsonb_build_object('review', jsonb_build_object('status', CASE WHEN gs % 5 = 0 THEN 'approved' ELSE 'pending' END)),
    jsonb_build_object('trigger', 'manual', 'seed_key', '{{ .SeedKey }}'),
    CASE WHEN gs % 11 = 0 THEN 'Seeded approval path failed at verification step.' ELSE NULL END,
    CASE ((gs - 1) % 7) + 1
        WHEN 1 THEN '{{ .MainAdminUserID }}'::uuid
        WHEN 2 THEN '{{ .SecurityManagerUserID }}'::uuid
        WHEN 3 THEN '{{ .DataStewardUserID }}'::uuid
        WHEN 4 THEN '{{ .LegalManagerUserID }}'::uuid
        WHEN 5 THEN '{{ .BoardSecretaryUserID }}'::uuid
        WHEN 6 THEN '{{ .ExecutiveUserID }}'::uuid
        ELSE '{{ .AuditorUserID }}'::uuid
    END,
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320)),
    CASE WHEN gs % 5 = 0 THEN date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 180) ELSE NULL END,
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 10),
    0
FROM generate_series(1, {{ .Scale.WorkflowInstanceCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    status = EXCLUDED.status,
    current_step_id = EXCLUDED.current_step_id,
    variables = EXCLUDED.variables,
    step_outputs = EXCLUDED.step_outputs,
    trigger_data = EXCLUDED.trigger_data,
    error_message = EXCLUDED.error_message,
    completed_at = EXCLUDED.completed_at,
    updated_at = EXCLUDED.updated_at;

INSERT INTO workflow_step_executions (
    id, instance_id, step_id, step_type, status, input_data, output_data,
    error_message, attempt, started_at, completed_at, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'workflow-step-execution-' || gs),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'workflow-instance-' || gs),
    CASE WHEN gs % 5 = 0 THEN 'notify' ELSE 'review' END,
    CASE WHEN gs % 5 = 0 THEN 'notification' ELSE 'human_task' END,
    CASE
        WHEN gs % 11 = 0 THEN 'failed'
        WHEN gs % 5 = 0 THEN 'completed'
        ELSE 'running'
    END,
    jsonb_build_object('sequence', gs, 'seeded', true),
    CASE WHEN gs % 5 = 0 THEN jsonb_build_object('result', 'approved') ELSE NULL END,
    CASE WHEN gs % 11 = 0 THEN 'Seeded step failure for escalation path.' ELSE NULL END,
    1 + (gs % 2),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320)),
    CASE WHEN gs % 5 = 0 THEN date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 60) ELSE NULL END,
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320))
FROM generate_series(1, {{ .Scale.WorkflowInstanceCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    status = EXCLUDED.status,
    output_data = EXCLUDED.output_data,
    error_message = EXCLUDED.error_message,
    completed_at = EXCLUDED.completed_at;

INSERT INTO workflow_tasks (
    id, tenant_id, instance_id, step_id, step_exec_id, name, description, status,
    assignee_id, assignee_role, claimed_by, claimed_at, form_schema, form_data,
    sla_deadline, sla_breached, escalated_to, escalation_role, delegated_by, delegated_at,
    priority, metadata, completed_at, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'workflow-task-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'workflow-instance-' || (((gs - 1) % {{ .Scale.WorkflowInstanceCount }}) + 1)),
    'review',
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'workflow-step-execution-' || (((gs - 1) % {{ .Scale.WorkflowInstanceCount }}) + 1)),
    format('Seeded Task %s', lpad(gs::text, 6, '0')),
    'Seeded human approval task for workflow dashboard demonstrations.',
    CASE
        WHEN gs % 9 = 0 THEN 'claimed'
        WHEN gs % 5 = 0 THEN 'completed'
        WHEN gs % 13 = 0 THEN 'escalated'
        ELSE 'pending'
    END,
    CASE ((gs - 1) % 7) + 1
        WHEN 1 THEN '{{ .MainAdminUserID }}'::uuid
        WHEN 2 THEN '{{ .SecurityManagerUserID }}'::uuid
        WHEN 3 THEN '{{ .DataStewardUserID }}'::uuid
        WHEN 4 THEN '{{ .LegalManagerUserID }}'::uuid
        WHEN 5 THEN '{{ .BoardSecretaryUserID }}'::uuid
        WHEN 6 THEN '{{ .ExecutiveUserID }}'::uuid
        ELSE '{{ .AuditorUserID }}'::uuid
    END,
    CASE ((gs - 1) % 6)
        WHEN 0 THEN 'tenant-admin'
        WHEN 1 THEN 'security-manager'
        WHEN 2 THEN 'data-steward'
        WHEN 3 THEN 'legal-manager'
        WHEN 4 THEN 'board-secretary'
        ELSE 'executive'
    END,
    CASE WHEN gs % 9 = 0 THEN '{{ .MainAdminUserID }}'::uuid ELSE NULL END,
    CASE WHEN gs % 9 = 0 THEN date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 20) ELSE NULL END,
    '[{"name":"decision","type":"select","options":["approve","reject"]}]'::jsonb,
    CASE WHEN gs % 5 = 0 THEN '{"decision":"approve","notes":"seeded completion"}'::jsonb ELSE NULL END,
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 240),
    gs % 17 = 0,
    CASE WHEN gs % 13 = 0 THEN '{{ .ExecutiveUserID }}'::uuid ELSE NULL END,
    CASE WHEN gs % 13 = 0 THEN 'executive' ELSE NULL END,
    CASE WHEN gs % 19 = 0 THEN '{{ .SecurityManagerUserID }}'::uuid ELSE NULL END,
    CASE WHEN gs % 19 = 0 THEN date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 35) ELSE NULL END,
    (gs % 5),
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs),
    CASE WHEN gs % 5 = 0 THEN date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 90) ELSE NULL END,
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320)),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 15)
FROM generate_series(1, {{ .Scale.WorkflowTaskCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    status = EXCLUDED.status,
    assignee_id = EXCLUDED.assignee_id,
    claimed_by = EXCLUDED.claimed_by,
    claimed_at = EXCLUDED.claimed_at,
    form_data = EXCLUDED.form_data,
    sla_deadline = EXCLUDED.sla_deadline,
    sla_breached = EXCLUDED.sla_breached,
    escalated_to = EXCLUDED.escalated_to,
    escalation_role = EXCLUDED.escalation_role,
    delegated_by = EXCLUDED.delegated_by,
    delegated_at = EXCLUDED.delegated_at,
    metadata = EXCLUDED.metadata,
    completed_at = EXCLUDED.completed_at,
    updated_at = EXCLUDED.updated_at;

INSERT INTO workflow_timers (
    id, instance_id, step_id, fire_at, fired, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'workflow-timer-' || gs),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'workflow-instance-' || (((gs - 1) % {{ .Scale.WorkflowInstanceCount }}) + 1)),
    'notify',
    now() + make_interval(hours => gs),
    false,
    now()
FROM generate_series(1, LEAST({{ .Scale.WorkflowInstanceCount }}, 256)) gs
ON CONFLICT (id) DO UPDATE SET
    fire_at = EXCLUDED.fire_at,
    fired = EXCLUDED.fired;
