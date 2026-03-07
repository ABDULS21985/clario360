export interface JsonObject {
  [key: string]: unknown;
}

export interface DataSource {
  id: string;
  name: string;
  type: string;
  status: string;
  connection_config: JsonObject;
  schema_metadata: JsonObject;
  last_synced_at?: string | null;
  sync_frequency?: string | null;
  created_at: string;
  updated_at: string;
}

export interface DataPipeline {
  id: string;
  name: string;
  description: string;
  type: string;
  status: string;
  source_id?: string | null;
  source_name?: string | null;
  target_id?: string | null;
  target_name?: string | null;
  schedule?: string | null;
  config: JsonObject;
  last_run_at?: string | null;
  next_run_at?: string | null;
  last_run_status?: string | null;
  last_run_records_failed?: number | null;
  last_run_records_processed?: number | null;
  last_run_completed_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface DataPipelineRun {
  id: string;
  pipeline_id: string;
  status: string;
  started_at: string;
  completed_at?: string | null;
  records_processed: number;
  records_failed: number;
  error_log?: string | null;
  metrics: JsonObject;
}

export interface Dataset {
  id: string;
  name: string;
  description: string;
  version: number;
  status: string;
  source_id?: string | null;
  source_name?: string | null;
  schema_definition: JsonObject;
  lineage: JsonObject;
  created_at: string;
  updated_at: string;
}

export interface QualityFailure {
  rule_name: string;
  model_name: string;
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info' | string;
  records_failed: number;
  failure_samples: unknown;
  checked_at: string;
}

export interface QualityDashboard {
  score: number;
  trend: number;
  total_rules: number;
  enabled_rules: number;
  results_last_7_days: number;
  failed_last_7_days: number;
  critical_failures: number;
  pass_rate: number;
  recent_failures: QualityFailure[];
}

export interface ActaCommittee {
  id: string;
  name: string;
  type: string;
  description: string;
  members: Array<Record<string, unknown>>;
  meeting_frequency?: string | null;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface ActaMeeting {
  id: string;
  committee_id: string;
  committee_name: string;
  title: string;
  description: string;
  scheduled_at: string;
  location?: string | null;
  virtual_link?: string | null;
  status: string;
  duration_minutes?: number | null;
  attendees: Array<Record<string, unknown>>;
  action_item_count: number;
  created_at: string;
  updated_at: string;
}

export interface ActaMeetingMinute {
  id: string;
  meeting_id: string;
  meeting_title: string;
  content: string;
  ai_summary?: string | null;
  ai_action_items: Array<Record<string, unknown>>;
  status: string;
  approved_by?: string | null;
  approved_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface ActaActionItem {
  id: string;
  meeting_id: string;
  meeting_title: string;
  title: string;
  description: string;
  assigned_to?: string | null;
  due_date?: string | null;
  status: string;
  completed_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface ActaTemplate {
  id: string;
  name: string;
  type: string;
  definition: JsonObject;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface LexContract {
  id: string;
  title: string;
  type: string;
  status: string;
  parties: Array<Record<string, unknown>>;
  effective_date?: string | null;
  expiry_date?: string | null;
  value?: number | null;
  currency: string;
  file_url?: string | null;
  metadata: JsonObject;
  created_at: string;
  updated_at: string;
}

export interface LexDocument {
  id: string;
  title: string;
  type: string;
  content: string;
  file_url?: string | null;
  status: string;
  version: number;
  parent_id?: string | null;
  tags: string[];
  created_at: string;
  updated_at: string;
}

export interface ComplianceRule {
  id: string;
  name: string;
  description: string;
  jurisdiction?: string | null;
  regulation_reference?: string | null;
  rule_logic: JsonObject;
  severity: 'critical' | 'high' | 'medium' | 'low' | string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface ComplianceAlert {
  id: string;
  rule_id: string;
  entity_type: string;
  entity_id: string;
  title: string;
  description: string;
  severity: 'critical' | 'high' | 'medium' | 'low' | string;
  status: string;
  created_at: string;
  resolved_at?: string | null;
}

export interface ComplianceDashboard {
  total_rules: number;
  enabled_rules: number;
  open_alerts: number;
  by_severity: Record<string, number>;
  by_status: Record<string, number>;
  recent_alerts: ComplianceAlert[];
  contracts_expiring_30d: number;
}

export interface ComplianceCheckResult {
  rule_id: string;
  rule_name: string;
  severity: string;
  status: string;
  message: string;
  alert_id?: string | null;
}

export interface VisusDashboard {
  id: string;
  name: string;
  description: string;
  layout: JsonObject;
  is_default: boolean;
  owner_user_id?: string | null;
  shared_with: Array<Record<string, unknown>>;
  widget_count: number;
  created_at: string;
  updated_at: string;
}

export interface VisusWidget {
  id: string;
  dashboard_id: string;
  type: string;
  title: string;
  config: JsonObject;
  position: JsonObject;
  refresh_interval_seconds?: number | null;
  created_at: string;
  updated_at: string;
}

export interface VisusReport {
  id: string;
  name: string;
  type: string;
  config: JsonObject;
  schedule?: string | null;
  last_generated_at?: string | null;
  file_url?: string | null;
  created_at: string;
  updated_at: string;
}

export interface VisusReportGeneration {
  report_id: string;
  snapshot_id: string;
  generated_at: string;
  file_url: string;
  metadata: JsonObject;
}
