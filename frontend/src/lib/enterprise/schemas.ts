import { z } from 'zod';

const uuidSchema = z.string().uuid('Invalid identifier.');
const optionalUuidSchema = z.string().uuid('Invalid identifier.').optional().nullable();
const optionalStringSchema = z.string().trim().optional().nullable();

export const committeeSchema = z.object({
  name: z.string().trim().min(3, 'Committee name is required.'),
  type: z.enum(['board', 'audit', 'risk', 'compensation', 'nomination', 'executive', 'governance', 'ad_hoc']),
  description: z.string().trim().min(5, 'Description is required.'),
  chair_user_id: uuidSchema,
  vice_chair_user_id: optionalUuidSchema,
  secretary_user_id: optionalUuidSchema,
  meeting_frequency: z.enum(['weekly', 'bi_weekly', 'monthly', 'quarterly', 'semi_annual', 'annual', 'ad_hoc']),
  quorum_percentage: z.number().int().min(1).max(100),
  quorum_type: z.enum(['percentage', 'fixed_count']),
  quorum_fixed_count: z.number().int().min(1).max(100).optional().nullable(),
  charter: optionalStringSchema,
  established_date: optionalStringSchema,
  dissolution_date: optionalStringSchema,
  tags: z.array(z.string().trim()).default([]),
  metadata: z.record(z.unknown()).default({}),
  chair_name: z.string().trim().min(2, 'Chair name is required.'),
  chair_email: z.string().trim().email('Valid chair email is required.'),
  vice_chair_name: optionalStringSchema,
  vice_chair_email: optionalStringSchema,
  secretary_name: optionalStringSchema,
  secretary_email: optionalStringSchema,
}).superRefine((value, ctx) => {
  if (value.quorum_type === 'fixed_count' && !value.quorum_fixed_count) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['quorum_fixed_count'],
      message: 'Fixed quorum count is required.',
    });
  }
});

export const committeeMemberSchema = z.object({
  user_id: uuidSchema,
  user_name: z.string().trim().min(2, 'Member name is required.'),
  user_email: z.string().trim().email('Valid member email is required.'),
  role: z.enum(['chair', 'vice_chair', 'secretary', 'member', 'observer']),
});

export const meetingSchema = z.object({
  committee_id: uuidSchema,
  title: z.string().trim().min(3, 'Meeting title is required.'),
  description: z.string().trim().min(3, 'Meeting description is required.'),
  scheduled_at: z.string().min(1, 'Scheduled start is required.'),
  scheduled_end_at: optionalStringSchema,
  duration_minutes: z.number().int().min(15).max(480),
  location: optionalStringSchema,
  location_type: z.enum(['physical', 'virtual', 'hybrid']),
  virtual_link: optionalStringSchema,
  virtual_platform: optionalStringSchema,
  tags: z.array(z.string().trim()).default([]),
  metadata: z.record(z.unknown()).default({}),
});

export const cancelMeetingSchema = z.object({
  reason: z.string().trim().min(5, 'Cancellation reason is required.'),
});

export const postponeMeetingSchema = z.object({
  new_scheduled_at: z.string().min(1, 'New meeting date is required.'),
  new_scheduled_end_at: optionalStringSchema,
  reason: z.string().trim().min(5, 'Postponement reason is required.'),
});

export const attendanceSchema = z.object({
  user_id: uuidSchema,
  status: z.enum(['present', 'absent', 'proxy', 'excused']),
  notes: optionalStringSchema,
  proxy_user_id: optionalUuidSchema,
  proxy_user_name: optionalStringSchema,
  proxy_authorized_by: optionalUuidSchema,
}).superRefine((value, ctx) => {
  if (value.status === 'proxy' && !value.proxy_user_name) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['proxy_user_name'],
      message: 'Proxy attendee name is required.',
    });
  }
});

export const bulkAttendanceSchema = z.object({
  attendance: z.array(attendanceSchema).min(1, 'At least one attendance record is required.'),
});

export const agendaItemSchema = z.object({
  title: z.string().trim().min(3, 'Agenda title is required.'),
  description: z.string().trim().min(3, 'Agenda description is required.'),
  item_number: optionalStringSchema,
  presenter_user_id: optionalUuidSchema,
  presenter_name: optionalStringSchema,
  duration_minutes: z.number().int().min(5).max(180),
  order_index: z.number().int().min(1).optional().nullable(),
  parent_item_id: optionalUuidSchema,
  requires_vote: z.boolean().default(false),
  vote_type: z.enum(['unanimous', 'majority', 'two_thirds', 'roll_call']).optional().nullable(),
  attachment_ids: z.array(uuidSchema).default([]),
  category: z.enum(['regular', 'special', 'information', 'decision', 'discussion', 'ratification']).optional().nullable(),
  confidential: z.boolean().default(false),
});

export const agendaNotesSchema = z.object({
  notes: z.string().trim().min(1, 'Discussion notes are required.'),
});

export const agendaVoteSchema = z.object({
  vote_type: z.enum(['unanimous', 'majority', 'two_thirds', 'roll_call']),
  votes_for: z.number().int().min(0),
  votes_against: z.number().int().min(0),
  votes_abstained: z.number().int().min(0),
  notes: z.string().trim().min(3, 'Vote notes are required.'),
});

export const minutesSchema = z.object({
  content: z.string().trim().min(20, 'Minutes content is required.'),
});

export const reviewNotesSchema = z.object({
  notes: z.string().trim().min(5, 'Review notes are required.'),
});

export const actionItemSchema = z.object({
  meeting_id: uuidSchema,
  agenda_item_id: optionalUuidSchema,
  committee_id: uuidSchema,
  title: z.string().trim().min(3, 'Action item title is required.'),
  description: z.string().trim().min(3, 'Action item description is required.'),
  priority: z.enum(['critical', 'high', 'medium', 'low']),
  assigned_to: uuidSchema,
  assignee_name: z.string().trim().min(2, 'Assignee name is required.'),
  due_date: z.string().min(1, 'Due date is required.'),
  tags: z.array(z.string().trim()).default([]),
  metadata: z.record(z.unknown()).default({}),
});

export const actionStatusSchema = z.object({
  status: z.enum(['pending', 'in_progress', 'completed', 'overdue', 'cancelled', 'deferred']),
  completion_notes: optionalStringSchema,
  completion_evidence: z.array(uuidSchema).default([]),
});

export const extendDueDateSchema = z.object({
  new_due_date: z.string().min(1, 'New due date is required.'),
  reason: z.string().trim().min(5, 'Extension reason is required.'),
});

export const attachmentReferenceSchema = z.object({
  file_id: uuidSchema,
  name: z.string().trim().min(1, 'Attachment name is required.'),
  content_type: optionalStringSchema,
  uploaded_by: optionalUuidSchema,
});

export const lexContractSchema = z.object({
  title: z.string().trim().min(3, 'Contract title is required.'),
  contract_number: optionalStringSchema,
  type: z.enum(['service_agreement', 'nda', 'employment', 'vendor', 'license', 'lease', 'partnership', 'consulting', 'procurement', 'sla', 'mou', 'amendment', 'renewal', 'other']),
  description: z.string().trim().min(5, 'Description is required.'),
  party_a_name: z.string().trim().min(2, 'Party A is required.'),
  party_a_entity: optionalStringSchema,
  party_b_name: z.string().trim().min(2, 'Counterparty is required.'),
  party_b_entity: optionalStringSchema,
  party_b_contact: optionalStringSchema,
  total_value: z.number().nonnegative().optional().nullable(),
  currency: z.string().trim().length(3, 'Currency must be a 3-letter ISO code.'),
  payment_terms: optionalStringSchema,
  effective_date: optionalStringSchema,
  expiry_date: optionalStringSchema,
  renewal_date: optionalStringSchema,
  auto_renew: z.boolean().default(false),
  renewal_notice_days: z.number().int().min(0).max(365).default(30),
  owner_user_id: uuidSchema,
  owner_name: z.string().trim().min(2, 'Owner name is required.'),
  legal_reviewer_id: optionalUuidSchema,
  legal_reviewer_name: optionalStringSchema,
  department: optionalStringSchema,
  tags: z.array(z.string().trim()).default([]),
  metadata: z.record(z.unknown()).default({}),
  document: z.object({
    file_id: uuidSchema,
    file_name: z.string().trim().min(1),
    file_size_bytes: z.number().int().nonnegative(),
    content_hash: z.string().trim().min(1),
    extracted_text: z.string().default(''),
    change_summary: z.string().default(''),
  }).optional().nullable(),
});

export const lexContractStatusSchema = z.object({
  status: z.enum(['draft', 'internal_review', 'legal_review', 'negotiation', 'pending_signature', 'active', 'suspended', 'expired', 'terminated', 'renewed', 'cancelled']),
});

export const lexClauseReviewSchema = z.object({
  status: z.enum(['pending', 'reviewed', 'flagged', 'accepted', 'rejected']),
  notes: z.string().trim().min(3, 'Review notes are required.'),
});

export const lexDocumentSchema = z.object({
  title: z.string().trim().min(3, 'Document title is required.'),
  type: z.enum(['policy', 'regulation', 'template', 'memo', 'opinion', 'filing', 'correspondence', 'resolution', 'power_of_attorney', 'other']),
  description: z.string().trim().min(3, 'Description is required.'),
  category: optionalStringSchema,
  confidentiality: z.enum(['public', 'internal', 'confidential', 'privileged']),
  contract_id: optionalUuidSchema,
  tags: z.array(z.string().trim()).default([]),
  metadata: z.record(z.unknown()).default({}),
  document: z.object({
    file_id: uuidSchema,
    file_name: z.string().trim().min(1),
    file_size_bytes: z.number().int().nonnegative(),
    content_hash: z.string().trim().min(1),
    extracted_text: z.string().default(''),
    change_summary: z.string().default(''),
  }).optional().nullable(),
});

export const lexComplianceRuleSchema = z.object({
  name: z.string().trim().min(3, 'Rule name is required.'),
  description: z.string().trim().min(5, 'Rule description is required.'),
  rule_type: z.enum(['expiry_warning', 'missing_clause', 'risk_threshold', 'review_overdue', 'unsigned_contract', 'value_threshold', 'jurisdiction_check', 'data_protection_required', 'custom']),
  severity: z.enum(['critical', 'high', 'medium', 'low']),
  config: z.record(z.unknown()).default({}),
  contract_types: z.array(z.string()).default([]),
  enabled: z.boolean().default(true),
});

export const lexAlertResolveSchema = z.object({
  status: z.enum(['open', 'acknowledged', 'investigating', 'resolved', 'dismissed']),
  resolution_notes: optionalStringSchema,
});

export const visusDashboardSchema = z.object({
  name: z.string().trim().min(3, 'Dashboard name is required.'),
  description: z.string().trim().min(3, 'Description is required.'),
  grid_columns: z.number().int().min(1).max(12).default(12),
  visibility: z.enum(['private', 'team', 'organization', 'public']),
  shared_with: z.array(uuidSchema).default([]),
  is_default: z.boolean().default(false),
  tags: z.array(z.string().trim()).default([]),
  metadata: z.record(z.unknown()).default({}),
});

export const visusWidgetSchema = z.object({
  title: z.string().trim().min(3, 'Widget title is required.'),
  subtitle: optionalStringSchema,
  type: z.enum(['kpi_card', 'line_chart', 'bar_chart', 'area_chart', 'pie_chart', 'gauge', 'table', 'alert_feed', 'text', 'sparkline', 'heatmap', 'status_grid', 'trend_indicator']),
  config: z.record(z.unknown()).default({}),
  position: z.object({
    x: z.number().int().min(0).max(11),
    y: z.number().int().min(0).max(200),
    w: z.number().int().min(1).max(12),
    h: z.number().int().min(1).max(8),
  }),
  refresh_interval_seconds: z.number().int().min(0).default(60),
});

export const visusKpiSchema = z.object({
  name: z.string().trim().min(3, 'KPI name is required.'),
  description: z.string().trim().min(3, 'Description is required.'),
  category: z.enum(['security', 'data', 'governance', 'legal', 'operations', 'general']),
  suite: z.enum(['cyber', 'data', 'acta', 'lex', 'platform', 'custom']),
  icon: optionalStringSchema,
  query_endpoint: z.string().trim().min(1, 'Query endpoint is required.'),
  query_params: z.record(z.unknown()).default({}),
  value_path: z.string().trim().min(1, 'Value path is required.'),
  unit: z.enum(['count', 'percentage', 'hours', 'minutes', 'score', 'currency', 'ratio', 'bytes']),
  format_pattern: optionalStringSchema,
  target_value: z.number().optional().nullable(),
  warning_threshold: z.number().optional().nullable(),
  critical_threshold: z.number().optional().nullable(),
  direction: z.enum(['higher_is_better', 'lower_is_better']),
  calculation_type: z.enum(['direct', 'delta', 'percentage_change', 'average_over_period', 'sum_over_period']),
  calculation_window: optionalStringSchema,
  snapshot_frequency: z.enum(['every_15m', 'hourly', 'every_4h', 'daily', 'weekly']),
  enabled: z.boolean().default(true),
  tags: z.array(z.string().trim()).default([]),
});

export const visusReportSchema = z.object({
  name: z.string().trim().min(3, 'Report name is required.'),
  description: z.string().trim().min(3, 'Description is required.'),
  report_type: z.enum(['executive_summary', 'security_posture', 'data_intelligence', 'governance', 'legal', 'custom']),
  sections: z.array(z.string().trim()).min(1, 'At least one section is required.'),
  period: z.string().trim().min(1, 'Reporting period is required.'),
  custom_period_start: optionalStringSchema,
  custom_period_end: optionalStringSchema,
  schedule: optionalStringSchema,
  recipients: z.array(uuidSchema).default([]),
  auto_send: z.boolean().default(false),
});

export const visusAlertStatusSchema = z.object({
  status: z.enum(['new', 'viewed', 'acknowledged', 'actioned', 'dismissed', 'escalated']),
  action_notes: optionalStringSchema,
  dismiss_reason: optionalStringSchema,
});

export type CommitteeFormValues = z.infer<typeof committeeSchema>;
export type CommitteeMemberFormValues = z.infer<typeof committeeMemberSchema>;
export type MeetingFormValues = z.infer<typeof meetingSchema>;
export type AgendaItemFormValues = z.infer<typeof agendaItemSchema>;
export type AgendaVoteFormValues = z.infer<typeof agendaVoteSchema>;
export type MinutesFormValues = z.infer<typeof minutesSchema>;
export type ActionItemFormValues = z.infer<typeof actionItemSchema>;
export type LexContractFormValues = z.infer<typeof lexContractSchema>;
export type LexDocumentFormValues = z.infer<typeof lexDocumentSchema>;
export type LexComplianceRuleFormValues = z.infer<typeof lexComplianceRuleSchema>;
export type VisusDashboardFormValues = z.infer<typeof visusDashboardSchema>;
export type VisusWidgetFormValues = z.infer<typeof visusWidgetSchema>;
export type VisusKpiFormValues = z.infer<typeof visusKpiSchema>;
export type VisusReportFormValues = z.infer<typeof visusReportSchema>;
