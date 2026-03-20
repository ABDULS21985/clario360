import type { PaginationMeta } from '@/types/api';

export interface UebaRiskFactor {
  alert_id: string;
  alert_type: string;
  severity: string;
  confidence: number;
  impact: number;
  description: string;
  created_at: string;
  signal_types: string[];
  event_count: number;
}

export interface UebaFrequencyEntry {
  name: string;
  frequency: number;
  last_accessed: string;
}

export interface UebaBaseline {
  access_times: {
    peak_hours: number[];
    active_hours_count: number;
  };
  data_volume: {
    daily_bytes_mean: number;
    daily_rows_mean: number;
    max_single_query_bytes: number;
    max_single_query_rows: number;
  };
  access_patterns: {
    databases_accessed: string[];
    tables_accessed: UebaFrequencyEntry[];
    query_types: Record<string, number>;
  };
  source_ips: string[];
  failure_rate: {
    failure_rate_percent: number;
  };
}

export interface UebaProfile {
  id: string;
  entity_id: string;
  entity_name?: string;
  entity_type: string;
  risk_score: number;
  risk_level: string;
  alert_count_7d: number;
  alert_count_30d: number;
  profile_maturity: string;
  status: string;
  last_seen_at: string;
  baseline: UebaBaseline;
  risk_factors: UebaRiskFactor[];
}

export interface UebaSignal {
  signal_type: string;
  title: string;
  description: string;
  severity: string;
  confidence: number;
  deviation_z: number;
  expected_value: string;
  actual_value: string;
  event_id: string;
  mitre_technique: string;
}

export interface UebaAlert {
  id: string;
  entity_id: string;
  entity_name?: string;
  entity_type: string;
  alert_type: string;
  severity: string;
  confidence: number;
  risk_score_before: number;
  risk_score_after: number;
  risk_score_delta: number;
  title: string;
  description: string;
  triggering_signals: UebaSignal[];
  triggering_event_ids: string[];
  baseline_comparison: Record<string, unknown>;
  correlated_signal_count: number;
  correlation_window_start: string;
  correlation_window_end: string;
  mitre_technique_ids: string[];
  mitre_tactic?: string;
  status: string;
  created_at: string;
}

export interface UebaTrendDatum {
  bucket: string;
  alert_type: string;
  count: number;
}

export interface UebaChartDatum {
  label: string;
  value: number;
}

export interface UebaRiskRankingItem {
  entity_id: string;
  entity_name: string;
  entity_type: string;
  risk_score: number;
  risk_level: string;
  alert_count_7d: number;
  alert_count_30d: number;
  profile_maturity: string;
  last_seen_at: string;
  status: string;
}

export interface UebaDashboard {
  kpis: {
    active_profiles: number;
    high_risk_entities: number;
    alerts_7d: number;
    average_risk_score: number;
  };
  risk_ranking: UebaRiskRankingItem[];
  alert_type_distribution: UebaChartDatum[];
  alert_trend: UebaTrendDatum[];
  profiles: UebaRiskRankingItem[];
}

export interface UebaProfileDetailResponse {
  profile: UebaProfile;
  baseline_comparison: Record<string, unknown>;
  risk_history: Array<{
    timestamp: string;
    score: number;
    alert_id?: string;
    severity?: string;
    alert_type?: string;
  }>;
}

export interface UebaHeatmapResponse {
  entity_id: string;
  days: number;
  matrix: number[][];
}

export interface UebaTimelineEvent {
  id: string;
  action: string;
  source_type: string;
  database_name?: string;
  schema_name?: string;
  table_name?: string;
  source_ip?: string;
  bytes_accessed?: number;
  rows_accessed?: number;
  duration_ms?: number;
  success: boolean;
  anomaly_count: number;
  anomaly_signals: UebaSignal[];
  event_timestamp: string;
}

export interface UebaTimelineResponse {
  data: UebaTimelineEvent[];
  meta: PaginationMeta;
}
