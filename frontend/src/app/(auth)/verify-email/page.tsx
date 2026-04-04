'use client';

import React, { Suspense, useEffect, useRef, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import {
  AlertCircle,
  Mail,
  RotateCcw,
  ShieldCheck,
  Sparkles,
  TimerReset,
  Workflow,
} from 'lucide-react';

import {
  AuthCallout,
  AuthFormSurface,
  AuthGuardGrid,
  AuthInsightGrid,
  AuthLoadingState,
  AuthPageIntro,
  type AuthInsightItem,
} from '@/components/auth/auth-page-primitives';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';
import { setAccessToken } from '@/lib/auth';
import { apiPost } from '@/lib/api';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';
import { isApiError } from '@/types/api';
import { useAuthStore } from '@/stores/auth-store';

const OTP_LENGTH = 6;
const RESEND_COOLDOWN_SECONDS = 60;

const VERIFY_INSIGHTS: AuthInsightItem[] = [
  {
    icon: Mail,
    label: 'Proof channel',
    value: 'Email verified',
    detail: 'The admin session is unlocked only after the mailbox proves control of the account.',
  },
  {
    icon: ShieldCheck,
    label: 'Session bootstrap',
    value: 'Token exchange',
    detail: 'A successful code exchange provisions the browser session before setup begins.',
  },
  {
    icon: Workflow,
    label: 'Next motion',
    value: 'Setup wizard',
    detail: 'Once verified, the user continues directly into tenant onboarding without another login step.',
  },
];

const VERIFY_GUARDS = [
  'Verification expires automatically if the code is not used in time',
  'Resend requests are throttled to slow abuse and duplicate delivery',
  'A verified code creates the admin session before setup is launched',
] as const;

function VerifyEmailForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const email = searchParams?.get('email') ?? '';
  const verificationTTL = Number(searchParams?.get('ttl') ?? '600');

  const [otp, setOtp] = useState<string[]>(Array(OTP_LENGTH).fill(''));
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isResending, setIsResending] = useState(false);
  const [apiError, setApiError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [expiresInSeconds, setExpiresInSeconds] = useState(
    Number.isFinite(verificationTTL) && verificationTTL > 0 ? verificationTTL : 600,
  );
  const [resendCooldown, setResendCooldown] = useState(RESEND_COOLDOWN_SECONDS);
  const inputRefs = useRef<(HTMLInputElement | null)[]>([]);
  const refreshSession = useAuthStore((s) => s.refreshSession);

  useEffect(() => {
    inputRefs.current[0]?.focus();
  }, []);

  useEffect(() => {
    if (expiresInSeconds <= 0) {
      return undefined;
    }

    const timer = window.setInterval(() => {
      setExpiresInSeconds((current) => Math.max(current - 1, 0));
    }, 1000);

    return () => window.clearInterval(timer);
  }, [expiresInSeconds]);

  useEffect(() => {
    if (resendCooldown <= 0) {
      return undefined;
    }

    const timer = window.setInterval(() => {
      setResendCooldown((current) => Math.max(current - 1, 0));
    }, 1000);

    return () => window.clearInterval(timer);
  }, [resendCooldown]);

  const handleChange = (idx: number, value: string) => {
    const digit = value.replace(/\D/g, '').slice(-1);
    const next = [...otp];
    next[idx] = digit;
    setOtp(next);
    if (digit && idx < OTP_LENGTH - 1) {
      inputRefs.current[idx + 1]?.focus();
    }
    if (digit && idx === OTP_LENGTH - 1 && next.every((entry) => entry !== '')) {
      void submitOtp(next.join(''));
    }
  };

  const handleKeyDown = (idx: number, e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Backspace' && !otp[idx] && idx > 0) {
      inputRefs.current[idx - 1]?.focus();
    }
  };

  const handlePaste = (e: React.ClipboardEvent<HTMLInputElement>) => {
    e.preventDefault();
    const text = e.clipboardData.getData('text').replace(/\D/g, '').slice(0, OTP_LENGTH);
    if (text.length === 0) return;
    const next = [...otp];
    for (let index = 0; index < text.length; index += 1) {
      next[index] = text[index];
    }
    setOtp(next);
    const lastFilledIdx = Math.min(text.length - 1, OTP_LENGTH - 1);
    inputRefs.current[lastFilledIdx]?.focus();
    if (next.every((entry) => entry !== '')) {
      void submitOtp(next.join(''));
    }
  };

  const submitOtp = async (code: string) => {
    if (!email) {
      setApiError('Email is missing. Please go back and register again.');
      return;
    }
    setApiError(null);
    setIsSubmitting(true);
    try {
      const resp = await apiPost<{
        access_token: string;
        refresh_token: string;
        tenant_id: string;
        message: string;
      }>(API_ENDPOINTS.ONBOARDING_VERIFY_EMAIL, { email, otp: code });

      setAccessToken(resp.access_token);

      await fetch(API_ENDPOINTS.BFF_SESSION, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          access_token: resp.access_token,
          refresh_token: resp.refresh_token,
        }),
      });

      try {
        await refreshSession();
      } catch {
        // Best effort only
      }

      router.push(`${ROUTES.SETUP}?step=1`);
    } catch (err) {
      setApiError(isApiError(err) ? err.message : 'Verification failed. Please try again.');
      setOtp(Array(OTP_LENGTH).fill(''));
      inputRefs.current[0]?.focus();
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const code = otp.join('');
    if (code.length < OTP_LENGTH) {
      setApiError('Please enter all 6 digits.');
      return;
    }
    void submitOtp(code);
  };

  const handleResend = async () => {
    if (!email) return;
    setApiError(null);
    setSuccessMessage(null);
    setIsResending(true);
    try {
      await apiPost(API_ENDPOINTS.ONBOARDING_RESEND_OTP, { email });
      setSuccessMessage('A new code has been sent to your email.');
      setExpiresInSeconds(
        Number.isFinite(verificationTTL) && verificationTTL > 0 ? verificationTTL : 600,
      );
      setResendCooldown(RESEND_COOLDOWN_SECONDS);
      setOtp(Array(OTP_LENGTH).fill(''));
      inputRefs.current[0]?.focus();
    } catch (err) {
      setApiError(isApiError(err) ? err.message : 'Failed to resend code. Please try again.');
    } finally {
      setIsResending(false);
    }
  };

  const countdown = `${Math.floor(expiresInSeconds / 60)
    .toString()
    .padStart(2, '0')}:${(expiresInSeconds % 60).toString().padStart(2, '0')}`;

  return (
    <div className="space-y-8">
      <AuthPageIntro
        badge="Email verification"
        badgeIcon={Sparkles}
        title="Confirm administrator identity"
        description="Use the code sent to your inbox to activate the initial session and continue into tenant setup without another auth step."
        statusLabel="Code window"
        statusValue={`Expires in ${countdown}`}
      />

      <AuthInsightGrid items={VERIFY_INSIGHTS} />

      {apiError ? (
        <Alert
          variant="destructive"
          role="alert"
          className="border-red-200 bg-red-50 text-red-900 [&>svg]:text-red-600"
        >
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{apiError}</AlertDescription>
        </Alert>
      ) : null}

      {successMessage ? (
        <Alert
          role="status"
          className="border-emerald-200 bg-emerald-50 text-emerald-900 [&>svg]:text-emerald-600"
        >
          <AlertDescription>{successMessage}</AlertDescription>
        </Alert>
      ) : null}

      <AuthFormSurface>
        <form onSubmit={handleSubmit} className="space-y-5" noValidate>
          <fieldset disabled={isSubmitting} className="space-y-4">
            <legend className="sr-only">Enter your 6-digit verification code</legend>
            <div className="rounded-[24px] border border-slate-200 bg-white/90 p-5 text-center">
              <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-[#0f5132]/10 text-[#0f5132]">
                <Mail className="h-6 w-6" />
              </div>
              <p className="mt-4 text-sm leading-7 text-slate-600">
                We sent a 6-digit verification code to{' '}
                {email ? (
                  <span className="font-semibold text-slate-900">{email}</span>
                ) : (
                  'your email address'
                )}
                .
              </p>
              <div className="mt-6 flex justify-center gap-2">
                {otp.map((digit, idx) => (
                  <input
                    key={idx}
                    ref={(el) => {
                      inputRefs.current[idx] = el;
                    }}
                    type="text"
                    inputMode="numeric"
                    pattern="\d"
                    maxLength={1}
                    value={digit}
                    onChange={(e) => handleChange(idx, e.target.value)}
                    onKeyDown={(e) => handleKeyDown(idx, e)}
                    onPaste={idx === 0 ? handlePaste : undefined}
                    aria-label={`Digit ${idx + 1} of ${OTP_LENGTH}`}
                    className="h-12 w-12 rounded-2xl border border-slate-200 bg-white text-center text-lg font-semibold shadow-sm focus:border-[#0f5132] focus:outline-none focus:ring-2 focus:ring-[#0f5132]/25 disabled:opacity-50"
                  />
                ))}
              </div>
            </div>
          </fieldset>

          <Button
            type="submit"
            className="h-12 w-full rounded-2xl bg-[#0f5132] text-base font-semibold shadow-[0_18px_40px_rgba(15,81,50,0.22)] transition-transform hover:-translate-y-0.5 hover:bg-[#0c432b]"
            disabled={isSubmitting || otp.join('').length < OTP_LENGTH}
            aria-busy={isSubmitting}
          >
            {isSubmitting ? (
              <>
                <Spinner size="sm" className="mr-2" />
                Verifying...
              </>
            ) : (
              'Verify email'
            )}
          </Button>
        </form>

        <AuthCallout icon={TimerReset} title="Resend control">
          If the message does not arrive, request another code. Resends are delayed deliberately to
          prevent delivery spam and reduce abuse.
        </AuthCallout>

        <div className="rounded-[24px] border border-slate-200 bg-white/90 p-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <p className="text-sm font-medium text-slate-900">Didn&apos;t receive the code?</p>
              <p className="text-sm text-slate-500">
                Check spam first, then request a fresh code if needed.
              </p>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={handleResend}
              disabled={isResending || resendCooldown > 0}
              className="gap-2 rounded-2xl border-slate-200 bg-white px-4"
            >
              {isResending ? <Spinner size="sm" /> : <RotateCcw className="h-4 w-4" />}
              {resendCooldown > 0 ? `Resend in ${resendCooldown}s` : 'Resend code'}
            </Button>
          </div>
        </div>

        <AuthGuardGrid items={VERIFY_GUARDS} />
      </AuthFormSurface>
    </div>
  );
}

export default function VerifyEmailPage() {
  return (
    <Suspense
      fallback={
        <AuthLoadingState
          label="Preparing verification"
          detail="We are loading the email verification flow and checking the initial session state."
        />
      }
    >
      <VerifyEmailForm />
    </Suspense>
  );
}
