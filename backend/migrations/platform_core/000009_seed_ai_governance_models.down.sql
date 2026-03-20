-- Remove seeded AI governance models and their versions across ALL tenants.
-- Deletes by slug (the stable identifier) rather than hardcoded UUIDs.

DO $$
DECLARE
  v_rec RECORD;
  v_slugs TEXT[] := ARRAY[
    'cyber-sigma-evaluator',
    'cyber-anomaly-detector',
    'cyber-risk-scorer',
    'cyber-ueba-detector',
    'cyber-vciso-classifier',
    'cyber-vciso-llm',
    'cyber-vciso-predictive',
    'cyber-asset-classifier',
    'cyber-ctem-prioritizer',
    'data-pii-classifier',
    'data-contradiction-detector',
    'data-quality-scorer',
    'acta-minutes-generator',
    'acta-action-extractor',
    'lex-clause-extractor',
    'lex-risk-analyzer',
    'visus-kpi-monitor',
    'visus-recommendation-engine'
  ];
BEGIN
  FOR v_rec IN SELECT id FROM tenants LOOP
    PERFORM set_config('app.current_tenant_id', v_rec.id::text, true);

    -- Versions cascade from models, but explicit delete avoids relying on FK cascade.
    DELETE FROM ai_model_versions
    WHERE tenant_id = v_rec.id
      AND model_id IN (SELECT id FROM ai_models WHERE tenant_id = v_rec.id AND slug = ANY(v_slugs));

    DELETE FROM ai_models
    WHERE tenant_id = v_rec.id
      AND slug = ANY(v_slugs);
  END LOOP;
END $$;
