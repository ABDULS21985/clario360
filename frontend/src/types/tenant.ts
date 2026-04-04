/**
 * Tenant types — aligned with backend TenantResponse (iam/dto/tenant_dto.go).
 *
 * Backend returns: id, name, slug, domain, settings (JSONB), status,
 *   subscription_tier, created_at, updated_at.
 */

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  domain: string | null;
  status: TenantStatus;
  subscription_tier: SubscriptionTier;
  /** Opaque JSONB — may be empty `{}` for tenants created without settings. */
  settings: TenantSettings;
  created_at: string;
  updated_at: string;
}

/**
 * Backend enum: active, inactive, suspended, trial, onboarding, deprovisioned.
 * See model/tenant.go TenantStatus constants.
 */
export type TenantStatus =
  | 'active'
  | 'inactive'
  | 'suspended'
  | 'trial'
  | 'onboarding'
  | 'deprovisioned';

/**
 * Backend enum: free, starter, professional, enterprise.
 * See model/tenant.go SubscriptionTier constants.
 */
export type SubscriptionTier = 'free' | 'starter' | 'professional' | 'enterprise';

/**
 * Settings is stored as JSONB in the tenants table.
 * All fields are optional because older tenants may have been created
 * with an empty `{}` settings object.
 */
export interface TenantSettings {
  max_users?: number;
  max_storage_gb?: number;
  enabled_suites?: string[];
  mfa_required?: boolean;
  session_timeout_minutes?: number;
  password_policy?: TenantPasswordPolicy;
  ip_whitelist?: string[];
  custom_domain?: string | null;
  branding?: TenantBranding;
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

/**
 * Provision request — maps to backend ProvisionTenantRequest.
 * Creates tenant + owner user with tenant-admin role.
 */
export interface ProvisionTenantRequest {
  name: string;
  slug: string;
  subscription_tier: SubscriptionTier;
  owner_email: string;
  owner_name: string;
  /** Optional. If omitted the backend generates a random password returned in the response. */
  owner_password?: string;
  settings?: Partial<TenantSettings>;
}

/** Returned by POST /api/v1/tenants/provision — includes the one-time temp_password. */
export interface ProvisionTenantResponse extends Tenant {
  temp_password: string;
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
