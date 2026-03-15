import type { LucideIcon } from "lucide-react";
import {
  AlertCircle,
  Eye,
  Search,
  Clock,
  CheckCircle,
  XCircle,
  MinusCircle,
  ArrowUpCircle,
  Play,
  Pause,
  FileEdit,
  UserCheck,
  Ban,
  CheckCheck,
  AlertTriangle,
  Shield,
  User,
  Building2,
  File,
  GitBranch,
  Scale,
  Workflow,
} from "lucide-react";

export interface StatusConfigItem {
  label: string;
  color: string; // Tailwind color name: "red", "green", "yellow", "blue", "gray", "orange", "purple"
  icon: LucideIcon;
}

export type StatusConfig = Record<string, StatusConfigItem>;

export const alertStatusConfig: StatusConfig = {
  new: { label: "New", color: "red", icon: AlertCircle },
  acknowledged: { label: "Acknowledged", color: "orange", icon: Eye },
  investigating: { label: "Investigating", color: "yellow", icon: Search },
  in_progress: { label: "In Progress", color: "blue", icon: Clock },
  resolved: { label: "Resolved", color: "green", icon: CheckCircle },
  closed: { label: "Closed", color: "gray", icon: XCircle },
  false_positive: { label: "False Positive", color: "gray", icon: MinusCircle },
  escalated: { label: "Escalated", color: "purple", icon: ArrowUpCircle },
};

export const pipelineStatusConfig: StatusConfig = {
  active: { label: "Active", color: "green", icon: Play },
  paused: { label: "Paused", color: "yellow", icon: Pause },
  failed: { label: "Failed", color: "red", icon: XCircle },
  completed: { label: "Completed", color: "blue", icon: CheckCircle },
  draft: { label: "Draft", color: "gray", icon: FileEdit },
};

export const sourceStatusConfig: StatusConfig = {
  active: { label: 'Active', color: 'green', icon: CheckCircle },
  syncing: { label: 'Syncing', color: 'blue', icon: Play },
  inactive: { label: 'Inactive', color: 'gray', icon: Pause },
  error: { label: 'Error', color: 'red', icon: AlertCircle },
};

export const datasetStatusConfig: StatusConfig = {
  active: { label: 'Active', color: 'green', icon: CheckCircle },
  published: { label: 'Published', color: 'blue', icon: CheckCheck },
  draft: { label: 'Draft', color: 'gray', icon: FileEdit },
  archived: { label: 'Archived', color: 'gray', icon: XCircle },
  deprecated: { label: 'Deprecated', color: 'orange', icon: AlertTriangle },
};

export const taskStatusConfig: StatusConfig = {
  pending: { label: "Pending", color: "yellow", icon: Clock },
  claimed: { label: "In Progress", color: "blue", icon: UserCheck },
  completed: { label: "Completed", color: "green", icon: CheckCircle },
  rejected: { label: "Rejected", color: "red", icon: XCircle },
  escalated: { label: "Escalated", color: "purple", icon: ArrowUpCircle },
  cancelled: { label: "Cancelled", color: "gray", icon: Ban },
};

export const userStatusConfig: StatusConfig = {
  active: { label: "Active", color: "green", icon: CheckCircle },
  suspended: { label: "Suspended", color: "red", icon: Ban },
  inactive: { label: "Inactive", color: "gray", icon: XCircle },
  pending_verification: { label: "Pending", color: "yellow", icon: Clock },
};

export const fileStatusConfig: StatusConfig = {
  pending: { label: "Pending", color: "yellow", icon: Clock },
  processing: { label: "Processing", color: "blue", icon: Search },
  available: { label: "Available", color: "green", icon: CheckCircle },
  quarantined: { label: "Quarantined", color: "red", icon: AlertTriangle },
  deleted: { label: "Deleted", color: "gray", icon: XCircle },
};

export const tenantStatusConfig: StatusConfig = {
  active: { label: "Active", color: "green", icon: CheckCircle },
  inactive: { label: "Inactive", color: "gray", icon: XCircle },
  suspended: { label: "Suspended", color: "red", icon: Ban },
  trial: { label: "Trial", color: "yellow", icon: Clock },
  onboarding: { label: "Onboarding", color: "blue", icon: Clock },
  deprovisioned: { label: "Deprovisioned", color: "gray", icon: XCircle },
};

export const tenantPlanConfig: StatusConfig = {
  free: { label: "Free", color: "gray", icon: Building2 },
  starter: { label: "Starter", color: "blue", icon: Building2 },
  professional: { label: "Professional", color: "purple", icon: Building2 },
  enterprise: { label: "Enterprise", color: "teal", icon: Building2 },
};

export const apiKeyStatusConfig: StatusConfig = {
  active: { label: "Active", color: "green", icon: CheckCircle },
  revoked: { label: "Revoked", color: "red", icon: Ban },
  expired: { label: "Expired", color: "gray", icon: Clock },
};

export const invitationStatusConfig: StatusConfig = {
  pending: { label: "Pending", color: "yellow", icon: Clock },
  accepted: { label: "Accepted", color: "green", icon: CheckCircle },
  expired: { label: "Expired", color: "gray", icon: Clock },
  cancelled: { label: "Cancelled", color: "red", icon: Ban },
};

export const workflowDefinitionStatusConfig: StatusConfig = {
  draft: { label: "Draft", color: "gray", icon: FileEdit },
  active: { label: "Active", color: "green", icon: CheckCircle },
  archived: { label: "Archived", color: "gray", icon: Ban },
};

export const workflowStatusConfig: StatusConfig = {
  running: { label: "Running", color: "blue", icon: Play },
  completed: { label: "Completed", color: "green", icon: CheckCircle },
  failed: { label: "Failed", color: "red", icon: XCircle },
  cancelled: { label: "Cancelled", color: "gray", icon: Ban },
  suspended: { label: "Suspended", color: "yellow", icon: Pause },
};

export const contractStatusConfig: StatusConfig = {
  draft: { label: "Draft", color: "gray", icon: FileEdit },
  review: { label: "In Review", color: "yellow", icon: Eye },
  internal_review: { label: "Internal Review", color: "yellow", icon: Eye },
  legal_review: { label: "Legal Review", color: "orange", icon: Scale },
  negotiation: { label: 'Negotiation', color: 'orange', icon: Workflow },
  pending_signature: { label: 'Pending Signature', color: 'blue', icon: FileEdit },
  active: { label: 'Active', color: 'green', icon: CheckCircle },
  suspended: { label: 'Suspended', color: 'yellow', icon: Pause },
  approved: { label: "Approved", color: "green", icon: CheckCircle },
  signed: { label: "Signed", color: "blue", icon: CheckCheck },
  expired: { label: "Expired", color: "red", icon: AlertCircle },
  renewed: { label: 'Renewed', color: 'teal', icon: CheckCheck },
  terminated: { label: "Terminated", color: "gray", icon: XCircle },
  cancelled: { label: 'Cancelled', color: 'gray', icon: Ban },
};

export const committeeStatusConfig: StatusConfig = {
  active: { label: 'Active', color: 'green', icon: CheckCircle },
  inactive: { label: 'Inactive', color: 'gray', icon: Pause },
  dissolved: { label: 'Dissolved', color: 'red', icon: XCircle },
};

export const meetingStatusConfig: StatusConfig = {
  draft: { label: 'Draft', color: 'gray', icon: FileEdit },
  scheduled: { label: 'Scheduled', color: 'blue', icon: Clock },
  in_progress: { label: 'In Progress', color: 'yellow', icon: Play },
  completed: { label: 'Completed', color: 'green', icon: CheckCircle },
  cancelled: { label: 'Cancelled', color: 'gray', icon: Ban },
  postponed: { label: 'Postponed', color: 'orange', icon: Pause },
};

export const minuteStatusConfig: StatusConfig = {
  draft: { label: 'Draft', color: 'gray', icon: FileEdit },
  review: { label: 'In Review', color: 'yellow', icon: Eye },
  revision_requested: { label: 'Revision Requested', color: 'orange', icon: AlertTriangle },
  approved: { label: 'Approved', color: 'green', icon: CheckCircle },
  published: { label: 'Published', color: 'blue', icon: CheckCheck },
};

export const documentStatusConfig: StatusConfig = {
  draft: { label: 'Draft', color: 'gray', icon: FileEdit },
  review: { label: 'In Review', color: 'yellow', icon: Eye },
  approved: { label: 'Approved', color: 'green', icon: CheckCircle },
  active: { label: 'Active', color: 'green', icon: CheckCircle },
  archived: { label: 'Archived', color: 'gray', icon: XCircle },
  superseded: { label: 'Superseded', color: 'orange', icon: AlertTriangle },
};

export const actionItemStatusConfig: StatusConfig = {
  pending: { label: 'Pending', color: 'gray', icon: Clock },
  in_progress: { label: 'In Progress', color: 'blue', icon: Play },
  completed: { label: 'Completed', color: 'green', icon: CheckCircle },
  overdue: { label: 'Overdue', color: 'red', icon: AlertCircle },
  cancelled: { label: 'Cancelled', color: 'gray', icon: Ban },
  deferred: { label: 'Deferred', color: 'yellow', icon: Pause },
};

export const clauseReviewStatusConfig: StatusConfig = {
  pending: { label: 'Pending', color: 'gray', icon: Clock },
  reviewed: { label: 'Reviewed', color: 'blue', icon: Eye },
  flagged: { label: 'Flagged', color: 'red', icon: AlertTriangle },
  accepted: { label: 'Accepted', color: 'green', icon: CheckCircle },
  rejected: { label: 'Rejected', color: 'orange', icon: XCircle },
};

export const complianceStatusConfig: StatusConfig = {
  compliant: { label: 'Compliant', color: 'green', icon: CheckCircle },
  non_compliant: { label: 'Non-Compliant', color: 'red', icon: AlertTriangle },
  warning: { label: 'Warning', color: 'yellow', icon: AlertCircle },
  not_applicable: { label: 'Not Applicable', color: 'gray', icon: MinusCircle },
};

export const visusAlertStatusConfig: StatusConfig = {
  new: { label: 'New', color: 'red', icon: AlertCircle },
  viewed: { label: 'Viewed', color: 'blue', icon: Eye },
  acknowledged: { label: 'Acknowledged', color: 'yellow', icon: Search },
  actioned: { label: 'Actioned', color: 'green', icon: CheckCircle },
  dismissed: { label: 'Dismissed', color: 'gray', icon: XCircle },
  escalated: { label: 'Escalated', color: 'purple', icon: ArrowUpCircle },
};

export const riskStatusConfig: StatusConfig = {
  identified: { label: 'Identified', color: 'yellow', icon: AlertCircle },
  assessed: { label: 'Assessed', color: 'blue', icon: Eye },
  mitigating: { label: 'Mitigating', color: 'orange', icon: Play },
  accepted: { label: 'Accepted', color: 'green', icon: CheckCircle },
  closed: { label: 'Closed', color: 'gray', icon: XCircle },
};

export const riskTreatmentConfig: StatusConfig = {
  mitigate: { label: 'Mitigate', color: 'blue', icon: Shield },
  transfer: { label: 'Transfer', color: 'purple', icon: ArrowUpCircle },
  accept: { label: 'Accept', color: 'green', icon: CheckCircle },
  avoid: { label: 'Avoid', color: 'orange', icon: Ban },
};

export const policyStatusConfig: StatusConfig = {
  draft: { label: 'Draft', color: 'gray', icon: FileEdit },
  review: { label: 'In Review', color: 'yellow', icon: Eye },
  approved: { label: 'Approved', color: 'green', icon: CheckCircle },
  published: { label: 'Published', color: 'blue', icon: CheckCheck },
  retired: { label: 'Retired', color: 'gray', icon: Ban },
};

export const policyExceptionStatusConfig: StatusConfig = {
  pending: { label: 'Pending', color: 'yellow', icon: Clock },
  approved: { label: 'Approved', color: 'green', icon: CheckCircle },
  rejected: { label: 'Rejected', color: 'red', icon: XCircle },
  expired: { label: 'Expired', color: 'gray', icon: Clock },
};

export const vendorStatusConfig: StatusConfig = {
  active: { label: 'Active', color: 'green', icon: CheckCircle },
  onboarding: { label: 'Onboarding', color: 'blue', icon: Play },
  under_review: { label: 'Under Review', color: 'yellow', icon: Eye },
  offboarding: { label: 'Offboarding', color: 'orange', icon: Clock },
  terminated: { label: 'Terminated', color: 'gray', icon: XCircle },
};

export const questionnaireStatusConfig: StatusConfig = {
  draft: { label: 'Draft', color: 'gray', icon: FileEdit },
  sent: { label: 'Sent', color: 'blue', icon: ArrowUpCircle },
  in_progress: { label: 'In Progress', color: 'yellow', icon: Play },
  completed: { label: 'Completed', color: 'green', icon: CheckCircle },
  expired: { label: 'Expired', color: 'red', icon: Clock },
};

export const evidenceStatusConfig: StatusConfig = {
  current: { label: 'Current', color: 'green', icon: CheckCircle },
  stale: { label: 'Stale', color: 'yellow', icon: AlertTriangle },
  expired: { label: 'Expired', color: 'red', icon: XCircle },
};

export const budgetItemStatusConfig: StatusConfig = {
  proposed: { label: 'Proposed', color: 'yellow', icon: Clock },
  approved: { label: 'Approved', color: 'green', icon: CheckCircle },
  in_progress: { label: 'In Progress', color: 'blue', icon: Play },
  completed: { label: 'Completed', color: 'green', icon: CheckCheck },
  deferred: { label: 'Deferred', color: 'gray', icon: Pause },
};

export const maturityAssessmentStatusConfig: StatusConfig = {
  draft: { label: 'Draft', color: 'gray', icon: FileEdit },
  in_progress: { label: 'In Progress', color: 'blue', icon: Play },
  completed: { label: 'Completed', color: 'green', icon: CheckCircle },
};

export const awarenessStatusConfig: StatusConfig = {
  scheduled: { label: 'Scheduled', color: 'gray', icon: Clock },
  active: { label: 'Active', color: 'blue', icon: Play },
  completed: { label: 'Completed', color: 'green', icon: CheckCircle },
};

export const iamFindingStatusConfig: StatusConfig = {
  open: { label: 'Open', color: 'red', icon: AlertCircle },
  in_progress: { label: 'In Progress', color: 'blue', icon: Play },
  resolved: { label: 'Resolved', color: 'green', icon: CheckCircle },
  accepted: { label: 'Accepted', color: 'gray', icon: CheckCheck },
};

export const playbookStatusConfig: StatusConfig = {
  draft: { label: 'Draft', color: 'gray', icon: FileEdit },
  approved: { label: 'Approved', color: 'green', icon: CheckCircle },
  tested: { label: 'Tested', color: 'blue', icon: CheckCheck },
  retired: { label: 'Retired', color: 'red', icon: Ban },
};

export const simulationResultConfig: StatusConfig = {
  pass: { label: 'Pass', color: 'green', icon: CheckCircle },
  partial: { label: 'Partial', color: 'orange', icon: AlertTriangle },
  fail: { label: 'Fail', color: 'red', icon: XCircle },
};

export const integrationStatusConfig: StatusConfig = {
  connected: { label: 'Connected', color: 'green', icon: CheckCircle },
  disconnected: { label: 'Disconnected', color: 'gray', icon: XCircle },
  error: { label: 'Error', color: 'red', icon: AlertCircle },
  pending: { label: 'Pending', color: 'yellow', icon: Clock },
};

export const integrationHealthConfig: StatusConfig = {
  healthy: { label: 'Healthy', color: 'green', icon: CheckCircle },
  degraded: { label: 'Degraded', color: 'yellow', icon: AlertTriangle },
  unavailable: { label: 'Unavailable', color: 'red', icon: XCircle },
};

export const ownershipStatusConfig: StatusConfig = {
  assigned: { label: 'Assigned', color: 'blue', icon: UserCheck },
  pending_review: { label: 'Pending Review', color: 'yellow', icon: Clock },
  reviewed: { label: 'Reviewed', color: 'green', icon: CheckCircle },
};

export const approvalStatusConfig: StatusConfig = {
  pending: { label: 'Pending', color: 'yellow', icon: Clock },
  approved: { label: 'Approved', color: 'green', icon: CheckCircle },
  rejected: { label: 'Rejected', color: 'red', icon: XCircle },
  escalated: { label: 'Escalated', color: 'purple', icon: ArrowUpCircle },
};

export const obligationStatusConfig: StatusConfig = {
  compliant: { label: 'Compliant', color: 'green', icon: CheckCircle },
  partially_compliant: { label: 'Partially Compliant', color: 'orange', icon: AlertTriangle },
  non_compliant: { label: 'Non-Compliant', color: 'red', icon: XCircle },
  not_assessed: { label: 'Not Assessed', color: 'gray', icon: MinusCircle },
};

export const controlTestResultConfig: StatusConfig = {
  effective: { label: 'Effective', color: 'green', icon: CheckCircle },
  partially_effective: { label: 'Partially Effective', color: 'orange', icon: AlertTriangle },
  ineffective: { label: 'Ineffective', color: 'red', icon: XCircle },
  not_tested: { label: 'Not Tested', color: 'gray', icon: MinusCircle },
};

// Re-export icons that are referenced in status configs for convenience
export {
  Shield,
  User,
  Building2,
  File,
  GitBranch,
  Workflow,
};
