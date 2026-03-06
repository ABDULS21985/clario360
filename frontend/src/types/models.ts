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
  title: string;
  body: string;
  category: NotificationCategory;
  priority: NotificationPriority;
  action_url?: string;
  read: boolean;
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
