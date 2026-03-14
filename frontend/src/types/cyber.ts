// Cyber suite TypeScript types — mirrors backend model package.

// ─── Common ──────────────────────────────────────────────────────────────────

export type CyberSeverity = 'critical' | 'high' | 'medium' | 'low' | 'info';
export type AlertStatus =
  | 'new'
  | 'acknowledged'
  | 'investigating'
  | 'in_progress'
  | 'resolved'
  | 'closed'
  | 'false_positive'
  | 'escalated'
  | 'merged';

export type AssetType =
  | 'server'
  | 'endpoint'
  | 'network_device'
  | 'cloud_resource'
  | 'iot_device'
  | 'application'
  | 'database'
  | 'container';

export type AssetStatus = 'active' | 'inactive' | 'decommissioned' | 'unknown';
export type Criticality = 'critical' | 'high' | 'medium' | 'low';

// ─── Dashboard ────────────────────────────────────────────────────────────────

export interface KPICards {
  open_alerts: number;
  critical_alerts: number;
  open_vulnerabilities: number;
  critical_vulnerabilities: number;
  active_threats: number;
  mttr_hours: number;
  mean_resolve_hours: number;
  risk_score: number;
  risk_grade: string;
  alerts_delta: number;
  vulns_delta: number;
}

export interface AlertTimelinePoint {
  bucket: string;
  count: number;
}

export interface AlertTimelineData {
  granularity: string;
  points: AlertTimelinePoint[];
}

export interface SeverityDistribution {
  counts: Record<string, number>;
  total: number;
}

export interface DailyMetric {
  date: string;
  count: number;
}

export interface AlertSummary {
  id: string;
  title: string;
  severity: CyberSeverity;
  status: AlertStatus;
  asset_id?: string;
  asset_name?: string;
  assigned_to?: string;
  created_at: string;
  mitre_technique_id?: string;
  mitre_technique_name?: string;
  confidence_score?: number;
}

export interface AssetAlertSummary {
  asset_id: string;
  asset_name: string;
  asset_type: string;
  criticality: Criticality;
  alert_count: number;
  critical_open: number;
  open_vuln_count?: number;
}

export interface AnalystWorkloadEntry {
  user_id: string;
  name: string;
  open_assigned: number;
  critical_open: number;
  resolved_this_week: number;
  avg_resolve_hours?: number;
}

export interface MITREHeatmapCell {
  tactic_id: string;
  tactic_name: string;
  technique_id: string;
  technique_name: string;
  alert_count: number;
  critical_count: number;
  last_seen: string;
  has_detection: boolean;
}

export interface MITREHeatmapData {
  cells: MITREHeatmapCell[];
  max_count: number;
}

export interface ComponentScore {
  score: number;
  weight: number;
  weighted: number;
  trend: string;
  trend_delta: number;
  description: string;
}

export interface RiskComponents {
  vulnerability_risk: ComponentScore;
  threat_exposure: ComponentScore;
  configuration_risk: ComponentScore;
  attack_surface_risk: ComponentScore;
  compliance_gap_risk: ComponentScore;
}

export interface RiskContributor {
  type: string;
  id: string;
  title: string;
  score: number;
  impact_percent: number;
  severity: string;
  asset_id?: string;
  asset_name?: string;
  remediation: string;
  link: string;
}

export interface RiskRecommendation {
  priority: number;
  title: string;
  description: string;
  component: string;
  estimated_score_reduction: number;
  effort: string;
  category: string;
  actions: string[];
}

export interface OrganizationRiskScore {
  tenant_id: string;
  overall_score: number;
  grade: string;
  trend: string;
  trend_delta: number;
  components: RiskComponents;
  top_contributors: RiskContributor[];
  recommendations: RiskRecommendation[];
  context: {
    total_assets: number;
    total_open_vulnerabilities: number;
    total_open_alerts: number;
    total_active_threats: number;
    internet_facing_assets: number;
    critical_assets: number;
  };
  calculated_at: string;
}

export interface SOCDashboard {
  kpis: KPICards;
  alert_timeline: AlertTimelineData;
  severity_distribution: SeverityDistribution;
  alert_trend: DailyMetric[];
  vulnerability_trend: DailyMetric[];
  recent_alerts: AlertSummary[];
  top_attacked_assets: AssetAlertSummary[];
  analyst_workload: AnalystWorkloadEntry[];
  mitre_heatmap: MITREHeatmapData;
  risk_score?: OrganizationRiskScore;
  cached_at?: string;
  calculated_at: string;
  partial_failures?: string[];
}

// ─── Asset ────────────────────────────────────────────────────────────────────

export interface CyberAsset {
  id: string;
  tenant_id: string;
  name: string;
  type: AssetType;
  ip_address?: string;
  hostname?: string;
  mac_address?: string;
  os?: string;
  os_version?: string;
  owner?: string;
  department?: string;
  location?: string;
  criticality: Criticality;
  status: AssetStatus;
  tags: string[];
  discovery_source?: string;
  discovered_at?: string;
  last_seen_at?: string;
  vulnerability_count?: number;
  critical_vuln_count?: number;
  high_vuln_count?: number;
  alert_count?: number;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface AssetStats {
  total: number;
  by_type: Record<string, number>;
  by_criticality: Record<string, number>;
  by_status: Record<string, number>;
  assets_with_vulns: number;
  assets_discovered_this_week: number;
}

export interface AssetRelationship {
  id: string;
  source_asset_id: string;
  source_asset_name: string;
  source_asset_type: string;
  source_criticality: Criticality;
  target_asset_id: string;
  target_asset_name: string;
  target_asset_type: string;
  target_criticality: Criticality;
  relationship_type: string;
  description?: string;
  created_at: string;
}

export interface AssetScan {
  id: string;
  tenant_id: string;
  scan_type: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  target?: string;
  assets_found: number;
  assets_updated: number;
  started_at?: string;
  completed_at?: string;
  error?: string;
  created_at: string;
}

// ─── Alert ────────────────────────────────────────────────────────────────────

export interface AlertEvidence {
  label: string;
  field: string;
  value: unknown;
  description: string;
}

export interface IndicatorEvidence {
  type: string;
  value: string;
  source: string;
  confidence: number;
  field: string;
}

export interface ConfidenceFactor {
  factor: string;
  impact: number;
  description: string;
}

export interface AlertExplanation {
  summary: string;
  reason: string;
  evidence: AlertEvidence[];
  matched_conditions: string[];
  confidence_factors: ConfidenceFactor[];
  recommended_actions: string[];
  false_positive_indicators: string[];
  indicator_matches?: IndicatorEvidence[];
  details?: Record<string, unknown>;
}

export interface CyberAlert {
  id: string;
  tenant_id: string;
  title: string;
  description: string;
  severity: CyberSeverity;
  status: AlertStatus;
  source: string;
  rule_id?: string;
  asset_id?: string;
  asset_ids: string[];
  assigned_to?: string;
  assigned_at?: string;
  escalated_to?: string;
  escalated_at?: string;
  explanation: AlertExplanation;
  confidence_score: number;
  mitre_tactic_id?: string;
  mitre_tactic_name?: string;
  mitre_technique_id?: string;
  mitre_technique_name?: string;
  event_count: number;
  first_event_at: string;
  last_event_at: string;
  resolved_at?: string;
  resolution_notes?: string;
  false_positive_reason?: string;
  tags: string[];
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  // enriched fields
  asset_name?: string;
  assigned_to_name?: string;
  rule_name?: string;
}

export interface AlertStats {
  by_severity: Array<{ name: string; count: number }>;
  by_status: Array<{ name: string; count: number }>;
  by_rule: Array<{ name: string; count: number }>;
  by_technique: Array<{ name: string; count: number }>;
  open_count: number;
  resolved_count: number;
}

export interface AlertComment {
  id: string;
  alert_id: string;
  user_id: string;
  user_name: string;
  user_email: string;
  content: string;
  is_system: boolean;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface AlertTimelineEntry {
  id: string;
  alert_id: string;
  action: string;
  actor_id?: string;
  actor_name?: string;
  old_value?: string;
  new_value?: string;
  description: string;
  metadata?: Record<string, unknown>;
  created_at: string;
}

// ─── Vulnerability ────────────────────────────────────────────────────────────

export interface Vulnerability {
  id: string;
  tenant_id: string;
  asset_id: string;
  asset_name?: string;
  asset_type?: string;
  asset_criticality?: string;
  cve_id?: string;
  title: string;
  description: string;
  severity: CyberSeverity;
  cvss_score?: number;
  cvss_vector?: string;
  status: 'open' | 'in_progress' | 'mitigated' | 'resolved' | 'accepted' | 'false_positive';
  detected_at: string;
  resolved_at?: string;
  source: string;
  remediation?: string;
  proof?: string;
  metadata?: Record<string, unknown>;
  age_days: number;
  has_exploit: boolean;
  created_at: string;
  updated_at: string;
}

// ─── Threat ───────────────────────────────────────────────────────────────────

export interface ThreatIndicator {
  id: string;
  type: string;
  value: string;
  severity: CyberSeverity;
  source: string;
  confidence: number;
  first_seen: string;
  last_seen: string;
  tags: string[];
}

export interface Threat {
  id: string;
  tenant_id: string;
  name: string;
  type: string;
  severity: CyberSeverity;
  status: 'active' | 'contained' | 'eradicated' | 'monitoring' | 'closed';
  description: string;
  first_seen: string;
  last_seen: string;
  indicator_count: number;
  affected_asset_count: number;
  indicators?: ThreatIndicator[];
  tags: string[];
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface IndicatorCheckResult {
  value: string;
  type: string;
  matched: boolean;
  threat_name?: string;
  severity?: CyberSeverity;
  source?: string;
  confidence?: number;
}

// ─── Detection Rule ───────────────────────────────────────────────────────────

export interface DetectionRule {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  type: 'sigma' | 'threshold' | 'correlation' | 'anomaly';
  severity: CyberSeverity;
  enabled: boolean;
  mitre_technique_ids: string[];
  mitre_tactic_ids: string[];
  trigger_count: number;
  false_positive_rate: number;
  last_triggered?: string;
  condition?: string;
  rule_content?: SigmaRuleContent | ThresholdRuleContent | AnomalyRuleContent | CorrelationRuleContent;
  base_confidence?: number;
  is_auto_disabled?: boolean;
  tp_count?: number;
  fp_count?: number;
  metadata?: Record<string, unknown>;
  is_template: boolean;
  tags: string[];
  created_at: string;
  updated_at: string;
}

export interface RuleTemplate {
  id: string;
  name: string;
  description: string;
  type: string;
  severity: CyberSeverity;
  mitre_technique_ids: string[];
  category: string;
}

// ─── CTEM ─────────────────────────────────────────────────────────────────────

export type CTEMPhase = 'scoping' | 'discovery' | 'prioritization' | 'validation' | 'mobilization';
export type CTEMPhaseStatus = 'pending' | 'running' | 'completed' | 'failed';

export interface CTEMPhaseInfo {
  phase: CTEMPhase;
  status: CTEMPhaseStatus;
  started_at?: string;
  completed_at?: string;
  progress_percent?: number;
  error?: string;
}

export interface CTEMFinding {
  id: string;
  assessment_id: string;
  title: string;
  description: string;
  severity: CyberSeverity;
  priority_score: number;
  status: 'open' | 'in_remediation' | 'resolved' | 'accepted' | 'false_positive';
  asset_id?: string;
  asset_name?: string;
  cvss_score?: number;
  exploit_available: boolean;
  attack_path?: string[];
  remediation_steps?: string[];
  created_at: string;
  updated_at: string;
}

export interface CTEMAssessment {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  status: 'draft' | 'running' | 'completed' | 'failed' | 'cancelled';
  current_phase?: CTEMPhase;
  phases: CTEMPhaseInfo[];
  scope: {
    all_assets: boolean;
    asset_types?: string[];
    tags?: string[];
    departments?: string[];
    asset_ids?: string[];
    excluded_asset_ids?: string[];
    include_external_exposure: boolean;
  };
  findings_summary?: {
    critical: number;
    high: number;
    medium: number;
    low: number;
    total: number;
  };
  exposure_score?: number;
  findings?: CTEMFinding[];
  error?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface ExposureScore {
  score: number;
  grade: string;
  trend: string;
  trend_delta: number;
  calculated_at: string;
}

export interface ExposureScorePoint {
  date: string;
  score: number;
  grade: string;
}

// ─── Remediation ──────────────────────────────────────────────────────────────

export type RemediationStatus =
  | 'draft'
  | 'pending_approval'
  | 'approved'
  | 'rejected'
  | 'revision_requested'
  | 'dry_run_running'
  | 'dry_run_completed'
  | 'dry_run_failed'
  | 'execution_pending'
  | 'executing'
  | 'executed'
  | 'verification_running'
  | 'verified'
  | 'verification_failed'
  | 'rollback_pending'
  | 'rolling_back'
  | 'rolled_back'
  | 'rollback_failed'
  | 'closed';

export interface RemediationStep {
  number: number;
  action: string;
  description?: string;
  target?: string;
  expected?: string;
}

export interface RemediationPlan {
  steps: RemediationStep[];
  target_version?: string;
  requires_reboot?: boolean;
  reversible?: boolean;
  risk_level?: string;
  estimated_downtime?: string;
  rollback_steps?: RemediationStep[];
  target_config?: Record<string, unknown>;
}

export interface SimulatedChange {
  asset_id: string;
  asset_name: string;
  change_type: string;
  description: string;
  before_value?: string;
  after_value?: string;
  reversible?: boolean;
}

export interface ImpactEstimate {
  downtime: string;
  services_affected: number;
  risk_level: string;
  recommend_window: string;
}

export interface DryRunResult {
  success: boolean;
  simulated_changes: SimulatedChange[];
  warnings: string[];
  blockers: string[];
  affected_services: string[];
  estimated_impact: ImpactEstimate;
  duration_ms: number;
}

export interface StepResult {
  step_number: number;
  action: string;
  status: 'success' | 'failure' | 'skipped';
  output?: string;
  error?: string;
  duration_ms: number;
}

export interface AppliedChange {
  asset_id: string;
  change_type: string;
  description: string;
  old_value?: string;
  new_value?: string;
}

export interface ExecutionResult {
  success: boolean;
  steps_total: number;
  steps_executed: number;
  step_results: StepResult[];
  changes_applied: AppliedChange[];
  duration_ms: number;
}

export interface VerificationCheck {
  name: string;
  expected: string;
  actual: string;
  passed: boolean;
  notes?: string;
}

export interface VerificationResult {
  verified: boolean;
  checks: VerificationCheck[];
  failure_reason?: string;
  duration_ms: number;
}

export interface RemediationAction {
  id: string;
  tenant_id: string;
  alert_id?: string;
  vulnerability_id?: string;
  assessment_id?: string;
  type: string;
  severity: CyberSeverity;
  title: string;
  description: string;
  status: RemediationStatus;
  plan: RemediationPlan;
  affected_asset_ids: string[];
  execution_mode: string;
  requires_approval_from?: string;
  approved_by?: string;
  approved_at?: string;
  rejected_by?: string;
  rejected_at?: string;
  dry_run_at?: string;
  dry_run_result?: DryRunResult;
  executed_at?: string;
  execution_result?: ExecutionResult;
  verification_result?: VerificationResult;
  rollback_deadline?: string;
  rollback_reason?: string;
  pre_execution_state?: unknown;
  tags: string[];
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  created_by?: string;
  created_by_name?: string;
}

export interface RemediationAuditEntry {
  id: string;
  remediation_id: string;
  action: string;
  actor_id?: string;
  actor_name?: string;
  details?: Record<string, unknown>;
  created_at: string;
}

export interface RemediationStats {
  total: number;
  by_status: Record<string, number>;
  by_severity: Record<string, number>;
  by_type: Record<string, number>;
  pending_approval: number;
  execution_pending: number;
}

// ─── DSPM ─────────────────────────────────────────────────────────────────────

export interface DSPMPostureFinding {
  control: string;
  severity: CyberSeverity;
  description: string;
  guidance: string;
}

export interface ComplianceTag {
  framework: string;
  article: string;
  category: string;
  requirement: string;
  impact: string;
  severity: CyberSeverity;
}

export interface DataAsset {
  id: string;
  tenant_id: string;
  asset_id: string;
  asset_name: string;
  asset_type: string;
  scan_id?: string | null;
  data_classification: string;
  sensitivity_score: number;
  contains_pii: boolean;
  pii_column_count: number;
  estimated_record_count?: number | null;
  posture_score: number;
  risk_score: number;
  encrypted_at_rest?: boolean | null;
  encrypted_in_transit?: boolean | null;
  access_control_type?: string | null;
  network_exposure?: string | null;
  backup_configured?: boolean | null;
  audit_logging?: boolean | null;
  last_access_review?: string | null;
  consumer_count?: number;
  producer_count?: number;
  database_type?: string | null;
  schema_info?: Record<string, unknown>;
  posture_findings: DSPMPostureFinding[];
  pii_types: string[];
  metadata?: Record<string, unknown> & {
    compliance_tags?: ComplianceTag[];
  };
  last_scanned_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface DSPMScan {
  id: string;
  tenant_id: string;
  status: string;
  assets_scanned: number;
  pii_assets_found: number;
  high_risk_found: number;
  findings_count: number;
  started_at: string;
  completed_at?: string | null;
  duration_ms?: number | null;
  created_by: string;
  created_at: string;
}

export interface DSPMDashboard {
  total_data_assets: number;
  pii_assets_count: number;
  high_risk_assets_count: number;
  avg_posture_score: number;
  avg_risk_score: number;
  unencrypted_count: number;
  no_access_control_count: number;
  internet_facing_count: number;
  classification_breakdown: Record<string, number>;
  exposure_breakdown: Record<string, number>;
  top_risky_assets: DataAsset[];
  recent_scans: DSPMScan[];
  pii_type_frequency: Record<string, number>;
}

export interface ShadowCopyMatch {
  source_asset_id: string;
  source_asset_name: string;
  source_table: string;
  target_asset_id: string;
  target_asset_name: string;
  target_table: string;
  fingerprint: string;
  match_type: string;
  similarity: number;
  has_lineage: boolean;
}

export interface ShadowDetectionResult {
  tenant_id: string;
  matches: ShadowCopyMatch[];
  sources_count: number;
  tables_count: number;
  duration: number | string;
  summary: string;
}

export type RootCauseAnalysisType = 'security_alert' | 'pipeline_failure' | 'quality_issue';

export interface RootCauseEvidence {
  label: string;
  field: string;
  value: unknown;
  description: string;
}

export interface RootCauseStep {
  order: number;
  event_id: string;
  event_type: string;
  source: string;
  description: string;
  timestamp: string;
  severity?: string;
  mitre_phase?: string;
  mitre_technique_id?: string;
  evidence: RootCauseEvidence[];
  is_root_cause: boolean;
  metadata?: Record<string, unknown>;
}

export interface RootCauseTimelineEvent {
  id: string;
  timestamp: string;
  source: string;
  type: string;
  summary: string;
  severity?: string;
  source_ip?: string;
  user_id?: string;
  asset_id?: string;
  mitre_phase?: string;
  mitre_technique_id?: string;
  metadata?: Record<string, unknown>;
}

export interface RootCauseAffectedAsset {
  asset_id: string;
  asset_name: string;
  asset_type: string;
  criticality: string;
  impact_type: string;
}

export interface RootCauseDataRisk {
  asset_id: string;
  asset_name: string;
  classification: string;
  contains_pii: boolean;
  pii_types?: string[];
}

export interface RootCauseImpactAssessment {
  direct_assets: RootCauseAffectedAsset[];
  transitive_assets: RootCauseAffectedAsset[];
  total_affected: number;
  data_at_risk: RootCauseDataRisk[];
  users_at_risk: number;
  business_impact: string;
  summary: string;
}

export interface RootCauseRecommendation {
  priority: number;
  category: string;
  action: string;
  rationale: string;
  root_cause_type: string;
}

export interface RootCauseAnalysis {
  id: string;
  tenant_id: string;
  type: RootCauseAnalysisType;
  incident_id: string;
  status: string;
  root_cause?: RootCauseStep | null;
  causal_chain: RootCauseStep[];
  timeline: RootCauseTimelineEvent[];
  impact?: RootCauseImpactAssessment | null;
  recommendations: RootCauseRecommendation[];
  confidence: number;
  summary: string;
  analyzed_at: string;
  duration_ms: number;
}

// ─── vCISO ────────────────────────────────────────────────────────────────────

export interface VCISOCriticalIssue {
  id: string;
  title: string;
  severity: CyberSeverity;
  impact: string;
  recommendation: string;
  link?: string;
}

export interface VCISORecommendation {
  id: string;
  priority: number;
  title: string;
  description: string;
  category: string;
  impact: string;
  effort: 'low' | 'medium' | 'high';
  estimated_risk_reduction: number;
  actions: string[];
}

export interface ComplianceFramework {
  name: string;
  coverage_percent: number;
  controls_passed: number;
  controls_total: number;
  status: 'compliant' | 'partial' | 'non_compliant';
}

export interface ThreatLandscape {
  active_threat_count: number;
  top_tactic: string;
  top_technique: string;
  recent_indicators: number;
  threat_by_type: Record<string, number>;
}

export interface RiskPostureSummary {
  overall_score: number;
  grade: string;
  trend: string;
  trend_delta: number;
  components: Record<string, number>;
}

export interface VCISOBriefing {
  id: string;
  tenant_id: string;
  period_start: string;
  period_end: string;
  risk_posture: RiskPostureSummary;
  critical_issues: VCISOCriticalIssue[];
  threat_landscape: ThreatLandscape;
  recommendations: VCISORecommendation[];
  compliance_status: ComplianceFramework[];
  executive_summary: string;
  generated_at: string;
  previous_briefing_id?: string;
  previous_risk_score?: number;
}

export type VCISOEnginePreference = 'auto' | 'llm' | 'rule_based';
export type VCISOEngine = 'llm' | 'rule_based' | 'fallback';

export type VCISOResponseType =
  | 'text'
  | 'table'
  | 'chart'
  | 'kpi'
  | 'dashboard'
  | 'list'
  | 'investigation';

export interface VCISOSuggestedAction {
  label: string;
  type: 'navigate' | 'execute_tool' | 'confirm';
  params: Record<string, string>;
}

export interface VCISOResponsePayload {
  text: string;
  data?: unknown;
  data_type: VCISOResponseType;
  actions: VCISOSuggestedAction[];
}

export interface VCISOResponseMeta {
  intent?: string;
  confidence?: number;
  tool_calls_count?: number;
  reasoning_steps?: number;
  latency_ms?: number;
  synthesis_latency_ms?: number;
  tokens_used?: number;
  grounding?: string;
  engine?: string;
  routing_reason?: string;
}

export interface VCISOChatResponse {
  conversation_id: string;
  message_id: string;
  response: VCISOResponsePayload;
  intent: string;
  confidence: number;
  engine?: string;
  meta?: VCISOResponseMeta;
}

export interface VCISOSuggestion {
  text: string;
  category: string;
  priority: number;
  reason: string;
}

export interface VCISOConversationListItem {
  id: string;
  title: string;
  status: string;
  message_count: number;
  last_message_at?: string | null;
  created_at: string;
}

export interface VCISOConversationMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  intent?: string | null;
  response_type?: VCISOResponseType | null;
  actions: VCISOSuggestedAction[];
  tool_result?: unknown;
  engine?: string | null;
  meta?: VCISOResponseMeta | null;
  created_at: string;
}

export interface VCISOConversationDetail {
  id: string;
  title: string;
  status: string;
  message_count: number;
  last_message_at?: string | null;
  created_at: string;
  messages: VCISOConversationMessage[];
}

export interface VCISOLLMAuditToolCall {
  name: string;
  arguments: Record<string, unknown>;
  result_summary: string;
  success: boolean;
  latency_ms: number;
  called_at: string;
}

export interface VCISOLLMAuditReasoningStep {
  step: number;
  action: string;
  detail: string;
  tool_names?: string[];
}

export interface VCISOLLMAuditResponse {
  message_id: string;
  provider: string;
  model: string;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  tool_calls: VCISOLLMAuditToolCall[];
  reasoning_trace: VCISOLLMAuditReasoningStep[];
  grounding_result: string;
  engine_used: string;
  routing_reason?: string;
  created_at: string;
}

export interface VCISOLLMUsage {
  calls_today: number;
  tokens_today: number;
  cost_today: number;
  calls_this_month: number;
  cost_this_month: number;
}

export interface VCISOLLMHealth {
  provider: string;
  model: string;
  status: string;
  latency_ms: number;
  rate_limit_remaining: number;
}

export interface VCISOLLMConfigRequest {
  provider: string;
  model: string;
  temperature: number;
}

export interface VCISOLLMConfigResponse {
  provider: string;
  model: string;
  temperature: number;
}

export interface VCISOLLMPromptVersion {
  id: string;
  version: string;
  description?: string;
  active: boolean;
  created_by: string;
  created_at: string;
}

export interface VCISOLLMPromptVersionRequest {
  version: string;
  prompt_text: string;
  description: string;
}

// ─── MITRE ────────────────────────────────────────────────────────────────────

export interface MITRETechniqueCoverage {
  technique_id: string;
  technique_name: string;
  tactic_id: string;
  tactic_name: string;
  rule_count: number;
  alert_count: number;
  has_detection: boolean;
  last_alert?: string;
  description?: string;
  platforms?: string[];
}

export interface MITRETactic {
  id: string;
  name: string;
  short_name: string;
  description?: string;
  technique_count: number;
  covered_count: number;
}

export interface MITRECoverage {
  tactics: Array<{
    id: string;
    name: string;
    short_name?: string;
    technique_count: number;
    covered_count: number;
  }>;
  techniques: MITRETechniqueCoverage[];
  total_techniques: number;
  covered_techniques: number;
  coverage_percent: number;
  active_techniques?: number;
  passive_techniques?: number;
  total_alerts_90d?: number;
}

// ─── Risk Heatmap ─────────────────────────────────────────────────────────────

export interface RiskHeatmapCell {
  asset_type: string;
  severity: CyberSeverity;
  count: number;
  affected_asset_count: number;
  total_assets_of_type: number;
}

export interface RiskHeatmapData {
  cells: RiskHeatmapCell[];
  asset_types: string[];
  total_vulnerabilities: number;
  generated_at: string;
}

// ─── Rule Content Types ────────────────────────────────────────────────────────

export interface RuleCondition {
  field: string;
  operator: string;
  value: string;
}

export interface RuleSelection {
  name: string;
  conditions: RuleCondition[];
}

export interface SigmaRuleContent {
  selections: RuleSelection[];
  filters?: RuleSelection[];
  condition: string;
  timeframe?: string;
  threshold?: number;
}

export interface ThresholdRuleContent {
  filter_conditions: RuleCondition[];
  group_by?: string;
  metric: 'count' | 'sum' | 'distinct';
  metric_field?: string;
  threshold: number;
  window: string;
}

export interface AnomalyRuleContent {
  metric: string;
  group_by?: string;
  window: string;
  z_score_threshold: number;
  min_baseline_samples: number;
  direction: 'above' | 'below' | 'both';
}

export interface CorrelationEventType {
  name: string;
  conditions: RuleCondition[];
}

export interface CorrelationRuleContent {
  event_types: CorrelationEventType[];
  sequence: string[];
  group_by?: string;
  window: string;
  min_count?: Record<string, number>;
}

// ─── vCISO Governance — Risk Register ────────────────────────────────────────

export type RiskLikelihood = 'rare' | 'unlikely' | 'possible' | 'likely' | 'almost_certain';
export type RiskImpact = 'negligible' | 'minor' | 'moderate' | 'major' | 'catastrophic';
export type RiskStatus = 'identified' | 'assessed' | 'mitigating' | 'accepted' | 'closed';
export type RiskTreatment = 'mitigate' | 'transfer' | 'accept' | 'avoid';

export interface VCISORiskEntry {
  id: string;
  tenant_id: string;
  title: string;
  description: string;
  category: string;
  inherent_score: number;
  residual_score: number;
  likelihood: RiskLikelihood;
  impact: RiskImpact;
  status: RiskStatus;
  treatment: RiskTreatment;
  owner_id: string;
  owner_name: string;
  review_date: string;
  business_services: string[];
  department: string;
  treatment_plan: string;
  controls: string[];
  acceptance_rationale?: string;
  acceptance_approved_by?: string;
  acceptance_approved_by_name?: string;
  acceptance_expiry?: string;
  tags: string[];
  created_at: string;
  updated_at: string;
}

export interface VCISORiskStats {
  total: number;
  by_status: Record<string, number>;
  by_treatment: Record<string, number>;
  by_likelihood: Record<string, number>;
  by_impact: Record<string, number>;
  avg_inherent_score: number;
  avg_residual_score: number;
  overdue_reviews: number;
  accepted_count: number;
}

// ─── vCISO Governance — Policies ─────────────────────────────────────────────

export type PolicyStatus = 'draft' | 'review' | 'approved' | 'published' | 'retired';
export type PolicyDomain =
  | 'access_control'
  | 'incident_response'
  | 'data_protection'
  | 'acceptable_use'
  | 'business_continuity'
  | 'risk_management'
  | 'vendor_management'
  | 'change_management'
  | 'security_awareness'
  | 'network_security'
  | 'encryption'
  | 'physical_security'
  | 'other';

export interface VCISOPolicy {
  id: string;
  tenant_id: string;
  title: string;
  domain: PolicyDomain;
  version: string;
  status: PolicyStatus;
  content: string;
  owner_id: string;
  owner_name: string;
  reviewer_id?: string;
  reviewer_name?: string;
  approved_by?: string;
  approved_by_name?: string;
  approved_at?: string;
  review_due: string;
  last_reviewed_at?: string;
  tags: string[];
  exceptions_count: number;
  created_at: string;
  updated_at: string;
}

export type PolicyExceptionStatus = 'pending' | 'approved' | 'rejected' | 'expired';

export interface VCISOPolicyException {
  id: string;
  tenant_id: string;
  policy_id: string;
  policy_title: string;
  title: string;
  description: string;
  justification: string;
  compensating_controls: string;
  status: PolicyExceptionStatus;
  requested_by: string;
  requested_by_name: string;
  approved_by?: string;
  approved_by_name?: string;
  decision_notes?: string;
  expires_at: string;
  created_at: string;
  updated_at: string;
}

// ─── vCISO Governance — Third-Party Risk ─────────────────────────────────────

export type VendorRiskTier = 'critical' | 'high' | 'medium' | 'low';
export type VendorStatus = 'active' | 'onboarding' | 'under_review' | 'offboarding' | 'terminated';

export interface VCISOVendor {
  id: string;
  tenant_id: string;
  name: string;
  category: string;
  risk_tier: VendorRiskTier;
  status: VendorStatus;
  risk_score: number;
  last_assessment_date?: string;
  next_review_date: string;
  contact_name?: string;
  contact_email?: string;
  services_provided: string[];
  data_shared: string[];
  compliance_frameworks: string[];
  controls_met: number;
  controls_total: number;
  open_findings: number;
  created_at: string;
  updated_at: string;
}

export type QuestionnaireStatus = 'draft' | 'sent' | 'in_progress' | 'completed' | 'expired';
export type QuestionnaireType = 'vendor' | 'customer' | 'audit' | 'internal';

export interface VCISOQuestionnaire {
  id: string;
  tenant_id: string;
  title: string;
  type: QuestionnaireType;
  status: QuestionnaireStatus;
  vendor_id?: string;
  vendor_name?: string;
  total_questions: number;
  answered_questions: number;
  due_date: string;
  completed_at?: string;
  score?: number;
  assigned_to?: string;
  assigned_to_name?: string;
  created_at: string;
  updated_at: string;
}

// ─── vCISO Governance — Evidence ─────────────────────────────────────────────

export type EvidenceType = 'screenshot' | 'log' | 'config' | 'report' | 'policy' | 'certificate' | 'other';
export type EvidenceSource = 'manual' | 'automated';
export type EvidenceStatus = 'current' | 'stale' | 'expired';

export interface VCISOEvidence {
  id: string;
  tenant_id: string;
  title: string;
  description: string;
  type: EvidenceType;
  source: EvidenceSource;
  status: EvidenceStatus;
  frameworks: string[];
  control_ids: string[];
  file_name?: string;
  file_size?: number;
  file_url?: string;
  collected_at: string;
  expires_at?: string;
  collector_name?: string;
  last_verified_at?: string;
  created_at: string;
  updated_at: string;
}

export interface VCISOEvidenceStats {
  total: number;
  by_status: Record<string, number>;
  by_type: Record<string, number>;
  by_source: Record<string, number>;
  stale_count: number;
  expired_count: number;
  frameworks_covered: number;
  controls_with_evidence: number;
  controls_without_evidence: number;
}

// ─── vCISO Governance — Maturity & Benchmarking ──────────────────────────────

export type MaturityCategory = 'people' | 'process' | 'technology' | 'governance' | 'security' | 'operations';

export interface VCISOMaturityDimension {
  name: string;
  category: MaturityCategory;
  current_level: number;
  target_level: number;
  score: number;
  findings: string[];
  recommendations: string[];
}

export interface VCISOMaturityAssessment {
  id: string;
  tenant_id: string;
  framework: string;
  status: 'draft' | 'in_progress' | 'completed';
  overall_score: number;
  overall_level: number;
  dimensions: VCISOMaturityDimension[];
  assessor_name?: string;
  assessed_at: string;
  created_at: string;
  updated_at: string;
}

export interface VCISOBenchmark {
  dimension: string;
  category: MaturityCategory;
  organization_score: number;
  industry_average: number;
  industry_top_quartile: number;
  peer_average: number;
  gap: number;
}

export type BudgetItemStatus = 'proposed' | 'approved' | 'in_progress' | 'completed' | 'deferred';
export type BudgetItemType = 'capex' | 'opex';

export interface VCISOBudgetItem {
  id: string;
  tenant_id: string;
  title: string;
  category: string;
  type: BudgetItemType;
  amount: number;
  currency: string;
  status: BudgetItemStatus;
  risk_reduction_estimate: number;
  priority: number;
  justification: string;
  linked_risk_ids: string[];
  linked_recommendation_ids: string[];
  fiscal_year: string;
  quarter?: string;
  owner_name?: string;
  created_at: string;
  updated_at: string;
}

export interface VCISOBudgetSummary {
  total_proposed: number;
  total_approved: number;
  total_spent: number;
  total_risk_reduction: number;
  by_category: Record<string, number>;
  by_status: Record<string, number>;
  currency: string;
}

// ─── vCISO Governance — Awareness & IAM ──────────────────────────────────────

export type AwarenessProgramType = 'training' | 'phishing_simulation' | 'policy_attestation';
export type AwarenessProgramStatus = 'scheduled' | 'active' | 'completed';

export interface VCISOAwarenessProgram {
  id: string;
  tenant_id: string;
  name: string;
  type: AwarenessProgramType;
  status: AwarenessProgramStatus;
  total_users: number;
  completed_users: number;
  passed_users: number;
  failed_users: number;
  completion_rate: number;
  pass_rate: number;
  start_date: string;
  end_date: string;
  created_at: string;
  updated_at: string;
}

export type IAMFindingType = 'mfa_gap' | 'orphaned_account' | 'privileged_access' | 'sod_violation' | 'stale_access' | 'excessive_permissions';
export type IAMFindingStatus = 'open' | 'in_progress' | 'resolved' | 'accepted';

export interface VCISOIAMFinding {
  id: string;
  tenant_id: string;
  type: IAMFindingType;
  severity: CyberSeverity;
  title: string;
  description: string;
  affected_users: number;
  status: IAMFindingStatus;
  remediation?: string;
  discovered_at: string;
  resolved_at?: string;
  created_at: string;
  updated_at: string;
}

export interface VCISOIAMSummary {
  total_findings: number;
  by_type: Record<string, number>;
  by_severity: Record<string, number>;
  mfa_coverage_percent: number;
  privileged_accounts: number;
  orphaned_accounts: number;
  stale_access_count: number;
}

// ─── vCISO Governance — Incident Readiness ───────────────────────────────────

export type EscalationTriggerType = 'severity' | 'time' | 'count' | 'custom';
export type EscalationTarget = 'management' | 'legal' | 'regulator' | 'board' | 'custom';

export interface VCISOEscalationRule {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  trigger_type: EscalationTriggerType;
  trigger_condition: string;
  escalation_target: EscalationTarget;
  target_contacts: string[];
  notification_channels: string[];
  enabled: boolean;
  last_triggered_at?: string;
  trigger_count: number;
  created_at: string;
  updated_at: string;
}

export type PlaybookStatus = 'draft' | 'approved' | 'tested' | 'retired';
export type SimulationResult = 'pass' | 'partial' | 'fail';

export interface VCISOPlaybook {
  id: string;
  tenant_id: string;
  name: string;
  scenario: string;
  status: PlaybookStatus;
  last_tested_at?: string;
  next_test_date: string;
  owner_id: string;
  owner_name: string;
  steps_count: number;
  dependencies: string[];
  rto_hours?: number;
  rpo_hours?: number;
  last_simulation_result?: SimulationResult;
  created_at: string;
  updated_at: string;
}

// ─── vCISO Governance — Compliance Deep Dive ─────────────────────────────────

export type ObligationType = 'legal' | 'regulatory' | 'contractual' | 'industry_standard';
export type ObligationStatus = 'compliant' | 'partially_compliant' | 'non_compliant' | 'not_assessed';

export interface VCISORegulatoryObligation {
  id: string;
  tenant_id: string;
  name: string;
  type: ObligationType;
  jurisdiction: string;
  description: string;
  requirements: string[];
  status: ObligationStatus;
  mapped_controls: number;
  total_requirements: number;
  met_requirements: number;
  owner_id?: string;
  owner_name?: string;
  effective_date: string;
  review_date: string;
  created_at: string;
  updated_at: string;
}

export type ControlTestResult = 'effective' | 'partially_effective' | 'ineffective' | 'not_tested';
export type ControlTestType = 'design' | 'operating_effectiveness';

export interface VCISOControlTest {
  id: string;
  tenant_id: string;
  control_id: string;
  control_name: string;
  framework: string;
  test_type: ControlTestType;
  result: ControlTestResult;
  tester_name: string;
  test_date: string;
  next_test_date: string;
  findings: string;
  evidence_ids: string[];
  created_at: string;
  updated_at: string;
}

export type ControlFailureImpact = 'critical' | 'high' | 'medium' | 'low';

export interface VCISOControlDependency {
  control_id: string;
  control_name: string;
  framework: string;
  depends_on: string[];
  depended_by: string[];
  risk_domains: string[];
  compliance_domains: string[];
  failure_impact: ControlFailureImpact;
}

// ─── vCISO Governance — Integrations ─────────────────────────────────────────

export type IntegrationType = 'ticketing' | 'cloud_security' | 'asset_management' | 'data_protection' | 'siem' | 'iam';
export type IntegrationStatus = 'connected' | 'disconnected' | 'error' | 'pending';
export type IntegrationHealth = 'healthy' | 'degraded' | 'unavailable';

export interface VCISOIntegration {
  id: string;
  tenant_id: string;
  name: string;
  type: IntegrationType;
  provider: string;
  status: IntegrationStatus;
  last_sync_at?: string;
  sync_frequency: string;
  items_synced: number;
  config: Record<string, unknown>;
  health_status: IntegrationHealth;
  error_message?: string;
  created_at: string;
  updated_at: string;
}

// ─── vCISO Governance — Workflows ────────────────────────────────────────────

export type OwnershipStatus = 'assigned' | 'pending_review' | 'reviewed';

export interface VCISOControlOwnership {
  id: string;
  tenant_id: string;
  control_id: string;
  control_name: string;
  framework: string;
  owner_id: string;
  owner_name: string;
  delegate_id?: string;
  delegate_name?: string;
  status: OwnershipStatus;
  last_reviewed_at?: string;
  next_review_date: string;
  created_at: string;
  updated_at: string;
}

export type ApprovalRequestType = 'risk_acceptance' | 'policy_exception' | 'remediation' | 'budget' | 'vendor_onboarding';
export type ApprovalRequestStatus = 'pending' | 'approved' | 'rejected' | 'escalated';
export type ApprovalPriority = 'critical' | 'high' | 'medium' | 'low';

export interface VCISOApprovalRequest {
  id: string;
  tenant_id: string;
  type: ApprovalRequestType;
  title: string;
  description: string;
  status: ApprovalRequestStatus;
  requested_by: string;
  requested_by_name: string;
  approver_id: string;
  approver_name: string;
  priority: ApprovalPriority;
  decision_notes?: string;
  decided_at?: string;
  deadline: string;
  linked_entity_type: string;
  linked_entity_id: string;
  created_at: string;
  updated_at: string;
}

// ─── Export ───────────────────────────────────────────────────────────────────

export interface ExportJob {
  job_id: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  progress?: number;
  download_url?: string;
  error?: string;
  created_at: string;
  completed_at?: string;
}
