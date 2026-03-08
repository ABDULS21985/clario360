'use client';

import React, { useState, useEffect, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Eye, EyeOff, AlertCircle, UserPlus, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Spinner } from '@/components/ui/spinner';
import { PasswordStrengthMeter } from '@/components/auth/password-strength-meter';
import { apiGet, apiPost } from '@/lib/api';
import { isApiError } from '@/types/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { setAccessToken } from '@/lib/auth';
import { useAuthStore } from '@/stores/auth-store';

// ──────────────────────────────────────────────────────────────
// Schema
// ──────────────────────────────────────────────────────────────

const acceptSchema = z
  .object({
    first_name: z.string().min(1, 'First name is required').max(100),
    last_name: z.string().min(1, 'Last name is required').max(100),
    password: z
      .string()
      .min(12, 'Password must be at least 12 characters')
      .max(128, 'Password must be at most 128 characters'),
    confirm_password: z.string(),
  })
  .refine((d) => d.password === d.confirm_password, {
    message: 'Passwords do not match',
    path: ['confirm_password'],
  });

type AcceptFormData = z.infer<typeof acceptSchema>;

// ──────────────────────────────────────────────────────────────
// Invite details type
// ──────────────────────────────────────────────────────────────

interface InviteDetails {
  invitation_id: string;
  tenant_id: string;
  email: string;
  role_name: string;
  organization_name: string;
  inviter_name: string;
  expires_at: string;
  message?: string;
}

// ──────────────────────────────────────────────────────────────
// Main form
// ──────────────────────────────────────────────────────────────

function InviteForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams?.get('token') ?? '';

  const [details, setDetails] = useState<InviteDetails | null>(null);
  const [loadingDetails, setLoadingDetails] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [apiError, setApiError] = useState<string | null>(null);
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);

  const refreshSession = useAuthStore((s) => s.refreshSession);

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<AcceptFormData>({ resolver: zodResolver(acceptSchema), mode: 'onBlur' });

  const password = watch('password', '');

  // Load invite details
  useEffect(() => {
    if (!token) {
      setLoadError('No invitation token provided. Please check the link in your email.');
      setLoadingDetails(false);
      return;
    }

    const load = async () => {
      try {
        const data = await apiGet<InviteDetails>(
          `${API_ENDPOINTS.ONBOARDING_INVITATIONS_VALIDATE}?token=${encodeURIComponent(token)}`
        );
        setDetails(data);
      } catch (err) {
        if (isApiError(err)) {
          if (err.status === 401 || err.status === 410) {
            setLoadError('This invitation has expired or is no longer valid. Please contact your administrator.');
          } else {
            setLoadError(err.message || 'Failed to load invitation details.');
          }
        } else {
          setLoadError('Failed to load invitation details. Please try again.');
        }
      } finally {
        setLoadingDetails(false);
      }
    };

    load();
  }, [token]);

  const onSubmit = async (data: AcceptFormData) => {
    setApiError(null);
    try {
      const resp = await apiPost<{
        access_token: string;
        refresh_token: string;
        tenant_id: string;
        message: string;
      }>(API_ENDPOINTS.ONBOARDING_INVITATIONS_ACCEPT, {
        token,
        first_name: data.first_name,
        last_name: data.last_name,
        password: data.password,
      });

      // Store access token in memory
      setAccessToken(resp.access_token);

      await fetch(API_ENDPOINTS.BFF_SESSION, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ access_token: resp.access_token, refresh_token: resp.refresh_token }),
      });

      // Exchange for session cookie via BFF
      try {
        await refreshSession();
      } catch {
        // best-effort
      }

      router.push('/dashboard');
    } catch (err) {
      setApiError(isApiError(err) ? err.message : 'Failed to accept invitation. Please try again.');
    }
  };

  // ─── Loading state ───────────────────────────────────────────
  if (loadingDetails) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-8">
        <Spinner />
        <p className="text-sm text-muted-foreground">Loading invitation…</p>
      </div>
    );
  }

  // ─── Load error ──────────────────────────────────────────────
  if (loadError || !details) {
    return (
      <div className="space-y-4">
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{loadError ?? 'Invalid invitation.'}</AlertDescription>
        </Alert>
        <p className="text-center text-sm text-muted-foreground">
          Need help?{' '}
          <a href="mailto:support@clario360.com" className="text-primary hover:underline">
            Contact support
          </a>
        </p>
      </div>
    );
  }

  const expiresDate = new Date(details.expires_at).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });

  // ─── Form ────────────────────────────────────────────────────
  return (
    <div className="space-y-6">
      {/* Invite summary card */}
      <div className="rounded-lg border border-[#1B5E20]/20 bg-[#1B5E20]/5 p-4">
        <div className="flex items-start gap-3">
          <div className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full bg-[#1B5E20]/10">
            <UserPlus className="h-5 w-5 text-[#1B5E20]" />
          </div>
          <div>
            <p className="text-sm font-medium text-foreground">
              {details.inviter_name} invited you to{' '}
              <span className="font-semibold">{details.organization_name}</span>
            </p>
            <p className="mt-0.5 text-xs text-muted-foreground">
              Role: <span className="font-medium">{details.role_name}</span> &middot; Expires{' '}
              {expiresDate}
            </p>
            {details.email && (
              <p className="mt-0.5 text-xs text-muted-foreground">
                For: <span className="font-medium">{details.email}</span>
              </p>
            )}
            {details.message && (
              <p className="mt-2 text-sm italic text-muted-foreground">
                &ldquo;{details.message}&rdquo;
              </p>
            )}
          </div>
        </div>
      </div>

      <div className="space-y-1">
        <h1 className="text-2xl font-bold">Create your account</h1>
        <p className="text-sm text-muted-foreground">
          Set up your password to get started.
        </p>
      </div>

      {apiError && (
        <Alert variant="destructive" role="alert">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{apiError}</AlertDescription>
        </Alert>
      )}

      <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-4">
        {/* Name row */}
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="first_name">First name *</Label>
            <Input
              id="first_name"
              type="text"
              autoComplete="given-name"
              aria-invalid={!!errors.first_name}
              {...register('first_name')}
            />
            {errors.first_name && (
              <p className="text-sm text-destructive">{errors.first_name.message}</p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="last_name">Last name *</Label>
            <Input
              id="last_name"
              type="text"
              autoComplete="family-name"
              aria-invalid={!!errors.last_name}
              {...register('last_name')}
            />
            {errors.last_name && (
              <p className="text-sm text-destructive">{errors.last_name.message}</p>
            )}
          </div>
        </div>

        {/* Password */}
        <div className="space-y-2">
          <Label htmlFor="password">Password *</Label>
          <div className="relative">
            <Input
              id="password"
              type={showPassword ? 'text' : 'password'}
              autoComplete="new-password"
              aria-invalid={!!errors.password}
              className="pr-10"
              {...register('password')}
            />
            <button
              type="button"
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              onClick={() => setShowPassword((p) => !p)}
              aria-label={showPassword ? 'Hide password' : 'Show password'}
            >
              {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
          </div>
          {errors.password && (
            <p className="text-sm text-destructive">{errors.password.message}</p>
          )}
          <PasswordStrengthMeter password={password} />
        </div>

        {/* Confirm password */}
        <div className="space-y-2">
          <Label htmlFor="confirm_password">Confirm password *</Label>
          <div className="relative">
            <Input
              id="confirm_password"
              type={showConfirm ? 'text' : 'password'}
              autoComplete="new-password"
              aria-invalid={!!errors.confirm_password}
              className="pr-10"
              {...register('confirm_password')}
            />
            <button
              type="button"
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              onClick={() => setShowConfirm((p) => !p)}
              aria-label={showConfirm ? 'Hide password' : 'Show password'}
            >
              {showConfirm ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
          </div>
          {errors.confirm_password && (
            <p className="text-sm text-destructive">{errors.confirm_password.message}</p>
          )}
        </div>

        <Button
          type="submit"
          className="w-full"
          disabled={isSubmitting}
          aria-busy={isSubmitting}
        >
          {isSubmitting ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Creating account…
            </>
          ) : (
            'Accept invitation & create account'
          )}
        </Button>
      </form>

      <p className="text-center text-xs text-muted-foreground">
        By accepting, you agree to Clario 360&apos;s{' '}
        <a href="/terms" className="underline hover:text-foreground">
          Terms of Service
        </a>{' '}
        and{' '}
        <a href="/privacy" className="underline hover:text-foreground">
          Privacy Policy
        </a>
        .
      </p>
    </div>
  );
}

export default function InvitePage() {
  return (
    <Suspense fallback={<div className="flex justify-center py-8"><Spinner /></div>}>
      <InviteForm />
    </Suspense>
  );
}
