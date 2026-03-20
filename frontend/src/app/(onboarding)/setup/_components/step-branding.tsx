'use client';

import { useEffect, useRef, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { AlertCircle, CheckCircle2, ChevronLeft, ImagePlus, Loader2, ShieldCheck, Upload, X } from 'lucide-react';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { isApiError } from '@/types/api';

import { ColorPicker } from './color-picker';
import { brandingSchema, type BrandingFormValues } from './shared';

export function StepBranding({
  initialValues,
  savedLogoFileId,
  onBack,
  onSaved,
  onPersist,
}: {
  initialValues: BrandingFormValues;
  savedLogoFileId?: string | null;
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
  const [logoError, setLogoError] = useState<string | null>(null);
  const [logoFile, setLogoFile] = useState<File | null>(null);
  const [logoPreviewUrl, setLogoPreviewUrl] = useState<string | null>(null);
  const [isDraggingLogo, setIsDraggingLogo] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const values = watch();

  useEffect(() => {
    const subscription = watch((nextValues) => {
      onPersist(nextValues as BrandingFormValues);
    });
    return () => subscription.unsubscribe();
  }, [watch, onPersist]);

  useEffect(() => {
    if (!logoFile) {
      setLogoPreviewUrl(null);
      return;
    }
    const objectUrl = URL.createObjectURL(logoFile);
    setLogoPreviewUrl(objectUrl);
    return () => URL.revokeObjectURL(objectUrl);
  }, [logoFile]);

  const handleLogoSelection = (fileList: FileList | File[]) => {
    const file = Array.from(fileList)[0];
    if (!file) {
      return;
    }

    const isSupportedType =
      file.type === 'image/png' ||
      file.type === 'image/svg+xml' ||
      file.name.toLowerCase().endsWith('.png') ||
      file.name.toLowerCase().endsWith('.svg');
    if (!isSupportedType) {
      setLogoError('Logo must be a PNG or SVG image.');
      return;
    }
    if (file.size > 2 * 1024 * 1024) {
      setLogoError('Logo must be 2MB or smaller.');
      return;
    }

    setLogoError(null);
    setLogoFile(file);
  };

  const submit = handleSubmit(async (branding) => {
    setApiError(null);
    try {
      const payload = new FormData();
      payload.append('primary_color', branding.primary_color);
      payload.append('accent_color', branding.accent_color);
      if (logoFile) {
        payload.append('logo', logoFile);
      }

      await apiPost(API_ENDPOINTS.ONBOARDING_BRANDING, payload);
      setLogoFile(null);
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

      <div className="grid gap-5 lg:grid-cols-[1.2fr_0.8fr]">
        <div className="space-y-3">
          <div
            className={`rounded-3xl border-2 border-dashed p-6 transition ${
              isDraggingLogo
                ? 'border-[#0f5132] bg-[#0f5132]/5'
                : 'border-[#0f5132]/20 bg-[#f5fbf7]'
            }`}
            onDragOver={(event) => {
              event.preventDefault();
              setIsDraggingLogo(true);
            }}
            onDragLeave={() => setIsDraggingLogo(false)}
            onDrop={(event) => {
              event.preventDefault();
              setIsDraggingLogo(false);
              handleLogoSelection(event.dataTransfer.files);
            }}
          >
            <input
              ref={inputRef}
              type="file"
              accept="image/png,image/svg+xml,.png,.svg"
              className="hidden"
              onChange={(event) => {
                if (event.target.files) {
                  handleLogoSelection(event.target.files);
                }
                event.target.value = '';
              }}
            />

            <div className="flex items-start gap-4">
              <div className="rounded-2xl bg-[#0f5132] p-3 text-white">
                <ImagePlus className="h-5 w-5" />
              </div>
              <div className="flex-1">
                <p className="text-sm font-semibold text-slate-900">Upload your organization logo</p>
                <p className="mt-1 text-sm text-slate-500">
                  PNG or SVG only, up to 2MB. We store it with your tenant branding so dashboards and welcome
                  surfaces can reuse it.
                </p>
                <div className="mt-4 flex flex-wrap gap-3">
                  <Button type="button" variant="outline" onClick={() => inputRef.current?.click()}>
                    <Upload className="mr-2 h-4 w-4" />
                    Choose Logo
                  </Button>
                  <div className="rounded-full bg-white px-3 py-2 text-xs font-medium text-slate-500 shadow-sm">
                    Drag and drop supported
                  </div>
                </div>
              </div>
            </div>
          </div>

          {logoError ? (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{logoError}</AlertDescription>
            </Alert>
          ) : null}
        </div>

        <div className="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
          <p className="text-sm font-semibold text-slate-900">Logo preview</p>
          <div className="mt-4 flex min-h-40 items-center justify-center rounded-2xl border border-slate-200 bg-slate-50 p-4">
            {logoPreviewUrl ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={logoPreviewUrl} alt="Selected logo preview" className="max-h-24 max-w-full object-contain" />
            ) : savedLogoFileId ? (
              <div className="flex items-center gap-2 text-sm text-slate-600">
                <CheckCircle2 className="h-4 w-4 text-[#0f5132]" />
                Current logo is already stored for this tenant.
              </div>
            ) : (
              <div className="text-center text-sm text-slate-500">
                <p>No logo selected yet.</p>
                <p className="mt-1">Your colors will still be applied if you skip this step.</p>
              </div>
            )}
          </div>
          {logoFile ? (
            <div className="mt-3 flex items-center justify-between rounded-2xl border border-slate-200 px-3 py-2 text-sm text-slate-600">
              <span className="truncate pr-3">{logoFile.name}</span>
              <Button type="button" variant="ghost" size="sm" onClick={() => setLogoFile(null)}>
                <X className="h-4 w-4" />
              </Button>
            </div>
          ) : null}
        </div>
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
