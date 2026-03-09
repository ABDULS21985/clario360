'use client';

import Link from 'next/link';
import { ChevronRight } from 'lucide-react';
import { useBreadcrumbs } from '@/hooks/use-breadcrumbs';

export function Breadcrumbs() {
  const crumbs = useBreadcrumbs();

  if (crumbs.length <= 1) return null;

  return (
    <nav aria-label="Breadcrumb" className="min-w-0">
      <ol role="list" className="flex min-w-0 flex-wrap items-center gap-1.5 text-sm">
        {crumbs.map((crumb, idx) => (
          <li key={crumb.href} className="flex items-center gap-1">
            {idx > 0 && (
              <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/70" aria-hidden="true" />
            )}
            {crumb.isLast ? (
              <span
                aria-current="page"
                className="inline-flex max-w-[220px] items-center truncate rounded-full border border-emerald-200/80 bg-emerald-50/90 px-3 py-1 text-xs font-semibold tracking-[0.02em] text-emerald-950 shadow-sm"
              >
                {crumb.label}
              </span>
            ) : (
              <Link
                href={crumb.href}
                className="inline-flex max-w-[180px] items-center truncate rounded-full border border-transparent bg-white/70 px-3 py-1 text-xs font-medium text-muted-foreground shadow-sm transition-all hover:border-border/70 hover:bg-white hover:text-foreground"
              >
                {crumb.label}
              </Link>
            )}
          </li>
        ))}
      </ol>
    </nav>
  );
}
