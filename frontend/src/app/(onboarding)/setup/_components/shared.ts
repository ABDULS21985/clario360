'use client';

import { z } from 'zod';

export const wizardDraftKey = 'clario360:onboarding-wizard';

export const organizationSchema = z.object({
  organization_name: z.string().min(2).max(100),
  industry: z.string().min(1),
  country: z.string().length(2),
  city: z.string().max(120).optional().or(z.literal('')),
  organization_size: z.string().min(1),
});

export const brandingSchema = z.object({
  primary_color: z.string().regex(/^#[0-9A-Fa-f]{6}$/),
  accent_color: z.string().regex(/^#[0-9A-Fa-f]{6}$/),
});

export type OrganizationFormValues = z.infer<typeof organizationSchema>;
export type BrandingFormValues = z.infer<typeof brandingSchema>;

export type WizardProgress = {
  tenant_id: string;
  current_step: number;
  steps_completed: number[];
  wizard_completed: boolean;
  email_verified: boolean;
  organization_name?: string | null;
  industry?: string | null;
  country: string;
  city?: string | null;
  organization_size?: string | null;
  logo_file_id?: string | null;
  primary_color?: string | null;
  accent_color?: string | null;
  active_suites: string[];
  provisioning_status: 'pending' | 'provisioning' | 'completed' | 'failed';
  provisioning_error?: string | null;
};

export type ProvisioningStep = {
  step_number: number;
  step_name: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'skipped';
  error_message?: string | null;
};

export type ProvisioningStatus = {
  tenant_id: string;
  status: 'pending' | 'provisioning' | 'completed' | 'failed';
  error?: string | null;
  progress_pct: number;
  completed_steps: number;
  total_steps: number;
  steps: ProvisioningStep[];
};

export type RoleRecord = {
  id: string;
  name: string;
  slug: string;
};

export type InvitationDraft = {
  email: string;
  role_slug: string;
  message?: string;
};

export type WizardDraft = {
  organization?: OrganizationFormValues;
  branding?: BrandingFormValues;
  team?: InvitationDraft[];
  suites?: string[];
};

export const INDUSTRIES = [
  { value: 'financial', label: 'Financial Services' },
  { value: 'government', label: 'Government' },
  { value: 'healthcare', label: 'Healthcare' },
  { value: 'technology', label: 'Technology' },
  { value: 'energy', label: 'Energy' },
  { value: 'telecom', label: 'Telecom' },
  { value: 'education', label: 'Education' },
  { value: 'retail', label: 'Retail' },
  { value: 'manufacturing', label: 'Manufacturing' },
  { value: 'other', label: 'Other' },
] as const;

export const ORG_SIZES = [
  { value: '1-50', label: '1-50' },
  { value: '51-200', label: '51-200' },
  { value: '201-1000', label: '201-1000' },
  { value: '1000+', label: '1000+' },
] as const;

export const SUITES = [
  {
    id: 'cyber',
    title: 'Cybersecurity',
    description: 'Threat detection, asset management, SOC dashboards',
    accent: 'from-[#0f5132] to-[#2b7d59]',
  },
  {
    id: 'data',
    title: 'Data Intelligence',
    description: 'Data quality, pipeline orchestration, contradiction detection',
    accent: 'from-[#155e75] to-[#0ea5b7]',
  },
  {
    id: 'acta',
    title: 'Board Governance',
    description: 'Meeting automation, minutes, compliance tracking',
    accent: 'from-[#7c2d12] to-[#ea580c]',
  },
  {
    id: 'lex',
    title: 'Legal Operations',
    description: 'Contract management, clause analysis, expiry monitoring',
    accent: 'from-[#6b21a8] to-[#9333ea]',
  },
  {
    id: 'visus',
    title: 'Executive Intelligence',
    description: 'Cross-suite dashboards, KPIs, executive reports',
    accent: 'from-[#9a3412] to-[#d97706]',
  },
] as const;

export const COUNTRY_OPTIONS = ['SA', 'AE', 'US', 'GB', 'NG', 'ZA', 'EG', 'KE', 'DE', 'FR'];

export function loadDraft(): WizardDraft {
  if (typeof window === 'undefined') {
    return {};
  }

  try {
    const stored = window.localStorage.getItem(wizardDraftKey);
    return stored ? (JSON.parse(stored) as WizardDraft) : {};
  } catch {
    return {};
  }
}

export function saveDraft(nextDraft: WizardDraft): void {
  if (typeof window === 'undefined') {
    return;
  }
  window.localStorage.setItem(wizardDraftKey, JSON.stringify(nextDraft));
}

export function clearDraft(): void {
  if (typeof window === 'undefined') {
    return;
  }
  window.localStorage.removeItem(wizardDraftKey);
}
