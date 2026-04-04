'use client';

import { cn } from '@/lib/utils';

interface SidebarSectionProps {
  label: string;
  collapsed: boolean;
  children: React.ReactNode;
}

export function SidebarSection({ label, collapsed, children }: SidebarSectionProps) {
  return (
    <div role="group" aria-label={label || 'Navigation'} className="mb-4">
      {label && !collapsed && (
        <p className="mb-2 px-3 text-[11px] font-semibold uppercase tracking-[0.28em] text-slate-400">
          {label}
        </p>
      )}
      {label && collapsed && <div className="mx-3 mb-2 h-px bg-white/10" />}
      <div className={cn('space-y-1')}>{children}</div>
    </div>
  );
}
