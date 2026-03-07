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
  deactivated: { label: "Deactivated", color: "gray", icon: XCircle },
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
  suspended: { label: "Suspended", color: "red", icon: Ban },
  deactivated: { label: "Deactivated", color: "gray", icon: XCircle },
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
  negotiation: { label: 'Negotiation', color: 'orange', icon: Workflow },
  active: { label: 'Active', color: 'green', icon: CheckCircle },
  approved: { label: "Approved", color: "green", icon: CheckCircle },
  signed: { label: "Signed", color: "blue", icon: CheckCheck },
  expired: { label: "Expired", color: "red", icon: AlertCircle },
  terminated: { label: "Terminated", color: "gray", icon: XCircle },
};

export const committeeStatusConfig: StatusConfig = {
  active: { label: 'Active', color: 'green', icon: CheckCircle },
  inactive: { label: 'Inactive', color: 'gray', icon: Pause },
  archived: { label: 'Archived', color: 'gray', icon: XCircle },
};

export const meetingStatusConfig: StatusConfig = {
  scheduled: { label: 'Scheduled', color: 'blue', icon: Clock },
  in_progress: { label: 'In Progress', color: 'yellow', icon: Play },
  completed: { label: 'Completed', color: 'green', icon: CheckCircle },
  cancelled: { label: 'Cancelled', color: 'gray', icon: Ban },
};

export const minuteStatusConfig: StatusConfig = {
  draft: { label: 'Draft', color: 'gray', icon: FileEdit },
  review: { label: 'In Review', color: 'yellow', icon: Eye },
  approved: { label: 'Approved', color: 'green', icon: CheckCircle },
  published: { label: 'Published', color: 'blue', icon: CheckCheck },
};

export const documentStatusConfig: StatusConfig = {
  draft: { label: 'Draft', color: 'gray', icon: FileEdit },
  review: { label: 'In Review', color: 'yellow', icon: Eye },
  approved: { label: 'Approved', color: 'green', icon: CheckCircle },
  archived: { label: 'Archived', color: 'gray', icon: XCircle },
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
