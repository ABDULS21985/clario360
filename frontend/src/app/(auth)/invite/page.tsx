'use client';

import React, { Suspense, useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  AlertCircle,
  BadgeCheck,
  Building2,
  Eye,
  EyeOff,
  Loader2,
  Mail,
  Quote,
  UserPlus,
  Workflow,
} from 'lucide-react';

import {
  AUTH_INPUT_CLASSNAME,
  AuthActionStrip,
  AuthCallout,
  AuthFormSurface,
  AuthGuardGrid,
  AuthInsightGrid,
  AuthLoadingState,
  AuthPageIntro,
  type AuthInsightItem,
} from '@/components/auth/auth-page-primitives';
import { PasswordStrengthMeter } from '@/components/auth/password-strength-meter';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { setAccessToken } from '@/lib/auth';
import { apiGet, apiPost } from '@/lib/api';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';
import { isApiError } from '@/types/api';
import { useAuthStore } from '@/stores/auth-store';

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
  .refine((data) => data.password === data.confirm_password, {
    message: 'Passwords do not match',
    path: ['confirm_password'],
  });

type AcceptFormData = z.infer<typeof acceptSchema>;

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

const INVITE_INSIGHTS: AuthInsightItem[] = [
  {
    icon: UserPlus,
    label: 'Access source',
    value: 'Admin initiated',
    detail: 'This account is being provisioned from an invitation issued by an existing workspace administrator.',
  },
  {
    icon: BadgeCheck,
    label: 'Role binding',
    value: 'Pre-scoped',
    detail: 'The role assignment is attached to the invitation before the account is created.',
  },
  {
    icon: Workflow,
    label: 'Landing path',
    value: 'Direct to workspace',
    detail: 'Once accepted, the session is created and the user is routed straight into the product.',
  },
];

const INVITE_GUARDS = [
  'Invitation validity is checked before account creation begins',
  'Role and organization scope are preserved from the original invite',
  'Accepted invitations exchange directly into an authenticated session',
] as const;

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

  const refreshSession = useAuthStore((state) => state.refreshSession);

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<AcceptFormData>({
    resolver: zodResolver(acceptSchema),
    mode: 'onBlur',
  });

  const password = watch('password', '');

  useEffect(() => {
    if (!token) {
      setLoadError('No invitation token provided. Please check the link in your email.');
      setLoadingDetails(false);
      return;
    }

    const load = async () => {
      try {
        const data = await apiGet<InviteDetails>(
          `${API_ENDPOINTS.ONBOARDING_INVITATIONS_VALIDATE}?token=${encodeURIComponent(token)}`,
        );
        setDetails(data);
      } catch (err) {
        if (isApiError(err)) {
          if (err.status === 401 || err.status === 410) {
            setLoadError(
              'This invitation has expired or is no longer valid. Please contact your administrator.',
            );
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

    void load();
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

      router.push(ROUTES.DASHBOARD);
    } catch (err) {
      setApiError(
        isApiError(err) ? err.message : 'Failed to accept invitation. Please try again.',
      );
    }
  };

  if (loadingDetails) {
    return (
      <div className="space-y-8">
        <AuthPageIntro
          badge="Invitation access"
          badgeIcon={UserPlus}
          title="Loading invitation details"
          description="We are validating the token and resolving the organization, role, and inviter context before account setup begins."
          statusLabel="Invitation state"
          statusValue="Checking validity"
        />
        <AuthInsightGrid items={INVITE_INSIGHTS} />
        <AuthLoadingState
          label="Preparing account acceptance"
          detail="The invitation is being verified against the workspace before you continue."
        />
      </div>
    );
  }

  if (loadError || !details) {
    return (
      <div className="space-y-8">
        <AuthPageIntro
          badge="Invitation access"
          badgeIcon={AlertCircle}
          title="This invitation is not available"
          description="The token could not be resolved into a valid invitation, so account creation cannot continue."
          statusLabel="Invitation state"
          statusValue="Invalid or expired"
        />

        <AuthInsightGrid items={INVITE_INSIGHTS} />

        <Alert
          variant="destructive"
          className="border-red-200 bg-red-50 text-red-900 [&>svg]:text-red-600"
        >
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{loadError ?? 'Invalid invitation.'}</AlertDescription>
        </Alert>

        <AuthActionStrip
          description="You will need a fresh invitation or administrator help to continue."
          href={ROUTES.LOGIN}
          cta="Back to sign in"
        />
      </div>
    );
  }

  const expiresDate = new Date(details.expires_at).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });

  return (
    <div className="space-y-8">
      <AuthPageIntro
        badge="Invitation access"
        badgeIcon={UserPlus}
        title="Create your account from an invitation"
        description="Your role, organization, and access path are already prepared. Finish the account setup and enter the workspace directly."
        statusLabel="Assigned role"
        statusValue={details.role_name}
      />

      <AuthInsightGrid items={INVITE_INSIGHTS} />

      <AuthCallout icon={Building2} title="Invitation summary">
        <div className="space-y-1">
          <p>
            <span className="font-semibold text-slate-900">{details.inviter_name}</span> invited
            you to join <span className="font-semibold text-slate-900">{details.organization_name}</span>.
          </p>
          <p>
            This invitation is tied to{' '}
            <span className="font-semibold text-slate-900">{details.email}</span> and expires on{' '}
            <span className="font-semibold text-slate-900">{expiresDate}</span>.
          </p>
        </div>
      </AuthCallout>

      {details.message ? (
        <AuthCallout icon={Quote} title="Inviter note" tone="warning">
          &ldquo;{details.message}&rdquo;
        </AuthCallout>
      ) : null}

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

      <AuthFormSurface>
        <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-5">
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
                <p className="text-sm text-destructive">{errors.first_name.message}</p>
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
                <p className="text-sm text-destructive">{errors.last_name.message}</p>
              ) : null}
            </div>
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
                aria-invalid={!!errors.password}
                className={`${AUTH_INPUT_CLASSNAME} pr-12`}
                {...register('password')}
              />
              <button
                type="button"
                className="absolute right-4 top-1/2 -translate-y-1/2 text-slate-400 transition-colors hover:text-slate-700"
                onClick={() => setShowPassword((show) => !show)}
                aria-label={showPassword ? 'Hide password' : 'Show password'}
              >
                {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
            {errors.password ? (
              <p className="text-sm text-destructive">{errors.password.message}</p>
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
                aria-invalid={!!errors.confirm_password}
                className={`${AUTH_INPUT_CLASSNAME} pr-12`}
                {...register('confirm_password')}
              />
              <button
                type="button"
                className="absolute right-4 top-1/2 -translate-y-1/2 text-slate-400 transition-colors hover:text-slate-700"
                onClick={() => setShowConfirm((show) => !show)}
                aria-label={showConfirm ? 'Hide password' : 'Show password'}
              >
                {showConfirm ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
            {errors.confirm_password ? (
              <p className="text-sm text-destructive">{errors.confirm_password.message}</p>
            ) : null}
          </div>

          <AuthCallout icon={Mail} title="Workspace routing">
            When the invitation is accepted, the platform exchanges the response into a session and
            routes you directly to the assigned workspace.
          </AuthCallout>

          <Button
            type="submit"
            className="h-12 w-full rounded-2xl bg-[#0f5132] text-base font-semibold shadow-[0_18px_40px_rgba(15,81,50,0.22)] transition-transform hover:-translate-y-0.5 hover:bg-[#0c432b]"
            disabled={isSubmitting}
            aria-busy={isSubmitting}
          >
            {isSubmitting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Creating account...
              </>
            ) : (
              'Accept invitation and create account'
            )}
          </Button>
        </form>

        <AuthGuardGrid items={INVITE_GUARDS} />
      </AuthFormSurface>

      <AuthActionStrip
        description="Need a different link or help from the workspace administrator?"
        href={ROUTES.LOGIN}
        cta="Back to sign in"
      />
    </div>
  );
}

export default function InvitePage() {
  return (
    <Suspense
      fallback={
        <AuthLoadingState
          label="Loading invitation"
          detail="We are validating the invitation token and resolving the target workspace."
        />
      }
    >
      <InviteForm />
    </Suspense>
  );
}
