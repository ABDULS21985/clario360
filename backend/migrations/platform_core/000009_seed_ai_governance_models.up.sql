-- Seed AI governance models and production versions for EVERY tenant.
-- Loops over all tenants so each one gets the full model catalogue.
-- Uses set_config per iteration so RLS policies are satisfied.

DO $$
DECLARE
  v_rec   RECORD;
  v_actor UUID := 'aaaaaaaa-0000-0000-0000-aaaaaaaaaaaa';  -- system actor
BEGIN
  FOR v_rec IN SELECT id FROM tenants LOOP
    PERFORM set_config('app.current_tenant_id', v_rec.id::text, true);

    -- ── Models ──────────────────────────────────────────────────────────

    INSERT INTO ai_models (tenant_id, name, slug, description, model_type, suite, owner_team, risk_tier, status, tags, metadata, created_by)
    VALUES
      (v_rec.id, 'Threat Detection - Sigma Rule Evaluator',   'cyber-sigma-evaluator',       'Deterministic Sigma-style rule evaluation for security events.',                                    'rule_based',       'cyber', 'security-operations',         'critical', 'active', ARRAY['cyber','detection','sigma'],                    '{"used_by":"detection_engine"}'::jsonb,    v_actor),
      (v_rec.id, 'Anomaly Detection - Statistical Baseline',   'cyber-anomaly-detector',      'Statistical deviation detector using baseline windows and z-score thresholds.',                     'anomaly_detector', 'cyber', 'security-operations',         'high',     'active', ARRAY['cyber','anomaly','baseline'],                   '{"used_by":"detection_engine"}'::jsonb,    v_actor),
      (v_rec.id, 'Risk Scoring - Multi-Factor Composite',      'cyber-risk-scorer',           'Weighted transparent scoring model for cyber organizational risk.',                                 'scorer',           'cyber', 'security-risk',               'high',     'active', ARRAY['cyber','risk','scorer'],                        '{"used_by":"risk_service"}'::jsonb,        v_actor),
      (v_rec.id, 'UEBA Behavioral Anomaly Detector',           'cyber-ueba-detector',         'Profiles user and entity behavior over time using exponential moving averages.',                    'anomaly_detector', 'cyber', 'security-operations',         'high',     'active', ARRAY['cyber','ueba','anomaly','behavioral'],          '{"used_by":"ueba_engine"}'::jsonb,         v_actor),
      (v_rec.id, 'Virtual CISO Intent Classifier',             'cyber-vciso-classifier',      'Classifies natural language security queries into intents using regex and keyword matching.',        'rule_based',       'cyber', 'security-operations',         'medium',   'active', ARRAY['cyber','vciso','chat','rule-based'],            '{"used_by":"vciso_chat_engine"}'::jsonb,   v_actor),
      (v_rec.id, 'Virtual CISO LLM Engine',                    'cyber-vciso-llm',             'LLM-powered conversational AI for complex security queries with function calling and grounded responses.', 'llm_agentic', 'cyber', 'security-operations',         'high',     'active', ARRAY['cyber','vciso','chat','llm','agentic'],         '{"used_by":"vciso_llm_engine"}'::jsonb,    v_actor),
      (v_rec.id, 'vCISO Predictive Threat Engine',             'cyber-vciso-predictive',      'Six explainable predictive security models for forecasting threats and risks.',                     'ml_classifier',    'cyber', 'security-operations',         'high',     'active', ARRAY['cyber','vciso','predictive','forecasting'],     '{"used_by":"vciso_predict_engine"}'::jsonb, v_actor),
      (v_rec.id, 'Asset Auto-Classifier',                      'cyber-asset-classifier',      'Rule-based asset criticality classifier.',                                                         'rule_based',       'cyber', 'security-asset-management',   'medium',   'active', ARRAY['cyber','asset','classification'],               '{"used_by":"asset_service"}'::jsonb,       v_actor),
      (v_rec.id, 'CTEM Prioritization',                        'cyber-ctem-prioritizer',      'Weighted prioritization model for CTEM findings.',                                                 'scorer',           'cyber', 'security-exposure-management','high',     'active', ARRAY['cyber','ctem','prioritization'],                '{"used_by":"ctem_engine"}'::jsonb,         v_actor),
      (v_rec.id, 'PII Classifier',                             'data-pii-classifier',         'Deterministic PII classifier for schema discovery.',                                               'rule_based',       'data',  'data-governance',             'high',     'active', ARRAY['data','pii','classification'],                  '{"used_by":"schema_discovery"}'::jsonb,    v_actor),
      (v_rec.id, 'Contradiction Detector',                     'data-contradiction-detector', 'Transparent logical contradiction detector across sources.',                                        'rule_based',       'data',  'data-governance',             'medium',   'active', ARRAY['data','quality','contradiction'],               '{"used_by":"contradiction_service"}'::jsonb, v_actor),
      (v_rec.id, 'Data Quality Scorer',                        'data-quality-scorer',         'Transparent weighted scorer for enterprise data quality.',                                          'scorer',           'data',  'data-quality',                'medium',   'active', ARRAY['data','quality','scorer'],                      '{"used_by":"quality_service"}'::jsonb,     v_actor),
      (v_rec.id, 'Meeting Minutes Generator',                  'acta-minutes-generator',      'Template-driven meeting minutes generation.',                                                      'nlp_extractor',    'acta',  'governance-operations',       'medium',   'active', ARRAY['acta','minutes','template'],                    '{"used_by":"minutes_service"}'::jsonb,     v_actor),
      (v_rec.id, 'Action Item Extractor',                      'acta-action-extractor',       'Pattern-based action item extraction from meeting notes.',                                         'nlp_extractor',    'acta',  'governance-operations',       'low',      'active', ARRAY['acta','actions','extractor'],                   '{"used_by":"minutes_generator"}'::jsonb,   v_actor),
      (v_rec.id, 'Contract Clause Extractor',                  'lex-clause-extractor',        'Pattern-driven clause extractor for legal documents.',                                             'rule_based',       'lex',   'legal-operations',            'high',     'active', ARRAY['lex','clause','extractor'],                     '{"used_by":"contract_analyzer"}'::jsonb,   v_actor),
      (v_rec.id, 'Contract Risk Analyzer',                     'lex-risk-analyzer',           'Transparent weighted risk analysis for contracts.',                                                'scorer',           'lex',   'legal-operations',            'high',     'active', ARRAY['lex','risk','analysis'],                        '{"used_by":"contract_analyzer"}'::jsonb,   v_actor),
      (v_rec.id, 'KPI Threshold Monitor',                      'visus-kpi-monitor',           'Threshold evaluator for KPI status transitions.',                                                 'statistical',      'visus', 'executive-reporting',         'medium',   'active', ARRAY['visus','kpi','monitor'],                        '{"used_by":"kpi_engine"}'::jsonb,          v_actor),
      (v_rec.id, 'Executive Recommendation Engine',            'visus-recommendation-engine', 'Rule-based recommendation engine for executive reporting.',                                        'recommender',      'visus', 'executive-reporting',         'low',      'active', ARRAY['visus','recommendation','reporting'],           '{"used_by":"report_generator"}'::jsonb,    v_actor)
    ON CONFLICT (tenant_id, slug) WHERE deleted_at IS NULL DO NOTHING;

    -- ── Production Versions ─────────────────────────────────────────────
    -- Join back on slug to pick up the generated model IDs.

    INSERT INTO ai_model_versions (tenant_id, model_id, version_number, status, description, artifact_type, artifact_config, artifact_hash, explainability_type, training_metrics, created_by, promoted_to_production_at, promoted_by)
    SELECT v_rec.id, m.id, 1, 'production', 'Initial seeded production version',
           v.artifact_type, v.artifact_config, v.artifact_hash, v.explainability_type,
           '{}'::jsonb, v_actor, NOW(), v_actor
    FROM (VALUES
      ('cyber-sigma-evaluator',       'rule_set',           '{"engine":"sigma","supports":["selection","condition","timeframe"]}'::jsonb,                                                                                           'seed-sigma-v1',          'rule_trace'),
      ('cyber-anomaly-detector',      'statistical_config', '{"window":"7d","threshold_z":3.0}'::jsonb,                                                                                                                            'seed-anomaly-v1',        'statistical_deviation'),
      ('cyber-risk-scorer',           'statistical_config', '{"components":{"vulnerability":0.30,"threat":0.25,"configuration":0.20,"surface":0.15,"compliance":0.10}}'::jsonb,                                                     'seed-risk-v1',           'feature_importance'),
      ('cyber-ueba-detector',         'statistical_config', '{"baseline_strategy":"ema_welford","signals":["time","volume","table_access","ip","failures","bulk_access","privilege"],"correlation_window":"1h"}'::jsonb,              'seed-ueba-v1',           'statistical_deviation'),
      ('cyber-vciso-classifier',      'rule_set',           '{"strategy":"regex_then_keyword","intent_count":19,"entities":["alert_id","asset_name","asset_ip","time_range","severity","count","framework","description"]}'::jsonb,   'seed-vciso-v1',          'rule_trace'),
      ('cyber-vciso-llm',             'template_config',    '{"providers":["openai","anthropic","azure","local"],"tool_count":25,"guardrails":["grounding","pii_filter","prompt_injection","rate_limits"]}'::jsonb,                   'seed-llm-v1',            'reasoning_trace'),
      ('cyber-vciso-predictive',      'serialized_model',   '{"engine_type":"ml_ensemble","models":["alert_volume","asset_risk","vulnerability_exploit","technique_trend","insider_trajectory","campaign_detection"]}'::jsonb,        'seed-predict-v1',        'feature_importance'),
      ('cyber-asset-classifier',      'rule_set',           '{"strategy":"priority_first_match"}'::jsonb,                                                                                                                           'seed-asset-v1',          'rule_trace'),
      ('cyber-ctem-prioritizer',      'statistical_config', '{"weights":{"impact":0.55,"exploitability":0.45}}'::jsonb,                                                                                                             'seed-ctem-v1',           'feature_importance'),
      ('data-pii-classifier',         'rule_set',           '{"sources":["column_name","sample_value_patterns"]}'::jsonb,                                                                                                           'seed-pii-v1',            'rule_trace'),
      ('data-contradiction-detector', 'rule_set',           '{"strategies":["logical","semantic","temporal","analytical"]}'::jsonb,                                                                                                  'seed-contradiction-v1',  'rule_trace'),
      ('data-quality-scorer',         'statistical_config', '{"weights":{"critical":4,"high":3,"medium":2,"low":1}}'::jsonb,                                                                                                        'seed-quality-v1',        'feature_importance'),
      ('acta-minutes-generator',      'template_config',    '{"template":"minutes_markdown","summary_builder":"deterministic"}'::jsonb,                                                                                              'seed-minutes-v1',        'template_based'),
      ('acta-action-extractor',       'rule_set',           '{"patterns":["ACTION:","will","agreed that"]}'::jsonb,                                                                                                                 'seed-action-v1',         'rule_trace'),
      ('lex-clause-extractor',        'rule_set',           '{"clause_catalog_size":19}'::jsonb,                                                                                                                                    'seed-clause-v1',         'rule_trace'),
      ('lex-risk-analyzer',           'statistical_config', '{"factors":["clause_risk","missing_clause","value","expiry","compliance"]}'::jsonb,                                                                                     'seed-risk-analyzer-v1',  'feature_importance'),
      ('visus-kpi-monitor',           'statistical_config', '{"logic":"directional_threshold"}'::jsonb,                                                                                                                             'seed-kpi-v1',            'statistical_deviation'),
      ('visus-recommendation-engine', 'rule_set',           '{"triggers":["critical_kpi","overdue_action","expiring_contract","coverage_gap"]}'::jsonb,                                                                              'seed-rec-v1',            'rule_trace')
    ) AS v(slug, artifact_type, artifact_config, artifact_hash, explainability_type)
    JOIN ai_models m ON m.tenant_id = v_rec.id AND m.slug = v.slug AND m.deleted_at IS NULL
    WHERE NOT EXISTS (
      SELECT 1 FROM ai_model_versions mv WHERE mv.model_id = m.id AND mv.version_number = 1
    );

  END LOOP;
END $$;
