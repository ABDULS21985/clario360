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
  last_scan_id?: string;
  // Computed fields from backend JOINs
  open_vulnerability_count?: number;
  vulnerability_count?: number;
  critical_vuln_count?: number;
  high_vuln_count?: number;
  alert_count?: number;
  highest_vulnerability_severity?: string;
  relationship_count?: number;
  metadata?: Record<string, unknown>;
  created_by?: string;
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
  config?: Record<string, unknown>;
  status: 'running' | 'completed' | 'failed' | 'cancelled';
  assets_discovered: number;
  assets_found: number;  // alias for assets_discovered
  assets_new: number;
  assets_updated: number;
  error_count: number;
  errors?: string[];
  target?: string;       // derived from config.targets
  error?: string;        // first error string
  started_at: string;
  completed_at?: string;
  duration_ms?: number;
  created_by: string;
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
  type: IndicatorType;
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
  rule_name?: string;
  rule_type?: 'sigma' | 'threshold' | 'correlation' | 'anomaly';
  asset_id?: string;
  asset_ids: string[];
  asset_name?: string;
  asset_ip_address?: string;
  asset_hostname?: string;
  asset_os?: string;
  asset_owner?: string;
  asset_criticality?: Criticality;
  assigned_to?: string;
  assigned_to_name?: string;
  assigned_to_email?: string;
  assigned_at?: string;
  escalated_to?: string;
  escalated_to_name?: string;
  escalated_to_email?: string;
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
}

export interface AlertStats {
  total: number;
  by_severity: NamedCount[];
  by_status: NamedCount[];
  by_rule: NamedCount[];
  by_rule_type: NamedCount[];
  by_technique: NamedCount[];
  open_count: number;
  resolved_count: number;
  mttr_hours: number;
  mtta_hours: number;
  false_positive_rate: number;
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

export type ThreatStatus = 'active' | 'contained' | 'eradicated' | 'monitoring' | 'closed';
export type ThreatSeverity = Exclude<CyberSeverity, 'info'>;
export type ThreatType =
  | 'malware'
  | 'phishing'
  | 'apt'
  | 'ransomware'
  | 'ddos'
  | 'insider_threat'
  | 'supply_chain'
  | 'zero_day'
  | 'brute_force'
  | 'other';

export type IndicatorType =
  | 'ip'
  | 'domain'
  | 'url'
  | 'email'
  | 'file_hash_md5'
  | 'file_hash_sha1'
  | 'file_hash_sha256'
  | 'certificate'
  | 'registry_key'
  | 'user_agent'
  | 'cidr';

export type IndicatorSource = 'manual' | 'stix_feed' | 'osint' | 'internal' | 'vendor';

export interface NamedCount {
  name: string;
  count: number;
}

export interface ThreatIndicator {
  id: string;
  tenant_id: string;
  threat_id?: string;
  threat_name?: string;
  threat_type?: ThreatType;
  threat_status?: ThreatStatus;
  type: IndicatorType;
  value: string;
  description: string;
  severity: ThreatSeverity;
  source: IndicatorSource | string;
  confidence: number;
  active: boolean;
  first_seen_at: string;
  last_seen_at: string;
  expires_at?: string;
  tags: string[];
  metadata?: Record<string, unknown>;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export interface Threat {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  type: ThreatType;
  severity: ThreatSeverity;
  status: ThreatStatus;
  threat_actor?: string;
  campaign?: string;
  mitre_tactic_ids: string[];
  mitre_technique_ids: string[];
  indicator_count: number;
  affected_asset_count: number;
  alert_count: number;
  first_seen_at: string;
  last_seen_at: string;
  contained_at?: string;
  indicators?: ThreatIndicator[];
  tags: string[];
  metadata?: Record<string, unknown>;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export interface ThreatStats {
  total: number;
  active: number;
  indicators_total: number;
  contained_this_month: number;
  by_type: NamedCount[];
  by_status: NamedCount[];
  by_severity: NamedCount[];
}

export interface ThreatTrendPoint {
  date: string;
  total: number;
  active: number;
  contained: number;
}

export interface ThreatTimelineEntry {
  id: string;
  kind: string;
  title: string;
  description?: string;
  timestamp: string;
  variant?: 'default' | 'success' | 'warning' | 'error';
}

export interface CreateIndicatorInput {
  type: IndicatorType;
  value: string;
  severity: ThreatSeverity;
  confidence: number;
  source?: string;
  description?: string;
  tags?: string[];
}

export interface CreateThreatInput {
  name: string;
  type: ThreatType;
  severity: ThreatSeverity;
  description?: string;
  threat_actor?: string;
  campaign?: string;
  mitre_tactic_ids?: string[];
  mitre_technique_ids?: string[];
  tags?: string[];
  indicators?: CreateIndicatorInput[];
}

export interface IndicatorCheckResult {
  value: string;
  indicators: ThreatIndicator[];
}

export interface StandaloneIndicatorInput {
  type: IndicatorType;
  value: string;
  severity: ThreatSeverity;
  source: IndicatorSource;
  confidence: number;
  description?: string;
  threat_id?: string;
  expires_at?: string;
  tags?: string[];
  metadata?: Record<string, unknown>;
}

export interface IndicatorStats {
  total: number;
  active: number;
  expiring_soon: number;
  by_source: NamedCount[];
}

export interface IndicatorEnrichment {
  dns?: Record<string, unknown>;
  geolocation?: Record<string, unknown>;
  cves?: string[];
  whois?: Record<string, unknown>;
  reputation_score?: number;
  metadata?: Record<string, unknown>;
}

export interface IndicatorDetectionMatch {
  id: string;
  kind: string;
  title: string;
  description?: string;
  severity?: ThreatSeverity;
  status?: string;
  asset_id?: string;
  asset_name?: string;
  match_field?: string;
  match_value?: string;
  timestamp: string;
}

export type ThreatFeedType = 'stix' | 'taxii' | 'misp' | 'csv_url' | 'manual';
export type ThreatFeedAuthType = 'none' | 'api_key' | 'basic' | 'certificate';
export type ThreatFeedInterval = 'hourly' | 'every_6h' | 'daily' | 'weekly' | 'manual';
export type ThreatFeedStatus = 'active' | 'paused' | 'error';

export interface ThreatFeedConfig {
  id: string;
  tenant_id: string;
  name: string;
  type: ThreatFeedType;
  url?: string;
  auth_type: ThreatFeedAuthType;
  auth_config?: Record<string, unknown>;
  sync_interval: ThreatFeedInterval;
  default_severity: ThreatSeverity;
  default_confidence: number;
  default_tags: string[];
  indicator_types: string[];
  enabled: boolean;
  status: ThreatFeedStatus;
  last_sync_at?: string;
  last_sync_status?: string;
  last_error?: string;
  next_sync_at?: string;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export interface ThreatFeedSyncHistory {
  id: string;
  tenant_id: string;
  feed_id: string;
  status: string;
  indicators_parsed: number;
  indicators_imported: number;
  indicators_skipped: number;
  indicators_failed: number;
  duration_ms: number;
  error_message?: string;
  metadata?: Record<string, unknown>;
  started_at: string;
  completed_at?: string;
}

export interface ThreatFeedSyncSummary {
  feed_id: string;
  feed_name: string;
  indicators_parsed: number;
  indicators_imported: number;
  indicators_skipped: number;
  indicators_failed: number;
}

export interface ThreatFeedConfigInput {
  name: string;
  type: ThreatFeedType;
  url?: string;
  auth_type: ThreatFeedAuthType;
  auth_config?: Record<string, unknown>;
  sync_interval: ThreatFeedInterval;
  default_severity: ThreatSeverity;
  default_confidence: number;
  default_tags?: string[];
  indicator_types?: string[];
  enabled: boolean;
}

// ─── Detection Rule ───────────────────────────────────────────────────────────

export type DetectionRuleType = 'sigma' | 'threshold' | 'correlation' | 'anomaly';

export interface DetectionRule {
  id: string;
  tenant_id?: string | null;
  name: string;
  description: string;
  rule_type: DetectionRuleType;
  type?: DetectionRuleType;
  severity: CyberSeverity;
  enabled: boolean;
  mitre_tactic_ids: string[];
  mitre_technique_ids: string[];
  trigger_count: number;
  false_positive_count: number;
  true_positive_count: number;
  false_positive_rate?: number;
  true_positive_rate?: number;
  last_triggered_at?: string;
  last_triggered?: string;
  rule_content: SigmaRuleContent | ThresholdRuleContent | AnomalyRuleContent | CorrelationRuleContent | Record<string, unknown>;
  base_confidence: number;
  tp_count?: number;
  fp_count?: number;
  is_template: boolean;
  tags: string[];
  template_id?: string | null;
  created_by?: string | null;
  created_at: string;
  updated_at: string;
}

export interface RuleTemplate {
  id: string;
  name: string;
  description: string;
  rule_type: DetectionRuleType;
  type?: DetectionRuleType;
  severity: CyberSeverity;
  mitre_tactic_ids?: string[];
  mitre_technique_ids: string[];
  rule_content?: Record<string, unknown>;
  tags?: string[];
  template_id?: string | null;
}

export interface DetectionRuleStats {
  total: number;
  active: number;
  by_type: NamedCount[];
  by_severity: NamedCount[];
  true_positive_rate: number;
  alerts_last_30_days: number;
}

export interface DetectionRuleAlertTrendPoint {
  date: string;
  count: number;
}

export interface DetectionRuleTopAsset {
  asset_id?: string | null;
  asset_name: string;
  alert_count: number;
}

export interface DetectionRulePerformance {
  alerts_last_30_days: number;
  alerts_last_90_days: number;
  severity_distribution: NamedCount[];
  alert_trend: DetectionRuleAlertTrendPoint[];
  top_assets: DetectionRuleTopAsset[];
  true_positive_rate: number;
  false_positive_rate: number;
}

export interface DetectionRuleTestMatch {
  rule_id: string;
  events: Array<Record<string, unknown>>;
  match_details: Record<string, unknown>;
  timestamp: string;
}

export interface DetectionRuleTestResult {
  matches: DetectionRuleTestMatch[];
  count: number;
}

// ─── CTEM ─────────────────────────────────────────────────────────────────────

export type CTEMPhase = 'scoping' | 'discovery' | 'prioritization' | 'validation' | 'mobilization';
export type CTEMPhaseStatus = 'pending' | 'running' | 'completed' | 'failed' | 'skipped' | 'cancelled';

/** Backend PhaseProgress — the raw shape returned from the API as a map value. */
export interface PhaseProgress {
  status: CTEMPhaseStatus;
  started_at?: string;
  completed_at?: string;
  items_processed: number;
  items_total: number;
  errors?: string[];
  result?: Record<string, unknown>;
}

/** Frontend-friendly phase info used by PhaseStepper (includes the phase key). */
export interface CTEMPhaseInfo {
  phase: CTEMPhase;
  status: CTEMPhaseStatus;
  started_at?: string;
  completed_at?: string;
  progress_percent?: number;
  error?: string;
}

/**
 * Normalizes phases from the backend map[string]PhaseProgress (JSON object)
 * into the CTEMPhaseInfo[] array expected by frontend components.
 */
export function normalizeCTEMPhases(
  phases: Record<string, PhaseProgress> | CTEMPhaseInfo[] | null | undefined,
): CTEMPhaseInfo[] {
  if (!phases) return [];
  // Already an array — return as-is
  if (Array.isArray(phases)) return phases;
  // Object map — convert to array
  return Object.entries(phases).map(([key, progress]) => ({
    phase: key as CTEMPhase,
    status: progress.status,
    started_at: progress.started_at,
    completed_at: progress.completed_at,
    progress_percent:
      progress.items_total > 0
        ? Math.round((progress.items_processed / progress.items_total) * 100)
        : undefined,
    error: progress.errors?.[0],
  }));
}

export type CTEMFindingType =
  | 'vulnerability'
  | 'misconfiguration'
  | 'attack_path'
  | 'exposure'
  | 'weak_credential'
  | 'missing_patch'
  | 'expired_certificate'
  | 'insecure_protocol';

export type CTEMFindingCategory = 'technical' | 'configuration' | 'architectural' | 'operational';

export type CTEMValidationStatus =
  | 'pending'
  | 'validated'
  | 'compensated'
  | 'not_exploitable'
  | 'requires_manual';

export type CTEMRemediationType =
  | 'patch'
  | 'configuration'
  | 'architecture'
  | 'upgrade'
  | 'decommission'
  | 'accept_risk';

export type CTEMRemediationEffort = 'low' | 'medium' | 'high';

export interface CTEMFinding {
  id: string;
  tenant_id: string;
  assessment_id: string;
  type: CTEMFindingType;
  category: CTEMFindingCategory;
  severity: CyberSeverity;
  title: string;
  description: string;
  evidence?: Record<string, unknown>;
  affected_asset_ids?: string[];
  affected_asset_count: number;
  primary_asset_id?: string;
  vulnerability_ids?: string[];
  cve_ids?: string[];
  business_impact_score: number;
  business_impact_factors?: Record<string, unknown>;
  exploitability_score: number;
  exploitability_factors?: Record<string, unknown>;
  priority_score: number;
  priority_group: number;
  priority_rank?: number;
  validation_status: CTEMValidationStatus;
  compensating_controls?: string[];
  validation_notes?: string;
  validated_at?: string;
  remediation_type?: CTEMRemediationType;
  remediation_description?: string;
  remediation_effort?: CTEMRemediationEffort;
  remediation_group_id?: string;
  estimated_days?: number;
  status: 'open' | 'in_remediation' | 'remediated' | 'accepted_risk' | 'false_positive' | 'deferred';
  status_changed_by?: string;
  status_changed_at?: string;
  status_notes?: string;
  attack_path?: string[] | Record<string, unknown>;
  attack_path_length?: number;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;

  // Legacy frontend aliases (for backward compatibility with existing components)
  asset_id?: string;
  asset_name?: string;
  cvss_score?: number;
  exploit_available?: boolean;
  remediation_steps?: string[];
}

export interface CTEMAssessment {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  status: 'created' | 'scoping' | 'discovery' | 'prioritizing' | 'validating' | 'mobilizing' | 'completed' | 'failed' | 'cancelled';
  current_phase?: CTEMPhase;
  /** Backend returns map[string]PhaseProgress; use normalizeCTEMPhases() before rendering. */
  phases: Record<string, PhaseProgress> | CTEMPhaseInfo[];
  scope: {
    asset_types?: string[];
    asset_tags?: string[];
    asset_ids?: string[];
    departments?: string[];
    cidr_ranges?: string[];
    exclude_asset_ids?: string[];
  };
  resolved_asset_ids?: string[];
  resolved_asset_count?: number;
  findings_summary?: {
    critical: number;
    high: number;
    medium: number;
    low: number;
    total: number;
  };
  exposure_score?: number;
  score_breakdown?: Record<string, unknown>;
  findings?: CTEMFinding[];
  error_message?: string;
  error_phase?: string;
  scheduled?: boolean;
  schedule_cron?: string;
  parent_assessment_id?: string;
  tags?: string[];
  duration_ms?: number;
  created_by?: string;
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
  | 'execution_failed'
  | 'verification_pending'
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
  users_affected: number;
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

export interface RollbackResult {
  success: boolean;
  steps_reverted: number;
  duration_ms: number;
  error?: string;
}

export interface RemediationAction {
  id: string;
  tenant_id: string;
  alert_id?: string;
  vulnerability_id?: string;
  assessment_id?: string;
  ctem_finding_id?: string;
  remediation_group_id?: string;
  type: string;
  severity: CyberSeverity;
  title: string;
  description: string;
  status: RemediationStatus;
  plan: RemediationPlan;
  affected_asset_ids: string[];
  affected_asset_count?: number;
  execution_mode: string;
  // Approval chain
  submitted_by?: string;
  submitted_at?: string;
  requires_approval_from?: string;
  approved_by?: string;
  approved_at?: string;
  approval_notes?: string;
  rejected_by?: string;
  rejected_at?: string;
  rejection_reason?: string;
  revision_requested?: boolean;
  // Dry run
  dry_run_at?: string;
  dry_run_duration_ms?: number;
  dry_run_result?: DryRunResult;
  // Execution
  pre_execution_state?: unknown;
  executed_by?: string;
  execution_started_at?: string;
  execution_completed_at?: string;
  execution_duration_ms?: number;
  execution_result?: ExecutionResult;
  // Verification
  verified_by?: string;
  verified_at?: string;
  verification_result?: VerificationResult;
  // Rollback
  rollback_result?: RollbackResult;
  rollback_reason?: string;
  rollback_approved_by?: string;
  rolled_back_at?: string;
  rollback_deadline?: string;
  // Workflow
  workflow_instance_id?: string;
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

/** Matches backend cyber/model/remediation.go RemediationStats (flat fields, not maps). */
export interface RemediationStats {
  total: number;
  draft: number;
  pending_approval: number;
  approved: number;
  dry_run_completed: number;
  executing: number;
  executed: number;
  verified: number;
  verification_failed: number;
  rolled_back: number;
  failed: number;
  closed: number;
  avg_execution_hours: number;
  verification_success_rate: number;
  rollback_rate: number;
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

export interface MITRETacticItem {
  id: string;
  name: string;
  short_name: string;
  description: string;
}

export interface MITRETechniqueItem {
  id: string;
  name: string;
  description: string;
  tactic_ids: string[];
  platforms: string[];
  data_sources: string[];
}

export interface MITRETechniqueCoverage {
  technique_id: string;
  technique_name: string;
  tactic_ids: string[];
  tactic_id?: string;
  tactic_name?: string;
  rule_count: number;
  rule_names?: string[];
  alert_count: number;
  threat_count: number;
  active_threat_count: number;
  has_detection: boolean;
  coverage_state: 'covered' | 'noisy' | 'gap' | 'idle';
  high_fp_rule_count: number;
  last_alert_at?: string;
  description: string;
  platforms: string[];
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
  active_techniques: number;
  passive_techniques: number;
  critical_gap_count: number;
}

export interface MITRERuleReference {
  id: string;
  name: string;
  rule_type: DetectionRuleType;
  severity: CyberSeverity;
  enabled: boolean;
  trigger_count: number;
  true_positive_count: number;
  false_positive_count: number;
  last_triggered_at?: string;
}

export interface MITREThreatReference {
  id: string;
  name: string;
  type: ThreatType;
  severity: CyberSeverity;
  status: ThreatStatus;
  last_seen_at: string;
}

export interface MITREAlertReference {
  id: string;
  title: string;
  severity: CyberSeverity;
  status: AlertStatus;
  confidence_score: number;
  asset_name?: string;
  created_at: string;
}

export interface MITRETechniqueDetail {
  id: string;
  name: string;
  description: string;
  tactic_ids: string[];
  platforms: string[];
  data_sources: string[];
  coverage_state: 'covered' | 'noisy' | 'gap' | 'idle';
  rule_count: number;
  alert_count: number;
  threat_count: number;
  active_threat_count: number;
  high_fp_rule_count: number;
  last_alert_at?: string;
  linked_rules: MITRERuleReference[];
  linked_threats: MITREThreatReference[];
  recent_alerts: MITREAlertReference[];
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
  min_failed_count?: number;
}

// ─── vCISO Governance — Risk Register ────────────────────────────────────────

export type RiskLikelihood = 'low' | 'medium' | 'high' | 'critical';
export type RiskImpact = 'low' | 'medium' | 'high' | 'critical';
export type RiskStatus = 'open' | 'mitigated' | 'accepted' | 'closed';
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
  verified_by?: string;
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
  id: string;
  tenant_id: string;
  dimension: string;
  category: MaturityCategory;
  organization_score: number;
  industry_average: number;
  industry_top_quartile: number;
  peer_average: number;
  gap: number;
  framework: string;
  created_at: string;
  updated_at: string;
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
  id: string;
  tenant_id: string;
  control_id: string;
  control_name: string;
  framework: string;
  depends_on: string[];
  depended_by: string[];
  risk_domains: string[];
  compliance_domains: string[];
  failure_impact: ControlFailureImpact;
  created_at: string;
  updated_at: string;
}

// ─── vCISO Governance — Integrations ─────────────────────────────────────────

export type CyberIntegrationType = 'ticketing' | 'cloud_security' | 'asset_management' | 'data_protection' | 'siem' | 'iam';
export type CyberIntegrationStatus = 'connected' | 'disconnected' | 'error' | 'pending';
export type IntegrationHealth = 'healthy' | 'degraded' | 'unavailable';

export interface VCISOIntegration {
  id: string;
  tenant_id: string;
  name: string;
  type: CyberIntegrationType;
  provider: string;
  status: CyberIntegrationStatus;
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

// ─── DSPM Access Intelligence ─────────────────────────────────────────────────

export type IdentityType = 'user' | 'service_account' | 'role' | 'group' | 'api_key' | 'application';
export type DataClassification = 'public' | 'internal' | 'confidential' | 'restricted';
export type PermissionType = 'read' | 'write' | 'admin' | 'delete' | 'create' | 'alter' | 'execute' | 'full_control';
export type PermissionSource = 'direct_grant' | 'role_inherited' | 'group_inherited' | 'policy_inherited' | 'wildcard_grant';
export type AccessMappingStatus = 'active' | 'revoked' | 'expired' | 'pending_review';
export type RiskLevel = 'low' | 'medium' | 'high' | 'critical';
export type IdentityProfileStatus = 'active' | 'inactive' | 'under_review' | 'remediated';
export type AccessPolicyType = 'max_idle_days' | 'classification_restrict' | 'separation_of_duties' | 'time_bound_access' | 'blast_radius_limit' | 'periodic_review';
export type PolicyEnforcement = 'alert' | 'block' | 'auto_remediate';
export type RecommendationType = 'revoke' | 'downgrade' | 'time_bound' | 'review';

export interface AccessMapping {
  id: string;
  tenant_id: string;
  identity_type: IdentityType;
  identity_id: string;
  identity_name: string;
  identity_source: string;
  data_asset_id: string;
  data_asset_name: string;
  data_classification: DataClassification;
  permission_type: PermissionType;
  permission_source: PermissionSource;
  permission_path: string[];
  is_wildcard: boolean;
  last_used_at: string | null;
  usage_count_30d: number;
  usage_count_90d: number;
  is_stale: boolean;
  sensitivity_weight: number;
  access_risk_score: number;
  status: AccessMappingStatus;
  expires_at: string | null;
  discovered_at: string;
  last_verified_at: string;
  created_at: string;
  updated_at: string;
}

export interface IdentityProfile {
  id: string;
  tenant_id: string;
  identity_type: IdentityType;
  identity_id: string;
  identity_name: string;
  identity_email: string;
  identity_source: string;
  total_assets_accessible: number;
  sensitive_assets_count: number;
  permission_count: number;
  overprivileged_count: number;
  stale_permission_count: number;
  blast_radius_score: number;
  blast_radius_level: RiskLevel;
  access_risk_score: number;
  access_risk_level: RiskLevel;
  risk_factors: Record<string, unknown>[];
  last_activity_at: string | null;
  avg_daily_access_count: number;
  access_pattern_summary: Record<string, unknown> | null;
  recommendations: Record<string, unknown>[];
  status: IdentityProfileStatus;
  last_review_at: string | null;
  next_review_due: string | null;
  created_at: string;
  updated_at: string;
}

export interface AccessAuditEntry {
  id: string;
  tenant_id: string;
  identity_type: string;
  identity_id: string;
  data_asset_id: string;
  action: string;
  source_ip: string;
  query_hash: string;
  rows_affected: number | null;
  duration_ms: number | null;
  success: boolean;
  access_mapping_id: string | null;
  table_name: string;
  database_name: string;
  event_timestamp: string;
  created_at: string;
}

export interface AccessPolicy {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  policy_type: AccessPolicyType;
  rule_config: Record<string, unknown>;
  enforcement: PolicyEnforcement;
  severity: CyberSeverity;
  enabled: boolean;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface AssetExposure {
  data_asset_id: string;
  data_asset_name: string;
  data_classification: DataClassification;
  permission_type: PermissionType;
  sensitivity_weight: number;
  permission_breadth: number;
  weighted_score: number;
}

export interface EscalationPath {
  identity_id: string;
  identity_name: string;
  pattern: string;
  from_permission: string;
  to_permission: string;
  asset_id: string;
  asset_name: string;
  data_classification: DataClassification;
  mitre_technique: string;
  severity: CyberSeverity;
}

export interface BlastRadius {
  identity_id: string;
  identity_name: string;
  identity_type: IdentityType;
  total_assets_exposed: number;
  sensitive_assets: number;
  weighted_score: number;
  level: RiskLevel;
  exposed_classifications: Record<string, number>;
  top_risky_assets: AssetExposure[];
  escalation_paths: EscalationPath[];
  recommended_actions: string[];
}

export interface OverprivilegeResult {
  identity_id: string;
  identity_name: string;
  identity_type: IdentityType;
  data_asset_id: string;
  data_asset_name: string;
  data_classification: DataClassification;
  permission_type: PermissionType;
  last_used_at: string | null;
  usage_count_90d: number;
  sensitivity_weight: number;
  severity: CyberSeverity;
  confidence: number;
  recommendation: string;
  mitre_technique: string;
}

export interface StaleAccessResult {
  identity_id: string;
  identity_name: string;
  identity_type: IdentityType;
  stale_count: number;
  total_sensitivity_risk: number;
  mappings: AccessMapping[];
}

export interface CrossAssetResult {
  identity_id: string;
  identity_name: string;
  identity_type: IdentityType;
  distinct_assets: number;
  distinct_classifications: number;
  distinct_asset_types: number;
  breadth_score: number;
  classifications: string[];
  asset_types: string[];
}

export interface AccessRecommendation {
  type: RecommendationType;
  permission_id: string;
  asset_name: string;
  permission_type: PermissionType;
  data_classification: DataClassification;
  reason: string;
  impact: string;
  risk_reduction_estimate: number;
}

export interface PolicyViolation {
  policy_id: string;
  policy_name: string;
  policy_type: AccessPolicyType;
  identity_id: string;
  identity_name: string;
  identity_type: IdentityType;
  violation_type: string;
  severity: CyberSeverity;
  details: string;
  enforcement: PolicyEnforcement;
  action_taken: string;
}

export interface AccessDashboard {
  total_identities: number;
  high_risk_identities: number;
  overprivileged_mappings: number;
  stale_permissions: number;
  avg_blast_radius: number;
  policy_violations: number;
  total_mappings: number;
  active_mappings: number;
  risk_distribution: Record<string, number>;
  classification_access: Record<string, number>;
  top_risky_identities: IdentityProfile[];
}

// ─── DSPM Remediation Engine ─────────────────────────────────────────────────

export type DSPMFindingType =
  | 'posture_gap'
  | 'overprivileged_access'
  | 'stale_access'
  | 'classification_drift'
  | 'shadow_copy'
  | 'policy_violation'
  | 'encryption_missing'
  | 'exposure_risk'
  | 'pii_unprotected'
  | 'retention_expired'
  | 'blast_radius_excessive';

export type DSPMRemediationStatus =
  | 'open'
  | 'in_progress'
  | 'awaiting_approval'
  | 'completed'
  | 'failed'
  | 'cancelled'
  | 'rolled_back'
  | 'exception_granted';

export interface DSPMRemediation {
  id: string;
  tenant_id: string;
  finding_type: DSPMFindingType;
  finding_id?: string;
  data_asset_id?: string;
  data_asset_name?: string;
  identity_id?: string;
  playbook_id: string;
  title: string;
  description: string;
  severity: CyberSeverity;
  steps: DSPMRemediationStep[];
  current_step: number;
  total_steps: number;
  assigned_to?: string;
  assigned_team?: string;
  sla_due_at?: string;
  sla_breached: boolean;
  risk_score_before?: number;
  risk_score_after?: number;
  risk_reduction?: number;
  pre_action_state?: Record<string, unknown>;
  rollback_available: boolean;
  rolled_back: boolean;
  status: DSPMRemediationStatus;
  cyber_alert_id?: string;
  created_by?: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
  compliance_tags: string[];
}

export interface DSPMRemediationStep {
  step_id: string;
  order: number;
  action: string;
  description: string;
  params?: Record<string, unknown>;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'skipped';
  started_at?: string;
  completed_at?: string;
  result?: Record<string, unknown>;
  error?: string;
}

export interface DSPMRemediationStats {
  total_open: number;
  total_critical_open: number;
  total_in_progress: number;
  completed_last_7_days: number;
  sla_breaches: number;
  avg_resolution_hours: number;
  by_status: Record<string, number>;
  by_severity: Record<string, number>;
  by_finding_type: Record<string, number>;
  total_risk_reduction: number;
}

export interface DSPMRemediationDashboard {
  stats: DSPMRemediationStats;
  recent_remediations: DSPMRemediation[];
  burndown_data: { date: string; open: number; closed: number }[];
}

export interface DSPMRemediationHistory {
  id: string;
  tenant_id: string;
  remediation_id: string;
  action: string;
  actor_id?: string;
  actor_type: 'user' | 'system' | 'policy_engine' | 'scheduler';
  details: Record<string, unknown>;
  entry_hash: string;
  prev_hash?: string;
  created_at: string;
}

export interface DSPMStepResult {
  step_id: string;
  action: string;
  status: 'completed' | 'failed';
  started_at: string;
  completed_at?: string;
  duration_ms: number;
  result?: Record<string, unknown>;
  error?: string;
}

// ─── DSPM Data Policies ─────────────────────────────────────────────────────

export type DSPMPolicyCategory =
  | 'encryption'
  | 'classification'
  | 'retention'
  | 'exposure'
  | 'pii_protection'
  | 'access_review'
  | 'backup'
  | 'audit_logging';

export type DSPMPolicyEnforcement = 'alert' | 'auto_remediate' | 'block';

export interface DSPMDataPolicy {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  category: DSPMPolicyCategory;
  rule: Record<string, unknown>;
  enforcement: DSPMPolicyEnforcement;
  auto_playbook_id?: string;
  severity: CyberSeverity;
  scope_classification?: string[];
  scope_asset_types?: string[];
  enabled: boolean;
  last_evaluated_at?: string;
  violation_count: number;
  compliance_frameworks?: string[];
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export interface DSPMPolicyViolation {
  policy_id: string;
  policy_name: string;
  category: string;
  asset_id: string;
  asset_name: string;
  asset_type: string;
  classification: string;
  severity: string;
  description: string;
  enforcement: string;
  compliance_frameworks?: string[];
}

export interface DSPMPolicyImpact {
  total_assets_evaluated: number;
  violations_found: number;
  affected_assets: DSPMPolicyViolation[];
}

// ─── DSPM Risk Exceptions ───────────────────────────────────────────────────

export type DSPMExceptionType =
  | 'posture_finding'
  | 'policy_violation'
  | 'overprivileged_access'
  | 'exposure_risk'
  | 'encryption_gap';

export type DSPMApprovalStatus = 'pending' | 'approved' | 'rejected' | 'expired';

export interface DSPMRiskException {
  id: string;
  tenant_id: string;
  exception_type: DSPMExceptionType;
  remediation_id?: string;
  data_asset_id?: string;
  policy_id?: string;
  justification: string;
  business_reason?: string;
  compensating_controls?: string;
  risk_score: number;
  risk_level: string;
  requested_by: string;
  approved_by?: string;
  approval_status: DSPMApprovalStatus;
  approved_at?: string;
  rejection_reason?: string;
  expires_at: string;
  review_interval_days: number;
  next_review_at?: string;
  last_reviewed_at?: string;
  review_count: number;
  status: 'active' | 'expired' | 'revoked' | 'superseded';
  created_at: string;
  updated_at: string;
}

// ─── DSPM Advanced Intelligence ──────────────────────────────────────────────

export type LineageEdgeType =
  | 'etl_pipeline'
  | 'replication'
  | 'api_transfer'
  | 'manual_copy'
  | 'query_derived'
  | 'stream'
  | 'export'
  | 'inferred';

export type LineageEdgeStatus = 'active' | 'inactive' | 'broken' | 'deprecated';

export interface LineageEdge {
  id: string;
  tenant_id: string;
  source_asset_id: string;
  source_asset_name?: string;
  source_table?: string;
  target_asset_id: string;
  target_asset_name?: string;
  target_table?: string;
  edge_type: LineageEdgeType;
  transformation?: string;
  pipeline_id?: string;
  pipeline_name?: string;
  source_classification?: string;
  target_classification?: string;
  classification_changed: boolean;
  pii_types_transferred: string[];
  confidence: number;
  evidence: Record<string, unknown>;
  status: LineageEdgeStatus;
  last_transfer_at?: string;
  transfer_count_30d: number;
  created_at: string;
  updated_at: string;
}

export interface LineageNode {
  asset_id: string;
  asset_name: string;
  classification: string;
  pii_types: string[];
  upstream_count: number;
  downstream_count: number;
}

export interface LineageGraph {
  nodes: LineageNode[];
  edges: LineageEdge[];
  total_nodes: number;
  total_edges: number;
  pii_flow_count: number;
}

export interface ImpactResult {
  asset_id: string;
  asset_name: string;
  depth: number;
  classification: string;
  pii_types: string[];
  edge_type: string;
}

export type AIUsageType =
  | 'training_data'
  | 'evaluation_data'
  | 'inference_input'
  | 'rag_knowledge_base'
  | 'prompt_context'
  | 'feature_store'
  | 'embedding_source';

export type AIRiskLevel = 'low' | 'medium' | 'high' | 'critical';
export type AnonymizationLevel = 'none' | 'pseudonymized' | 'anonymized' | 'differential_privacy';
export type AIUsageStatus = 'active' | 'inactive' | 'blocked' | 'under_review';

export interface AIRiskFactor {
  factor: string;
  weight: number;
  score: number;
  description: string;
}

export interface AIDataUsage {
  id: string;
  tenant_id: string;
  data_asset_id: string;
  data_asset_name?: string;
  data_classification?: string;
  contains_pii: boolean;
  pii_types: string[];
  usage_type: AIUsageType;
  model_id?: string;
  model_name?: string;
  model_slug?: string;
  pipeline_id?: string;
  pipeline_name?: string;
  ai_risk_score: number;
  ai_risk_level: AIRiskLevel;
  risk_factors: AIRiskFactor[];
  consent_verified: boolean;
  data_minimization: boolean;
  anonymization_level?: AnonymizationLevel;
  retention_compliant: boolean;
  status: AIUsageStatus;
  first_detected_at: string;
  last_detected_at: string;
  created_at: string;
  updated_at: string;
}

export interface AISecurityDashboard {
  total_ai_data_usages: number;
  high_risk_count: number;
  pii_in_ai_count: number;
  consent_gap_count: number;
  risk_distribution: Record<string, number>;
  usage_type_distribution: Record<string, number>;
  top_risky_usages: AIDataUsage[];
}

export interface ModelDataAssessment {
  model_slug: string;
  model_name: string;
  data_usages: AIDataUsage[];
  total_risk_score: number;
  consent_coverage: number;
  anonymization_coverage: number;
  recommendations: string[];
}

export type CostMethodology = 'ibm_ponemon' | 'custom';

export interface CostBreakdown {
  detection_and_escalation: number;
  notification: number;
  post_breach_response: number;
  lost_business: number;
  regulatory_fines: number;
}

export interface FinancialImpact {
  id: string;
  tenant_id: string;
  data_asset_id: string;
  estimated_breach_cost: number;
  cost_per_record: number;
  record_count: number;
  cost_breakdown: CostBreakdown;
  methodology: CostMethodology;
  methodology_details: Record<string, unknown>;
  applicable_regulations: string[];
  max_regulatory_fine: number;
  breach_probability_annual: number;
  annual_expected_loss: number;
  calculated_at: string;
  created_at: string;
  updated_at: string;
}

export interface PortfolioRisk {
  tenant_id: string;
  total_breach_cost: number;
  total_expected_loss: number;
  max_single_breach: number;
  avg_breach_probability: number;
  asset_count: number;
  top_risks: FinancialImpact[];
}

export type DSPMComplianceFramework = 'gdpr' | 'hipaa' | 'soc2' | 'pci_dss' | 'saudi_pdpl' | 'iso27001';
export type ControlStatus = 'compliant' | 'partial' | 'non_compliant' | 'not_applicable';
export type TrendDirection = 'improving' | 'stable' | 'declining';

export interface ControlDetail {
  control_id: string;
  control_name: string;
  description: string;
  status: ControlStatus;
  evidence: string[];
  gaps: string[];
}

export interface CompliancePosture {
  id: string;
  tenant_id: string;
  framework: DSPMComplianceFramework;
  overall_score: number;
  controls_total: number;
  controls_compliant: number;
  controls_partial: number;
  controls_non_compliant: number;
  controls_not_applicable: number;
  control_details: ControlDetail[];
  score_7d_ago?: number;
  score_30d_ago?: number;
  score_90d_ago?: number;
  trend_direction?: TrendDirection;
  estimated_fine_exposure: number;
  fine_currency: string;
  evaluated_at: string;
  created_at: string;
  updated_at: string;
}

export interface ComplianceGap {
  framework: string;
  control_id: string;
  control_name: string;
  status: ControlStatus;
  description: string;
  gaps: string[];
}

export interface ResidencyViolation {
  asset_id: string;
  asset_name: string;
  regulation: string;
  requirement: string;
  location: string;
  allowed_locations: string[];
}

export interface AuditReport {
  tenant_id: string;
  framework: string;
  generated_at: string;
  overall_score: number;
  total_controls: number;
  compliant_controls: number;
  partial_controls: number;
  non_compliant_controls: number;
  asset_count: number;
  assets: AuditAssetEntry[];
  exceptions: AuditException[];
  score_history: AuditScorePoint[];
}

export interface AuditAssetEntry {
  asset_id: string;
  asset_name: string;
  classification: string;
  pii_types: string[];
  encryption_at_rest: boolean;
  encryption_in_transit: boolean;
}

export interface AuditException {
  control_id: string;
  control_name: string;
  reason: string;
  approved_by: string;
  expires_at: string;
}

export interface AuditScorePoint {
  date: string;
  score: number;
}

export type ProliferationStatus = 'contained' | 'spreading' | 'uncontrolled';

export interface DataProliferation {
  asset_id: string;
  asset_name: string;
  classification: string;
  total_copies: number;
  authorized_copies: number;
  unauthorized_copies: number;
  spread_events: SpreadEvent[];
  status: ProliferationStatus;
}

export interface SpreadEvent {
  target_asset_id: string;
  target_asset_name: string;
  edge_type: string;
  classification_changed: boolean;
  authorized: boolean;
  detected_at: string;
}

export interface ProliferationOverview {
  total_tracked_assets: number;
  spreading_count: number;
  uncontrolled_count: number;
  total_unauthorized_copies: number;
  proliferations: DataProliferation[];
}

export interface ClassificationDrift {
  asset_id: string;
  asset_name: string;
  events: DriftEvent[];
  drift_count: number;
  current_classification: string;
}

export interface DriftEvent {
  old_classification?: string;
  new_classification: string;
  change_type: string;
  detected_by: string;
  created_at: string;
}

export interface ClassificationHistory {
  id: string;
  tenant_id: string;
  data_asset_id: string;
  old_classification?: string;
  new_classification: string;
  old_pii_types: string[];
  new_pii_types: string[];
  change_type: string;
  detected_by: string;
  confidence: number;
  evidence: Record<string, unknown>;
  actor_id?: string;
  actor_type: string;
  created_at: string;
}

export interface EnhancedClassification {
  asset_id: string;
  asset_name: string;
  classification: string;
  confidence: number;
  pii_types: string[];
  method: string;
  needs_human_review: boolean;
  evidence: Record<string, unknown>;
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

// ─── Security Events ─────────────────────────────────────────────────────────

export interface SecurityEvent {
  id: string;
  tenant_id: string;
  timestamp: string;
  source: string;
  type: string;
  severity: CyberSeverity;
  source_ip?: string;
  dest_ip?: string;
  dest_port?: number;
  protocol?: string;
  username?: string;
  process?: string;
  parent_process?: string;
  command_line?: string;
  file_path?: string;
  file_hash?: string;
  asset_id?: string;
  raw_event: Record<string, unknown>;
  matched_rules: string[];
  processed_at: string;
}

export interface EventStats {
  total: number;
  by_source: NamedCount[];
  by_type: NamedCount[];
  by_severity: NamedCount[];
}

// ─── Analytics / Predictive ──────────────────────────────────────────────────

export interface ThreatForecastItem {
  technique_id: string;
  technique_name: string;
  trend: 'increasing' | 'stable' | 'decreasing';
  growth_rate: number;
  // Matches backend model.ConfidenceInterval JSON: { p10, p50, p90 }
  forecast: { p10: number; p50: number; p90: number };
}

// Matches backend model.ForecastPoint JSON shape
export interface AlertForecastPoint {
  timestamp: string;
  value: number;
  bounds: { p10: number; p50: number; p90: number };
}

export interface CampaignCluster {
  // Backend sends cluster_id as a string (uuid-style)
  cluster_id: string;
  alert_ids: string[];
  alert_titles: string[];
  start_at: string;
  end_at: string;
  stage: string;
  mitre_techniques: string[];
  shared_iocs: string[];
  // Backend field is "confidence_interval" (model.ConfidenceInterval)
  confidence_interval: { p10: number; p50: number; p90: number };
}

export interface AnalyticsLandscape {
  active_threat_count: number;
  total_threats: number;
  indicators_total: number;
  top_threat_type: string;
  by_type: NamedCount[];
  by_severity: NamedCount[];
}
