'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  Building2,
  Palette,
  Users,
  LayoutGrid,
  CheckCircle2,
  ChevronRight,
  ChevronLeft,
  Loader2,
  Plus,
  Trash2,
  AlertCircle,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Progress } from '@/components/ui/progress';
import { Checkbox } from '@/components/ui/checkbox';
import { Spinner } from '@/components/ui/spinner';
import { apiPost, apiGet } from '@/lib/api';
import { isApiError } from '@/types/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { cn } from '@/lib/utils';

// ──────────────────────────────────────────────────────────────
// Schemas
// ──────────────────────────────────────────────────────────────

const orgSchema = z.object({
  organization_name: z.string().min(2).max(100),
  industry: z.string().min(1, 'Please select an industry'),
  country: z.string().length(2, 'Enter a 2-letter country code'),
  city: z.string().max(120).optional(),
  organization_size: z.string().min(1, 'Please select an organization size'),
});
type OrgFormData = z.infer<typeof orgSchema>;

const brandingSchema = z.object({
  primary_color: z
    .string()
    .regex(/^#[0-9A-Fa-f]{6}$/, 'Must be a valid hex color like #1B5E20')
    .optional()
    .or(z.literal('')),
  accent_color: z
    .string()
    .regex(/^#[0-9A-Fa-f]{6}$/, 'Must be a valid hex color like #C6A962')
    .optional()
    .or(z.literal('')),
});
type BrandingFormData = z.infer<typeof brandingSchema>;

// ──────────────────────────────────────────────────────────────
// Constants
// ──────────────────────────────────────────────────────────────

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
];

const ORG_SIZES = [
  { value: '1-50', label: '1–50 employees' },
  { value: '51-200', label: '51–200 employees' },
  { value: '201-1000', label: '201–1 000 employees' },
  { value: '1000+', label: '1 000+ employees' },
];

const SUITES = [
  {
    id: 'cyber',
    name: 'Cyber',
    description: 'Threat detection, alerts, asset management, and SIEM.',
    color: 'border-blue-500',
  },
  {
    id: 'data',
    name: 'Data',
    description: 'Data pipelines, classification, and governance.',
    color: 'border-purple-500',
  },
  {
    id: 'acta',
    name: 'Acta',
    description: 'Meeting management, governance, and voting.',
    color: 'border-green-500',
  },
  {
    id: 'lex',
    name: 'Lex',
    description: 'Compliance rules, policies, and regulatory reporting.',
    color: 'border-amber-500',
  },
  {
    id: 'visus',
    name: 'Visus',
    description: 'KPI dashboards, widgets, and analytics.',
    color: 'border-rose-500',
  },
];

const ROLE_SLUGS = [
  { value: 'tenant-admin', label: 'Admin' },
  { value: 'analyst', label: 'Analyst' },
  { value: 'viewer', label: 'Viewer' },
];

const STEPS = [
  { number: 1, label: 'Organization', icon: Building2 },
  { number: 2, label: 'Branding', icon: Palette },
  { number: 3, label: 'Team', icon: Users },
  { number: 4, label: 'Suites', icon: LayoutGrid },
  { number: 5, label: 'Ready', icon: CheckCircle2 },
] as const;

type ProvisioningStatus = 'pending' | 'provisioning' | 'completed' | 'failed';

interface ProvisioningStatusResponse {
  status: ProvisioningStatus;
  progress_pct: number;
  current_step?: { step_name: string };
  error?: string;
}

// ──────────────────────────────────────────────────────────────
// Step 1: Organization
// ──────────────────────────────────────────────────────────────

function StepOrganization({
  onNext,
}: {
  onNext: () => void;
}) {
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<OrgFormData>({ resolver: zodResolver(orgSchema) });
  const [apiError, setApiError] = useState<string | null>(null);

  const onSubmit = async (data: OrgFormData) => {
    setApiError(null);
    try {
      await apiPost(API_ENDPOINTS.ONBOARDING_ORGANIZATION, data);
      onNext();
    } catch (err) {
      setApiError(isApiError(err) ? err.message : 'Failed to save organization details.');
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
      {apiError && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{apiError}</AlertDescription>
        </Alert>
      )}

      <div className="space-y-2">
        <Label htmlFor="organization_name">Organization name *</Label>
        <Input
          id="organization_name"
          placeholder="Acme Corp"
          {...register('organization_name')}
          aria-invalid={!!errors.organization_name}
        />
        {errors.organization_name && (
          <p className="text-sm text-destructive">{errors.organization_name.message}</p>
        )}
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="industry">Industry *</Label>
          <select
            id="industry"
            {...register('industry')}
            className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          >
            <option value="">Select industry…</option>
            {INDUSTRIES.map((i) => (
              <option key={i.value} value={i.value}>{i.label}</option>
            ))}
          </select>
          {errors.industry && (
            <p className="text-sm text-destructive">{errors.industry.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="organization_size">Organization size *</Label>
          <select
            id="organization_size"
            {...register('organization_size')}
            className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          >
            <option value="">Select size…</option>
            {ORG_SIZES.map((s) => (
              <option key={s.value} value={s.value}>{s.label}</option>
            ))}
          </select>
          {errors.organization_size && (
            <p className="text-sm text-destructive">{errors.organization_size.message}</p>
          )}
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="country">Country code *</Label>
          <Input
            id="country"
            placeholder="US"
            maxLength={2}
            {...register('country')}
            aria-invalid={!!errors.country}
          />
          {errors.country && (
            <p className="text-sm text-destructive">{errors.country.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="city">City</Label>
          <Input id="city" placeholder="New York" {...register('city')} />
        </div>
      </div>

      <div className="flex justify-end pt-2">
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
          Continue
          <ChevronRight className="ml-1 h-4 w-4" />
        </Button>
      </div>
    </form>
  );
}

// ──────────────────────────────────────────────────────────────
// Step 2: Branding
// ──────────────────────────────────────────────────────────────

function StepBranding({
  onNext,
  onBack,
}: {
  onNext: () => void;
  onBack: () => void;
}) {
  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<BrandingFormData>({ resolver: zodResolver(brandingSchema) });
  const [apiError, setApiError] = useState<string | null>(null);

  const primaryColor = watch('primary_color') || '#1B5E20';
  const accentColor = watch('accent_color') || '#C6A962';

  const onSubmit = async (data: BrandingFormData) => {
    setApiError(null);
    try {
      const payload: Record<string, string> = {};
      if (data.primary_color) payload.primary_color = data.primary_color;
      if (data.accent_color) payload.accent_color = data.accent_color;
      await apiPost(API_ENDPOINTS.ONBOARDING_BRANDING, payload);
      onNext();
    } catch (err) {
      setApiError(isApiError(err) ? err.message : 'Failed to save branding.');
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
      {apiError && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{apiError}</AlertDescription>
        </Alert>
      )}

      <p className="text-sm text-muted-foreground">
        Customize your platform colors. You can always change these later.
      </p>

      <div className="grid grid-cols-2 gap-6">
        <div className="space-y-3">
          <Label htmlFor="primary_color">Primary color</Label>
          <div className="flex items-center gap-3">
            <div
              className="h-10 w-10 rounded-md border shadow-sm"
              style={{ backgroundColor: primaryColor }}
            />
            <Input
              id="primary_color"
              placeholder="#1B5E20"
              maxLength={7}
              {...register('primary_color')}
              aria-invalid={!!errors.primary_color}
            />
          </div>
          {errors.primary_color && (
            <p className="text-sm text-destructive">{errors.primary_color.message}</p>
          )}
        </div>

        <div className="space-y-3">
          <Label htmlFor="accent_color">Accent color</Label>
          <div className="flex items-center gap-3">
            <div
              className="h-10 w-10 rounded-md border shadow-sm"
              style={{ backgroundColor: accentColor }}
            />
            <Input
              id="accent_color"
              placeholder="#C6A962"
              maxLength={7}
              {...register('accent_color')}
              aria-invalid={!!errors.accent_color}
            />
          </div>
          {errors.accent_color && (
            <p className="text-sm text-destructive">{errors.accent_color.message}</p>
          )}
        </div>
      </div>

      <div className="flex justify-between pt-2">
        <Button type="button" variant="outline" onClick={onBack}>
          <ChevronLeft className="mr-1 h-4 w-4" />
          Back
        </Button>
        <div className="flex gap-2">
          <Button type="button" variant="ghost" onClick={onNext}>
            Skip
          </Button>
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            Continue
            <ChevronRight className="ml-1 h-4 w-4" />
          </Button>
        </div>
      </div>
    </form>
  );
}

// ──────────────────────────────────────────────────────────────
// Step 3: Team
// ──────────────────────────────────────────────────────────────

interface TeamMember {
  email: string;
  role_slug: string;
  message: string;
}

function StepTeam({
  onNext,
  onBack,
}: {
  onNext: () => void;
  onBack: () => void;
}) {
  const [members, setMembers] = useState<TeamMember[]>([{ email: '', role_slug: 'analyst', message: '' }]);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [apiError, setApiError] = useState<string | null>(null);

  const addMember = () => {
    if (members.length < 10) {
      setMembers((prev) => [...prev, { email: '', role_slug: 'analyst', message: '' }]);
    }
  };

  const removeMember = (idx: number) => {
    setMembers((prev) => prev.filter((_, i) => i !== idx));
  };

  const updateMember = (idx: number, field: keyof TeamMember, value: string) => {
    setMembers((prev) => prev.map((m, i) => (i === idx ? { ...m, [field]: value } : m)));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setApiError(null);
    setIsSubmitting(true);
    try {
      const invitations = members
        .filter((m) => m.email.trim() !== '')
        .map((m) => ({
          email: m.email.trim(),
          role_slug: m.role_slug,
          message: m.message.trim(),
        }));
      await apiPost(API_ENDPOINTS.ONBOARDING_TEAM, { invitations });
      onNext();
    } catch (err) {
      setApiError(isApiError(err) ? err.message : 'Failed to send invitations.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      {apiError && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{apiError}</AlertDescription>
        </Alert>
      )}

      <p className="text-sm text-muted-foreground">
        Invite your team members. You can skip this step and invite people later.
      </p>

      <div className="space-y-3">
        {members.map((member, idx) => (
          <div key={idx} className="flex items-start gap-2">
            <div className="flex-1 space-y-1">
              <Input
                type="email"
                placeholder="colleague@company.com"
                value={member.email}
                onChange={(e) => updateMember(idx, 'email', e.target.value)}
                aria-label={`Email for member ${idx + 1}`}
              />
            </div>
            <select
              value={member.role_slug}
              onChange={(e) => updateMember(idx, 'role_slug', e.target.value)}
              aria-label={`Role for member ${idx + 1}`}
              className="h-10 rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              {ROLE_SLUGS.map((r) => (
                <option key={r.value} value={r.value}>{r.label}</option>
              ))}
            </select>
            {members.length > 1 && (
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={() => removeMember(idx)}
                aria-label="Remove"
              >
                <Trash2 className="h-4 w-4 text-muted-foreground" />
              </Button>
            )}
          </div>
        ))}
      </div>

      {members.length < 10 && (
        <Button type="button" variant="outline" size="sm" onClick={addMember}>
          <Plus className="mr-1 h-4 w-4" />
          Add another
        </Button>
      )}

      <div className="flex justify-between pt-2">
        <Button type="button" variant="outline" onClick={onBack}>
          <ChevronLeft className="mr-1 h-4 w-4" />
          Back
        </Button>
        <div className="flex gap-2">
          <Button type="button" variant="ghost" onClick={onNext}>
            Skip
          </Button>
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            Send invitations
            <ChevronRight className="ml-1 h-4 w-4" />
          </Button>
        </div>
      </div>
    </form>
  );
}

// ──────────────────────────────────────────────────────────────
// Step 4: Suites
// ──────────────────────────────────────────────────────────────

function StepSuites({
  onNext,
  onBack,
}: {
  onNext: () => void;
  onBack: () => void;
}) {
  const [selected, setSelected] = useState<string[]>(['cyber']);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [apiError, setApiError] = useState<string | null>(null);

  const toggle = (id: string) => {
    setSelected((prev) =>
      prev.includes(id) ? prev.filter((s) => s !== id) : [...prev, id]
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (selected.length === 0) {
      setApiError('Please select at least one suite.');
      return;
    }
    setApiError(null);
    setIsSubmitting(true);
    try {
      await apiPost(API_ENDPOINTS.ONBOARDING_SUITES, { active_suites: selected });
      onNext();
    } catch (err) {
      setApiError(isApiError(err) ? err.message : 'Failed to save suites.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      {apiError && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{apiError}</AlertDescription>
        </Alert>
      )}

      <p className="text-sm text-muted-foreground">
        Choose which product suites to activate. You can enable or disable suites later.
      </p>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        {SUITES.map((suite) => {
          const isSelected = selected.includes(suite.id);
          return (
            <button
              key={suite.id}
              type="button"
              onClick={() => toggle(suite.id)}
              className={cn(
                'flex cursor-pointer items-start gap-3 rounded-lg border-2 p-4 text-left transition-colors',
                isSelected
                  ? `${suite.color} bg-green-50 dark:bg-green-950/30`
                  : 'border-gray-200 hover:border-gray-300 dark:border-gray-700 dark:hover:border-gray-600'
              )}
              aria-pressed={isSelected}
            >
              <Checkbox
                checked={isSelected}
                onCheckedChange={() => toggle(suite.id)}
                aria-hidden="true"
                tabIndex={-1}
              />
              <div>
                <p className="font-medium">{suite.name}</p>
                <p className="mt-0.5 text-xs text-muted-foreground">{suite.description}</p>
              </div>
            </button>
          );
        })}
      </div>

      <p className="text-xs text-muted-foreground">
        {selected.length} suite{selected.length !== 1 ? 's' : ''} selected
      </p>

      <div className="flex justify-between pt-2">
        <Button type="button" variant="outline" onClick={onBack}>
          <ChevronLeft className="mr-1 h-4 w-4" />
          Back
        </Button>
        <Button type="submit" disabled={isSubmitting || selected.length === 0}>
          {isSubmitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
          Continue
          <ChevronRight className="ml-1 h-4 w-4" />
        </Button>
      </div>
    </form>
  );
}

// ──────────────────────────────────────────────────────────────
// Step 5: Ready (Provisioning progress)
// ──────────────────────────────────────────────────────────────

function StepReady({ onBack }: { onBack: () => void }) {
  const router = useRouter();
  const [status, setStatus] = useState<ProvisioningStatus>('pending');
  const [progressPct, setProgressPct] = useState(0);
  const [currentStep, setCurrentStep] = useState<string>('Initializing…');
  const [error, setError] = useState<string | null>(null);
  const [isCompleting, setIsCompleting] = useState(false);

  const pollStatus = useCallback(async () => {
    try {
      const data = await apiGet<ProvisioningStatusResponse>(API_ENDPOINTS.ONBOARDING_PROGRESS);
      setStatus(data.provisioning_status as ProvisioningStatus ?? 'pending');
      // ProvisioningStatus is nested in the progress; adapt based on actual API shape
    } catch {
      // ignore poll errors
    }
  }, []);

  useEffect(() => {
    // Complete wizard first
    const complete = async () => {
      setIsCompleting(true);
      try {
        await apiPost(API_ENDPOINTS.ONBOARDING_COMPLETE, {});
      } catch {
        // may already be completed
      }
      setIsCompleting(false);
    };
    complete();
  }, []);

  useEffect(() => {
    if (status === 'completed') return;
    if (status === 'failed') return;

    const interval = setInterval(async () => {
      try {
        const data = await apiGet<{
          provisioning_status: ProvisioningStatus;
          progress_pct: number;
          current_step?: { step_name: string };
          provisioning_error?: string;
        }>(API_ENDPOINTS.ONBOARDING_PROGRESS);
        const pStatus = data.provisioning_status;
        setStatus(pStatus);
        setProgressPct(data.progress_pct ?? 0);
        if (data.current_step?.step_name) {
          setCurrentStep(data.current_step.step_name);
        }
        if (data.provisioning_error) {
          setError(data.provisioning_error);
        }
        if (pStatus === 'completed' || pStatus === 'failed') {
          clearInterval(interval);
        }
      } catch {
        // ignore
      }
    }, 2500);

    return () => clearInterval(interval);
  }, [status]);

  const handleGoToDashboard = () => {
    router.push('/dashboard');
  };

  return (
    <div className="space-y-6 text-center">
      {isCompleting && (
        <div className="flex items-center justify-center gap-2 text-muted-foreground">
          <Spinner size="sm" />
          <span className="text-sm">Finalizing…</span>
        </div>
      )}

      {!isCompleting && (
        <>
          {status === 'completed' ? (
            <div className="space-y-4">
              <div className="flex justify-center">
                <CheckCircle2 className="h-16 w-16 text-green-500" />
              </div>
              <div>
                <h2 className="text-xl font-semibold">Your workspace is ready!</h2>
                <p className="mt-1 text-sm text-muted-foreground">
                  All services have been provisioned. Welcome to Clario 360.
                </p>
              </div>
              <Button onClick={handleGoToDashboard} size="lg" className="w-full">
                Go to Dashboard
                <ChevronRight className="ml-1 h-4 w-4" />
              </Button>
            </div>
          ) : status === 'failed' ? (
            <div className="space-y-4">
              {error && (
                <Alert variant="destructive">
                  <AlertCircle className="h-4 w-4" />
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}
              <p className="text-sm text-muted-foreground">
                Provisioning encountered an issue. Our team has been notified. You may still proceed to the dashboard.
              </p>
              <div className="flex gap-2">
                <Button variant="outline" onClick={onBack} className="flex-1">
                  <ChevronLeft className="mr-1 h-4 w-4" />
                  Back
                </Button>
                <Button onClick={handleGoToDashboard} className="flex-1">
                  Go to Dashboard
                </Button>
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              <div className="flex justify-center">
                <Loader2 className="h-12 w-12 animate-spin text-[#1B5E20]" />
              </div>
              <div>
                <h2 className="text-xl font-semibold">Setting up your workspace…</h2>
                <p className="mt-1 text-sm text-muted-foreground">
                  This usually takes less than a minute.
                </p>
              </div>
              <div className="space-y-2">
                <Progress value={progressPct} className="h-2" />
                <p className="text-xs text-muted-foreground">{currentStep}</p>
              </div>
              <p className="text-xs text-muted-foreground">
                You can safely close this tab. We will send you an email when your workspace is ready.
              </p>
            </div>
          )}
        </>
      )}
    </div>
  );
}

// ──────────────────────────────────────────────────────────────
// Main page
// ──────────────────────────────────────────────────────────────

export default function SetupPage() {
  const [currentStep, setCurrentStep] = useState(1);

  const stepTitle: Record<number, string> = {
    1: 'Tell us about your organization',
    2: 'Customize your brand',
    3: 'Invite your team',
    4: 'Choose your suites',
    5: 'Almost there!',
  };

  const stepDescription: Record<number, string> = {
    1: 'This helps us tailor the platform to your needs.',
    2: "Add your brand colors. You can always update these later.",
    3: 'Invite colleagues now or skip and do it later.',
    4: 'Select the product suites you want to activate.',
    5: 'We are provisioning your workspace.',
  };

  const progressValue = ((currentStep - 1) / (STEPS.length - 1)) * 100;

  return (
    <div className="w-full max-w-2xl">
      {/* Step progress bar */}
      <div className="mb-8">
        <div className="mb-4 flex items-center justify-between">
          {STEPS.map((step) => {
            const Icon = step.icon;
            const isActive = step.number === currentStep;
            const isCompleted = step.number < currentStep;
            return (
              <div key={step.number} className="flex flex-col items-center gap-1">
                <div
                  className={cn(
                    'flex h-9 w-9 items-center justify-center rounded-full border-2 transition-colors',
                    isCompleted
                      ? 'border-[#1B5E20] bg-[#1B5E20] text-white'
                      : isActive
                      ? 'border-[#1B5E20] bg-white text-[#1B5E20] dark:bg-gray-900'
                      : 'border-gray-300 bg-white text-gray-400 dark:border-gray-600 dark:bg-gray-800'
                  )}
                >
                  {isCompleted ? (
                    <CheckCircle2 className="h-4 w-4" />
                  ) : (
                    <Icon className="h-4 w-4" />
                  )}
                </div>
                <span
                  className={cn(
                    'hidden text-[10px] font-medium sm:block',
                    isActive ? 'text-[#1B5E20]' : 'text-gray-400'
                  )}
                >
                  {step.label}
                </span>
              </div>
            );
          })}
        </div>
        <Progress value={progressValue} className="h-1.5" />
      </div>

      {/* Card */}
      <div className="rounded-xl border border-gray-200 bg-white p-8 shadow-sm dark:border-gray-800 dark:bg-gray-900">
        <div className="mb-6">
          <h1 className="text-xl font-semibold">{stepTitle[currentStep]}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{stepDescription[currentStep]}</p>
        </div>

        {currentStep === 1 && (
          <StepOrganization onNext={() => setCurrentStep(2)} />
        )}
        {currentStep === 2 && (
          <StepBranding onNext={() => setCurrentStep(3)} onBack={() => setCurrentStep(1)} />
        )}
        {currentStep === 3 && (
          <StepTeam onNext={() => setCurrentStep(4)} onBack={() => setCurrentStep(2)} />
        )}
        {currentStep === 4 && (
          <StepSuites onNext={() => setCurrentStep(5)} onBack={() => setCurrentStep(3)} />
        )}
        {currentStep === 5 && (
          <StepReady onBack={() => setCurrentStep(4)} />
        )}
      </div>

      {/* Step counter */}
      <p className="mt-4 text-center text-xs text-gray-400">
        Step {currentStep} of {STEPS.length}
      </p>
    </div>
  );
}
