ALTER TABLE ai_models DROP CONSTRAINT IF EXISTS ai_models_model_type_check;
ALTER TABLE ai_models
    ADD CONSTRAINT ai_models_model_type_check
    CHECK (model_type IN (
        'rule_based', 'statistical', 'ml_classifier', 'ml_regressor',
        'nlp_extractor', 'anomaly_detector', 'scorer', 'recommender',
        'llm_agentic'
    ));

ALTER TABLE ai_model_versions DROP CONSTRAINT IF EXISTS ai_model_versions_explainability_type_check;
ALTER TABLE ai_model_versions
    ADD CONSTRAINT ai_model_versions_explainability_type_check
    CHECK (explainability_type IN (
        'rule_trace', 'feature_importance', 'statistical_deviation',
        'template_based', 'reasoning_trace'
    ));
