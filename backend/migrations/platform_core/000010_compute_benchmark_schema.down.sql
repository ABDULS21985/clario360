-- Rollback: remove all AI benchmark infrastructure tables.

DROP TABLE IF EXISTS ai_compute_cost_models CASCADE;
DROP TABLE IF EXISTS ai_benchmark_runs CASCADE;
DROP TABLE IF EXISTS ai_benchmark_suites CASCADE;
DROP TABLE IF EXISTS ai_inference_servers CASCADE;

-- Restore original artifact_type constraint.
ALTER TABLE ai_model_versions DROP CONSTRAINT IF EXISTS ai_model_versions_artifact_type_check;
ALTER TABLE ai_model_versions
    ADD CONSTRAINT ai_model_versions_artifact_type_check
    CHECK (artifact_type IN (
        'go_function', 'rule_set', 'statistical_config', 'template_config',
        'serialized_model'
    ));