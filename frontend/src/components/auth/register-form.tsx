'use client';

import React, { useState, useCallback, useRef } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Eye, EyeOff, CheckCircle2, XCircle, Loader2, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Spinner } from '@/components/ui/spinner';
import { PasswordStrengthMeter } from './password-strength-meter';
import { registerSchema, type RegisterFormData } from '@/lib/validators';
import { apiPost, apiGet } from '@/lib/api';
import { isApiError } from '@/types/api';
import { API_ENDPOINTS } from '@/lib/constants';

type TenantMode = 'create' | 'join';
type EmailStatus = 'idle' | 'checking' | 'available' | 'taken';

export function RegisterForm() {
  const router = useRouter();
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [tenantMode, setTenantMode] = useState<TenantMode>('create');
  const [emailStatus, setEmailStatus] = useState<EmailStatus>('idle');
  const [apiError, setApiError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    setError,
    formState: { errors },
  } = useForm<RegisterFormData>({
    resolver: zodResolver(registerSchema),
    mode: 'onBlur',
  });

  const password = watch('password', '');
  const email = watch('email', '');

  const checkEmailAvailability = useCallback((emailValue: string) => {
    if (!emailValue || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(emailValue)) {
      setEmailStatus('idle');
      return;
    }
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(async () => {
      setEmailStatus('checking');
      try {
        await apiGet(`${API_ENDPOINTS.AUTH_CHECK_EMAIL}?email=${encodeURIComponent(emailValue)}`);
        setEmailStatus('available');
      } catch (err) {
        if (isApiError(err) && err.status === 409) {
          setEmailStatus('taken');
        } else {
          setEmailStatus('idle');
        }
      }
    }, 500);
  }, []);

  const onSubmit = async (data: RegisterFormData) => {
    setApiError(null);
    setIsSubmitting(true);
    try {
      const payload = {
        email: data.email,
        password: data.password,
        first_name: data.first_name,
        last_name: data.last_name,
        ...(tenantMode === 'create' && data.tenant_name
          ? { tenant_name: data.tenant_name }
          : {}),
        ...(tenantMode === 'join' && data.invite_code
          ? { invite_code: data.invite_code }
          : {}),
      };
      await apiPost(API_ENDPOINTS.AUTH_REGISTER, payload);
      router.push('/login?registered=true');
    } catch (err) {
      if (isApiError(err)) {
        if (err.status === 409 || err.code === 'EMAIL_TAKEN') {
          setError('email', { message: 'This email address is already registered.' });
          setEmailStatus('taken');
        } else if (err.details) {
          // Field-level errors from backend
          Object.entries(err.details).forEach(([field, messages]) => {
            setError(field as keyof RegisterFormData, {
              message: messages[0] ?? 'Invalid value',
            });
          });
        } else {
          setApiError(err.message ?? 'Registration failed. Please try again.');
        }
      } else {
        setApiError('An unexpected error occurred. Please try again.');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Create your account</h1>
        <p className="text-sm text-muted-foreground">
          Get started with Clario 360
        </p>
      </div>

      {apiError && (
        <Alert variant="destructive" role="alert" aria-live="assertive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{apiError}</AlertDescription>
        </Alert>
      )}

      <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-4">
        {/* Name row */}
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="first_name">First name</Label>
            <Input
              id="first_name"
              type="text"
              autoComplete="given-name"
              aria-describedby={errors.first_name ? 'first-name-error' : undefined}
              aria-invalid={!!errors.first_name}
              {...register('first_name')}
            />
            {errors.first_name && (
              <p id="first-name-error" className="text-sm text-destructive" role="alert">
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
              aria-describedby={errors.last_name ? 'last-name-error' : undefined}
              aria-invalid={!!errors.last_name}
              {...register('last_name')}
            />
            {errors.last_name && (
              <p id="last-name-error" className="text-sm text-destructive" role="alert">
                {errors.last_name.message}
              </p>
            )}
          </div>
        </div>

        {/* Email with availability check */}
        <div className="space-y-2">
          <Label htmlFor="email">Email</Label>
          <div className="relative">
            <Input
              id="email"
              type="email"
              autoComplete="email"
              aria-describedby={errors.email ? 'email-error' : undefined}
              aria-invalid={!!errors.email}
              className="pr-10"
              {...register('email', {
                onChange: (e) => checkEmailAvailability(e.target.value),
              })}
            />
            <span className="absolute right-3 top-1/2 -translate-y-1/2">
              {emailStatus === 'checking' && <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />}
              {emailStatus === 'available' && <CheckCircle2 className="h-4 w-4 text-green-500" />}
              {emailStatus === 'taken' && <XCircle className="h-4 w-4 text-destructive" />}
            </span>
          </div>
          {errors.email && (
            <p id="email-error" className="text-sm text-destructive" role="alert">
              {errors.email.message}
            </p>
          )}
          {emailStatus === 'taken' && !errors.email && (
            <p className="text-sm text-destructive">This email is already registered.</p>
          )}
        </div>

        {/* Password */}
        <div className="space-y-2">
          <Label htmlFor="password">Password</Label>
          <div className="relative">
            <Input
              id="password"
              type={showPassword ? 'text' : 'password'}
              autoComplete="new-password"
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
          <PasswordStrengthMeter password={password} />
        </div>

        {/* Confirm password */}
        <div className="space-y-2">
          <Label htmlFor="confirm_password">Confirm password</Label>
          <div className="relative">
            <Input
              id="confirm_password"
              type={showConfirm ? 'text' : 'password'}
              autoComplete="new-password"
              aria-describedby={errors.confirm_password ? 'confirm-error' : undefined}
              aria-invalid={!!errors.confirm_password}
              className="pr-10"
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
            <p id="confirm-error" className="text-sm text-destructive" role="alert">
              {errors.confirm_password.message}
            </p>
          )}
        </div>

        {/* Tenant mode toggle */}
        <fieldset className="space-y-3">
          <legend className="text-sm font-medium">Organization</legend>
          <div className="flex gap-4">
            <label className="flex cursor-pointer items-center gap-2 text-sm">
              <input
                type="radio"
                name="tenant_mode"
                value="create"
                checked={tenantMode === 'create'}
                onChange={() => {
                  setTenantMode('create');
                  setValue('invite_code', '');
                }}
                className="accent-primary"
              />
              Create a new organization
            </label>
            <label className="flex cursor-pointer items-center gap-2 text-sm">
              <input
                type="radio"
                name="tenant_mode"
                value="join"
                checked={tenantMode === 'join'}
                onChange={() => {
                  setTenantMode('join');
                  setValue('tenant_name', '');
                }}
                className="accent-primary"
              />
              Join an existing organization
            </label>
          </div>

          {tenantMode === 'create' ? (
            <div className="space-y-2">
              <Label htmlFor="tenant_name">Organization name</Label>
              <Input
                id="tenant_name"
                type="text"
                placeholder="Acme Corp"
                aria-describedby={errors.tenant_name ? 'tenant-error' : undefined}
                aria-invalid={!!errors.tenant_name}
                {...register('tenant_name')}
              />
              {errors.tenant_name && (
                <p id="tenant-error" className="text-sm text-destructive" role="alert">
                  {errors.tenant_name.message}
                </p>
              )}
            </div>
          ) : (
            <div className="space-y-2">
              <Label htmlFor="invite_code">Invite code</Label>
              <Input
                id="invite_code"
                type="text"
                placeholder="Enter your invite code"
                aria-describedby={errors.invite_code ? 'invite-error' : undefined}
                aria-invalid={!!errors.invite_code}
                {...register('invite_code')}
              />
              {errors.invite_code && (
                <p id="invite-error" className="text-sm text-destructive" role="alert">
                  {errors.invite_code.message}
                </p>
              )}
            </div>
          )}
        </fieldset>

        <Button
          type="submit"
          className="w-full"
          disabled={isSubmitting}
          aria-busy={isSubmitting}
        >
          {isSubmitting ? (
            <>
              <Spinner size="sm" className="mr-2" />
              Creating account&hellip;
            </>
          ) : (
            'Create account'
          )}
        </Button>
      </form>

      <p className="text-center text-sm text-muted-foreground">
        Already have an account?{' '}
        <Link href="/login" className="text-primary font-medium hover:underline">
          Sign in
        </Link>
      </p>
    </div>
  );
}
