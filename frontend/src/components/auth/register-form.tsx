'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Eye, EyeOff, AlertCircle, Building2, MailCheck } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Spinner } from '@/components/ui/spinner';
import { PasswordStrengthMeter } from './password-strength-meter';
import { registerSchema, type RegisterFormData } from '@/lib/validators';
import { apiPost } from '@/lib/api';
import { isApiError } from '@/types/api';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';

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

export function RegisterForm() {
  const router = useRouter();
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [apiError, setApiError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<RegisterFormData>({
    resolver: zodResolver(registerSchema),
    mode: 'onBlur',
    defaultValues: {
      country: 'SA',
      industry: 'financial',
    },
  });

  const password = watch('password', '');

  const onSubmit = async (data: RegisterFormData) => {
    setApiError(null);
    setIsSubmitting(true);

    try {
      const response = await apiPost<{
        tenant_id: string;
        email: string;
        message: string;
        verification_ttl_seconds: number;
      }>(API_ENDPOINTS.ONBOARDING_REGISTER, {
        organization_name: data.organization_name,
        admin_email: data.email,
        admin_first_name: data.first_name,
        admin_last_name: data.last_name,
        admin_password: data.password,
        country: data.country.toUpperCase(),
        industry: data.industry,
      });

      const params = new URLSearchParams({
        email: data.email,
        tenantId: response.tenant_id,
      });
      router.push(`${ROUTES.VERIFY_EMAIL}?${params.toString()}`);
    } catch (err) {
      setApiError(
        isApiError(err) ? err.message : 'Registration failed. Please try again.',
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="space-y-2">
        <div className="inline-flex items-center gap-2 rounded-full border border-[#0f5132]/15 bg-[#0f5132]/5 px-3 py-1 text-xs font-medium text-[#0f5132]">
          <Building2 className="h-3.5 w-3.5" />
          New organization setup
        </div>
        <h1 className="text-2xl font-bold tracking-tight">Launch your Clario 360 workspace</h1>
        <p className="text-sm text-muted-foreground">
          Create your organization, verify your email, and continue into setup.
        </p>
      </div>

      {apiError && (
        <Alert variant="destructive" role="alert" aria-live="assertive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{apiError}</AlertDescription>
        </Alert>
      )}

      <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-5">
        <div className="space-y-2">
          <Label htmlFor="organization_name">Organization name</Label>
          <Input
            id="organization_name"
            type="text"
            placeholder="Acme Corp"
            aria-invalid={!!errors.organization_name}
            {...register('organization_name')}
          />
          {errors.organization_name && (
            <p className="text-sm text-destructive" role="alert">
              {errors.organization_name.message}
            </p>
          )}
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="industry">Industry</Label>
            <select
              id="industry"
              {...register('industry')}
              className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            >
              {INDUSTRIES.map((industry) => (
                <option key={industry.value} value={industry.value}>
                  {industry.label}
                </option>
              ))}
            </select>
            {errors.industry && (
              <p className="text-sm text-destructive" role="alert">
                {errors.industry.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="country">Country code</Label>
            <Input
              id="country"
              type="text"
              maxLength={2}
              placeholder="SA"
              aria-invalid={!!errors.country}
              {...register('country')}
            />
            {errors.country && (
              <p className="text-sm text-destructive" role="alert">
                {errors.country.message}
              </p>
            )}
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="first_name">First name</Label>
            <Input
              id="first_name"
              type="text"
              autoComplete="given-name"
              aria-invalid={!!errors.first_name}
              {...register('first_name')}
            />
            {errors.first_name && (
              <p className="text-sm text-destructive" role="alert">
                {errors.first_name.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="last_name">Last name</Label>
            <Input
              id="last_name"
              type="text"
              autoComplete="family-name"
              aria-invalid={!!errors.last_name}
              {...register('last_name')}
            />
            {errors.last_name && (
              <p className="text-sm text-destructive" role="alert">
                {errors.last_name.message}
              </p>
            )}
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="email">Work email</Label>
          <Input
            id="email"
            type="email"
            autoComplete="email"
            aria-invalid={!!errors.email}
            {...register('email')}
          />
          {errors.email && (
            <p className="text-sm text-destructive" role="alert">
              {errors.email.message}
            </p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="password">Password</Label>
          <div className="relative">
            <Input
              id="password"
              type={showPassword ? 'text' : 'password'}
              autoComplete="new-password"
              className="pr-10"
              aria-invalid={!!errors.password}
              {...register('password')}
            />
            <button
              type="button"
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              onClick={() => setShowPassword((prev) => !prev)}
              aria-label={showPassword ? 'Hide password' : 'Show password'}
            >
              {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
          </div>
          {errors.password && (
            <p className="text-sm text-destructive" role="alert">
              {errors.password.message}
            </p>
          )}
          <PasswordStrengthMeter password={password} />
        </div>

        <div className="space-y-2">
          <Label htmlFor="confirm_password">Confirm password</Label>
          <div className="relative">
            <Input
              id="confirm_password"
              type={showConfirm ? 'text' : 'password'}
              autoComplete="new-password"
              className="pr-10"
              aria-invalid={!!errors.confirm_password}
              {...register('confirm_password')}
            />
            <button
              type="button"
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              onClick={() => setShowConfirm((prev) => !prev)}
              aria-label={showConfirm ? 'Hide password' : 'Show password'}
            >
              {showConfirm ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
          </div>
          {errors.confirm_password && (
            <p className="text-sm text-destructive" role="alert">
              {errors.confirm_password.message}
            </p>
          )}
        </div>

        <div className="rounded-lg border border-[#c5a04e]/25 bg-[#c5a04e]/10 p-3 text-sm text-muted-foreground">
          <div className="flex items-start gap-2">
            <MailCheck className="mt-0.5 h-4 w-4 text-[#8f6a12]" />
            <p>
              After signup, we&apos;ll email a 6-digit verification code before you enter the setup wizard.
            </p>
          </div>
        </div>

        <Button
          type="submit"
          className="w-full"
          disabled={isSubmitting}
          aria-busy={isSubmitting}
        >
          {isSubmitting ? (
            <>
              <Spinner size="sm" className="mr-2" />
              Creating workspace…
            </>
          ) : (
            'Continue to verification'
          )}
        </Button>
      </form>

      <p className="text-center text-sm text-muted-foreground">
        Already have an account?{' '}
        <Link href={ROUTES.LOGIN} className="font-medium text-primary hover:underline">
          Sign in
        </Link>
      </p>
    </div>
  );
}
