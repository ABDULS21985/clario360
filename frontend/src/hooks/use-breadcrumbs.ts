'use client';

import { usePathname } from 'next/navigation';
import { useMemo } from 'react';
import { isUUID, segmentToLabel } from '@/config/breadcrumb-map';

export interface Breadcrumb {
  label: string;
  href: string;
  isLast: boolean;
  isDynamic: boolean;
}

export function useBreadcrumbs(): Breadcrumb[] {
  const pathname = usePathname();
  const currentPath = pathname ?? '/dashboard';

  return useMemo(() => {
    const segments = currentPath.split('/').filter(Boolean);

    if (segments.length === 0) {
      return [{ label: 'Home', href: '/dashboard', isLast: true, isDynamic: false }];
    }

    const crumbs: Breadcrumb[] = [
      { label: 'Home', href: '/dashboard', isLast: false, isDynamic: false },
    ];

    let accumulatedPath = '';
    for (let i = 0; i < segments.length; i++) {
      const segment = segments[i];
      accumulatedPath += '/' + segment;

      // Skip adding a "Dashboard" crumb when path starts with /dashboard
      // — the "Home" crumb already points there.
      if (accumulatedPath === '/dashboard') continue;

      const isLast = i === segments.length - 1;
      const isDynamic = isUUID(segment);

      crumbs.push({
        label: isDynamic ? 'Details' : segmentToLabel(segment),
        href: accumulatedPath,
        isLast,
        isDynamic,
      });
    }

    // Ensure the last crumb is marked correctly after filtering.
    if (crumbs.length > 0) {
      crumbs[crumbs.length - 1].isLast = true;
    }

    return crumbs;
  }, [currentPath]);
}
