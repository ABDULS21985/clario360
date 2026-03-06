'use client';

import { cn } from '@/lib/utils';

interface SidebarSectionProps {
  label: string;
  collapsed: boolean;
  children: React.ReactNode;
}

export function SidebarSection({ label, collapsed, children }: SidebarSectionProps) {
  return (
    <div role="group" aria-label={label || 'Navigation'} className="mb-2">
      {label && !collapsed && (
        <p className="mb-1 px-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground/60">
          {label}
        </p>
      )}
      {label && collapsed && <div className="mx-3 mb-1 h-px bg-border" />}
      <div className={cn('space-y-0.5')}>{children}</div>
    </div>
  );
}
