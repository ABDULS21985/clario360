'use client';

import { useAuth } from '@/hooks/use-auth';
import { PageHeader } from '@/components/common/page-header';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { formatDate } from '@/lib/utils';

export default function SettingsPage() {
  const { user } = useAuth();

  if (!user) return null;

  const fullName = `${user.first_name} ${user.last_name}`.trim() || '—';

  return (
    <div className="space-y-6 max-w-2xl">
      <PageHeader title="Profile Settings" description="Manage your account information" />

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Account Information</CardTitle>
          <CardDescription>Your profile details</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4 text-sm">
          <div className="grid grid-cols-2 gap-x-4 gap-y-3">
            <div>
              <p className="text-xs text-muted-foreground">Full name</p>
              <p className="font-medium">{fullName}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Email</p>
              <p className="font-medium">{user.email}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Status</p>
              <Badge variant={user.status === 'active' ? 'success' : 'warning'}>
                {user.status}
              </Badge>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">MFA</p>
              <Badge variant={user.mfa_enabled ? 'success' : 'outline'}>
                {user.mfa_enabled ? 'Enabled' : 'Disabled'}
              </Badge>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Member since</p>
              <p>{formatDate(user.created_at)}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Last login</p>
              <p>{user.last_login_at ? formatDate(user.last_login_at) : 'Never'}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Roles</CardTitle>
          <CardDescription>Your assigned roles and permissions</CardDescription>
        </CardHeader>
        <CardContent>
          {user.roles.length === 0 ? (
            <p className="text-sm text-muted-foreground">No roles assigned.</p>
          ) : (
            <div className="space-y-2">
              {user.roles.map((role) => (
                <div key={role.id} className="flex items-center justify-between rounded-md border px-3 py-2">
                  <span className="text-sm font-medium">{role.name}</span>
                  <Badge variant={role.is_system ? 'default' : 'outline'} className="text-xs">
                    {role.is_system ? 'System' : 'Custom'}
                  </Badge>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
