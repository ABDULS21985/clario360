import * as React from 'react';
import { cn } from '@/lib/utils';

interface PageHeaderProps {
  title: React.ReactNode;
  description?: React.ReactNode;
  actions?: React.ReactNode;
  className?: string;
}

export function PageHeader({ title, description, actions, className }: PageHeaderProps) {
  return (
    <div
      className={cn(
        'relative overflow-hidden rounded-[30px] border border-[color:var(--panel-border)] bg-[linear-gradient(135deg,rgba(255,255,255,0.94),rgba(246,250,247,0.82))] p-6 shadow-[var(--card-shadow)] backdrop-blur-xl sm:p-7',
        className,
      )}
    >
      <div className="pointer-events-none absolute inset-y-0 right-0 hidden w-1/3 bg-[radial-gradient(circle_at_center,rgba(16,185,129,0.12),transparent_55%)] lg:block" />
      <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-emerald-400/70 to-transparent" />
      <div className="relative flex flex-col gap-6 lg:flex-row lg:items-start lg:justify-between">
        <div className="space-y-4">
          <span className="inline-flex items-center gap-2 rounded-full border border-emerald-200 bg-emerald-50/90 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.24em] text-emerald-900">
            <span className="h-2 w-2 rounded-full bg-emerald-500" aria-hidden="true" />
            Enterprise Workspace
          </span>
          <div className="space-y-2">
            <h1 className="text-3xl font-semibold tracking-[-0.05em] text-slate-950 sm:text-[2.15rem]">
              {title}
            </h1>
            {description && (
              <p className="max-w-3xl text-sm leading-7 text-slate-600 sm:text-base">
                {description}
              </p>
            )}
          </div>
        </div>
        {actions && (
          <div className="flex shrink-0 flex-wrap items-center gap-2 lg:justify-end">
            {actions}
          </div>
        )}
      </div>
    </div>
  );
}
