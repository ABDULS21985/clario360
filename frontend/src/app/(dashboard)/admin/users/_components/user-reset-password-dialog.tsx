'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { toast } from 'sonner';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Checkbox } from '@/components/ui/checkbox';
import { PasswordStrengthMeter } from '@/components/auth/password-strength-meter';
import { apiPost, apiPut } from '@/lib/api';
import { isApiError } from '@/types/api';
import type { User } from '@/types/models';

const tempPasswordSchema = z.object({
  temp_password: z
    .string()
    .min(12, 'At least 12 characters')
    .regex(/[A-Z]/, 'Requires uppercase')
    .regex(/[a-z]/, 'Requires lowercase')
    .regex(/[0-9]/, 'Requires number')
    .regex(/[^a-zA-Z0-9]/, 'Requires special character'),
  require_change: z.boolean(),
});

type TempPasswordData = z.infer<typeof tempPasswordSchema>;

interface UserResetPasswordDialogProps {
  user: User;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function UserResetPasswordDialog({
  user,
  open,
  onOpenChange,
  onSuccess,
}: UserResetPasswordDialogProps) {
  const [mode, setMode] = useState<'email' | 'temp'>('email');
  const [loading, setLoading] = useState(false);

  const {
    register,
    handleSubmit,
    watch,
    reset,
    formState: { errors },
  } = useForm<TempPasswordData>({
    resolver: zodResolver(tempPasswordSchema),
    defaultValues: { require_change: true },
  });

  const tempPassword = watch('temp_password') ?? '';

  const handleClose = (open: boolean) => {
    if (!open) {
      reset();
      setMode('email');
    }
    onOpenChange(open);
  };

  const handleEmailReset = async () => {
    setLoading(true);
    try {
      await apiPost('/api/v1/auth/forgot-password', { email: user.email });
      toast.success(`Password reset email sent to ${user.email}.`);
      handleClose(false);
      onSuccess();
    } catch (err) {
      const msg = isApiError(err) ? err.message : 'Failed to send reset email.';
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  const handleTempPassword = async (data: TempPasswordData) => {
    setLoading(true);
    try {
      await apiPut(`/api/v1/users/${user.id}/password`, {
        temp_password: data.temp_password,
        require_change: data.require_change,
      });
      toast.success(`Temporary password set for ${user.first_name} ${user.last_name}.`);
      handleClose(false);
      onSuccess();
    } catch (err) {
      const msg = isApiError(err) ? err.message : 'Failed to set temporary password.';
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  const name = `${user.first_name} ${user.last_name}`.trim();

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Reset Password for {name}</DialogTitle>
          <DialogDescription>
            Choose how to reset the password for {user.email}.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <RadioGroup
            value={mode}
            onValueChange={(v) => setMode(v as 'email' | 'temp')}
            className="space-y-3"
          >
            <div className="flex items-start gap-3 rounded-md border p-3 cursor-pointer">
              <RadioGroupItem value="email" id="mode-email" className="mt-0.5" />
              <Label htmlFor="mode-email" className="cursor-pointer space-y-1">
                <p className="font-medium">Send password reset email</p>
                <p className="text-sm text-muted-foreground">
                  {`${name}'s current password will remain active until they use the reset link.`}
                </p>
              </Label>
            </div>
            <div className="flex items-start gap-3 rounded-md border p-3 cursor-pointer">
              <RadioGroupItem value="temp" id="mode-temp" className="mt-0.5" />
              <Label htmlFor="mode-temp" className="cursor-pointer space-y-1">
                <p className="font-medium">Set temporary password</p>
                <p className="text-sm text-muted-foreground">
                  You set a password that the user must change on next login.
                </p>
              </Label>
            </div>
          </RadioGroup>

          {mode === 'temp' && (
            <form id="temp-pw-form" onSubmit={handleSubmit(handleTempPassword)} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="temp_password">Temporary Password</Label>
                <Input
                  id="temp_password"
                  type="password"
                  {...register('temp_password')}
                  disabled={loading}
                />
                {errors.temp_password && (
                  <p className="text-sm text-destructive">{errors.temp_password.message}</p>
                )}
                <PasswordStrengthMeter password={tempPassword} />
              </div>
              <div className="flex items-center gap-2">
                <Checkbox
                  id="require_change"
                  {...register('require_change')}
                  defaultChecked
                />
                <Label htmlFor="require_change" className="cursor-pointer text-sm">
                  Require password change on next login
                </Label>
              </div>
            </form>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => handleClose(false)} disabled={loading}>
            Cancel
          </Button>
          {mode === 'email' ? (
            <Button onClick={handleEmailReset} disabled={loading}>
              {loading ? 'Sending...' : 'Send Reset Email'}
            </Button>
          ) : (
            <Button type="submit" form="temp-pw-form" disabled={loading}>
              {loading ? 'Setting...' : 'Set Password'}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
