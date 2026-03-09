export interface Tenant {
  id: string;
  name: string;
  slug: string;
  domain: string | null;
  status: TenantStatus;
  plan: TenantPlan;
  settings: TenantSettings;
  branding: TenantBranding;
  owner_id: string;
  created_at: string;
  updated_at: string;
  provisioned_at: string | null;
  deprovisioned_at: string | null;
  user_count: number;
  storage_used_bytes: number;
}

export type TenantStatus = 'active' | 'suspended' | 'provisioning' | 'deprovisioning' | 'deprovisioned';
export type TenantPlan = 'starter' | 'professional' | 'enterprise' | 'custom';

export interface TenantSettings {
  max_users: number;
  max_storage_gb: number;
  enabled_suites: string[];
  mfa_required: boolean;
  session_timeout_minutes: number;
  password_policy: TenantPasswordPolicy;
  ip_whitelist: string[];
  custom_domain: string | null;
}

export interface TenantBranding {
  logo_url: string | null;
  primary_color: string;
  accent_color: string;
  company_name: string;
}

export interface TenantPasswordPolicy {
  min_length: number;
  require_uppercase: boolean;
  require_lowercase: boolean;
  require_numbers: boolean;
  require_special: boolean;
  max_age_days: number;
  history_count: number;
}

export interface ProvisionTenantRequest {
  name: string;
  slug: string;
  plan: TenantPlan;
  owner_email: string;
  owner_name: string;
  settings?: Partial<TenantSettings>;
}

export interface TenantUsage {
  tenant_id: string;
  period: string;
  active_users: number;
  api_calls: number;
  storage_used_bytes: number;
  bandwidth_bytes: number;
  suite_usage: Record<string, SuiteUsage>;
}

export interface SuiteUsage {
  suite: string;
  api_calls: number;
  active_users: number;
  last_accessed: string | null;
}
