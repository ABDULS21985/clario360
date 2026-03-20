'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  Activity,
  AlertCircle,
  ArrowRight,
  CheckCircle2,
  Eye,
  EyeOff,
  Lock,
  Mail,
  ShieldCheck,
  Sparkles,
  Workflow,
  type LucideIcon,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Spinner } from '@/components/ui/spinner';
import { MFACodeInput } from './mfa-code-input';
import { OAuthProviders } from './oauth-providers';
import { loginSchema, type LoginFormData } from '@/lib/validators';
import { useAuth } from '@/hooks/use-auth';
import { useCountdown } from '@/hooks/use-countdown';
import { isApiError } from '@/types/api';
import { cn } from '@/lib/utils';

type LoginStep = 'credentials' | 'mfa';
type MFAMode = 'totp' | 'recovery';

const ACCESS_SIGNALS: Array<{
  icon: LucideIcon;
  label: string;
  value: string;
  detail: string;
}> = [
  {
    icon: ShieldCheck,
    label: 'Identity shield',
    value: 'Adaptive MFA',
    detail: 'Risk-based verification stays armed across every suite.',
  },
  {
    icon: Activity,
    label: 'Telemetry context',
    value: 'Live',
    detail: 'Threat, audit, and workflow signals stay synchronized.',
  },
  {
    icon: Workflow,
    label: 'Execution layer',
    value: 'Unified',
    detail: 'Cyber, data, governance, and executive workstreams stay connected.',
  },
] as const;

const ACCESS_GUARDS = [
  'Anomaly detection on every session',
  'Device-aware sign-in monitoring',
  'Centralized audit trails and access review',
] as const;

function AccessSignalCard({
  icon: Icon,
  label,
  value,
  detail,
}: {
  icon: LucideIcon;
  label: string;
  value: string;
  detail: string;
}) {
  return (
    <div className="rounded-[22px] border border-slate-200/80 bg-white/[0.85] p-4 shadow-[0_18px_40px_rgba(15,23,42,0.05)]">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-[11px] uppercase tracking-[0.24em] text-slate-400">{label}</p>
          <p className="mt-3 text-lg font-semibold tracking-tight text-slate-900">{value}</p>
        </div>
        <div className="rounded-2xl bg-[#0f5132]/10 p-2.5 text-[#0f5132]">
          <Icon className="h-5 w-5" />
        </div>
      </div>
      <p className="mt-3 text-sm leading-6 text-slate-500">{detail}</p>
    </div>
  );
}

export function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const rawRedirect = searchParams?.get('redirect') ?? searchParams?.get('return_to') ?? '/dashboard';
  // Treat root redirect as dashboard (root page just redirects there anyway)
  const redirectTo = rawRedirect === '/' ? '/dashboard' : rawRedirect;
  const registeredBanner = searchParams?.get('registered') === 'true';

  const { login, verifyMFA } = useAuth();
  const { seconds: countdownSeconds, isRunning: isCountingDown, start: startCountdown } =
    useCountdown();

  const [step, setStep] = useState<LoginStep>('credentials');
  const [mfaToken, setMfaToken] = useState('');
  const [mfaMode, setMfaMode] = useState<MFAMode>('totp');
  const [recoveryCode, setRecoveryCode] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [apiError, setApiError] = useState<string | null>(null);
  const [mfaError, setMfaError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [mfaShake, setMfaShake] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
    setFocus,
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    mode: 'onBlur',
  });

  useEffect(() => {
    if (step === 'credentials') {
      setFocus('email');
    }
  }, [setFocus, step]);

  const triggerMFAShake = () => {
    setMfaShake(true);
    setTimeout(() => setMfaShake(false), 600);
  };

  const navigateAfterAuth = (target: string) => {
    if (/^https?:\/\//i.test(target)) {
      window.location.assign(target);
      return;
    }
    router.push(target);
  };

  const onSubmit = async (data: LoginFormData) => {
    setApiError(null);
    setIsSubmitting(true);
    try {
      const result = await login(data.email, data.password);
      if (result.requiresMFA && result.mfaToken) {
        setMfaToken(result.mfaToken);
        setStep('mfa');
      } else {
        navigateAfterAuth(redirectTo);
      }
    } catch (err) {
      if (isApiError(err)) {
        if (err.status === 429) {
          const retryAfter = err.details?.['retry_after']?.[0];
          const seconds = retryAfter ? parseInt(retryAfter, 10) : 60;
          startCountdown(seconds);
          setApiError(
            `Too many login attempts. Please try again in ${seconds} seconds.`,
          );
        } else if (err.status === 401) {
          setApiError('Invalid email or password.');
        } else if (err.code === 'ACCOUNT_LOCKED') {
          const minutes = err.details?.['lock_minutes']?.[0] ?? '?';
          setApiError(
            `Your account is locked. Please try again in ${minutes} minutes.`,
          );
        } else if (err.code === 'ACCOUNT_SUSPENDED') {
          setApiError(
            'Your account has been suspended. Contact your administrator.',
          );
        } else if (err.status === 0) {
          setApiError('Unable to connect to server. Please check your connection.');
        } else {
          setApiError(err.message ?? 'An unexpected error occurred. Please try again.');
        }
      } else {
        setApiError('An unexpected error occurred. Please try again.');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleMFAComplete = async (code: string) => {
    setMfaError(null);
    setIsSubmitting(true);
    try {
      await verifyMFA(mfaToken, code);
      navigateAfterAuth(redirectTo);
    } catch (err) {
      triggerMFAShake();
      if (isApiError(err)) {
        if (err.code === 'TOKEN_EXPIRED') {
          setMfaError('Your session has expired. Please sign in again.');
          setStep('credentials');
        } else {
          setMfaError('Invalid code. Please try again.');
        }
      } else {
        setMfaError('Invalid code. Please try again.');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleRecoverySubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!recoveryCode.trim()) return;
    await handleMFAComplete(recoveryCode.trim());
  };

  // ── Credentials step ──────────────────────────────────────────────────────

  if (step === 'credentials') {
    return (
      <div className="space-y-8">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="space-y-3">
            <div className="inline-flex items-center gap-2 rounded-full border border-[#0f5132]/15 bg-[#0f5132]/5 px-3 py-1 text-xs font-medium text-[#0f5132]">
              <Sparkles className="h-3.5 w-3.5" />
              Command center sign-in
            </div>
            <div className="space-y-2">
              <h1 className="text-3xl font-semibold tracking-tight text-slate-950 sm:text-[2.2rem]">
                Resume secure operations
              </h1>
              <p className="max-w-2xl text-sm leading-7 text-slate-600 sm:text-base">
                Sign in to investigate alerts, approve workflows, and monitor platform health from
                one surface that feels like the product you are about to enter.
              </p>
            </div>
          </div>

          <div className="rounded-[24px] border border-[#0f5132]/15 bg-[#0f5132]/5 px-4 py-3 shadow-sm">
            <p className="text-[11px] uppercase tracking-[0.28em] text-[#0f5132]/70">
              Workspace status
            </p>
            <div className="mt-2 flex items-center gap-2 text-sm font-semibold text-[#0f5132]">
              <span className="h-2.5 w-2.5 rounded-full bg-emerald-500 shadow-[0_0_0_4px_rgba(34,197,94,0.15)]" />
              Online and synchronized
            </div>
          </div>
        </div>

        <div className="grid gap-3 sm:grid-cols-3">
          {ACCESS_SIGNALS.map((item) => (
            <AccessSignalCard key={item.label} {...item} />
          ))}
        </div>

        {registeredBanner && (
          <Alert
            variant="success"
            className="border-emerald-200 bg-emerald-50 text-emerald-900 [&>svg]:text-emerald-600"
          >
            <AlertDescription>
              Registration successful! Please sign in.
            </AlertDescription>
          </Alert>
        )}

        {apiError && (
          <Alert
            variant="destructive"
            role="alert"
            aria-live="assertive"
            className="border-red-200 bg-red-50 text-red-900 [&>svg]:text-red-600"
          >
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              {apiError}
              {isCountingDown && (
                <span className="ml-1 font-mono font-semibold">
                  ({countdownSeconds}s)
                </span>
              )}
            </AlertDescription>
          </Alert>
        )}

        <form
          onSubmit={handleSubmit(onSubmit)}
          noValidate
          role="form"
          aria-label="Sign in"
          className="space-y-5 rounded-[28px] border border-slate-200/80 bg-slate-50/70 p-5 shadow-[inset_0_1px_0_rgba(255,255,255,0.9)] sm:p-6"
        >
          <div className="space-y-2">
            <Label htmlFor="email" className="text-sm font-medium text-slate-700">
              Work email
            </Label>
            <div className="relative">
              <Mail className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
              <Input
                id="email"
                type="email"
                autoComplete="email"
                autoFocus
                aria-describedby={errors.email ? 'email-error' : undefined}
                aria-invalid={!!errors.email}
                className="h-12 rounded-2xl border-slate-200 bg-white pl-11 pr-4 text-[15px] shadow-sm placeholder:text-slate-400 focus-visible:ring-[#0f5132]/25"
                placeholder="name@company.com"
                {...register('email')}
              />
            </div>
            {errors.email && (
              <p id="email-error" className="text-sm text-destructive" role="alert">
                {errors.email.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="password" className="text-sm font-medium text-slate-700">
              Password
            </Label>
            <div className="relative">
              <Lock className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
              <Input
                id="password"
                type={showPassword ? 'text' : 'password'}
                autoComplete="current-password"
                aria-describedby={errors.password ? 'password-error' : undefined}
                aria-invalid={!!errors.password}
                className="h-12 rounded-2xl border-slate-200 bg-white pl-11 pr-12 text-[15px] shadow-sm placeholder:text-slate-400 focus-visible:ring-[#0f5132]/25"
                placeholder="Enter your password"
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
            {errors.password && (
              <p id="password-error" className="text-sm text-destructive" role="alert">
                {errors.password.message}
              </p>
            )}
          </div>

          <div className="flex flex-col gap-3 rounded-[22px] border border-slate-200 bg-white/90 p-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-2">
              <Checkbox id="remember" className="rounded border-slate-300" />
              <Label htmlFor="remember" className="cursor-pointer font-normal">
                Remember me
              </Label>
            </div>
            <Link
              href="/forgot-password"
              className="text-sm font-medium text-[#0f5132] hover:underline"
            >
              Forgot your password?
            </Link>
          </div>

          <Button
            type="submit"
            className="h-12 w-full rounded-2xl bg-[#0f5132] text-base font-semibold shadow-[0_18px_40px_rgba(15,81,50,0.22)] transition-transform hover:-translate-y-0.5 hover:bg-[#0c432b]"
            disabled={isSubmitting || isCountingDown}
            aria-busy={isSubmitting}
          >
            {isSubmitting ? (
              <>
                <Spinner size="sm" className="mr-2" />
                Signing in&hellip;
              </>
            ) : (
              <>
                Sign in
                <ArrowRight className="ml-2 h-4 w-4" />
              </>
            )}
          </Button>

          <div className="grid gap-2 sm:grid-cols-3">
            {ACCESS_GUARDS.map((item) => (
              <div
                key={item}
                className="flex items-center gap-2 rounded-[18px] border border-white/80 bg-white/80 px-3 py-3 text-xs leading-5 text-slate-500"
              >
                <CheckCircle2 className="h-4 w-4 shrink-0 text-[#0f5132]" />
                <span>{item}</span>
              </div>
            ))}
          </div>
        </form>

        <OAuthProviders className="rounded-[28px] border border-slate-200/80 bg-white/95 p-5 shadow-[0_18px_45px_rgba(15,23,42,0.06)]" />

        <div className="flex flex-col gap-3 rounded-[24px] border border-slate-200/70 bg-slate-50/70 p-4 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-sm text-slate-600">
            Don&apos;t have an account? Create a workspace and continue into onboarding.
          </p>
          <Link
            href="/register"
            className="inline-flex items-center gap-2 text-sm font-semibold text-[#0f5132] hover:underline"
          >
            Create an account
            <ArrowRight className="h-4 w-4" />
          </Link>
        </div>
      </div>
    );
  }

  // ── MFA step ──────────────────────────────────────────────────────────────

  return (
    <div className="space-y-8">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="space-y-3">
          <div className="inline-flex items-center gap-2 rounded-full border border-[#0f5132]/15 bg-[#0f5132]/5 px-3 py-1 text-xs font-medium text-[#0f5132]">
            <ShieldCheck className="h-3.5 w-3.5" />
            Identity verification
          </div>
          <div className="space-y-2">
            <h1 className="text-3xl font-semibold tracking-tight text-slate-950 sm:text-[2.2rem]">
              Complete step two
            </h1>
            <p className="max-w-2xl text-sm leading-7 text-slate-600 sm:text-base">
              Your workspace requires an additional proof point before access is granted.
            </p>
          </div>
        </div>

        <div className="rounded-[24px] border border-[#0f5132]/15 bg-[#0f5132]/5 px-4 py-3 shadow-sm">
          <p className="text-[11px] uppercase tracking-[0.28em] text-[#0f5132]/70">
            Verification mode
          </p>
          <p className="mt-2 text-sm font-semibold text-[#0f5132]">
            {mfaMode === 'totp' ? 'Authenticator app' : 'Recovery code'}
          </p>
        </div>
      </div>

      <div className="grid gap-3 sm:grid-cols-3">
        {ACCESS_GUARDS.map((item) => (
          <div
            key={item}
            className="rounded-[22px] border border-slate-200/80 bg-white/[0.85] p-4 shadow-[0_18px_40px_rgba(15,23,42,0.05)]"
          >
            <div className="flex items-center gap-2 text-[#0f5132]">
              <CheckCircle2 className="h-4 w-4" />
              <p className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-400">
                Secure step
              </p>
            </div>
            <p className="mt-3 text-sm leading-6 text-slate-600">{item}</p>
          </div>
        ))}
      </div>

      {mfaError && (
        <Alert
          variant="destructive"
          role="alert"
          aria-live="assertive"
          className="border-red-200 bg-red-50 text-red-900 [&>svg]:text-red-600"
        >
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{mfaError}</AlertDescription>
        </Alert>
      )}

      <div className="space-y-5 rounded-[28px] border border-slate-200/80 bg-slate-50/70 p-5 shadow-[inset_0_1px_0_rgba(255,255,255,0.9)] sm:p-6">
        <div className="grid grid-cols-2 gap-2 rounded-[22px] border border-slate-200 bg-white p-1">
          <button
            type="button"
            className={cn(
              'rounded-[18px] px-4 py-3 text-sm font-medium transition-colors',
              mfaMode === 'totp'
                ? 'bg-[#0f5132] text-white shadow-sm'
                : 'text-slate-500 hover:bg-slate-50 hover:text-slate-900',
            )}
            onClick={() => {
              setMfaMode('totp');
              setMfaError(null);
            }}
          >
            Authenticator app
          </button>
          <button
            type="button"
            className={cn(
              'rounded-[18px] px-4 py-3 text-sm font-medium transition-colors',
              mfaMode === 'recovery'
                ? 'bg-[#0f5132] text-white shadow-sm'
                : 'text-slate-500 hover:bg-slate-50 hover:text-slate-900',
            )}
            onClick={() => {
              setMfaMode('recovery');
              setMfaError(null);
            }}
          >
            Recovery code
          </button>
        </div>

        {mfaMode === 'totp' ? (
          <div className="space-y-5">
            <div className="rounded-[24px] border border-slate-200 bg-white/90 p-5">
              <p className="text-sm leading-6 text-slate-600">
                {mfaMode === 'totp'
                  ? 'Enter the 6-digit code from your authenticator app.'
                  : 'Enter your recovery code.'}
              </p>
              <div
                className={cn(
                  'mt-6 flex justify-center transition-transform',
                  mfaShake && 'animate-[shake_0.5s_ease-in-out]',
                )}
              >
                <MFACodeInput
                  onComplete={handleMFAComplete}
                  disabled={isSubmitting}
                  error={!!mfaError}
                  className="justify-center"
                />
              </div>
              <p className="mt-5 text-center text-sm text-slate-500">
                Codes refresh roughly every 30 seconds in your authenticator app.
              </p>
            </div>
            {isSubmitting && (
              <div className="flex justify-center">
                <Spinner />
              </div>
            )}
          </div>
        ) : (
          <form onSubmit={handleRecoverySubmit} className="space-y-5">
            <div className="space-y-2">
              <Label htmlFor="recovery-code" className="text-sm font-medium text-slate-700">
                Recovery code
              </Label>
              <div className="relative">
                <Lock className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                <Input
                  id="recovery-code"
                  type="text"
                  autoComplete="off"
                  autoFocus
                  value={recoveryCode}
                  onChange={(e) => setRecoveryCode(e.target.value)}
                  placeholder="Enter your recovery code"
                  aria-describedby={mfaError ? 'mfa-error' : undefined}
                  className="h-12 rounded-2xl border-slate-200 bg-white pl-11 pr-4 text-[15px] shadow-sm placeholder:text-slate-400 focus-visible:ring-[#0f5132]/25"
                />
              </div>
            </div>
            <Button
              type="submit"
              className="h-12 w-full rounded-2xl bg-[#0f5132] text-base font-semibold shadow-[0_18px_40px_rgba(15,81,50,0.22)] transition-transform hover:-translate-y-0.5 hover:bg-[#0c432b]"
              disabled={isSubmitting || !recoveryCode.trim()}
              aria-busy={isSubmitting}
            >
              {isSubmitting ? <Spinner size="sm" className="mr-2" /> : null}
              Verify recovery code
            </Button>
          </form>
        )}
      </div>

      <p className="text-sm leading-7 text-slate-600">
          {mfaMode === 'totp'
            ? 'If you do not have access to your authenticator app, switch to a recovery code. Your workspace access will remain protected either way.'
            : 'Recovery codes are single-use credentials. Once access is restored, rotate them from account settings.'}
        </p>

      <div className="text-center">
        <button
          type="button"
          className="text-sm font-medium text-slate-500 hover:text-slate-900 hover:underline"
          onClick={() => {
            setStep('credentials');
            setMfaToken('');
            setMfaError(null);
          }}
        >
          ← Back to login
        </button>
      </div>
    </div>
  );
}
