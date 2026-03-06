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
