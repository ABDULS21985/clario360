'use client';

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  AlertCircle,
  ArrowRight,
  Building2,
  Eye,
  EyeOff,
  MailCheck,
  ShieldCheck,
  Sparkles,
  Workflow,
} from 'lucide-react';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Spinner } from '@/components/ui/spinner';
import { apiPost } from '@/lib/api';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';
import { registerSchema, type RegisterFormData } from '@/lib/validators';
import { isApiError } from '@/types/api';

import {
  AUTH_INPUT_CLASSNAME,
  AUTH_SELECT_CLASSNAME,
  AuthActionStrip,
  AuthCallout,
  AuthFormSurface,
  AuthGuardGrid,
  AuthInsightGrid,
  AuthPageIntro,
  type AuthInsightItem,
} from './auth-page-primitives';
import { PasswordStrengthMeter } from './password-strength-meter';

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

const REGISTER_INSIGHTS: AuthInsightItem[] = [
  {
    icon: Building2,
    label: 'Workspace model',
    value: 'Tenant ready',
    detail: 'Organization metadata, admin identity, and activation defaults are provisioned together.',
  },
  {
    icon: ShieldCheck,
    label: 'Identity proof',
    value: '6-digit verify',
    detail: 'Email verification gates access before the setup wizard starts.',
  },
  {
    icon: Workflow,
    label: 'Next motion',
    value: 'Guided setup',
    detail: 'Branding, team invites, and suite activation continue immediately after signup.',
  },
];

const REGISTER_GUARDS = [
  'Admin account is verified before tenant setup begins',
  'Password policy is enforced during workspace creation',
  'Provisioning continues directly into the onboarding wizard',
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
        ttl: String(response.verification_ttl_seconds),
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
    <div className="space-y-8">
      <AuthPageIntro
        badge="New organization setup"
        badgeIcon={Sparkles}
        title="Launch your Clario 360 workspace"
        description="Create your organization, verify your admin identity, and move directly into guided setup with the same polished experience as the main platform."
        statusLabel="Provisioning path"
        statusValue="Guided and secure"
      />

      <AuthInsightGrid items={REGISTER_INSIGHTS} />

      {apiError ? (
        <Alert
          variant="destructive"
          role="alert"
          aria-live="assertive"
          className="border-red-200 bg-red-50 text-red-900 [&>svg]:text-red-600"
        >
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{apiError}</AlertDescription>
        </Alert>
      ) : null}

      <AuthFormSurface>
        <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-5">
          <div className="space-y-2">
            <Label htmlFor="organization_name" className="text-sm font-medium text-slate-700">
              Organization name
            </Label>
            <Input
              id="organization_name"
              type="text"
              placeholder="Acme Corp"
              aria-invalid={!!errors.organization_name}
              className={AUTH_INPUT_CLASSNAME}
              {...register('organization_name')}
            />
            {errors.organization_name ? (
              <p className="text-sm text-destructive" role="alert">
                {errors.organization_name.message}
              </p>
            ) : null}
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="industry" className="text-sm font-medium text-slate-700">
                Industry
              </Label>
              <select
                id="industry"
                {...register('industry')}
                className={AUTH_SELECT_CLASSNAME}
              >
                {INDUSTRIES.map((industry) => (
                  <option key={industry.value} value={industry.value}>
                    {industry.label}
                  </option>
                ))}
              </select>
              {errors.industry ? (
                <p className="text-sm text-destructive" role="alert">
                  {errors.industry.message}
                </p>
              ) : null}
            </div>

            <div className="space-y-2">
              <Label htmlFor="country" className="text-sm font-medium text-slate-700">
                Country code
              </Label>
              <Input
                id="country"
                type="text"
                maxLength={2}
                placeholder="SA"
                aria-invalid={!!errors.country}
                className={AUTH_INPUT_CLASSNAME}
                {...register('country')}
              />
              {errors.country ? (
                <p className="text-sm text-destructive" role="alert">
                  {errors.country.message}
                </p>
              ) : null}
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="first_name" className="text-sm font-medium text-slate-700">
                First name
              </Label>
              <Input
                id="first_name"
                type="text"
                autoComplete="given-name"
                aria-invalid={!!errors.first_name}
                className={AUTH_INPUT_CLASSNAME}
                {...register('first_name')}
              />
              {errors.first_name ? (
                <p className="text-sm text-destructive" role="alert">
                  {errors.first_name.message}
                </p>
              ) : null}
            </div>

            <div className="space-y-2">
              <Label htmlFor="last_name" className="text-sm font-medium text-slate-700">
                Last name
              </Label>
              <Input
                id="last_name"
                type="text"
                autoComplete="family-name"
                aria-invalid={!!errors.last_name}
                className={AUTH_INPUT_CLASSNAME}
                {...register('last_name')}
              />
              {errors.last_name ? (
                <p className="text-sm text-destructive" role="alert">
                  {errors.last_name.message}
                </p>
              ) : null}
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="email" className="text-sm font-medium text-slate-700">
              Work email
            </Label>
            <Input
              id="email"
              type="email"
              autoComplete="email"
              aria-invalid={!!errors.email}
              className={AUTH_INPUT_CLASSNAME}
              {...register('email')}
            />
            {errors.email ? (
              <p className="text-sm text-destructive" role="alert">
                {errors.email.message}
              </p>
            ) : null}
          </div>

          <div className="space-y-2">
            <Label htmlFor="password" className="text-sm font-medium text-slate-700">
              Password
            </Label>
            <div className="relative">
              <Input
                id="password"
                type={showPassword ? 'text' : 'password'}
                autoComplete="new-password"
                className={`${AUTH_INPUT_CLASSNAME} pr-12`}
                aria-invalid={!!errors.password}
                {...register('password')}
              />
              <button
                type="button"
                className="absolute right-4 top-1/2 -translate-y-1/2 text-slate-400 transition-colors hover:text-slate-700"
                onClick={() => setShowPassword((prev) => !prev)}
                aria-label={showPassword ? 'Hide password' : 'Show password'}
              >
                {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
            {errors.password ? (
              <p className="text-sm text-destructive" role="alert">
                {errors.password.message}
              </p>
            ) : null}
            <PasswordStrengthMeter password={password} />
          </div>

          <div className="space-y-2">
            <Label htmlFor="confirm_password" className="text-sm font-medium text-slate-700">
              Confirm password
            </Label>
            <div className="relative">
              <Input
                id="confirm_password"
                type={showConfirm ? 'text' : 'password'}
                autoComplete="new-password"
                className={`${AUTH_INPUT_CLASSNAME} pr-12`}
                aria-invalid={!!errors.confirm_password}
                {...register('confirm_password')}
              />
              <button
                type="button"
                className="absolute right-4 top-1/2 -translate-y-1/2 text-slate-400 transition-colors hover:text-slate-700"
                onClick={() => setShowConfirm((prev) => !prev)}
                aria-label={showConfirm ? 'Hide password' : 'Show password'}
              >
                {showConfirm ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
            {errors.confirm_password ? (
              <p className="text-sm text-destructive" role="alert">
                {errors.confirm_password.message}
              </p>
            ) : null}
          </div>

          <AuthCallout icon={MailCheck} title="Verification handoff" tone="warning">
            After signup, we send a 6-digit verification code before the admin session is activated
            and the tenant setup wizard begins.
          </AuthCallout>

          <Button
            type="submit"
            className="h-12 w-full rounded-2xl bg-[#0f5132] text-base font-semibold shadow-[0_18px_40px_rgba(15,81,50,0.22)] transition-transform hover:-translate-y-0.5 hover:bg-[#0c432b]"
            disabled={isSubmitting}
            aria-busy={isSubmitting}
          >
            {isSubmitting ? (
              <>
                <Spinner size="sm" className="mr-2" />
                Creating workspace...
              </>
            ) : (
              <>
                Continue to verification
                <ArrowRight className="ml-2 h-4 w-4" />
              </>
            )}
          </Button>
        </form>

        <AuthGuardGrid items={REGISTER_GUARDS} />
      </AuthFormSurface>

      <AuthActionStrip
        description="Already have an account? Return to the secure sign-in flow."
        href={ROUTES.LOGIN}
        cta="Sign in"
      />
    </div>
  );
}
