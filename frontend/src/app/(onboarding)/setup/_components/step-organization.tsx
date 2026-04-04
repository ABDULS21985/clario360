'use client';

import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { AlertCircle, ArrowRight, ChevronLeft, Loader2 } from 'lucide-react';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { isApiError } from '@/types/api';

import { COUNTRY_OPTIONS, INDUSTRIES, ORG_SIZES, organizationSchema, type OrganizationFormValues } from './shared';

export function StepOrganization({
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

  useEffect(() => {
    const subscription = watch((values) => {
      onPersist(values as OrganizationFormValues);
    });
    return () => subscription.unsubscribe();
  }, [watch, onPersist]);

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
      {apiError ? (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{apiError}</AlertDescription>
        </Alert>
      ) : null}

      <div className="space-y-2">
        <Label htmlFor="organization_name">Organization name</Label>
        <Input id="organization_name" {...register('organization_name')} />
        {errors.organization_name ? <p className="text-sm text-destructive">{errors.organization_name.message}</p> : null}
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
          {errors.country ? <p className="text-sm text-destructive">{errors.country.message}</p> : null}
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
