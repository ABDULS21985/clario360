'use client';

import React, { useState, useEffect, useRef } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Eye, EyeOff, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Spinner } from '@/components/ui/spinner';
import { MFACodeInput } from './mfa-code-input';
import { loginSchema, type LoginFormData } from '@/lib/validators';
import { useAuth } from '@/hooks/use-auth';
import { useCountdown } from '@/hooks/use-countdown';
import { isApiError } from '@/types/api';
import { cn } from '@/lib/utils';

type LoginStep = 'credentials' | 'mfa';
type MFAMode = 'totp' | 'recovery';

export function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const redirectTo = searchParams?.get('redirect') ?? searchParams?.get('return_to') ?? '/dashboard';
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

  const emailRef = useRef<HTMLInputElement>(null);

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
    emailRef.current?.focus();
  }, []);

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
      <div className="space-y-6">
        <div className="space-y-1">
          <h1 className="text-2xl font-bold tracking-tight">Sign in</h1>
          <p className="text-sm text-muted-foreground">
            Enter your credentials to access your account
          </p>
        </div>

        {registeredBanner && (
          <Alert variant="success">
            <AlertDescription>
              Registration successful! Please sign in.
            </AlertDescription>
          </Alert>
        )}

        {apiError && (
          <Alert variant="destructive" role="alert" aria-live="assertive">
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
          className="space-y-4"
        >
          <div className="space-y-2">
            <Label htmlFor="email">Email</Label>
            <Input
              id="email"
              type="email"
              autoComplete="email"
              autoFocus
              aria-describedby={errors.email ? 'email-error' : undefined}
              aria-invalid={!!errors.email}
              {...register('email')}
            />
            {errors.email && (
              <p id="email-error" className="text-sm text-destructive" role="alert">
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
                autoComplete="current-password"
                aria-describedby={errors.password ? 'password-error' : undefined}
                aria-invalid={!!errors.password}
                className="pr-10"
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
              <p id="password-error" className="text-sm text-destructive" role="alert">
                {errors.password.message}
              </p>
            )}
          </div>

          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Checkbox id="remember" />
              <Label htmlFor="remember" className="cursor-pointer font-normal">
                Remember me
              </Label>
            </div>
            <Link
              href="/forgot-password"
              className="text-sm text-primary hover:underline"
            >
              Forgot your password?
            </Link>
          </div>

          <Button
            type="submit"
            className="w-full"
            disabled={isSubmitting || isCountingDown}
            aria-busy={isSubmitting}
          >
            {isSubmitting ? (
              <>
                <Spinner size="sm" className="mr-2" />
                Signing in&hellip;
              </>
            ) : (
              'Sign in'
            )}
          </Button>
        </form>

        <p className="text-center text-sm text-muted-foreground">
          Don&apos;t have an account?{' '}
          <Link href="/register" className="text-primary font-medium hover:underline">
            Create an account
          </Link>
        </p>
      </div>
    );
  }

  // ── MFA step ──────────────────────────────────────────────────────────────

  return (
    <div className="space-y-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Two-factor authentication</h1>
        <p className="text-sm text-muted-foreground">
          {mfaMode === 'totp'
            ? 'Enter the 6-digit code from your authenticator app.'
            : 'Enter your recovery code.'}
        </p>
      </div>

      {mfaError && (
        <Alert variant="destructive" role="alert" aria-live="assertive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{mfaError}</AlertDescription>
        </Alert>
      )}

      {mfaMode === 'totp' ? (
        <div className="space-y-4">
          <div
            className={cn(
              'flex justify-center transition-transform',
              mfaShake && 'animate-[shake_0.5s_ease-in-out]',
            )}
          >
            <MFACodeInput
              onComplete={handleMFAComplete}
              disabled={isSubmitting}
              error={!!mfaError}
            />
          </div>
          {isSubmitting && (
            <div className="flex justify-center">
              <Spinner />
            </div>
          )}
          <div className="text-center">
            <button
              type="button"
              className="text-sm text-primary hover:underline"
              onClick={() => { setMfaMode('recovery'); setMfaError(null); }}
            >
              Use a recovery code instead
            </button>
          </div>
        </div>
      ) : (
        <form onSubmit={handleRecoverySubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="recovery-code">Recovery code</Label>
            <Input
              id="recovery-code"
              type="text"
              autoComplete="off"
              autoFocus
              value={recoveryCode}
              onChange={(e) => setRecoveryCode(e.target.value)}
              placeholder="xxxxxxxx"
              aria-describedby={mfaError ? 'mfa-error' : undefined}
            />
          </div>
          <Button
            type="submit"
            className="w-full"
            disabled={isSubmitting || !recoveryCode.trim()}
            aria-busy={isSubmitting}
          >
            {isSubmitting ? <Spinner size="sm" className="mr-2" /> : null}
            Verify recovery code
          </Button>
          <div className="text-center">
            <button
              type="button"
              className="text-sm text-primary hover:underline"
              onClick={() => { setMfaMode('totp'); setMfaError(null); }}
            >
              Use authenticator app instead
            </button>
          </div>
        </form>
      )}

      <div className="text-center">
        <button
          type="button"
          className="text-sm text-muted-foreground hover:underline"
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
