'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { ArrowLeft, KeyRound, Mail, ShieldCheck, TimerReset, Workflow } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Spinner } from '@/components/ui/spinner';
import { apiPost } from '@/lib/api';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';
import { forgotPasswordSchema, type ForgotPasswordFormData } from '@/lib/validators';

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

const RECOVERY_INSIGHTS: AuthInsightItem[] = [
  {
    icon: KeyRound,
    label: 'Recovery path',
    value: 'Tokenized reset',
    detail: 'Password recovery is issued through a short-lived email token, not a static link.',
  },
  {
    icon: ShieldCheck,
    label: 'Lookup safety',
    value: 'Non-disclosing',
    detail: 'The response remains identical whether the email exists or not.',
  },
  {
    icon: Workflow,
    label: 'Next motion',
    value: 'Return to sign-in',
    detail: 'After reset, the user is routed back into the premium access flow.',
  },
];

const RECOVERY_GUARDS = [
  'Email enumeration is suppressed by design',
  'Recovery links are time-bound and single-purpose',
  'Successful reset returns the user to secure sign-in',
] as const;

export function ForgotPasswordForm() {
  const [submitted, setSubmitted] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    getValues,
    formState: { errors },
  } = useForm<ForgotPasswordFormData>({
    resolver: zodResolver(forgotPasswordSchema),
  });

  const onSubmit = async (data: ForgotPasswordFormData) => {
    setIsSubmitting(true);
    try {
      await apiPost(API_ENDPOINTS.AUTH_FORGOT_PASSWORD, { email: data.email });
    } catch {
      // Always show success to prevent email enumeration
    } finally {
      setIsSubmitting(false);
      setSubmitted(true);
    }
  };

  if (submitted) {
    return (
      <div className="space-y-8">
        <AuthPageIntro
          badge="Recovery issued"
          badgeIcon={TimerReset}
          title="Check your email"
          description="If an account exists for the email you entered, a recovery link is already on the way."
          statusLabel="Recovery status"
          statusValue="Awaiting email action"
        />

        <AuthInsightGrid items={RECOVERY_INSIGHTS} />

        <AuthCenteredState
          icon={Mail}
          title="Recovery message sent"
          description={
            <>
              If an account exists for{' '}
              <span className="font-semibold text-slate-900">{getValues('email')}</span>, you will
              receive a password reset link within a few minutes.
            </>
          }
          secondary="Do not forget to check spam or quarantine folders if the message does not appear right away."
        >
          <Link
            href={ROUTES.LOGIN}
            className="inline-flex items-center gap-2 text-sm font-semibold text-[#0f5132] hover:underline"
          >
            <ArrowLeft className="h-4 w-4" />
            Back to sign in
          </Link>
        </AuthCenteredState>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <AuthPageIntro
        badge="Credential recovery"
        badgeIcon={KeyRound}
        title="Reset access without exposing identity"
        description="Enter your work email and we will issue a time-bound recovery path while keeping account discovery protected."
        statusLabel="Recovery control"
        statusValue="Enumeration resistant"
      />

      <AuthInsightGrid items={RECOVERY_INSIGHTS} />

      <AuthFormSurface>
        <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-5">
          <div className="space-y-2">
            <Label htmlFor="email" className="text-sm font-medium text-slate-700">
              Work email
            </Label>
            <Input
              id="email"
              type="email"
              autoComplete="email"
              autoFocus
              aria-describedby={errors.email ? 'email-error' : undefined}
              aria-invalid={!!errors.email}
              className={AUTH_INPUT_CLASSNAME}
              {...register('email')}
            />
            {errors.email ? (
              <p id="email-error" className="text-sm text-destructive" role="alert">
                {errors.email.message}
              </p>
            ) : null}
          </div>

          <AuthCallout icon={ShieldCheck} title="Protected request flow">
            The experience looks the same for valid and invalid email addresses. That keeps account
            discovery from leaking through the recovery endpoint.
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
                Sending...
              </>
            ) : (
              'Send reset link'
            )}
          </Button>
        </form>

        <AuthGuardGrid items={RECOVERY_GUARDS} />
      </AuthFormSurface>

      <AuthActionStrip
        description="Remembered your password or already recovered access?"
        href={ROUTES.LOGIN}
        cta="Back to sign in"
      />
    </div>
  );
}
