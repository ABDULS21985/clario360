'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { toast } from 'sonner';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Spinner } from '@/components/ui/spinner';
import { PasswordStrengthMeter } from '@/components/auth/password-strength-meter';
import { changePasswordSchema, type ChangePasswordFormData } from '@/lib/validators/settings-validators';
import { apiPut } from '@/lib/api';
import { isApiError } from '@/types/api';

export function PasswordChangeForm() {
  const [loading, setLoading] = useState(false);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    reset,
    watch,
    setError,
    formState: { errors },
  } = useForm<ChangePasswordFormData>({
    resolver: zodResolver(changePasswordSchema),
  });

  const newPassword = watch('new_password') ?? '';

  const onSubmit = async (data: ChangePasswordFormData) => {
    setLoading(true);
    setSuccessMsg(null);
    try {
      await apiPut('/api/v1/users/me/password', {
        current_password: data.current_password,
        new_password: data.new_password,
      });
      toast.success('Password changed successfully.');
      setSuccessMsg('All other sessions have been logged out for security.');
      reset();
    } catch (err) {
      if (isApiError(err)) {
        if (err.status === 401) {
          setError('current_password', { message: 'Incorrect current password.' });
        } else if (err.details) {
          Object.entries(err.details).forEach(([field, messages]) => {
            setError(field as keyof ChangePasswordFormData, {
              message: (messages as string[])[0] ?? 'Invalid value',
            });
          });
        } else {
          toast.error(err.message);
        }
      } else {
        toast.error('Failed to change password.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Change Password</CardTitle>
        <CardDescription>
          Update your password. You will be logged out of all other sessions.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {successMsg && (
          <Alert className="mb-4">
            <AlertDescription className="text-sm">{successMsg}</AlertDescription>
          </Alert>
        )}
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="current_password">Current Password</Label>
            <Input
              id="current_password"
              type="password"
              {...register('current_password')}
              disabled={loading}
            />
            {errors.current_password && (
              <p className="text-sm text-destructive">{errors.current_password.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="new_password">New Password</Label>
            <Input
              id="new_password"
              type="password"
              {...register('new_password')}
              disabled={loading}
            />
            {errors.new_password && (
              <p className="text-sm text-destructive">{errors.new_password.message}</p>
            )}
            <PasswordStrengthMeter password={newPassword} />
          </div>

          <div className="space-y-2">
            <Label htmlFor="confirm_password">Confirm New Password</Label>
            <Input
              id="confirm_password"
              type="password"
              {...register('confirm_password')}
              disabled={loading}
            />
            {errors.confirm_password && (
              <p className="text-sm text-destructive">{errors.confirm_password.message}</p>
            )}
          </div>

          <div className="flex justify-end">
            <Button type="submit" size="sm" disabled={loading}>
              {loading ? <><Spinner className="mr-2 h-4 w-4" />Updating...</> : 'Update Password'}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
