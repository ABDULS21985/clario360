'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  AlertCircle,
  ArrowRight,
  CheckCircle,
  Eye,
  EyeOff,
  KeyRound,
  Lock,
  ShieldCheck,
  TimerReset,
  Workflow,
} from 'lucide-react';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Spinner } from '@/components/ui/spinner';
import { apiPost } from '@/lib/api';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';
import { resetPasswordSchema, type ResetPasswordFormData } from '@/lib/validators';
import { isApiError } from '@/types/api';

import {
  AUTH_INPUT_CLASSNAME,
  AuthActionStrip,
  AuthCallout,
  AuthCenteredState,
  AuthFormSurface,
  AuthGuardGrid,
  AuthInsightGrid,
  AuthPageIntro,
  type AuthInsightItem,
} from './auth-page-primitives';
import { PasswordStrengthMeter } from './password-strength-meter';

const RESET_INSIGHTS: AuthInsightItem[] = [
  {
    icon: KeyRound,
    label: 'Credential rotation',
    value: 'Immediate',
    detail: 'The new password becomes active as soon as the reset token is accepted.',
  },
  {
    icon: ShieldCheck,
    label: 'Token policy',
    value: 'Time bound',
    detail: 'Reset links expire automatically and are rejected if stale or malformed.',
  },
  {
    icon: Workflow,
    label: 'Return flow',
    value: 'Back to access',
    detail: 'Once complete, the user is returned to the sign-in experience with the updated secret.',
  },
];

const RESET_GUARDS = [
  'Reset links are validated before a new password is accepted',
  'Password confirmation stops accidental credential mismatch',
  'Successful reset returns the user into the secure access flow',
] as const;

export function ResetPasswordForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams?.get('token') ?? '';

  const [showPassword, setShowPassword] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [apiError, setApiError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<ResetPasswordFormData>({
    resolver: zodResolver(resetPasswordSchema),
  });

  const password = watch('password', '');

  const onSubmit = async (data: ResetPasswordFormData) => {
    if (!token) {
      setApiError('Invalid reset link. Please request a new one.');
      return;
    }
    setApiError(null);
    setIsSubmitting(true);
    try {
      await apiPost(API_ENDPOINTS.AUTH_RESET_PASSWORD, {
        token,
        new_password: data.password,
      });
      setSuccess(true);
      setTimeout(() => router.push(ROUTES.LOGIN), 3000);
    } catch (err) {
      if (isApiError(err)) {
        if (err.code === 'TOKEN_EXPIRED') {
          setApiError('This reset link has expired. Please request a new one.');
        } else if (err.code === 'TOKEN_INVALID') {
          setApiError('Invalid reset link. Please request a new password reset.');
        } else {
          setApiError(err.message ?? 'Failed to reset password. Please try again.');
        }
      } else {
        setApiError('An unexpected error occurred. Please try again.');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  if (!token) {
    return (
      <div className="space-y-8">
        <AuthPageIntro
          badge="Reset unavailable"
          badgeIcon={TimerReset}
          title="The reset token is missing"
          description="This page needs a valid recovery token before a new password can be accepted."
          statusLabel="Token state"
          statusValue="Invalid request"
        />

        <AuthInsightGrid items={RESET_INSIGHTS} />

        <Alert
          variant="destructive"
          className="border-red-200 bg-red-50 text-red-900 [&>svg]:text-red-600"
        >
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            Invalid or missing reset token. Please request a new password reset link.
          </AlertDescription>
        </Alert>

        <AuthActionStrip
          description="You need a fresh recovery email before you can set a new password."
          href={ROUTES.FORGOT_PASSWORD}
          cta="Request new reset link"
        />
      </div>
    );
  }

  if (success) {
    return (
      <div className="space-y-8">
        <AuthPageIntro
          badge="Reset complete"
          badgeIcon={CheckCircle}
          title="Password reset successful"
          description="The credential update has been accepted and the account is ready for sign-in."
          statusLabel="Redirect"
          statusValue="Returning to access flow"
        />

        <AuthInsightGrid items={RESET_INSIGHTS} />

        <AuthCenteredState
          icon={CheckCircle}
          title="Password updated"
          description="Your password has been rotated successfully. You will be redirected to sign in shortly."
          secondary="If the redirect does not happen automatically, use the action below."
        >
          <Link
            href={ROUTES.LOGIN}
            className="inline-flex items-center gap-2 text-sm font-semibold text-[#0f5132] hover:underline"
          >
            Sign in now
            <ArrowRight className="h-4 w-4" />
          </Link>
        </AuthCenteredState>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <AuthPageIntro
        badge="Credential reset"
        badgeIcon={Lock}
        title="Set a new password"
        description="Choose a strong password for the account and complete the recovery flow without leaving the premium auth experience."
        statusLabel="Token state"
        statusValue="Validated for update"
      />

      <AuthInsightGrid items={RESET_INSIGHTS} />

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
            <Label htmlFor="password" className="text-sm font-medium text-slate-700">
              New password
            </Label>
            <div className="relative">
              <Input
                id="password"
                type={showPassword ? 'text' : 'password'}
                autoComplete="new-password"
                autoFocus
                aria-describedby={errors.password ? 'password-error' : undefined}
                aria-invalid={!!errors.password}
                className={`${AUTH_INPUT_CLASSNAME} pr-12`}
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
              <p id="password-error" className="text-sm text-destructive" role="alert">
                {errors.password.message}
              </p>
            ) : null}
            <PasswordStrengthMeter password={password} />
          </div>

          <div className="space-y-2">
            <Label htmlFor="confirm_password" className="text-sm font-medium text-slate-700">
              Confirm new password
            </Label>
            <div className="relative">
              <Input
                id="confirm_password"
                type={showConfirm ? 'text' : 'password'}
                autoComplete="new-password"
                aria-describedby={errors.confirm_password ? 'confirm-error' : undefined}
                aria-invalid={!!errors.confirm_password}
                className={`${AUTH_INPUT_CLASSNAME} pr-12`}
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
              <p id="confirm-error" className="text-sm text-destructive" role="alert">
                {errors.confirm_password.message}
              </p>
            ) : null}
          </div>

          <AuthCallout icon={ShieldCheck} title="Reset hygiene">
            This action rotates the credential immediately. Use a password that is unique to this
            workspace and not reused elsewhere.
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
                Resetting...
              </>
            ) : (
              'Reset password'
            )}
          </Button>
        </form>

        <AuthGuardGrid items={RESET_GUARDS} />
      </AuthFormSurface>

      <AuthActionStrip
        description="Need to restart the recovery flow or request a fresh link instead?"
        href={ROUTES.FORGOT_PASSWORD}
        cta="Request new reset link"
      />
    </div>
  );
}
