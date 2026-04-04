'use client';

import React from 'react';
import { useAuth } from '@/hooks/use-auth';

interface PermissionGateProps {
  permission?: string;
  permissions?: string[];
  mode?: 'all' | 'any';
  requireRole?: string;
  fallback?: React.ReactNode;
  children: React.ReactNode;
}

export function PermissionGate({
  permission,
  permissions,
  mode = 'all',
  requireRole,
  fallback = null,
  children,
}: PermissionGateProps) {
  const { isAuthenticated, hasPermission, hasAnyPermission, hasAllPermissions, user } =
    useAuth();

  if (!isAuthenticated || !user) {
    return <>{fallback}</>;
  }

  // Role-based check
  if (requireRole) {
    const hasRole = (user.roles ?? []).some((r) => r.slug === requireRole);
    return hasRole ? <>{children}</> : <>{fallback}</>;
  }

  // Single permission
  if (permission) {
    return hasPermission(permission) ? <>{children}</> : <>{fallback}</>;
  }

  // Multiple permissions
  if (permissions && permissions.length > 0) {
    const granted =
      mode === 'any'
        ? hasAnyPermission(permissions)
        : hasAllPermissions(permissions);
    return granted ? <>{children}</> : <>{fallback}</>;
  }

  // No constraint specified — render children
  return <>{children}</>;
}
