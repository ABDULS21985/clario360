'use client';

import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { AlertCircle, ChevronLeft, Loader2, ShieldCheck } from 'lucide-react';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { isApiError } from '@/types/api';

import { ColorPicker } from './color-picker';
import { brandingSchema, type BrandingFormValues } from './shared';

export function StepBranding({
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
    handleSubmit,
    setValue,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<BrandingFormValues>({
    resolver: zodResolver(brandingSchema),
    defaultValues: initialValues,
  });
  const [apiError, setApiError] = useState<string | null>(null);
  const values = watch();

  useEffect(() => {
    const subscription = watch((nextValues) => {
      onPersist(nextValues as BrandingFormValues);
    });
    return () => subscription.unsubscribe();
  }, [watch, onPersist]);

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
      {apiError ? (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{apiError}</AlertDescription>
        </Alert>
      ) : null}

      <div className="rounded-2xl border border-dashed border-[#0f5132]/20 bg-[#0f5132]/5 p-4 text-sm text-slate-600">
        Logo upload is not available in this deployment yet. Brand colors will be applied immediately.
      </div>

      <div className="grid gap-5 md:grid-cols-2">
        <ColorPicker
          id="primary_color"
          label="Primary color"
          value={values.primary_color}
          onChange={(value) => setValue('primary_color', value, { shouldValidate: true, shouldDirty: true })}
          error={errors.primary_color?.message}
        />
        <ColorPicker
          id="accent_color"
          label="Accent color"
          value={values.accent_color}
          onChange={(value) => setValue('accent_color', value, { shouldValidate: true, shouldDirty: true })}
          error={errors.accent_color?.message}
        />
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
