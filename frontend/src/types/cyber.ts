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

export interface VCISOChatResponse {
  conversation_id: string;
  message_id: string;
  response: VCISOResponsePayload;
  intent: string;
  confidence: number;
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
