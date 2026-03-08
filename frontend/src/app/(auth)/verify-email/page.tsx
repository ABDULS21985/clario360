'use client';

import React, { useState, useRef, useEffect, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { AlertCircle, Mail, RotateCcw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Spinner } from '@/components/ui/spinner';
import { apiPost } from '@/lib/api';
import { isApiError } from '@/types/api';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';
import { setAccessToken } from '@/lib/auth';
import { useAuthStore } from '@/stores/auth-store';

const OTP_LENGTH = 6;
const RESEND_COOLDOWN_SECONDS = 60;

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
    // Auto-submit when all digits are filled
    if (digit && idx === OTP_LENGTH - 1 && next.every((d) => d !== '')) {
      submitOtp(next.join(''));
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
    for (let i = 0; i < text.length; i++) {
      next[i] = text[i];
    }
    setOtp(next);
    const lastFilledIdx = Math.min(text.length - 1, OTP_LENGTH - 1);
    inputRefs.current[lastFilledIdx]?.focus();
    if (next.every((d) => d !== '')) {
      submitOtp(next.join(''));
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

      // Store access token in memory
      setAccessToken(resp.access_token);

      await fetch(API_ENDPOINTS.BFF_SESSION, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ access_token: resp.access_token, refresh_token: resp.refresh_token }),
      });

      // Fetch user session and store in auth store
      try {
        await refreshSession();
      } catch {
        // session store is best-effort here
      }

      router.push(`${ROUTES.SETUP}?step=1`);
    } catch (err) {
      setApiError(isApiError(err) ? err.message : 'Verification failed. Please try again.');
      // Reset OTP inputs on error
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
    submitOtp(code);
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
    <div className="space-y-6">
      <div className="space-y-2 text-center">
        <div className="flex justify-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-[#1B5E20]/10">
            <Mail className="h-6 w-6 text-[#1B5E20]" />
          </div>
        </div>
        <h1 className="text-2xl font-bold">Check your email</h1>
        <p className="text-sm text-muted-foreground">
          We sent a 6-digit verification code to{' '}
          {email ? (
            <span className="font-medium text-foreground">{email}</span>
          ) : (
            'your email address'
          )}
          .
        </p>
        <p className="text-xs uppercase tracking-[0.2em] text-muted-foreground">
          Code expires in {countdown}
        </p>
      </div>

      {apiError && (
        <Alert variant="destructive" role="alert">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{apiError}</AlertDescription>
        </Alert>
      )}

      {successMessage && (
        <Alert role="status">
          <AlertDescription>{successMessage}</AlertDescription>
        </Alert>
      )}

      <form onSubmit={handleSubmit} className="space-y-4" noValidate>
        <fieldset disabled={isSubmitting} className="space-y-2">
          <legend className="sr-only">Enter your 6-digit verification code</legend>
          <div className="flex justify-center gap-2">
            {otp.map((digit, idx) => (
              <input
                key={idx}
                ref={(el) => { inputRefs.current[idx] = el; }}
                type="text"
                inputMode="numeric"
                pattern="\d"
                maxLength={1}
                value={digit}
                onChange={(e) => handleChange(idx, e.target.value)}
                onKeyDown={(e) => handleKeyDown(idx, e)}
                onPaste={idx === 0 ? handlePaste : undefined}
                aria-label={`Digit ${idx + 1} of ${OTP_LENGTH}`}
                className="h-12 w-12 rounded-md border border-input bg-background text-center text-lg font-semibold focus:border-[#1B5E20] focus:outline-none focus:ring-2 focus:ring-[#1B5E20]/30 disabled:opacity-50"
              />
            ))}
          </div>
        </fieldset>

        <Button
          type="submit"
          className="w-full"
          disabled={isSubmitting || otp.join('').length < OTP_LENGTH}
          aria-busy={isSubmitting}
        >
          {isSubmitting ? (
            <>
              <Spinner size="sm" className="mr-2" />
              Verifying…
            </>
          ) : (
            'Verify email'
          )}
        </Button>
      </form>

      <div className="text-center">
        <p className="mb-2 text-sm text-muted-foreground">Didn&apos;t receive the code?</p>
        <Button
          variant="ghost"
          size="sm"
          onClick={handleResend}
          disabled={isResending || resendCooldown > 0}
          className="gap-2"
        >
          {isResending ? (
            <Spinner size="sm" />
          ) : (
            <RotateCcw className="h-4 w-4" />
          )}
          {resendCooldown > 0 ? `Resend in ${resendCooldown}s` : 'Resend code'}
        </Button>
      </div>
    </div>
  );
}

export default function VerifyEmailPage() {
  return (
    <Suspense fallback={<div className="flex justify-center py-8"><Spinner /></div>}>
      <VerifyEmailForm />
    </Suspense>
  );
}
