export interface User {
  id: string;
  tenant_id: string;
  email: string;
  first_name: string;
  last_name: string;
  status: UserStatus;
  mfa_enabled: boolean;
  last_login_at: string | null;
  password_changed_at: string;
  roles: Role[];
  created_at: string;
  updated_at: string;
}

export type UserStatus = 'active' | 'suspended' | 'deactivated' | 'pending_verification';

export interface Role {
  id: string;
  tenant_id: string;
  name: string;
  slug: string;
  description: string;
  permissions: string[];
  is_system: boolean;
  created_at: string;
  updated_at: string;
}

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  status: TenantStatus;
  settings: TenantSettings;
  max_users: number;
  created_at: string;
  updated_at: string;
}

export type TenantStatus = 'active' | 'suspended' | 'deactivated';

export interface TenantSettings {
  self_registration_enabled: boolean;
  mfa_required: boolean;
  session_timeout_minutes: number;
  password_policy: PasswordPolicy;
  branding: BrandingSettings;
}

export interface PasswordPolicy {
  min_length: number;
  require_uppercase: boolean;
  require_lowercase: boolean;
  require_digit: boolean;
  require_special: boolean;
  max_age_days: number;
}

export interface BrandingSettings {
  primary_color: string;
  accent_color: string;
  logo_url: string | null;
  app_name: string;
}

export type SuiteName = 'cyber' | 'data' | 'acta' | 'lex' | 'visus';

export type NotificationCategory =
  | 'security'
  | 'data'
  | 'workflow'
  | 'system'
  | 'governance'
  | 'legal';

export type NotificationPriority = 'critical' | 'high' | 'medium' | 'low';

export interface Notification {
  id: string;
  type?: string;
  title: string;
  body: string;
  category: NotificationCategory;
  priority: NotificationPriority;
  data?: Record<string, unknown> | null;
  action_url?: string | null;
  read: boolean;
  read_at?: string | null;
  created_at: string;
}

export type ConnectionStatus =
  | 'connecting'
  | 'connected'
  | 'disconnected'
  | 'reconnecting'
  | 'failed';

export interface WorkflowTask {
  id: string;
  name: string;
  workflow_id: string;
  workflow_name: string;
  status: 'pending' | 'claimed' | 'completed' | 'failed' | 'overdue';
  assigned_to: string | null;
  due_at: string | null;
  created_at: string;
  updated_at: string;
}

export type FormFieldType = 'boolean' | 'text' | 'textarea' | 'select' | 'number' | 'date';

export interface FormField {
  name: string;
  type: FormFieldType;
  label: string;
  required: boolean;
  default?: unknown;
  options?: string[];
  placeholder?: string;
  description?: string;
}

export type HumanTaskStatus = 'pending' | 'claimed' | 'completed' | 'rejected' | 'escalated' | 'cancelled';
export type TaskPriority = 0 | 1 | 2;

export interface HumanTask {
  id: string;
  name: string;
  description: string;
  instance_id: string;
  definition_name: string;
  workflow_name: string;
  step_id: string;
  status: HumanTaskStatus;
  priority: TaskPriority;
  form_schema: FormField[];
  form_data: Record<string, unknown> | null;
  sla_deadline: string | null;
  sla_breached: boolean;
  claimed_by: string | null;
  claimed_by_name: string | null;
  assignee_role: string | null;
  assignee_id: string | null;
  metadata: Record<string, unknown>;
  claimed_at?: string | null;
  delegated_by?: string | null;
  delegated_at?: string | null;
  completed_at?: string | null;
  created_at: string;
  updated_at: string;
}

export type StepType = 'human_task' | 'service_task' | 'condition' | 'parallel_gateway' | 'timer' | 'end';
export type StepStatus = 'completed' | 'running' | 'failed' | 'pending' | 'skipped' | 'cancelled';

export interface StepDefinition {
  id: string;
  name: string;
  type: StepType;
  description?: string;
}

export interface StepExecution {
  id: string;
  step_id: string;
  step_name: string;
  step_type: StepType;
  status: StepStatus;
  started_at: string | null;
  completed_at: string | null;
  duration_seconds: number | null;
  attempt: number;
  input: Record<string, unknown> | null;
  output: Record<string, unknown> | null;
  error: string | null;
  assigned_to: string | null;
  completed_by: string | null;
}

export type WorkflowInstanceStatus = 'running' | 'completed' | 'failed' | 'cancelled' | 'suspended';

export interface WorkflowInstance {
  id: string;
  definition_id: string;
  definition_name: string;
  tenant_id: string;
  status: WorkflowInstanceStatus;
  current_step_id: string | null;
  current_step_name: string | null;
  total_steps: number;
  completed_steps: number;
  started_at: string;
  completed_at: string | null;
  started_by: string | null;
  started_by_name: string | null;
  variables: Record<string, unknown>;
  step_outputs: Record<string, Record<string, unknown>>;
  definition_steps: StepDefinition[];
  error_message?: string | null;
  updated_at?: string;
}

export interface TaskCounts {
  pending: number;
  claimed_by_me: number;
  completed: number;
  overdue: number;
  escalated: number;
}

export interface AuditLog {
  id: string;
  tenant_id: string;
  user_id: string | null;
  user_email: string;
  action: string;
  service?: string;
  resource_type: string;
  resource_id: string | null;
  old_value?: Record<string, unknown> | null;
  new_value?: Record<string, unknown> | null;
  severity?: 'info' | 'warning' | 'high' | 'critical';
  ip_address: string;
  user_agent: string;
  correlation_id?: string;
  entry_hash?: string;
  prev_hash?: string;
  metadata: Record<string, unknown>;
  created_at: string;
}

export interface FileItem {
  id: string;
  tenant_id: string;
  name: string;
  original_name: string;
  content_type: string;
  size: number;
  status: 'pending' | 'processing' | 'available' | 'quarantined' | 'deleted';
  uploaded_by: string;
  created_at: string;
  updated_at: string;
}

export interface Alert {
  id: string;
  tenant_id: string;
  title: string;
  description: string;
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info';
  status: 'new' | 'acknowledged' | 'investigating' | 'resolved' | 'false_positive';
  source: string;
  created_at: string;
  updated_at: string;
}
