'use client';

import { useEffect, useMemo, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { AlertCircle } from 'lucide-react';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Spinner } from '@/components/ui/spinner';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';
import { useAuthStore } from '@/stores/auth-store';
import { isApiError } from '@/types/api';

import { StepBranding } from './step-branding';
import { StepIndicator } from './step-indicator';
import { StepOrganization } from './step-organization';
import { StepReady } from './step-ready';
import { StepSuites } from './step-suites';
import { StepTeam } from './step-team';
import {
  clearDraft,
  loadDraft,
  saveDraft,
  type BrandingFormValues,
  type OrganizationFormValues,
  type RoleRecord,
  type WizardDraft,
  type WizardProgress,
} from './shared';

export function WizardContainer() {
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

  const organizationDefaults = useMemo<OrganizationFormValues>(
    () => ({
      organization_name: draft.organization?.organization_name ?? wizard?.organization_name ?? '',
      industry: draft.organization?.industry ?? wizard?.industry ?? 'financial',
      country: draft.organization?.country ?? wizard?.country ?? 'SA',
      city: draft.organization?.city ?? wizard?.city ?? '',
      organization_size: draft.organization?.organization_size ?? wizard?.organization_size ?? '1-50',
    }),
    [draft.organization, wizard],
  );

  const brandingDefaults = useMemo<BrandingFormValues>(
    () => ({
      primary_color: draft.branding?.primary_color ?? wizard?.primary_color ?? '#006B3F',
      accent_color: draft.branding?.accent_color ?? wizard?.accent_color ?? '#C5A04E',
    }),
    [draft.branding, wizard],
  );

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

      {currentStep === 1 ? (
        <StepOrganization
          initialValues={organizationDefaults}
          onSaved={async () => {
            await refreshWizard(2);
            syncStep(2);
          }}
          onPersist={(values) => persistDraft({ ...draft, organization: values })}
        />
      ) : null}

      {currentStep === 2 ? (
        <StepBranding
          initialValues={brandingDefaults}
          onBack={() => syncStep(1)}
          onSaved={async () => {
            await refreshWizard(3);
            syncStep(3);
          }}
          onPersist={(values) => persistDraft({ ...draft, branding: values })}
        />
      ) : null}

      {currentStep === 3 ? (
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
      ) : null}

      {currentStep === 4 ? (
        <StepSuites
          initialSelected={suiteDefaults}
          onBack={() => syncStep(3)}
          onSaved={async () => {
            await refreshWizard(5);
            syncStep(5);
          }}
          onPersist={(selected) => persistDraft({ ...draft, suites: selected })}
        />
      ) : null}

      {currentStep === 5 ? (
        <StepReady tenantID={wizard.tenant_id || tenant?.id || ''} initialStatus={wizard.provisioning_status} onBack={() => syncStep(4)} />
      ) : null}
    </div>
  );
}
