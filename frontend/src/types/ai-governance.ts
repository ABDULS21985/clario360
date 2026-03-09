export type JsonValue =
  | string
  | number
  | boolean
  | null
  | JsonValue[]
  | { [key: string]: JsonValue };

export type AIModelType =
  | 'rule_based'
  | 'statistical'
  | 'ml_classifier'
  | 'ml_regressor'
  | 'nlp_extractor'
  | 'anomaly_detector'
  | 'scorer'
  | 'recommender';

export type AIModelSuite = 'cyber' | 'data' | 'acta' | 'lex' | 'visus' | 'platform';

export type AIRiskTier = 'low' | 'medium' | 'high' | 'critical';

export type AIModelStatus = 'active' | 'deprecated' | 'retired';

export type AIVersionStatus =
  | 'development'
  | 'staging'
  | 'shadow'
  | 'production'
  | 'retired'
  | 'failed'
  | 'rolled_back';

export type AIExplainabilityType =
  | 'rule_trace'
  | 'feature_importance'
  | 'statistical_deviation'
  | 'template_based';

export type AIArtifactType =
  | 'go_function'
  | 'rule_set'
  | 'statistical_config'
  | 'template_config'
  | 'serialized_model';

export type AIDriftLevel = 'none' | 'low' | 'moderate' | 'significant';

export type AIShadowRecommendation = 'promote' | 'keep_shadow' | 'reject' | 'needs_review';
export type AIValidationDatasetType = 'historical' | 'custom' | 'live_replay';
export type AIValidationRecommendation = 'promote' | 'keep_testing' | 'reject';
export type AIValidationLabel = 'threat' | 'benign';

export interface AIModelVersion {
  id: string;
  tenant_id: string;
  model_id: string;
  model_slug?: string;
  model_name?: string;
  model_type?: AIModelType;
  suite?: AIModelSuite;
  risk_tier?: AIRiskTier;
  version_number: number;
  status: AIVersionStatus;
  description: string;
  artifact_type: AIArtifactType;
  artifact_config: JsonValue;
  artifact_hash: string;
  explainability_type: AIExplainabilityType;
  explanation_template?: string | null;
  training_data_desc?: string | null;
  training_data_hash?: string | null;
  training_metrics: JsonValue;
  prediction_count: number;
  avg_latency_ms?: number | null;
  avg_confidence?: number | null;
  accuracy_metric?: number | null;
  false_positive_rate?: number | null;
  false_negative_rate?: number | null;
  feedback_count: number;
  promoted_to_staging_at?: string | null;
  promoted_to_shadow_at?: string | null;
  promoted_to_production_at?: string | null;
  promoted_by?: string | null;
  retired_at?: string | null;
  retired_by?: string | null;
  retirement_reason?: string | null;
  rolled_back_at?: string | null;
  rolled_back_by?: string | null;
  rollback_reason?: string | null;
  failed_at?: string | null;
  failed_by?: string | null;
  failed_from_status?: AIVersionStatus | null;
  failure_reason?: string | null;
  replaced_version_id?: string | null;
  created_by: string;
  created_at: string;
  updated_at: string;
  latest_shadow_comparison_id?: string | null;
  latest_shadow_recommendation?: AIShadowRecommendation | null;
}

export interface AIRegisteredModel {
  id: string;
  tenant_id: string;
  name: string;
  slug: string;
  description: string;
  model_type: AIModelType;
  suite: AIModelSuite;
  owner_user_id?: string | null;
  owner_team?: string;
  risk_tier: AIRiskTier;
  status: AIModelStatus;
  tags: string[];
  metadata: JsonValue;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface AIRegisterModelPayload {
  name: string;
  slug: string;
  description: string;
  model_type: AIModelType;
  suite: AIModelSuite;
  owner_user_id?: string;
  owner_team?: string;
  risk_tier: AIRiskTier;
  tags: string[];
  metadata: JsonValue;
}

export interface AIUpdateModelPayload {
  name?: string;
  description?: string;
  owner_user_id?: string;
  owner_team?: string;
  risk_tier?: AIRiskTier;
  status?: AIModelStatus;
  tags?: string[];
  metadata?: JsonValue;
}

export interface AICreateVersionPayload {
  description: string;
  artifact_type: AIArtifactType;
  artifact_config: JsonValue;
  explainability_type: AIExplainabilityType;
  explanation_template?: string | null;
  training_data_desc?: string | null;
  training_data_hash?: string | null;
  training_metrics: JsonValue;
}

export interface AIModelWithVersions {
  model: AIRegisteredModel;
  production_version?: AIModelVersion | null;
  shadow_version?: AIModelVersion | null;
}

export interface AIExplanationFactor {
  name: string;
  value: string;
  impact: number;
  direction: 'positive' | 'negative';
  description: string;
}

export interface AIExplanation {
  structured: Record<string, JsonValue>;
  human_readable: string;
  factors: AIExplanationFactor[];
  confidence: number;
  explainer_type?: string;
  model_slug?: string;
  model_version?: number;
}

export interface AIPredictionLog {
  id: string;
  tenant_id: string;
  model_id: string;
  model_version_id: string;
  model_slug?: string;
  model_version_number?: number;
  input_hash: string;
  input_summary?: JsonValue;
  prediction: JsonValue;
  confidence?: number | null;
  explanation_structured: JsonValue;
  explanation_text: string;
  explanation_factors: AIExplanationFactor[];
  suite: string;
  use_case: string;
  entity_type?: string;
  entity_id?: string | null;
  is_shadow: boolean;
  shadow_production_version_id?: string | null;
  shadow_divergence?: JsonValue;
  feedback_correct?: boolean | null;
  feedback_by?: string | null;
  feedback_at?: string | null;
  feedback_notes?: string | null;
  feedback_corrected_output?: JsonValue;
  latency_ms: number;
  created_at: string;
}

export interface AIPredictionStats {
  model_id: string;
  model_slug: string;
  suite: string;
  use_case: string;
  total: number;
  shadow_total: number;
  avg_confidence?: number | null;
  avg_latency_ms?: number | null;
  correct_feedback: number;
  wrong_feedback: number;
}

export interface AIShadowDivergence {
  prediction_id: string;
  input_hash: string;
  use_case: string;
  entity_id?: string | null;
  production_output: JsonValue;
  shadow_output: JsonValue;
  production_confidence?: number | null;
  shadow_confidence?: number | null;
  reason: string;
  created_at: string;
}

export interface AIShadowComparison {
  id: string;
  tenant_id: string;
  model_id: string;
  production_version_id: string;
  shadow_version_id: string;
  period_start: string;
  period_end: string;
  total_predictions: number;
  agreement_count: number;
  disagreement_count: number;
  agreement_rate: number;
  production_metrics: JsonValue;
  shadow_metrics: JsonValue;
  metrics_delta: JsonValue;
  divergence_samples: AIShadowDivergence[];
  divergence_by_use_case: JsonValue;
  recommendation: AIShadowRecommendation;
  recommendation_reason: string;
  recommendation_factors: JsonValue;
  created_at: string;
}

export interface AIDriftAlert {
  type: string;
  severity: string;
  message: string;
  recommended?: string;
}

export interface AIDriftReport {
  id: string;
  tenant_id: string;
  model_id: string;
  model_version_id: string;
  model_slug?: string;
  period: string;
  period_start: string;
  period_end: string;
  output_psi?: number | null;
  output_drift_level?: AIDriftLevel;
  confidence_psi?: number | null;
  confidence_drift_level?: AIDriftLevel;
  current_volume: number;
  reference_volume: number;
  volume_change_pct?: number | null;
  current_p95_latency_ms?: number | null;
  reference_p95_latency_ms?: number | null;
  latency_change_pct?: number | null;
  current_accuracy?: number | null;
  reference_accuracy?: number | null;
  accuracy_change?: number | null;
  alerts: AIDriftAlert[];
  alert_count: number;
  created_at: string;
}

export interface AIPerformancePoint {
  period_start: string;
  volume: number;
  avg_latency_ms?: number | null;
  accuracy?: number | null;
}

export interface AILifecycleHistoryEntry {
  version_id: string;
  version_number: number;
  from_status?: AIVersionStatus | null;
  to_status: AIVersionStatus;
  changed_by?: string | null;
  reason?: string;
  changed_at: string;
}

export interface AIDashboardKPI {
  total_models: number;
  in_production: number;
  shadow_testing: number;
  predictions_24h: number;
  drift_alerts: number;
}

export interface AIDashboardModelRow {
  id: string;
  name: string;
  slug: string;
  suite: AIModelSuite;
  type: AIModelType;
  risk_tier: AIRiskTier;
  status: AIModelStatus;
  production_version?: AIModelVersion | null;
  shadow_version?: AIModelVersion | null;
  predictions_24h: number;
  avg_confidence?: number | null;
  drift_status: AIDriftLevel;
}

export interface AIDashboardData {
  kpis: AIDashboardKPI;
  models: AIDashboardModelRow[];
}

export interface AIValidationMetricsSummary {
  dataset_size: number;
  positive_count: number;
  negative_count: number;
  true_positives: number;
  false_positives: number;
  true_negatives: number;
  false_negatives: number;
  precision: number;
  recall: number;
  f1_score: number;
  false_positive_rate: number;
  false_negative_rate?: number;
  accuracy: number;
  auc?: number;
}

export interface AIROCPoint {
  threshold: number;
  fpr: number;
  tpr: number;
}

export interface AIValidationPredictionSample {
  prediction_id?: string | null;
  input_hash: string;
  input_summary: JsonValue;
  predicted_output: JsonValue;
  predicted_label: AIValidationLabel;
  expected_label: AIValidationLabel;
  confidence: number;
  severity?: string;
  rule_type?: string;
  explanation: string;
}

export interface AIValidationResult {
  id: string;
  version_id: string;
  dataset_type: AIValidationDatasetType;
  dataset_size: number;
  positive_count: number;
  negative_count: number;
  true_positives: number;
  false_positives: number;
  true_negatives: number;
  false_negatives: number;
  precision: number;
  recall: number;
  f1_score: number;
  false_positive_rate: number;
  accuracy: number;
  auc: number;
  roc_curve: AIROCPoint[];
  production_metrics?: AIValidationMetricsSummary | null;
  deltas?: Record<string, number> | null;
  by_severity: Record<string, AIValidationMetricsSummary>;
  by_rule_type?: Record<string, AIValidationMetricsSummary>;
  false_positive_samples: AIValidationPredictionSample[];
  false_negative_samples: AIValidationPredictionSample[];
  recommendation: AIValidationRecommendation;
  recommendation_reason: string;
  warnings: string[];
  validated_at: string;
  duration_ms: number;
}

export interface AIValidationPreview {
  dataset_type: AIValidationDatasetType;
  dataset_size: number;
  positive_count: number;
  negative_count: number;
  warnings: string[];
}
