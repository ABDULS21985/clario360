'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/hooks/use-auth';
import { LoadingSkeleton } from './loading-skeleton';

interface PermissionRedirectProps {
  permission: string;
  children: React.ReactNode;
}

export function PermissionRedirect({ permission, children }: PermissionRedirectProps) {
  const { hasPermission, isHydrated } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!isHydrated) return;
    if (permission !== '*:read' && !hasPermission(permission)) {
      router.replace('/dashboard');
    }
  }, [isHydrated, hasPermission, permission, router]);

  if (!isHydrated) {
    return <LoadingSkeleton variant="card" count={3} />;
  }

  if (permission !== '*:read' && !hasPermission(permission)) {
    return null;
  }

  return <>{children}</>;
}
