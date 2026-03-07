export interface JsonObject {
  [key: string]: JsonValue;
}

export type JsonValue =
  | string
  | number
  | boolean
  | null
  | JsonObject
  | JsonValue[];

export interface UserDirectoryEntry {
  id: string;
  first_name: string;
  last_name: string;
  email: string;
  status: string;
  roles: Array<{
    id: string;
    name: string;
    permissions: string[];
  }>;
}

export interface FileUploadRecord {
  id: string;
  tenant_id: string;
  original_name: string;
  sanitized_name: string;
  content_type: string;
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

export type ActaCommitteeType =
  | 'board'
  | 'audit'
  | 'risk'
  | 'compensation'
  | 'nomination'
  | 'executive'
  | 'governance'
  | 'ad_hoc';

export type ActaMeetingFrequency =
  | 'weekly'
  | 'bi_weekly'
  | 'monthly'
  | 'quarterly'
  | 'semi_annual'
  | 'annual'
  | 'ad_hoc';

export type ActaCommitteeStatus = 'active' | 'inactive' | 'dissolved';

export type ActaCommitteeMemberRole =
  | 'chair'
  | 'vice_chair'
  | 'secretary'
  | 'member'
  | 'observer';

export interface ActaCommitteeMember {
  id: string;
  tenant_id: string;
  committee_id: string;
  user_id: string;
  user_name: string;
  user_email: string;
  role: ActaCommitteeMemberRole;
  joined_at: string;
  left_at?: string | null;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface ActaCommitteeStats {
  active_members: number;
  upcoming_meetings: number;
  completed_meetings: number;
  open_action_items: number;
  overdue_action_items: number;
  pending_minutes_approval: number;
}

export interface ActaCommittee {
  id: string;
  tenant_id: string;
  name: string;
  type: ActaCommitteeType;
  description: string;
  chair_user_id: string;
  vice_chair_user_id?: string | null;
  secretary_user_id?: string | null;
  meeting_frequency: ActaMeetingFrequency;
  quorum_percentage: number;
  quorum_type: 'percentage' | 'fixed_count';
  quorum_fixed_count?: number | null;
  charter?: string | null;
  established_date?: string | null;
  dissolution_date?: string | null;
  status: ActaCommitteeStatus;
  tags: string[];
  metadata: JsonObject;
  created_by: string;
  created_at: string;
  updated_at: string;
  deleted_at?: string | null;
  members?: ActaCommitteeMember[];
  stats?: ActaCommitteeStats | null;
}

export type ActaMeetingStatus =
  | 'draft'
  | 'scheduled'
  | 'in_progress'
  | 'completed'
  | 'cancelled'
  | 'postponed';

export type ActaLocationType = 'physical' | 'virtual' | 'hybrid';

export type ActaAttendanceStatus =
  | 'invited'
  | 'confirmed'
  | 'declined'
  | 'present'
  | 'absent'
  | 'proxy'
  | 'excused';

export interface ActaAttendee {
  id: string;
  tenant_id: string;
  meeting_id: string;
  user_id: string;
  user_name: string;
  user_email: string;
  member_role: ActaCommitteeMemberRole;
  status: ActaAttendanceStatus;
  confirmed_at?: string | null;
  checked_in_at?: string | null;
  checked_out_at?: string | null;
  proxy_user_id?: string | null;
  proxy_user_name?: string | null;
  proxy_authorized_by?: string | null;
  notes?: string | null;
  created_at: string;
  updated_at: string;
}

export type ActaAgendaItemStatus =
  | 'pending'
  | 'discussed'
  | 'deferred'
  | 'approved'
  | 'rejected'
  | 'withdrawn'
  | 'for_noting';

export type ActaAgendaCategory =
  | 'regular'
  | 'special'
  | 'information'
  | 'decision'
  | 'discussion'
  | 'ratification';

export type ActaVoteType = 'unanimous' | 'majority' | 'two_thirds' | 'roll_call';

export type ActaVoteResult = 'approved' | 'rejected' | 'deferred' | 'tied';

export interface ActaAgendaItem {
  id: string;
  tenant_id: string;
  meeting_id: string;
  title: string;
  description: string;
  item_number?: string | null;
  presenter_user_id?: string | null;
  presenter_name?: string | null;
  duration_minutes: number;
  order_index: number;
  parent_item_id?: string | null;
  status: ActaAgendaItemStatus;
  notes?: string | null;
  requires_vote: boolean;
  vote_type?: ActaVoteType | null;
  votes_for?: number | null;
  votes_against?: number | null;
  votes_abstained?: number | null;
  vote_result?: ActaVoteResult | null;
  vote_notes?: string | null;
  attachment_ids: string[];
  category?: ActaAgendaCategory | null;
  confidential: boolean;
  created_at: string;
  updated_at: string;
}

export interface ActaMeetingAttachment {
  file_id: string;
  name: string;
  content_type?: string | null;
  uploaded_by?: string | null;
  uploaded_at: string;
}

export interface ActaExtractedAction {
  title: string;
  description: string;
  assigned_to: string;
  due_date?: string | null;
  priority: 'critical' | 'high' | 'medium' | 'low';
  source: string;
}

export type ActaMinutesStatus =
  | 'draft'
  | 'review'
  | 'revision_requested'
  | 'approved'
  | 'published';

export interface ActaMeetingMinutes {
  id: string;
  tenant_id: string;
  meeting_id: string;
  content: string;
  ai_summary?: string | null;
  status: ActaMinutesStatus;
  submitted_for_review_at?: string | null;
  submitted_by?: string | null;
  reviewed_by?: string | null;
  review_notes?: string | null;
  approved_by?: string | null;
  approved_at?: string | null;
  published_at?: string | null;
  version: number;
  previous_version_id?: string | null;
  ai_action_items: ActaExtractedAction[];
  ai_generated: boolean;
  created_by: string;
  created_at: string;
  updated_at: string;
  meeting_title?: string | null;
}

export type ActaActionItemPriority = 'critical' | 'high' | 'medium' | 'low';
export type ActaActionItemStatus =
  | 'pending'
  | 'in_progress'
  | 'completed'
  | 'overdue'
  | 'cancelled'
  | 'deferred';

export interface ActaActionItem {
  id: string;
  tenant_id: string;
  meeting_id: string;
  agenda_item_id?: string | null;
  committee_id: string;
  title: string;
  description: string;
  priority: ActaActionItemPriority;
  assigned_to: string;
  assignee_name: string;
  assigned_by: string;
  due_date: string;
  original_due_date: string;
  extended_count: number;
  extension_reason?: string | null;
  status: ActaActionItemStatus;
  completed_at?: string | null;
  completion_notes?: string | null;
  completion_evidence: string[];
  follow_up_meeting_id?: string | null;
  reviewed_at?: string | null;
  tags: string[];
  metadata: JsonObject;
  created_by: string;
  created_at: string;
  updated_at: string;
  meeting_title?: string | null;
}

export interface ActaActionItemStats {
  by_status: Record<string, number>;
  by_priority: Record<string, number>;
  open: number;
  overdue: number;
  completed: number;
}

export interface ActaActionItemSummary {
  id: string;
  title: string;
  committee_id: string;
  committee_name: string;
  assignee_name: string;
  due_date: string;
  priority: ActaActionItemPriority;
  status: ActaActionItemStatus;
}

export interface ActaMeetingSummary {
  id: string;
  committee_id: string;
  committee_name: string;
  title: string;
  status: ActaMeetingStatus;
  scheduled_at: string;
  duration_minutes: number;
  location?: string | null;
  quorum_met?: boolean | null;
}

export interface ActaCalendarDay {
  date: string;
  meetings: ActaMeetingSummary[];
}

export interface ActaMeeting {
  id: string;
  tenant_id: string;
  committee_id: string;
  committee_name: string;
  title: string;
  description: string;
  meeting_number?: number | null;
  scheduled_at: string;
  scheduled_end_at?: string | null;
  actual_start_at?: string | null;
  actual_end_at?: string | null;
  duration_minutes: number;
  location?: string | null;
  location_type: ActaLocationType;
  virtual_link?: string | null;
  virtual_platform?: string | null;
  status: ActaMeetingStatus;
  cancellation_reason?: string | null;
  quorum_required: number;
  attendee_count: number;
  present_count: number;
  quorum_met?: boolean | null;
  agenda_item_count: number;
  action_item_count: number;
  has_minutes: boolean;
  minutes_status?: string | null;
  workflow_instance_id?: string | null;
  tags: string[];
  metadata: JsonObject;
  created_by: string;
  created_at: string;
  updated_at: string;
  deleted_at?: string | null;
  attendees?: ActaAttendee[];
  agenda?: ActaAgendaItem[];
  attendance?: ActaAttendee[];
  latest_minutes?: ActaMeetingMinutes | null;
  attachments?: ActaMeetingAttachment[];
}

export type ActaComplianceCheckType =
  | 'meeting_frequency'
  | 'quorum_compliance'
  | 'minutes_completion'
  | 'action_item_tracking'
  | 'attendance_rate'
  | 'charter_review'
  | 'document_retention'
  | 'conflict_of_interest';

export type ActaComplianceStatus =
  | 'compliant'
  | 'non_compliant'
  | 'warning'
  | 'not_applicable';

export type ActaComplianceSeverity = 'critical' | 'high' | 'medium' | 'low';

export interface ActaComplianceCheck {
  id: string;
  tenant_id: string;
  committee_id?: string | null;
  check_type: ActaComplianceCheckType;
  check_name: string;
  status: ActaComplianceStatus;
  severity: ActaComplianceSeverity;
  description: string;
  finding?: string | null;
  recommendation?: string | null;
  evidence: JsonObject;
  period_start: string;
  period_end: string;
  checked_at: string;
  checked_by: string;
  created_at: string;
}

export interface ActaCommitteeCompliance {
  committee_id: string;
  committee_name: string;
  score: number;
  warnings: number;
  non_compliant: number;
}

export interface ActaComplianceReport {
  tenant_id: string;
  results: ActaComplianceCheck[];
  by_status: Record<string, number>;
  by_check_type: Record<string, number>;
  by_committee: ActaCommitteeCompliance[];
  score: number;
  non_compliant_count: number;
  warning_count: number;
  generated_at: string;
}

export interface ActaKPIs {
  active_committees: number;
  upcoming_meetings_30d: number;
  open_action_items: number;
  overdue_action_items: number;
  compliance_score: number;
  minutes_pending_approval: number;
  attendance_rate_avg: number;
}

export interface ActaMonthlyMeetingCount {
  month: string;
  count: number;
}

export interface ActaMonthlyAttendanceRate {
  month: string;
  rate_percent: number;
}

export interface ActaAuditEntry {
  timestamp: string;
  type: string;
  message: string;
  entity_id: string;
}

export interface ActaDashboard {
  kpis: ActaKPIs;
  upcoming_meetings: ActaMeetingSummary[];
  recent_meetings: ActaMeetingSummary[];
  action_items_by_status: Record<string, number>;
  action_items_by_priority: Record<string, number>;
  overdue_action_items: ActaActionItemSummary[];
  compliance_by_committee: ActaCommitteeCompliance[];
  compliance_score: number;
  meeting_frequency_chart: ActaMonthlyMeetingCount[];
  attendance_rate_chart: ActaMonthlyAttendanceRate[];
  recent_activity: ActaAuditEntry[];
  calculated_at: string;
}

export type LexContractType =
  | 'service_agreement'
  | 'nda'
  | 'employment'
  | 'vendor'
  | 'license'
  | 'lease'
  | 'partnership'
  | 'consulting'
  | 'procurement'
  | 'sla'
  | 'mou'
  | 'amendment'
  | 'renewal'
  | 'other';

export type LexContractStatus =
  | 'draft'
  | 'internal_review'
  | 'legal_review'
  | 'negotiation'
  | 'pending_signature'
  | 'active'
  | 'suspended'
  | 'expired'
  | 'terminated'
  | 'renewed'
  | 'cancelled';

export type LexAnalysisStatus = 'pending' | 'analyzing' | 'completed' | 'failed';
export type LexRiskLevel = 'critical' | 'high' | 'medium' | 'low' | 'none';

export interface LexContractRecord {
  id: string;
  tenant_id: string;
  title: string;
  contract_number?: string | null;
  type: LexContractType;
  description: string;
  party_a_name: string;
  party_a_entity?: string | null;
  party_b_name: string;
  party_b_entity?: string | null;
  party_b_contact?: string | null;
  total_value?: number | null;
  currency: string;
  payment_terms?: string | null;
  effective_date?: string | null;
  expiry_date?: string | null;
  renewal_date?: string | null;
  auto_renew: boolean;
  renewal_notice_days: number;
  signed_date?: string | null;
  status: LexContractStatus;
  previous_status?: LexContractStatus | null;
  status_changed_at?: string | null;
  status_changed_by?: string | null;
  owner_user_id: string;
  owner_name: string;
  legal_reviewer_id?: string | null;
  legal_reviewer_name?: string | null;
  risk_score?: number | null;
  risk_level: LexRiskLevel;
  analysis_status: LexAnalysisStatus;
  last_analyzed_at?: string | null;
  document_file_id?: string | null;
  document_text: string;
  current_version: number;
  parent_contract_id?: string | null;
  workflow_instance_id?: string | null;
  department?: string | null;
  tags: string[];
  metadata: JsonObject;
  created_by: string;
  created_at: string;
  updated_at: string;
  deleted_at?: string | null;
}

export interface LexContractSummary {
  id: string;
  title: string;
  type: LexContractType;
  status: LexContractStatus;
  party_b_name: string;
  risk_level: LexRiskLevel;
  risk_score?: number | null;
  expiry_date?: string | null;
  current_version: number;
  created_at: string;
  parties?: Array<Record<string, unknown>>;
  value?: number | null;
  currency?: string;
  effective_date?: string | null;
  file_url?: string | null;
  metadata?: JsonObject;
}

export interface LexContractVersion {
  id: string;
  tenant_id: string;
  contract_id: string;
  version: number;
  file_id: string;
  file_name: string;
  file_size_bytes: number;
  content_hash: string;
  extracted_text?: string | null;
  change_summary?: string | null;
  uploaded_by: string;
  uploaded_at: string;
}

export type LexClauseType =
  | 'indemnification'
  | 'termination'
  | 'limitation_of_liability'
  | 'confidentiality'
  | 'ip_ownership'
  | 'non_compete'
  | 'payment_terms'
  | 'warranty'
  | 'force_majeure'
  | 'dispute_resolution'
  | 'data_protection'
  | 'governing_law'
  | 'assignment'
  | 'insurance'
  | 'audit_rights'
  | 'sla'
  | 'auto_renewal'
  | 'representations'
  | 'non_solicitation'
  | 'other';

export type LexClauseReviewStatus =
  | 'pending'
  | 'reviewed'
  | 'flagged'
  | 'accepted'
  | 'rejected';

export interface LexClause {
  id: string;
  tenant_id: string;
  contract_id: string;
  clause_type: LexClauseType;
  title: string;
  content: string;
  section_reference?: string | null;
  page_number?: number | null;
  risk_level: LexRiskLevel;
  risk_score: number;
  risk_keywords: string[];
  analysis_summary?: string | null;
  recommendations: string[];
  compliance_flags: string[];
  review_status: LexClauseReviewStatus;
  reviewed_by?: string | null;
  reviewed_at?: string | null;
  review_notes?: string | null;
  extraction_confidence: number;
  created_at: string;
  updated_at: string;
}

export interface LexRiskFinding {
  title: string;
  description: string;
  severity: LexRiskLevel;
  clause_reference?: string | null;
  recommendation: string;
  clause_type?: LexClauseType | null;
}

export interface LexComplianceFlag {
  code: string;
  title: string;
  description: string;
  severity: LexRiskLevel;
  clause_reference?: string | null;
}

export interface LexExtractedParty {
  name: string;
  role: string;
  source: string;
}

export interface LexExtractedDate {
  label: string;
  value?: string | null;
  source: string;
}

export interface LexExtractedAmount {
  label: string;
  currency: string;
  value: number;
  source: string;
}

export interface LexContractRiskAnalysis {
  id: string;
  tenant_id: string;
  contract_id: string;
  contract_version: number;
  overall_risk: LexRiskLevel;
  risk_score: number;
  clause_count: number;
  high_risk_clause_count: number;
  missing_clauses: LexClauseType[];
  key_findings: LexRiskFinding[];
  recommendations: string[];
  compliance_flags: LexComplianceFlag[];
  extracted_parties: LexExtractedParty[];
  extracted_dates: LexExtractedDate[];
  extracted_amounts: LexExtractedAmount[];
  analysis_duration_ms: number;
  analyzed_by: string;
  analyzed_at: string;
  created_at: string;
}

export interface LexContractDetail {
  contract: LexContractRecord;
  clauses: LexClause[];
  latest_analysis?: LexContractRiskAnalysis | null;
  version_count: number;
}

export interface LexExpiringContractSummary {
  id: string;
  title: string;
  type: LexContractType;
  status: LexContractStatus;
  party_b_name: string;
  expiry_date: string;
  days_until_expiry: number;
  owner_name: string;
  legal_reviewer_name?: string | null;
}

export interface LexContractRiskSummary {
  id: string;
  title: string;
  type: LexContractType;
  status: LexContractStatus;
  risk_level: LexRiskLevel;
  risk_score: number;
  party_b_name: string;
  expiry_date?: string | null;
}

export interface LexTotalValueBreakdown {
  by_type: Record<string, number>;
  by_currency: Record<string, number>;
}

export interface LexMonthlyContractActivity {
  month: string;
  created: number;
  activated: number;
  expired: number;
  renewed: number;
}

export interface LexDashboardKPIs {
  active_contracts: number;
  expiring_in_30_days: number;
  expiring_in_7_days: number;
  high_risk_contracts: number;
  pending_review: number;
  open_compliance_alerts: number;
  total_active_value: number;
  compliance_score: number;
}

export interface LexDashboard {
  kpis: LexDashboardKPIs;
  contracts_by_type: Record<string, number>;
  contracts_by_status: Record<string, number>;
  expiring_contracts: LexExpiringContractSummary[];
  high_risk_contracts: LexContractRiskSummary[];
  recent_contracts: LexContractSummary[];
  compliance_alerts_by_status: Record<string, number>;
  total_contract_value: LexTotalValueBreakdown;
  monthly_activity: LexMonthlyContractActivity[];
  calculated_at: string;
}

export type LexDocumentType =
  | 'policy'
  | 'regulation'
  | 'template'
  | 'memo'
  | 'opinion'
  | 'filing'
  | 'correspondence'
  | 'resolution'
  | 'power_of_attorney'
  | 'other';

export type LexDocumentConfidentiality =
  | 'public'
  | 'internal'
  | 'confidential'
  | 'privileged';

export type LexDocumentStatus = 'draft' | 'active' | 'archived' | 'superseded';

export interface LexDocument {
  id: string;
  tenant_id: string;
  title: string;
  type: LexDocumentType;
  description: string;
  file_id?: string | null;
  file_name?: string | null;
  file_size_bytes?: number | null;
  category?: string | null;
  confidentiality: LexDocumentConfidentiality;
  contract_id?: string | null;
  current_version: number;
  version?: number;
  status: LexDocumentStatus;
  tags: string[];
  metadata: JsonObject;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface LexDocumentVersion {
  id: string;
  tenant_id: string;
  document_id: string;
  version: number;
  file_id: string;
  file_name: string;
  file_size_bytes: number;
  content_hash: string;
  change_summary?: string | null;
  uploaded_by: string;
  uploaded_at: string;
}

export type LexComplianceRuleType =
  | 'expiry_warning'
  | 'missing_clause'
  | 'risk_threshold'
  | 'review_overdue'
  | 'unsigned_contract'
  | 'value_threshold'
  | 'jurisdiction_check'
  | 'data_protection_required'
  | 'custom';

export type LexComplianceSeverity = 'critical' | 'high' | 'medium' | 'low';
export type LexComplianceAlertStatus =
  | 'open'
  | 'acknowledged'
  | 'investigating'
  | 'resolved'
  | 'dismissed';

export interface LexComplianceRule {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  rule_type: LexComplianceRuleType;
  severity: LexComplianceSeverity;
  config: JsonObject;
  contract_types: string[];
  enabled: boolean;
  jurisdiction?: string | null;
  regulation_reference?: string | null;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface LexComplianceAlert {
  id: string;
  tenant_id: string;
  rule_id?: string | null;
  contract_id?: string | null;
  title: string;
  description: string;
  severity: LexComplianceSeverity;
  status: LexComplianceAlertStatus;
  entity_type?: string | null;
  resolved_by?: string | null;
  resolved_at?: string | null;
  resolution_notes?: string | null;
  dedup_key?: string | null;
  evidence: JsonObject;
  created_at: string;
  updated_at: string;
}

export interface LexComplianceDashboard {
  rules_by_type: Record<string, number>;
  alerts_by_status: Record<string, number>;
  alerts_by_severity: Record<string, number>;
  open_alerts: number;
  resolved_alerts: number;
  contracts_in_scope: number;
  compliance_score: number;
  calculated_at: string;
}

export interface LexComplianceScore {
  tenant_id: string;
  score: number;
  open_alerts: number;
  resolved_alerts: number;
  rule_coverage: number;
  calculated_at: string;
}

export interface LexComplianceRunResult {
  tenant_id: string;
  score: number;
  alerts_created: number;
  alerts: LexComplianceAlert[];
  calculated_at: string;
}

export interface LexWorkflowSummary {
  workflow_instance_id: string;
  contract_id: string;
  contract_title: string;
  contract_status: LexContractStatus;
  workflow_status: string;
  current_step_id?: string | null;
  started_at: string;
  assignee_id?: string | null;
  assignee_role?: string | null;
  task_status?: string | null;
}

export type VisusDashboardVisibility = 'private' | 'team' | 'organization' | 'public';

export interface VisusWidgetPosition {
  x: number;
  y: number;
  w: number;
  h: number;
}

export type VisusWidgetType =
  | 'kpi_card'
  | 'line_chart'
  | 'bar_chart'
  | 'area_chart'
  | 'pie_chart'
  | 'gauge'
  | 'table'
  | 'alert_feed'
  | 'text'
  | 'sparkline'
  | 'heatmap'
  | 'status_grid'
  | 'trend_indicator';

export interface VisusWidget {
  id: string;
  tenant_id: string;
  dashboard_id: string;
  title: string;
  subtitle?: string | null;
  type: VisusWidgetType;
  config: JsonObject;
  position: VisusWidgetPosition;
  refresh_interval_seconds: number;
  created_at: string;
  updated_at: string;
}

export interface VisusDashboard {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  grid_columns: number;
  visibility: VisusDashboardVisibility;
  shared_with: string[];
  is_default: boolean;
  is_system: boolean;
  tags: string[];
  metadata: JsonObject;
  created_by: string;
  created_at: string;
  updated_at: string;
  deleted_at?: string | null;
  widgets?: VisusWidget[];
  widget_count?: number;
}

export type VisusKPICategory =
  | 'security'
  | 'data'
  | 'governance'
  | 'legal'
  | 'operations'
  | 'general';

export type VisusKPISuite = 'cyber' | 'data' | 'acta' | 'lex' | 'platform' | 'custom';
export type VisusKPIUnit =
  | 'count'
  | 'percentage'
  | 'hours'
  | 'minutes'
  | 'score'
  | 'currency'
  | 'ratio'
  | 'bytes';
export type VisusKPIDirection = 'higher_is_better' | 'lower_is_better';
export type VisusKPICalculationType =
  | 'direct'
  | 'delta'
  | 'percentage_change'
  | 'average_over_period'
  | 'sum_over_period';
export type VisusKPISnapshotFrequency =
  | 'every_15m'
  | 'hourly'
  | 'every_4h'
  | 'daily'
  | 'weekly';
export type VisusKPIStatus = 'normal' | 'warning' | 'critical' | 'unknown';

export interface VisusKPISnapshot {
  id: string;
  tenant_id: string;
  kpi_id: string;
  value: number;
  previous_value?: number | null;
  delta?: number | null;
  delta_percent?: number | null;
  status: VisusKPIStatus;
  period_start: string;
  period_end: string;
  fetch_success: boolean;
  fetch_latency_ms?: number | null;
  fetch_error?: string | null;
  created_at: string;
}

export interface VisusKPIDefinition {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  category: VisusKPICategory;
  suite: VisusKPISuite;
  icon?: string | null;
  query_endpoint: string;
  query_params: JsonObject;
  value_path: string;
  unit: VisusKPIUnit;
  format_pattern?: string | null;
  target_value?: number | null;
  warning_threshold?: number | null;
  critical_threshold?: number | null;
  direction: VisusKPIDirection;
  calculation_type: VisusKPICalculationType;
  calculation_window?: string | null;
  snapshot_frequency: VisusKPISnapshotFrequency;
  enabled: boolean;
  is_default: boolean;
  last_snapshot_at?: string | null;
  last_value?: number | null;
  last_status?: VisusKPIStatus | null;
  tags: string[];
  created_by: string;
  created_at: string;
  updated_at: string;
  latest_snapshot?: VisusKPISnapshot | null;
}

export interface VisusKPIGetResponse {
  definition: VisusKPIDefinition;
  history: VisusKPISnapshot[];
}

export type VisusReportType =
  | 'executive_summary'
  | 'security_posture'
  | 'data_intelligence'
  | 'governance'
  | 'legal'
  | 'custom';

export interface VisusReportDefinition {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  report_type: VisusReportType;
  sections: string[];
  period: string;
  custom_period_start?: string | null;
  custom_period_end?: string | null;
  schedule?: string | null;
  next_run_at?: string | null;
  recipients: string[];
  auto_send: boolean;
  last_generated_at?: string | null;
  total_generated: number;
  type?: string;
  file_url?: string | null;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export type VisusReportFileFormat = 'json' | 'pdf' | 'html';

export interface VisusReportSnapshot {
  id: string;
  snapshot_id?: string;
  tenant_id: string;
  report_id: string;
  report_data: JsonObject;
  narrative?: string | null;
  file_id?: string | null;
  file_format: VisusReportFileFormat;
  period_start: string;
  period_end: string;
  sections_included: string[];
  generation_time_ms?: number | null;
  suite_fetch_errors: Record<string, string>;
  generated_by?: string | null;
  generated_at: string;
}

export type VisusAlertCategory =
  | 'risk'
  | 'compliance'
  | 'data_quality'
  | 'governance'
  | 'legal'
  | 'operational'
  | 'financial'
  | 'strategic';

export type VisusAlertSeverity = 'critical' | 'high' | 'medium' | 'low' | 'info';
export type VisusAlertStatus =
  | 'new'
  | 'viewed'
  | 'acknowledged'
  | 'actioned'
  | 'dismissed'
  | 'escalated';

export interface VisusExecutiveAlert {
  id: string;
  tenant_id: string;
  title: string;
  description: string;
  category: VisusAlertCategory;
  severity: VisusAlertSeverity;
  source_suite: string;
  source_type: string;
  source_entity_id?: string | null;
  source_event_type?: string | null;
  status: VisusAlertStatus;
  viewed_at?: string | null;
  viewed_by?: string | null;
  actioned_at?: string | null;
  actioned_by?: string | null;
  action_notes?: string | null;
  dismissed_at?: string | null;
  dismissed_by?: string | null;
  dismiss_reason?: string | null;
  dedup_key?: string | null;
  occurrence_count: number;
  first_seen_at: string;
  last_seen_at: string;
  linked_kpi_id?: string | null;
  linked_dashboard_id?: string | null;
  metadata: JsonObject;
  created_at: string;
  updated_at: string;
}

export interface VisusAlertStats {
  by_category: Record<string, number>;
  by_severity: Record<string, number>;
  by_status: Record<string, number>;
  total: number;
}

export interface VisusSuiteStatus {
  available: boolean;
  last_success: string;
  latency_ms: number;
  error?: string | null;
}

export interface VisusExecutiveSummary {
  cyber_security?: JsonObject | null;
  data_intelligence?: JsonObject | null;
  governance?: JsonObject | null;
  legal?: JsonObject | null;
  kpis: VisusKPISnapshot[];
  alerts: VisusExecutiveAlert[];
  suite_health: Record<string, VisusSuiteStatus>;
  generated_at: string;
  cache_status: Record<string, string>;
}

export interface VisusWidgetTypeDefinition {
  type: VisusWidgetType;
  schema: JsonObject;
}

export interface VisusKpiCardWidgetData {
  value: number;
  status: VisusKPIStatus;
  trend: Array<{ at: string; value: number }>;
  target?: number | null;
  unit?: string | null;
  delta?: number | null;
  delta_percent?: number | null;
}

export interface VisusGaugeWidgetData {
  value: number;
  min: number;
  max: number;
  thresholds: {
    warning?: number | null;
    critical?: number | null;
  };
  status: VisusKPIStatus;
}

export interface VisusSparklineWidgetData {
  values: number[];
  min: number;
  max: number;
  current: number;
  trend_direction: 'up' | 'down' | 'flat';
}

export interface VisusTrendIndicatorWidgetData {
  value: number;
  direction: 'up' | 'down' | 'flat';
  change_percent: number;
  periods: VisusKPISnapshot[];
}

export interface VisusAlertFeedWidgetData {
  alerts: VisusExecutiveAlert[];
}

export interface VisusSeriesPoint {
  x: string;
  y: number;
}

export interface VisusSeries {
  name: string;
  data: VisusSeriesPoint[] | number[];
}

export interface VisusSeriesWidgetData {
  series: VisusSeries[];
  x_label?: string;
  y_label?: string;
  categories?: Array<string | number>;
}

export interface VisusPieWidgetData {
  slices: Array<{ label: string; value: number; color: string }>;
}

export interface VisusTableWidgetData {
  columns: Array<{ key: string; label: string }>;
  rows: Array<Record<string, JsonValue>>;
  total_count: number;
}

export interface VisusHeatmapWidgetData {
  cells: Array<{ x: string; y: string; value: number }>;
  x_labels: string[];
  y_labels: string[];
}

export interface VisusStatusGridWidgetData {
  items: Array<{
    label: string;
    status: string;
    value: number | string;
    unit?: string | null;
  }>;
}

export interface VisusTextWidgetData {
  content: string;
}

export type VisusWidgetData =
  | VisusKpiCardWidgetData
  | VisusGaugeWidgetData
  | VisusSparklineWidgetData
  | VisusTrendIndicatorWidgetData
  | VisusAlertFeedWidgetData
  | VisusSeriesWidgetData
  | VisusPieWidgetData
  | VisusTableWidgetData
  | VisusHeatmapWidgetData
  | VisusStatusGridWidgetData
  | VisusTextWidgetData;

export type ActaMeetingMinute = ActaMeetingMinutes;
export type LexContract = LexContractSummary;
export type VisusReport = VisusReportDefinition;
export type VisusReportGeneration = VisusReportSnapshot;
export type ComplianceDashboard = LexComplianceDashboard;
export type ComplianceRule = LexComplianceRule;

export interface ComplianceCheckResult {
  rule_id: string;
  rule_name: string;
  severity: string;
  status: string;
  message: string;
  alert_id?: string | null;
}
