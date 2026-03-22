import type { PaginationMeta } from '@/types/api';

export type JsonPrimitive = string | number | boolean | null;
export type JsonValue = JsonPrimitive | JsonObject | JsonValue[];
export interface JsonObject {
  [key: string]: JsonValue;
}

export interface DataEnvelope<T> {
  data: T;
}

export interface DataPaginatedEnvelope<T> {
  data: T[];
  meta: PaginationMeta;
}

export type DataSourceType =
  | 'postgresql'
  | 'mysql'
  | 'mssql'
  | 'api'
  | 'csv'
  | 's3'
  | 'clickhouse'
  | 'impala'
  | 'hive'
  | 'hdfs'
  | 'spark'
  | 'dagster'
  | 'dolt'
  | 'stream';

export type DataSourceStatus =
  | 'pending_test'
  | 'active'
  | 'inactive'
  | 'error'
  | 'syncing';

export type DataClassification =
  | 'public'
  | 'internal'
  | 'confidential'
  | 'restricted';

export interface ConnectionTestResult {
  success: boolean;
  latency_ms: number;
  version?: string;
  message: string;
  permissions?: string[];
  warnings?: string[];
}

export interface SourceStats {
  table_count: number;
  total_row_count: number;
  total_size_bytes: number;
  schema_discovered_at?: string | null;
  last_synced_at?: string | null;
  last_sync_status?: string | null;
}

export interface AggregateSourceStats {
  total_sources: number;
  by_type: Record<string, number>;
  by_status: Record<string, number>;
  sources_with_schema: number;
  total_rows: number;
  total_size_bytes: number;
}

export interface SourceConnectionConfig extends JsonObject {}

export interface ForeignKeyRef {
  schema?: string;
  table: string;
  column: string;
}

export interface ForeignKey {
  column: string;
  referenced_ref: ForeignKeyRef;
}

export interface SampleStats {
  null_count: number;
  distinct_count: number;
  looks_like_email: boolean;
  looks_like_phone: boolean;
  looks_like_credit_card: boolean;
  looks_like_ip: boolean;
  enum_values?: string[];
  min_value?: string;
  max_value?: string;
  observed_samples: number;
}

export interface DiscoveredColumn {
  name: string;
  data_type: string;
  native_type: string;
  mapped_type: string;
  subtype?: string;
  max_length?: number | null;
  nullable: boolean;
  default_value?: string | null;
  comment?: string;
  is_primary_key: boolean;
  is_foreign_key: boolean;
  foreign_key_ref?: ForeignKeyRef | null;
  sample_values?: string[];
  sample_stats?: SampleStats;
  inferred_pii: boolean;
  inferred_pii_type?: string;
  inferred_classification: DataClassification;
  detection_reasons?: string[];
}

export interface DiscoveredTable {
  schema_name?: string;
  name: string;
  type: string;
  comment?: string;
  columns: DiscoveredColumn[];
  primary_keys?: string[];
  foreign_keys?: ForeignKey[];
  estimated_rows?: number;
  size_bytes?: number;
  inferred_classification: DataClassification;
  contains_pii: boolean;
  pii_column_count: number;
  nullable_count: number;
  sampled_row_count: number;
  warnings?: string[];
}

export interface DiscoveredSchema {
  tables: DiscoveredTable[];
  table_count: number;
  column_count: number;
  contains_pii: boolean;
  highest_classification: DataClassification;
  warnings?: string[];
}

export interface SyncHistory {
  id: string;
  tenant_id: string;
  source_id: string;
  status: 'running' | 'success' | 'partial' | 'failed' | 'cancelled';
  sync_type: 'full' | 'incremental' | 'schema_only';
  tables_synced: number;
  rows_read: number;
  rows_written: number;
  bytes_transferred: number;
  errors: JsonValue;
  error_count: number;
  started_at: string;
  completed_at?: string | null;
  duration_ms?: number | null;
  triggered_by: 'manual' | 'schedule' | 'event' | 'api';
  triggered_by_user?: string | null;
  created_at: string;
}

export interface DataSource {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  type: DataSourceType;
  connection_config?: SourceConnectionConfig;
  encryption_key_id?: string;
  status: DataSourceStatus;
  last_error?: string | null;
  schema_metadata?: DiscoveredSchema | null;
  schema_discovered_at?: string | null;
  last_synced_at?: string | null;
  last_sync_status?: string | null;
  last_sync_error?: string | null;
  last_sync_duration_ms?: number | null;
  sync_frequency?: string | null;
  next_sync_at?: string | null;
  table_count?: number | null;
  total_row_count?: number | null;
  total_size_bytes?: number | null;
  tags: string[];
  metadata: JsonObject;
  created_by: string;
  created_at: string;
  updated_at: string;
  deleted_at?: string | null;
}

export type DataModelStatus = 'draft' | 'active' | 'deprecated' | 'archived';

export interface ValidationRule {
  type: string;
  field: string;
  params?: JsonObject;
  message?: string;
}

export interface ModelField {
  name: string;
  display_name: string;
  data_type: string;
  native_type: string;
  nullable: boolean;
  is_primary_key: boolean;
  is_foreign_key: boolean;
  foreign_key_ref?: ForeignKeyRef | null;
  description: string;
  default_value?: string | null;
  pii_type?: string;
  classification: DataClassification;
  sample_values?: string[];
  validation_rules: ValidationRule[];
}

export interface DataModel {
  id: string;
  tenant_id: string;
  name: string;
  display_name: string;
  description: string;
  status: DataModelStatus;
  schema_definition: ModelField[];
  source_id?: string | null;
  source_table?: string | null;
  quality_rules: ValidationRule[];
  data_classification: DataClassification;
  contains_pii: boolean;
  pii_columns: string[];
  field_count: number;
  version: number;
  previous_version_id?: string | null;
  tags: string[];
  metadata: JsonObject;
  created_by: string;
  created_at: string;
  updated_at: string;
  deleted_at?: string | null;
}

export interface ModelValidationError {
  field: string;
  code: string;
  message: string;
}

export interface ModelValidationResult {
  success: boolean;
  errors: ModelValidationError[];
}

export interface ModelLineage {
  model: DataModel;
  source?: DataSource | null;
  source_table?: DiscoveredTable | null;
  upstream_tables?: ForeignKeyRef[];
  consumers?: Array<Record<string, string>>;
}

export type PipelineType = 'etl' | 'elt' | 'batch' | 'streaming';
export type PipelineStatus = 'active' | 'paused' | 'disabled' | 'error';
export type PipelineRunStatus = 'running' | 'completed' | 'failed' | 'cancelled';
export type PipelinePhase = 'extracting' | 'transforming' | 'quality_check' | 'loading';
export type LoadStrategy = 'append' | 'full_replace' | 'incremental' | 'merge';
export type TransformationType =
  | 'rename'
  | 'cast'
  | 'filter'
  | 'map_values'
  | 'derive'
  | 'deduplicate'
  | 'aggregate';

export interface Transformation {
  type: TransformationType;
  config: JsonObject;
}

export interface QualityGate {
  name: string;
  metric:
    | 'null_percentage'
    | 'unique_percentage'
    | 'row_count_change'
    | 'min_row_count'
    | 'custom';
  column?: string;
  operator?: string;
  threshold?: number | null;
  min_value?: number | null;
  max_value?: number | null;
  expression?: string;
  severity?: string;
  description?: string;
}

export interface PipelineConfig {
  source_table?: string;
  source_query?: string;
  target_table?: string;
  target_model_id?: string | null;
  batch_size?: number;
  incremental_field?: string;
  incremental_value?: string | null;
  transformations?: Transformation[];
  quality_gates?: QualityGate[];
  fail_on_quality_gate?: boolean;
  load_strategy?: LoadStrategy;
  merge_keys?: string[];
  max_retries?: number;
  retry_backoff_sec?: number;
  metadata?: JsonObject;
}

export interface Pipeline {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  type: PipelineType;
  source_id: string;
  target_id?: string | null;
  config: PipelineConfig;
  schedule?: string | null;
  status: PipelineStatus;
  last_run_id?: string | null;
  last_run_at?: string | null;
  last_run_status?: PipelineRunStatus | null;
  last_run_error?: string | null;
  next_run_at?: string | null;
  total_runs: number;
  successful_runs: number;
  failed_runs: number;
  total_records_processed: number;
  avg_duration_ms?: number | null;
  tags: string[];
  created_by: string;
  created_at: string;
  updated_at: string;
  deleted_at?: string | null;
}

export interface QualityGateResult {
  name: string;
  metric: string;
  status: string;
  metric_value: number;
  threshold?: number | null;
  min_value?: number | null;
  max_value?: number | null;
  operator?: string;
  message?: string;
  severity?: string;
  evaluated_at: string;
}

export interface PipelineRun {
  id: string;
  tenant_id: string;
  pipeline_id: string;
  status: PipelineRunStatus;
  current_phase?: PipelinePhase | null;
  records_extracted: number;
  records_transformed: number;
  records_loaded: number;
  records_failed: number;
  records_filtered: number;
  records_deduplicated: number;
  bytes_read: number;
  bytes_written: number;
  quality_gate_results: QualityGateResult[];
  quality_gates_passed: number;
  quality_gates_failed: number;
  quality_gates_warned: number;
  started_at: string;
  extract_started_at?: string | null;
  extract_completed_at?: string | null;
  transform_started_at?: string | null;
  transform_completed_at?: string | null;
  load_started_at?: string | null;
  load_completed_at?: string | null;
  completed_at?: string | null;
  duration_ms?: number | null;
  error_phase?: string | null;
  error_message?: string | null;
  error_details?: JsonValue;
  triggered_by: 'manual' | 'schedule' | 'event' | 'api' | 'retry';
  triggered_by_user?: string | null;
  retry_count: number;
  incremental_from?: string | null;
  incremental_to?: string | null;
  created_at: string;
}

export interface PipelineRunLog {
  id: string;
  tenant_id: string;
  run_id: string;
  level: string;
  phase: string;
  message: string;
  details?: JsonValue;
  created_at: string;
}

export interface PipelineStats {
  total_pipelines: number;
  active_pipelines: number;
  paused_pipelines: number;
  error_pipelines: number;
  running_pipelines: number;
  completed_runs: number;
  failed_runs: number;
  success_rate: number;
  by_type: Record<string, number>;
  by_status: Record<string, number>;
  updated_at: string;
}

export type QualityRuleType =
  | 'not_null'
  | 'unique'
  | 'range'
  | 'regex'
  | 'referential'
  | 'enum'
  | 'freshness'
  | 'row_count'
  | 'custom_sql'
  | 'statistical';

export type QualitySeverity = 'critical' | 'high' | 'medium' | 'low';
export type QualityResultStatus = 'passed' | 'failed' | 'warning' | 'error';

export interface QualityRule {
  id: string;
  tenant_id: string;
  model_id: string;
  name: string;
  description: string;
  rule_type: QualityRuleType;
  severity: QualitySeverity;
  column_name?: string | null;
  config: JsonObject;
  schedule?: string | null;
  enabled: boolean;
  last_run_at?: string | null;
  last_status?: QualityResultStatus | null;
  consecutive_failures: number;
  tags: string[];
  created_by: string;
  created_at: string;
  updated_at: string;
  deleted_at?: string | null;
}

export interface QualityResult {
  id: string;
  tenant_id: string;
  rule_id: string;
  model_id: string;
  pipeline_run_id?: string | null;
  status: QualityResultStatus;
  records_checked: number;
  records_passed: number;
  records_failed: number;
  pass_rate?: number | null;
  failure_samples: JsonValue;
  failure_summary?: string | null;
  checked_at: string;
  duration_ms?: number | null;
  error_message?: string | null;
  created_at: string;
}

export interface ModelQualityScore {
  model_id: string;
  model_name: string;
  classification: string;
  score: number;
  total_rules: number;
  passed_rules: number;
  failed_rules: number;
  warning_rules: number;
  classification_weight: number;
}

export interface TopFailure {
  rule_id: string;
  rule_name: string;
  model_id: string;
  model_name: string;
  severity: string;
  status: string;
  records_failed: number;
}

export interface QualityScore {
  overall_score: number;
  grade: string;
  model_scores: ModelQualityScore[];
  total_rules: number;
  passed_rules: number;
  failed_rules: number;
  warning_rules: number;
  pass_rate: number;
  top_failures: TopFailure[];
  trend: string;
  trend_delta: number;
  calculated_at: string;
}

export interface QualityTrendPoint {
  day: string;
  score: number;
}

export interface QualityDashboard {
  score: QualityScore | null;
  recent_rules: QualityRule[];
  top_failures: TopFailure[];
  trend: QualityTrendPoint[];
}

export type ContradictionType = 'logical' | 'semantic' | 'temporal' | 'analytical';
export type ContradictionStatus =
  | 'detected'
  | 'investigating'
  | 'resolved'
  | 'accepted'
  | 'false_positive';
export type ContradictionResolutionAction =
  | 'source_a_corrected'
  | 'source_b_corrected'
  | 'both_corrected'
  | 'accepted_as_is'
  | 'data_reconciled'
  | 'false_positive';

export interface ContradictionSource {
  source_id?: string | null;
  source_name: string;
  model_id?: string | null;
  model_name: string;
  table_name?: string;
  column_name?: string;
  value?: JsonValue;
  last_synced_at?: string | null;
  status?: string;
  metadata?: JsonObject;
}

export interface Contradiction {
  id: string;
  tenant_id: string;
  scan_id?: string | null;
  type: ContradictionType;
  severity: QualitySeverity;
  confidence_score: number;
  title: string;
  description: string;
  source_a: ContradictionSource;
  source_b: ContradictionSource;
  entity_key_column?: string | null;
  entity_key_value?: string | null;
  affected_records: number;
  sample_records: JsonValue;
  resolution_guidance: string;
  authoritative_source?: string | null;
  status: ContradictionStatus;
  resolved_by?: string | null;
  resolved_at?: string | null;
  resolution_notes?: string | null;
  resolution_action?: ContradictionResolutionAction | null;
  metadata: JsonObject;
  created_at: string;
  updated_at: string;
}

export interface ContradictionScan {
  id: string;
  tenant_id: string;
  status: string;
  models_scanned: number;
  model_pairs_compared: number;
  contradictions_found: number;
  by_type: JsonValue;
  by_severity: JsonValue;
  started_at: string;
  completed_at?: string | null;
  duration_ms?: number | null;
  triggered_by: string;
  created_at: string;
}

export interface ContradictionStats {
  total: number;
  by_status: Record<string, number>;
  by_type: Record<string, number>;
  by_severity: Record<string, number>;
  average_confidence: number;
  open_contradictions: number;
  updated_at: string;
}

export interface LineageNode {
  id: string;
  type: string;
  entity_id: string;
  name: string;
  status?: string;
  metadata?: Record<string, JsonValue>;
  depth: number;
  in_degree: number;
  out_degree: number;
  is_critical: boolean;
}

export interface LineageEdge {
  id: string;
  source: string;
  target: string;
  relationship: string;
  transformation?: string;
  columns_affected?: string[];
  pipeline_id?: string | null;
  active: boolean;
  last_seen_at: string;
}

export interface LineageGraphStats {
  node_count: number;
  edge_count: number;
  max_depth: number;
  source_count: number;
  consumer_count: number;
  nodes_by_type: Record<string, number>;
}

export interface LineageGraph {
  nodes: LineageNode[];
  edges: LineageEdge[];
  stats: LineageGraphStats;
}

export interface ImpactedEntity {
  node: LineageNode;
  hop_distance: number;
  path_description: string;
  data_classification?: string;
}

export interface AffectedSuite {
  suite_name: string;
  capability: string;
  impact: string;
  severity: string;
}

export interface ImpactAnalysis {
  entity: LineageNode;
  directly_affected: ImpactedEntity[];
  indirectly_affected: ImpactedEntity[];
  affected_suites: AffectedSuite[];
  total_affected: number;
  severity: string;
  summary: string;
}

export interface LineageStatsSummary {
  node_count: number;
  edge_count: number;
  max_depth: number;
  source_count: number;
  consumer_count: number;
  nodes_by_type: Record<string, number>;
  critical_path_nodes: number;
  last_updated_at_unix_sec: number;
}

export type DarkDataAssetType =
  | 'database_table'
  | 'database_view'
  | 'file'
  | 'api_endpoint'
  | 'stream_topic';

export type DarkDataReason =
  | 'unmodeled'
  | 'orphaned_file'
  | 'stale'
  | 'ungoverned'
  | 'unclassified';

export type DarkDataGovernanceStatus =
  | 'unmanaged'
  | 'under_review'
  | 'governed'
  | 'archived'
  | 'scheduled_deletion';

export interface RiskFactor {
  factor: string;
  value: number;
  description?: string;
}

export interface DarkDataAsset {
  id: string;
  tenant_id: string;
  scan_id?: string | null;
  name: string;
  asset_type: DarkDataAssetType;
  source_id?: string | null;
  source_name?: string | null;
  schema_name?: string | null;
  table_name?: string | null;
  file_path?: string | null;
  reason: DarkDataReason;
  estimated_row_count?: number | null;
  estimated_size_bytes?: number | null;
  column_count?: number | null;
  contains_pii: boolean;
  pii_types: string[];
  inferred_classification?: DataClassification | null;
  last_accessed_at?: string | null;
  last_modified_at?: string | null;
  days_since_access?: number | null;
  risk_score: number;
  risk_factors: RiskFactor[];
  governance_status: DarkDataGovernanceStatus;
  governance_notes?: string | null;
  reviewed_by?: string | null;
  reviewed_at?: string | null;
  linked_model_id?: string | null;
  metadata: JsonObject;
  discovered_at: string;
  created_at: string;
  updated_at: string;
}

export interface DarkDataScan {
  id: string;
  tenant_id: string;
  status: 'running' | 'completed' | 'failed';
  sources_scanned: number;
  storage_scanned: boolean;
  assets_discovered: number;
  by_reason: JsonValue;
  by_type: JsonValue;
  pii_assets_found: number;
  high_risk_found: number;
  total_size_bytes: number;
  started_at: string;
  completed_at?: string | null;
  duration_ms?: number | null;
  triggered_by: string;
  created_at: string;
}

export interface DarkDataStatsSummary {
  total_assets: number;
  by_reason: Record<string, number>;
  by_type: Record<string, number>;
  by_governance_status: Record<string, number>;
  pii_assets: number;
  high_risk_assets: number;
  total_size_bytes: number;
  average_risk_score: number;
  governed_assets: number;
  scheduled_deletion_count: number;
}

export interface AnalyticsFilter {
  column: string;
  operator: string;
  value?: JsonValue;
}

export interface AnalyticsOrder {
  column: string;
  direction: 'asc' | 'desc';
}

export interface AnalyticsAggregation {
  function: string;
  column: string;
  alias: string;
  distinct?: boolean;
}

export interface AnalyticsQuery {
  columns?: string[];
  filters?: AnalyticsFilter[];
  group_by?: string[];
  aggregations?: AnalyticsAggregation[];
  order_by?: AnalyticsOrder[];
  limit?: number;
  offset?: number;
}

export interface QueryExplain {
  sql: string;
  count_sql: string;
  parameters: JsonValue[];
}

export interface ColumnMeta {
  name: string;
  data_type: string;
  classification: string;
  is_pii: boolean;
  masked: boolean;
}

export interface QueryMetadata {
  model_name: string;
  data_classification: string;
  pii_masking_applied: boolean;
  columns_masked?: string[];
  execution_time_ms: number;
  cached_result: boolean;
}

export interface QueryResult {
  columns: ColumnMeta[];
  rows: Array<Record<string, JsonValue>>;
  row_count: number;
  total_count: number;
  truncated: boolean;
  metadata: QueryMetadata;
}

export type SavedQueryVisibility = 'private' | 'team' | 'organization';

export interface SavedQuery {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  model_id: string;
  query_definition: AnalyticsQuery;
  last_run_at?: string | null;
  run_count: number;
  visibility: SavedQueryVisibility;
  tags: string[];
  created_by: string;
  created_at: string;
  updated_at: string;
  deleted_at?: string | null;
}

export interface AnalyticsAuditLog {
  id: string;
  tenant_id: string;
  user_id: string;
  model_id: string;
  source_id: string;
  query_definition: AnalyticsQuery;
  columns_accessed: string[];
  filters_applied: JsonValue;
  data_classification: string;
  pii_columns_accessed: string[];
  pii_masking_applied: boolean;
  rows_returned: number;
  truncated: boolean;
  execution_time_ms?: number | null;
  error_occurred: boolean;
  error_message?: string | null;
  saved_query_id?: string | null;
  ip_address?: string | null;
  user_agent?: string | null;
  executed_at: string;
}

export interface DailyMetric {
  day: string;
  value: number;
}

export interface PipelineRunSummary {
  id: string;
  pipeline_id: string;
  pipeline_name: string;
  status: string;
  started_at: string;
  completed_at?: string | null;
  duration_ms?: number | null;
}

export interface QualityScoreSummary {
  overall_score: number;
  grade: string;
  passed_rules: number;
  failed_rules: number;
  warning_rules: number;
  pass_rate: number;
}

export interface ModelQualitySummary {
  model_id: string;
  model_name: string;
  classification: string;
  score: number;
}

export interface QualityFailureSummary {
  rule_id: string;
  rule_name: string;
  model_id: string;
  model_name: string;
  severity: string;
  records_failed: number;
}

export interface DataKPIs {
  total_sources: number;
  active_pipelines: number;
  quality_score: number;
  quality_grade: string;
  open_contradictions: number;
  dark_data_assets: number;
  total_models: number;
  failed_pipelines_24h: number;
  sources_delta: number;
  quality_delta: number;
  contradictions_delta: number;
}

export interface DataSuiteDashboard {
  kpis: DataKPIs;
  sources_by_type: Record<string, number>;
  sources_by_status: Record<string, number>;
  pipelines_by_status: Record<string, number>;
  recent_runs: PipelineRunSummary[];
  pipeline_success_rate_30d: number;
  pipeline_trend_30d: DailyMetric[];
  quality_score: QualityScoreSummary;
  quality_trend_30d: DailyMetric[];
  quality_by_model: ModelQualitySummary[];
  top_quality_failures: QualityFailureSummary[];
  contradictions_by_type: Record<string, number>;
  contradictions_by_severity: Record<string, number>;
  open_contradictions: number;
  lineage_stats: Record<string, JsonValue>;
  dark_data_stats: Record<string, JsonValue>;
  cached_at?: string | null;
  calculated_at: string;
  partial_failures?: string[];
}

export interface UploadedFile {
  id: string;
  tenant_id: string;
  original_name: string;
  sanitized_name: string;
  content_type: string;
  detected_content_type?: string;
  size_bytes: number;
  checksum_sha256: string;
  encrypted: boolean;
  virus_scan_status: string;
  uploaded_by: string;
  suite: string;
  entity_type?: string | null;
  entity_id?: string | null;
  tags: string[];
  version_number: number;
  is_public: boolean;
  lifecycle_policy: string;
  expires_at?: string | null;
  created_at: string;
  updated_at: string;
}
