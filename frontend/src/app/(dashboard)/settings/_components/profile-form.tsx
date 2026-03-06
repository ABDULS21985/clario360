'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { toast } from 'sonner';
import { Lock } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Spinner } from '@/components/ui/spinner';
import { UserAvatar } from '@/components/shared/user-avatar';
import { profileFormSchema, type ProfileFormData } from '@/lib/validators/settings-validators';
import { apiPut } from '@/lib/api';
import { isApiError } from '@/types/api';
import { useAuth } from '@/hooks/use-auth';

export function ProfileForm() {
  const { user, updateProfile } = useAuth();
  const [loading, setLoading] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors, isDirty },
  } = useForm<ProfileFormData>({
    resolver: zodResolver(profileFormSchema),
    defaultValues: {
      first_name: user?.first_name ?? '',
      last_name: user?.last_name ?? '',
    },
  });

  if (!user) return null;

  const onSubmit = async (data: ProfileFormData) => {
    setLoading(true);
    try {
      await apiPut('/api/v1/users/me', data);
      await updateProfile(data);
      toast.success('Profile updated.');
    } catch (err) {
      const msg = isApiError(err) ? err.message : 'Failed to update profile.';
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Profile Information</CardTitle>
        <CardDescription>Update your display name.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="flex items-center gap-4">
            <UserAvatar user={user} size="lg" />
            <div>
              <p className="text-sm font-medium">{user.first_name} {user.last_name}</p>
              <p className="text-xs text-muted-foreground">{user.email}</p>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="first_name">First Name</Label>
              <Input id="first_name" {...register('first_name')} disabled={loading} />
              {errors.first_name && (
                <p className="text-sm text-destructive">{errors.first_name.message}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="last_name">Last Name</Label>
              <Input id="last_name" {...register('last_name')} disabled={loading} />
              {errors.last_name && (
                <p className="text-sm text-destructive">{errors.last_name.message}</p>
              )}
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="email">Email</Label>
            <div className="relative">
              <Input id="email" value={user.email} disabled className="pr-8" />
              <Lock className="absolute right-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            </div>
            <p className="text-xs text-muted-foreground">
              Email changes require a verification flow. Contact support.
            </p>
          </div>

          <div className="flex justify-end">
            <Button type="submit" size="sm" disabled={loading || !isDirty}>
              {loading ? <><Spinner className="mr-2 h-4 w-4" />Saving...</> : 'Save Changes'}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
