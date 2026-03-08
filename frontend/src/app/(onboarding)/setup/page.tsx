'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  AlertCircle,
  ArrowRight,
  Building2,
  CheckCircle2,
  ChevronLeft,
  ImagePlus,
  LayoutGrid,
  Loader2,
  ShieldCheck,
  Sparkles,
  Users,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Progress } from '@/components/ui/progress';
import { Checkbox } from '@/components/ui/checkbox';
import { Spinner } from '@/components/ui/spinner';
import { apiGet, apiPost } from '@/lib/api';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';
import { cn } from '@/lib/utils';
import { isApiError } from '@/types/api';
import { useAuthStore } from '@/stores/auth-store';

const wizardDraftKey = 'clario360:onboarding-wizard';

const organizationSchema = z.object({
  organization_name: z.string().min(2).max(100),
  industry: z.string().min(1),
  country: z.string().length(2),
  city: z.string().max(120).optional().or(z.literal('')),
  organization_size: z.string().min(1),
});

const brandingSchema = z.object({
  primary_color: z.string().regex(/^#[0-9A-Fa-f]{6}$/),
  accent_color: z.string().regex(/^#[0-9A-Fa-f]{6}$/),
});

type OrganizationFormValues = z.infer<typeof organizationSchema>;
type BrandingFormValues = z.infer<typeof brandingSchema>;

type WizardProgress = {
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
  primary_color?: string | null;
  accent_color?: string | null;
  active_suites: string[];
  provisioning_status: 'pending' | 'provisioning' | 'completed' | 'failed';
  provisioning_error?: string | null;
};

type ProvisioningStep = {
  step_number: number;
  step_name: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'skipped';
  error_message?: string | null;
};

type ProvisioningStatus = {
  tenant_id: string;
  status: 'pending' | 'provisioning' | 'completed' | 'failed';
  error?: string | null;
  progress_pct: number;
  completed_steps: number;
  total_steps: number;
  steps: ProvisioningStep[];
};

type RoleRecord = {
  id: string;
  name: string;
  slug: string;
};

type InvitationDraft = {
  email: string;
  role_slug: string;
  message?: string;
};

type WizardDraft = {
  organization?: OrganizationFormValues;
  branding?: BrandingFormValues;
  team?: InvitationDraft[];
  suites?: string[];
};

const INDUSTRIES = [
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

const ORG_SIZES = [
  { value: '1-50', label: '1-50' },
  { value: '51-200', label: '51-200' },
  { value: '201-1000', label: '201-1000' },
  { value: '1000+', label: '1000+' },
] as const;

const SUITES = [
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

const COUNTRY_OPTIONS = ['SA', 'AE', 'US', 'GB', 'NG', 'ZA', 'EG', 'KE', 'DE', 'FR'];

function loadDraft(): WizardDraft {
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

function saveDraft(nextDraft: WizardDraft): void {
  if (typeof window === 'undefined') {
    return;
  }
  window.localStorage.setItem(wizardDraftKey, JSON.stringify(nextDraft));
}

function clearDraft(): void {
  if (typeof window === 'undefined') {
    return;
  }
  window.localStorage.removeItem(wizardDraftKey);
}

function StepIndicator({ currentStep }: { currentStep: number }) {
  const steps = [
    { number: 1, label: 'Organization', icon: Building2 },
    { number: 2, label: 'Branding', icon: ImagePlus },
    { number: 3, label: 'Team', icon: Users },
    { number: 4, label: 'Suites', icon: LayoutGrid },
    { number: 5, label: 'Ready', icon: Sparkles },
  ] as const;

  return (
    <div className="mb-8">
      <div className="mb-4 flex items-center justify-between gap-2">
        {steps.map((step) => {
          const Icon = step.icon;
          const isActive = currentStep === step.number;
          const isComplete = currentStep > step.number;
          return (
            <div key={step.number} className="flex flex-1 flex-col items-center gap-2">
              <div
                className={cn(
                  'flex h-11 w-11 items-center justify-center rounded-full border text-sm transition-all',
                  isComplete && 'border-[#0f5132] bg-[#0f5132] text-white',
                  isActive && 'border-[#0f5132] bg-white text-[#0f5132] shadow-sm',
                  !isComplete && !isActive && 'border-slate-200 bg-white text-slate-400',
                )}
              >
                {isComplete ? <CheckCircle2 className="h-4 w-4" /> : <Icon className="h-4 w-4" />}
              </div>
              <span className={cn('text-[11px] uppercase tracking-[0.2em]', isActive ? 'text-[#0f5132]' : 'text-slate-400')}>
                {step.label}
              </span>
            </div>
          );
        })}
      </div>
      <Progress value={((currentStep - 1) / 4) * 100} className="h-2 bg-slate-100" />
    </div>
  );
}

function StepOrganization({
  initialValues,
  onBack,
  onSaved,
  onPersist,
}: {
  initialValues: OrganizationFormValues;
  onBack?: () => void;
  onSaved: () => Promise<void>;
  onPersist: (values: OrganizationFormValues) => void;
}) {
  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<OrganizationFormValues>({
    resolver: zodResolver(organizationSchema),
    defaultValues: initialValues,
  });
  const [apiError, setApiError] = useState<string | null>(null);
  const watched = watch();

  useEffect(() => {
    onPersist(watched as OrganizationFormValues);
  }, [watched, onPersist]);

  const submit = handleSubmit(async (values) => {
    setApiError(null);
    try {
      await apiPost(API_ENDPOINTS.ONBOARDING_ORGANIZATION, values);
      await onSaved();
    } catch (error) {
      setApiError(isApiError(error) ? error.message : 'Failed to save organization details.');
    }
  });

  return (
    <form onSubmit={submit} className="space-y-6">
      {apiError && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{apiError}</AlertDescription>
        </Alert>
      )}

      <div className="space-y-2">
        <Label htmlFor="organization_name">Organization name</Label>
        <Input id="organization_name" {...register('organization_name')} />
        {errors.organization_name && <p className="text-sm text-destructive">{errors.organization_name.message}</p>}
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="industry">Industry</Label>
          <select
            id="industry"
            {...register('industry')}
            className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
          >
            {INDUSTRIES.map((industry) => (
              <option key={industry.value} value={industry.value}>
                {industry.label}
              </option>
            ))}
          </select>
        </div>
        <div className="space-y-2">
          <Label htmlFor="country">Country</Label>
          <Input id="country" list="onboarding-country-options" maxLength={2} {...register('country')} />
          <datalist id="onboarding-country-options">
            {COUNTRY_OPTIONS.map((country) => (
              <option key={country} value={country} />
            ))}
          </datalist>
          {errors.country && <p className="text-sm text-destructive">{errors.country.message}</p>}
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="city">City</Label>
          <Input id="city" {...register('city')} />
        </div>
        <div className="space-y-2">
          <Label>Organization size</Label>
          <div className="grid gap-2 sm:grid-cols-2">
            {ORG_SIZES.map((size) => (
              <label key={size.value} className="flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-2 text-sm">
                <input type="radio" value={size.value} {...register('organization_size')} className="accent-[#0f5132]" />
                <span>{size.label}</span>
              </label>
            ))}
          </div>
        </div>
      </div>

      <div className="flex justify-between">
        <Button type="button" variant="ghost" disabled={!onBack} onClick={onBack}>
          <ChevronLeft className="mr-1 h-4 w-4" />
          Back
        </Button>
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
          Continue
          <ArrowRight className="ml-1 h-4 w-4" />
        </Button>
      </div>
    </form>
  );
}

function StepBranding({
  initialValues,
  onBack,
  onSaved,
  onPersist,
}: {
  initialValues: BrandingFormValues;
  onBack: () => void;
  onSaved: () => Promise<void>;
  onPersist: (values: BrandingFormValues) => void;
}) {
  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<BrandingFormValues>({
    resolver: zodResolver(brandingSchema),
    defaultValues: initialValues,
  });
  const [apiError, setApiError] = useState<string | null>(null);
  const values = watch();

  useEffect(() => {
    onPersist(values as BrandingFormValues);
  }, [values, onPersist]);

  const submit = handleSubmit(async (branding) => {
    setApiError(null);
    try {
      await apiPost(API_ENDPOINTS.ONBOARDING_BRANDING, branding);
      await onSaved();
    } catch (error) {
      setApiError(isApiError(error) ? error.message : 'Failed to save branding.');
    }
  });

  return (
    <form onSubmit={submit} className="space-y-6">
      {apiError && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{apiError}</AlertDescription>
        </Alert>
      )}

      <div className="rounded-2xl border border-dashed border-[#0f5132]/20 bg-[#0f5132]/5 p-4 text-sm text-slate-600">
        Logo upload is not available in this deployment yet. Brand colors will be applied immediately.
      </div>

      <div className="grid gap-5 md:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="primary_color">Primary color</Label>
          <div className="flex items-center gap-3">
            <input type="color" id="primary_color" className="h-11 w-14 rounded-md border border-slate-200 bg-white" {...register('primary_color')} />
            <Input {...register('primary_color')} />
          </div>
          {errors.primary_color && <p className="text-sm text-destructive">{errors.primary_color.message}</p>}
        </div>
        <div className="space-y-2">
          <Label htmlFor="accent_color">Accent color</Label>
          <div className="flex items-center gap-3">
            <input type="color" id="accent_color" className="h-11 w-14 rounded-md border border-slate-200 bg-white" {...register('accent_color')} />
            <Input {...register('accent_color')} />
          </div>
          {errors.accent_color && <p className="text-sm text-destructive">{errors.accent_color.message}</p>}
        </div>
      </div>

      <div className="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div className="flex items-center justify-between border-b border-slate-100 px-5 py-4">
          <div>
            <p className="text-sm font-semibold text-slate-900">Executive Overview Preview</p>
            <p className="text-xs text-slate-500">A quick feel for your chosen palette</p>
          </div>
          <ShieldCheck className="h-5 w-5 text-slate-400" />
        </div>
        <div className="grid gap-3 bg-slate-50 p-5 md:grid-cols-4">
          <div className="rounded-2xl p-4 text-white" style={{ backgroundColor: values.primary_color }}>
            <p className="text-xs uppercase tracking-[0.2em] text-white/70">Risk Score</p>
            <p className="mt-3 text-3xl font-semibold">84</p>
          </div>
          <div className="rounded-2xl border border-slate-200 bg-white p-4">
            <p className="text-xs uppercase tracking-[0.2em] text-slate-400">Critical Alerts</p>
            <p className="mt-3 text-3xl font-semibold text-slate-900">12</p>
          </div>
          <div className="rounded-2xl border border-slate-200 bg-white p-4">
            <p className="text-xs uppercase tracking-[0.2em] text-slate-400">Compliance</p>
            <p className="mt-3 text-3xl font-semibold text-slate-900">91%</p>
          </div>
          <div className="rounded-2xl p-4 text-slate-950" style={{ backgroundColor: values.accent_color }}>
            <p className="text-xs uppercase tracking-[0.2em] text-slate-900/60">Actions</p>
            <p className="mt-3 text-3xl font-semibold">7</p>
          </div>
        </div>
      </div>

      <div className="flex justify-between">
        <Button type="button" variant="outline" onClick={onBack}>
          <ChevronLeft className="mr-1 h-4 w-4" />
          Back
        </Button>
        <div className="flex gap-2">
          <Button type="button" variant="ghost" onClick={onSaved}>
            Skip
          </Button>
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            Continue
          </Button>
        </div>
      </div>
    </form>
  );
}

function StepTeam({
  roles,
  initialRows,
  onBack,
  onSaved,
  onPersist,
}: {
  roles: RoleRecord[];
  initialRows: InvitationDraft[];
  onBack: () => void;
  onSaved: () => Promise<void>;
  onPersist: (rows: InvitationDraft[]) => void;
}) {
  const [rows, setRows] = useState<InvitationDraft[]>(
    initialRows.length > 0 ? initialRows : [{ email: '', role_slug: roles[0]?.slug ?? 'viewer', message: '' }],
  );
  const [apiError, setApiError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [sentCount, setSentCount] = useState<number | null>(null);

  useEffect(() => {
    onPersist(rows);
  }, [rows, onPersist]);

  const updateRow = (index: number, field: keyof InvitationDraft, value: string) => {
    setRows((current) => current.map((row, rowIndex) => (rowIndex === index ? { ...row, [field]: value } : row)));
  };

  const addRow = () => {
    setRows((current) =>
      current.length >= 10 ? current : [...current, { email: '', role_slug: roles[0]?.slug ?? 'viewer', message: '' }],
    );
  };

  const removeRow = (index: number) => {
    setRows((current) => current.filter((_, rowIndex) => rowIndex !== index));
  };

  const submit = async (skip = false) => {
    setApiError(null);
    setSentCount(null);
    setIsSubmitting(true);
    try {
      const invitations = skip
        ? []
        : rows
            .filter((row) => row.email.trim() !== '')
            .map((row) => ({
              email: row.email.trim(),
              role_slug: row.role_slug,
              message: row.message?.trim() || undefined,
            }));

      const response = await apiPost<{ invitations_sent?: number; data?: unknown[]; count?: number }>(
        API_ENDPOINTS.ONBOARDING_TEAM,
        { invitations },
      );
      setSentCount(response.invitations_sent ?? response.count ?? invitations.length);
      await onSaved();
    } catch (error) {
      setApiError(isApiError(error) ? error.message : 'Failed to send invitations.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-6">
      {apiError && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{apiError}</AlertDescription>
        </Alert>
      )}
      {sentCount !== null && (
        <Alert className="border-[#0f5132]/20 bg-[#0f5132]/5 text-[#0f5132]">
          <CheckCircle2 className="h-4 w-4" />
          <AlertDescription>{sentCount} invitation{sentCount === 1 ? '' : 's'} sent.</AlertDescription>
        </Alert>
      )}

      <div className="space-y-3">
        {rows.map((row, index) => (
          <div key={index} className="grid gap-3 rounded-2xl border border-slate-200 bg-white p-4 md:grid-cols-[1.5fr_1fr_auto]">
            <div className="space-y-2">
              <Label>Email</Label>
              <Input
                type="email"
                value={row.email}
                onChange={(event) => updateRow(index, 'email', event.target.value)}
                placeholder="alice@company.com"
              />
            </div>
            <div className="space-y-2">
              <Label>Role</Label>
              <select
                value={row.role_slug}
                onChange={(event) => updateRow(index, 'role_slug', event.target.value)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              >
                {roles.map((role) => (
                  <option key={role.id} value={role.slug}>
                    {role.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex items-end">
              <Button type="button" variant="ghost" onClick={() => removeRow(index)} disabled={rows.length === 1}>
                Remove
              </Button>
            </div>
          </div>
        ))}
      </div>

      <div className="flex items-center justify-between">
        <Button type="button" variant="outline" onClick={addRow} disabled={rows.length >= 10}>
          Add another
        </Button>
        <span className="text-sm text-slate-500">{rows.length}/10 rows</span>
      </div>

      <div className="flex justify-between">
        <Button type="button" variant="outline" onClick={onBack}>
          <ChevronLeft className="mr-1 h-4 w-4" />
          Back
        </Button>
        <div className="flex gap-2">
          <Button type="button" variant="ghost" disabled={isSubmitting} onClick={() => void submit(true)}>
            Skip
          </Button>
          <Button type="button" disabled={isSubmitting} onClick={() => void submit(false)}>
            {isSubmitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            Continue
          </Button>
        </div>
      </div>
    </div>
  );
}

function StepSuites({
  initialSelected,
  onBack,
  onSaved,
  onPersist,
}: {
  initialSelected: string[];
  onBack: () => void;
  onSaved: () => Promise<void>;
  onPersist: (selected: string[]) => void;
}) {
  const [selected, setSelected] = useState<string[]>(initialSelected.length > 0 ? initialSelected : ['cyber', 'data', 'visus']);
  const [apiError, setApiError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    onPersist(selected);
  }, [selected, onPersist]);

  const toggleSuite = (suiteId: string) => {
    setSelected((current) =>
      current.includes(suiteId) ? current.filter((item) => item !== suiteId) : [...current, suiteId],
    );
  };

  const submit = async () => {
    if (selected.length === 0) {
      setApiError('Select at least one suite.');
      return;
    }

    setApiError(null);
    setIsSubmitting(true);
    try {
      await apiPost(API_ENDPOINTS.ONBOARDING_SUITES, { active_suites: selected });
      await onSaved();
    } catch (error) {
      setApiError(isApiError(error) ? error.message : 'Failed to save suite selection.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-6">
      {apiError && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{apiError}</AlertDescription>
        </Alert>
      )}

      <div className="grid gap-4 md:grid-cols-2">
        {SUITES.map((suite) => {
          const active = selected.includes(suite.id);
          return (
            <button
              key={suite.id}
              type="button"
              onClick={() => toggleSuite(suite.id)}
              className={cn(
                'overflow-hidden rounded-3xl border text-left transition-all',
                active ? 'border-[#0f5132] shadow-md' : 'border-slate-200 hover:border-slate-300',
              )}
            >
              <div className={cn('h-2 w-full bg-gradient-to-r', suite.accent)} />
              <div className="space-y-4 bg-white p-5">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-base font-semibold text-slate-900">{suite.title}</p>
                    <p className="mt-1 text-sm text-slate-500">{suite.description}</p>
                  </div>
                  <Checkbox checked={active} />
                </div>
              </div>
            </button>
          );
        })}
      </div>

      <div className="flex justify-between">
        <Button type="button" variant="outline" onClick={onBack}>
          <ChevronLeft className="mr-1 h-4 w-4" />
          Back
        </Button>
        <Button type="button" disabled={isSubmitting || selected.length === 0} onClick={() => void submit()}>
          {isSubmitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
          Continue
        </Button>
      </div>
    </div>
  );
}

function StepReady({
  tenantID,
  initialStatus,
  onBack,
}: {
  tenantID: string;
  initialStatus: WizardProgress['provisioning_status'];
  onBack: () => void;
}) {
  const router = useRouter();
  const [status, setStatus] = useState<ProvisioningStatus | null>(null);
  const [completeError, setCompleteError] = useState<string | null>(null);
  const [isCompleting, setIsCompleting] = useState(true);
  const [currentStatus, setCurrentStatus] = useState(initialStatus);

  useEffect(() => {
    let active = true;

    const completeWizard = async () => {
      try {
        await apiPost(API_ENDPOINTS.ONBOARDING_COMPLETE, {});
      } catch (error) {
        if (active) {
          setCompleteError(isApiError(error) ? error.message : 'Failed to finalize onboarding.');
        }
      } finally {
        if (active) {
          setIsCompleting(false);
        }
      }
    };

    void completeWizard();

    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (!tenantID) {
      return undefined;
    }

    let active = true;
    let timer: ReturnType<typeof setInterval> | null = null;

    const poll = async () => {
      try {
        const nextStatus = await apiGet<ProvisioningStatus>(`${API_ENDPOINTS.ONBOARDING_STATUS}/${tenantID}`);
        if (!active) {
          return;
        }
        setStatus(nextStatus);
        setCurrentStatus(nextStatus.status);

        if (nextStatus.status === 'completed') {
          clearDraft();
          if (timer) {
            clearInterval(timer);
          }
        }
        if (nextStatus.status === 'failed' && timer) {
          clearInterval(timer);
        }
      } catch {
        // keep polling through transient network failures
      }
    };

    void poll();
    timer = setInterval(() => {
      void poll();
    }, 2000);

    return () => {
      active = false;
      if (timer) {
        clearInterval(timer);
      }
    };
  }, [tenantID]);

  const steps = status?.steps ?? [];
  const progressPct = status?.progress_pct ?? (currentStatus === 'completed' ? 100 : 0);

  return (
    <div className="space-y-6">
      {completeError && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{completeError}</AlertDescription>
        </Alert>
      )}

      {currentStatus === 'completed' ? (
        <div className="space-y-6 text-center">
          <div className="mx-auto flex h-20 w-20 items-center justify-center rounded-full bg-[#0f5132]/10">
            <CheckCircle2 className="h-10 w-10 text-[#0f5132]" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold text-slate-900">Your Clario 360 platform is ready!</h2>
            <p className="mt-2 text-sm text-slate-500">Provisioning completed successfully. You can start from the dashboard or jump into a suite.</p>
          </div>
          <Button size="lg" className="w-full" onClick={() => router.push(ROUTES.DASHBOARD)}>
            Go to Dashboard
          </Button>
          <div className="grid gap-3 md:grid-cols-3">
            <button type="button" className="rounded-2xl border border-slate-200 bg-white p-4 text-left" onClick={() => router.push('/data/sources?create=true')}>
              <p className="font-medium text-slate-900">Connect your first data source</p>
              <p className="mt-1 text-sm text-slate-500">Start ingesting structured data.</p>
            </button>
            <button type="button" className="rounded-2xl border border-slate-200 bg-white p-4 text-left" onClick={() => router.push('/cyber/assets?scan=true')}>
              <p className="font-medium text-slate-900">Set up asset scanning</p>
              <p className="mt-1 text-sm text-slate-500">Bring cyber inventory online.</p>
            </button>
            <button type="button" className="rounded-2xl border border-slate-200 bg-white p-4 text-left" onClick={() => router.push('/acta/meetings?create=true')}>
              <p className="font-medium text-slate-900">Schedule a board meeting</p>
              <p className="mt-1 text-sm text-slate-500">Begin governance workflows.</p>
            </button>
          </div>
        </div>
      ) : (
        <div className="space-y-6">
          <div className="text-center">
            <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-[#0f5132]/10">
              {isCompleting ? <Loader2 className="h-8 w-8 animate-spin text-[#0f5132]" /> : <Sparkles className="h-8 w-8 text-[#0f5132]" />}
            </div>
            <h2 className="mt-4 text-2xl font-semibold text-slate-900">Provisioning your workspace</h2>
            <p className="mt-2 text-sm text-slate-500">We are setting up tenant defaults, security roles, dashboards, and storage.</p>
          </div>

          <div className="space-y-2">
            <Progress value={progressPct} className="h-2 bg-slate-100" />
            <p className="text-right text-xs uppercase tracking-[0.2em] text-slate-500">{progressPct}% complete</p>
          </div>

          <div className="space-y-3">
            {steps.map((step) => (
              <div key={step.step_number} className="flex items-start justify-between rounded-2xl border border-slate-200 bg-white px-4 py-3">
                <div>
                  <p className="font-medium text-slate-900">{step.step_name}</p>
                  {step.error_message && <p className="mt-1 text-sm text-destructive">{step.error_message}</p>}
                </div>
                <span
                  className={cn(
                    'rounded-full px-2.5 py-1 text-xs font-medium uppercase tracking-[0.15em]',
                    step.status === 'completed' && 'bg-[#0f5132]/10 text-[#0f5132]',
                    step.status === 'running' && 'bg-[#d97706]/10 text-[#d97706]',
                    step.status === 'failed' && 'bg-red-500/10 text-red-600',
                    step.status === 'pending' && 'bg-slate-100 text-slate-500',
                    step.status === 'skipped' && 'bg-slate-200 text-slate-600',
                  )}
                >
                  {step.status}
                </span>
              </div>
            ))}
          </div>

          {currentStatus === 'failed' && (
            <div className="flex justify-between">
              <Button variant="outline" onClick={onBack}>
                <ChevronLeft className="mr-1 h-4 w-4" />
                Back
              </Button>
              <Button onClick={() => router.push(ROUTES.DASHBOARD)}>Go to Dashboard</Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default function SetupPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const tenant = useAuthStore((state) => state.tenant);

  const [wizard, setWizard] = useState<WizardProgress | null>(null);
  const [roles, setRoles] = useState<RoleRecord[]>([]);
  const [draft, setDraft] = useState<WizardDraft>({});
  const [currentStep, setCurrentStep] = useState(1);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  useEffect(() => {
    setDraft(loadDraft());
  }, []);

  const syncStep = (step: number) => {
    const nextStep = Math.min(Math.max(step, 1), 5);
    setCurrentStep(nextStep);
    router.replace(`${ROUTES.SETUP}?step=${nextStep}`);
  };

  const refreshWizard = async (preferredStep?: number) => {
    const [wizardProgress, roleRecords] = await Promise.all([
      apiGet<WizardProgress>(API_ENDPOINTS.ONBOARDING_WIZARD),
      apiGet<RoleRecord[]>(API_ENDPOINTS.ROLES).catch(() => []),
    ]);

    setWizard(wizardProgress);
    setRoles(roleRecords);

    if (wizardProgress.wizard_completed && wizardProgress.provisioning_status === 'completed') {
      clearDraft();
      router.replace(ROUTES.DASHBOARD);
      return;
    }

    const queryStep = Number(searchParams.get('step'));
    const resolvedStep =
      preferredStep ??
      (Number.isFinite(queryStep) && queryStep >= 1 && queryStep <= 5
        ? queryStep
        : wizardProgress.wizard_completed
          ? 5
          : Math.max(1, wizardProgress.current_step || 1));

    setCurrentStep(resolvedStep);
  };

  useEffect(() => {
    let active = true;

    const load = async () => {
      setIsLoading(true);
      setLoadError(null);
      try {
        await refreshWizard();
      } catch (error) {
        if (active) {
          setLoadError(isApiError(error) ? error.message : 'Failed to load onboarding wizard.');
        }
      } finally {
        if (active) {
          setIsLoading(false);
        }
      }
    };

    void load();

    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (!wizard) {
      return;
    }
    const queryStep = Number(searchParams.get('step'));
    if (Number.isFinite(queryStep) && queryStep >= 1 && queryStep <= 5 && queryStep !== currentStep) {
      setCurrentStep(queryStep);
    }
  }, [searchParams, wizard, currentStep]);

  const organizationDefaults = useMemo<OrganizationFormValues>(() => ({
    organization_name: draft.organization?.organization_name ?? wizard?.organization_name ?? '',
    industry: draft.organization?.industry ?? wizard?.industry ?? 'financial',
    country: draft.organization?.country ?? wizard?.country ?? 'SA',
    city: draft.organization?.city ?? wizard?.city ?? '',
    organization_size: draft.organization?.organization_size ?? wizard?.organization_size ?? '1-50',
  }), [draft.organization, wizard]);

  const brandingDefaults = useMemo<BrandingFormValues>(() => ({
    primary_color: draft.branding?.primary_color ?? wizard?.primary_color ?? '#006B3F',
    accent_color: draft.branding?.accent_color ?? wizard?.accent_color ?? '#C5A04E',
  }), [draft.branding, wizard]);

  const teamDefaults = draft.team ?? [{ email: '', role_slug: roles[0]?.slug ?? 'viewer', message: '' }];
  const suiteDefaults = draft.suites ?? wizard?.active_suites ?? ['cyber', 'data', 'visus'];

  const persistDraft = (nextDraft: WizardDraft) => {
    setDraft(nextDraft);
    saveDraft(nextDraft);
  };

  if (isLoading) {
    return (
      <div className="flex w-full max-w-3xl flex-col items-center justify-center gap-3 rounded-[28px] border border-white/70 bg-white/90 px-8 py-16 shadow-[0_30px_80px_rgba(15,81,50,0.08)]">
        <Spinner />
        <p className="text-sm text-slate-500">Loading your onboarding wizard…</p>
      </div>
    );
  }

  if (loadError || !wizard) {
    return (
      <div className="w-full max-w-2xl rounded-[28px] border border-red-200 bg-white p-8 shadow-sm">
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{loadError ?? 'Unable to load onboarding state.'}</AlertDescription>
        </Alert>
      </div>
    );
  }

  return (
    <div className="w-full max-w-3xl rounded-[32px] border border-white/80 bg-white/90 p-8 shadow-[0_35px_90px_rgba(15,81,50,0.1)] backdrop-blur">
      <StepIndicator currentStep={currentStep} />

      <div className="mb-8">
        <h1 className="text-2xl font-semibold text-slate-900">
          {currentStep === 1 && 'Tell us about your organization'}
          {currentStep === 2 && 'Shape the look of your workspace'}
          {currentStep === 3 && 'Invite your team'}
          {currentStep === 4 && 'Select your active suites'}
          {currentStep === 5 && 'Finish provisioning'}
        </h1>
        <p className="mt-2 text-sm text-slate-500">
          {currentStep === 1 && 'These details seed your tenant profile and guide default settings.'}
          {currentStep === 2 && 'Colors can be changed later. They are applied across shared dashboards.'}
          {currentStep === 3 && 'Send invites now or skip and handle team setup later from the dashboard.'}
          {currentStep === 4 && 'At least one suite must be active. Cyber, Data, and Visus are enabled by default.'}
          {currentStep === 5 && 'We will poll the provisioning pipeline in real time until the platform is ready.'}
        </p>
      </div>

      {currentStep === 1 && (
        <StepOrganization
          initialValues={organizationDefaults}
          onSaved={async () => {
            persistDraft({ ...draft, organization: organizationDefaults });
            await refreshWizard(2);
            syncStep(2);
          }}
          onPersist={(values) => persistDraft({ ...draft, organization: values })}
        />
      )}

      {currentStep === 2 && (
        <StepBranding
          initialValues={brandingDefaults}
          onBack={() => syncStep(1)}
          onSaved={async () => {
            await refreshWizard(3);
            syncStep(3);
          }}
          onPersist={(values) => persistDraft({ ...draft, branding: values })}
        />
      )}

      {currentStep === 3 && (
        <StepTeam
          roles={roles.length > 0 ? roles : [{ id: 'viewer', name: 'Viewer', slug: 'viewer' }]}
          initialRows={teamDefaults}
          onBack={() => syncStep(2)}
          onSaved={async () => {
            await refreshWizard(4);
            syncStep(4);
          }}
          onPersist={(rows) => persistDraft({ ...draft, team: rows })}
        />
      )}

      {currentStep === 4 && (
        <StepSuites
          initialSelected={suiteDefaults}
          onBack={() => syncStep(3)}
          onSaved={async () => {
            await refreshWizard(5);
            syncStep(5);
          }}
          onPersist={(selected) => persistDraft({ ...draft, suites: selected })}
        />
      )}

      {currentStep === 5 && (
        <StepReady
          tenantID={wizard.tenant_id || tenant?.id || ''}
          initialStatus={wizard.provisioning_status}
          onBack={() => syncStep(4)}
        />
      )}
    </div>
  );
}
