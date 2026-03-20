'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { AlertCircle, CheckCircle2, ChevronLeft, Loader2, Sparkles } from 'lucide-react';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { apiGet, apiPost } from '@/lib/api';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';
import { isApiError } from '@/types/api';

import { clearDraft, type ProvisioningStatus, type WizardProgress } from './shared';
import { ProvisioningProgress } from './provisioning-progress';

export function StepReady({
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

  return (
    <div className="space-y-6">
      {completeError ? (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{completeError}</AlertDescription>
        </Alert>
      ) : null}

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
          <a href="https://docs.clario360.com" className="text-sm font-medium text-[#0f5132] hover:underline">
            Explore the documentation
          </a>
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

          <ProvisioningProgress status={status} fallbackStatus={currentStatus} />

          {currentStatus === 'failed' ? (
            <div className="flex justify-between">
              <Button variant="outline" onClick={onBack}>
                <ChevronLeft className="mr-1 h-4 w-4" />
                Back
              </Button>
              <Button onClick={() => router.push(ROUTES.DASHBOARD)}>Go to Dashboard</Button>
            </div>
          ) : null}
        </div>
      )}
    </div>
  );
}
