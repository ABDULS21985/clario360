'use client';

import { format } from 'date-fns';
import { useAuth } from '@/hooks/use-auth';

export function WelcomeHeader() {
  const { user, tenant } = useAuth();
  const firstName = user?.first_name || user?.email?.split('@')[0] || 'there';
  const today = format(new Date(), 'MMMM d, yyyy');

  return (
    <div className="flex items-start justify-between">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">
          Welcome back, {firstName}!
        </h1>
        {tenant && (
          <p className="text-sm text-muted-foreground mt-0.5">{tenant.name}</p>
        )}
      </div>
      <p className="text-sm text-muted-foreground shrink-0">{today}</p>
    </div>
  );
}
