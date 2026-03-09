import type { ReactNode } from 'react';
import Link from 'next/link';
import { ArrowRight, CheckCircle2, type LucideIcon } from 'lucide-react';

import { Spinner } from '@/components/ui/spinner';
import { cn } from '@/lib/utils';

export interface AuthInsightItem {
  icon: LucideIcon;
  label: string;
  value: string;
  detail: string;
}

interface AuthPageIntroProps {
  badge: string;
  badgeIcon: LucideIcon;
  title: string;
  description: string;
  statusLabel?: string;
  statusValue?: string;
}

interface AuthInsightGridProps {
  items: AuthInsightItem[];
}

interface AuthFormSurfaceProps {
  children: ReactNode;
  className?: string;
}

interface AuthGuardGridProps {
  items: readonly string[];
}

interface AuthCalloutProps {
  icon: LucideIcon;
  title?: string;
  children: ReactNode;
  tone?: 'default' | 'success' | 'warning' | 'danger';
  className?: string;
}

interface AuthActionStripProps {
  description: string;
  href: string;
  cta: string;
}

interface AuthCenteredStateProps {
  icon: LucideIcon;
  title: string;
  description: ReactNode;
  secondary?: ReactNode;
  className?: string;
  children?: ReactNode;
}

interface AuthLoadingStateProps {
  label: string;
  detail?: string;
  className?: string;
}

const CALLOUT_STYLES = {
  default: 'border-slate-200 bg-white/90 text-slate-600',
  success: 'border-emerald-200 bg-emerald-50 text-emerald-900',
  warning: 'border-amber-200 bg-amber-50 text-amber-900',
  danger: 'border-red-200 bg-red-50 text-red-900',
} as const;

export const AUTH_INPUT_CLASSNAME =
  'h-12 rounded-2xl border-slate-200 bg-white text-[15px] shadow-sm placeholder:text-slate-400 focus-visible:ring-[#0f5132]/25';

export const AUTH_SELECT_CLASSNAME =
  'flex h-12 w-full appearance-none rounded-2xl border border-slate-200 bg-white px-4 py-2 text-[15px] shadow-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#0f5132]/25 focus-visible:ring-offset-2';

function AuthInsightCard({ icon: Icon, label, value, detail }: AuthInsightItem) {
  return (
    <div className="rounded-[22px] border border-slate-200/80 bg-white/[0.85] p-4 shadow-[0_18px_40px_rgba(15,23,42,0.05)]">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-[11px] uppercase tracking-[0.24em] text-slate-400">{label}</p>
          <p className="mt-3 text-lg font-semibold tracking-tight text-slate-900">{value}</p>
        </div>
        <div className="rounded-2xl bg-[#0f5132]/10 p-2.5 text-[#0f5132]">
          <Icon className="h-5 w-5" />
        </div>
      </div>
      <p className="mt-3 text-sm leading-6 text-slate-500">{detail}</p>
    </div>
  );
}

export function AuthPageIntro({
  badge,
  badgeIcon: BadgeIcon,
  title,
  description,
  statusLabel,
  statusValue,
}: AuthPageIntroProps) {
  return (
    <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
      <div className="space-y-3">
        <div className="inline-flex items-center gap-2 rounded-full border border-[#0f5132]/15 bg-[#0f5132]/5 px-3 py-1 text-xs font-medium text-[#0f5132]">
          <BadgeIcon className="h-3.5 w-3.5" />
          {badge}
        </div>
        <div className="space-y-2">
          <h1 className="text-3xl font-semibold tracking-tight text-slate-950 sm:text-[2.2rem]">
            {title}
          </h1>
          <p className="max-w-2xl text-sm leading-7 text-slate-600 sm:text-base">
            {description}
          </p>
        </div>
      </div>

      {statusLabel && statusValue ? (
        <div className="rounded-[24px] border border-[#0f5132]/15 bg-[#0f5132]/5 px-4 py-3 shadow-sm">
          <p className="text-[11px] uppercase tracking-[0.28em] text-[#0f5132]/70">
            {statusLabel}
          </p>
          <div className="mt-2 flex items-center gap-2 text-sm font-semibold text-[#0f5132]">
            <span className="h-2.5 w-2.5 rounded-full bg-emerald-500 shadow-[0_0_0_4px_rgba(34,197,94,0.15)]" />
            {statusValue}
          </div>
        </div>
      ) : null}
    </div>
  );
}

export function AuthInsightGrid({ items }: AuthInsightGridProps) {
  return (
    <div className="grid gap-3 sm:grid-cols-3">
      {items.map((item) => (
        <AuthInsightCard key={`${item.label}-${item.value}`} {...item} />
      ))}
    </div>
  );
}

export function AuthFormSurface({ children, className }: AuthFormSurfaceProps) {
  return (
    <div
      className={cn(
        'space-y-5 rounded-[28px] border border-slate-200/80 bg-slate-50/70 p-5 shadow-[inset_0_1px_0_rgba(255,255,255,0.9)] sm:p-6',
        className,
      )}
    >
      {children}
    </div>
  );
}

export function AuthGuardGrid({ items }: AuthGuardGridProps) {
  return (
    <div className="grid gap-2 sm:grid-cols-3">
      {items.map((item) => (
        <div
          key={item}
          className="flex items-center gap-2 rounded-[18px] border border-white/80 bg-white/80 px-3 py-3 text-xs leading-5 text-slate-500"
        >
          <CheckCircle2 className="h-4 w-4 shrink-0 text-[#0f5132]" />
          <span>{item}</span>
        </div>
      ))}
    </div>
  );
}

export function AuthCallout({
  icon: Icon,
  title,
  children,
  tone = 'default',
  className,
}: AuthCalloutProps) {
  return (
    <div
      className={cn(
        'rounded-[24px] border px-4 py-4 text-sm',
        CALLOUT_STYLES[tone],
        className,
      )}
    >
      <div className="flex items-start gap-3">
        <div className="rounded-2xl bg-white/70 p-2 text-inherit shadow-sm">
          <Icon className="h-4 w-4" />
        </div>
        <div className="min-w-0">
          {title ? <p className="font-semibold">{title}</p> : null}
          <div className={cn('leading-6', title ? 'mt-1' : '')}>{children}</div>
        </div>
      </div>
    </div>
  );
}

export function AuthActionStrip({ description, href, cta }: AuthActionStripProps) {
  return (
    <div className="flex flex-col gap-3 rounded-[24px] border border-slate-200/70 bg-slate-50/70 p-4 sm:flex-row sm:items-center sm:justify-between">
      <p className="text-sm text-slate-600">{description}</p>
      <Link
        href={href}
        className="inline-flex items-center gap-2 text-sm font-semibold text-[#0f5132] hover:underline"
      >
        {cta}
        <ArrowRight className="h-4 w-4" />
      </Link>
    </div>
  );
}

export function AuthCenteredState({
  icon: Icon,
  title,
  description,
  secondary,
  className,
  children,
}: AuthCenteredStateProps) {
  return (
    <div
      className={cn(
        'space-y-6 rounded-[28px] border border-slate-200/80 bg-slate-50/70 p-6 text-center shadow-[inset_0_1px_0_rgba(255,255,255,0.9)] sm:p-8',
        className,
      )}
    >
      <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-[#0f5132]/10 text-[#0f5132]">
        <Icon className="h-8 w-8" />
      </div>
      <div className="space-y-2">
        <h1 className="text-2xl font-semibold tracking-tight text-slate-950">{title}</h1>
        <div className="text-sm leading-7 text-slate-600">{description}</div>
        {secondary ? <div className="text-xs text-slate-500">{secondary}</div> : null}
      </div>
      {children}
    </div>
  );
}

export function AuthLoadingState({ label, detail, className }: AuthLoadingStateProps) {
  return (
    <div
      className={cn(
        'space-y-5 rounded-[28px] border border-slate-200/80 bg-slate-50/70 p-6 text-center shadow-[inset_0_1px_0_rgba(255,255,255,0.9)] sm:p-8',
        className,
      )}
    >
      <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-[#0f5132]/10">
        <Spinner className="text-[#0f5132]" />
      </div>
      <div className="space-y-2">
        <h1 className="text-2xl font-semibold tracking-tight text-slate-950">{label}</h1>
        {detail ? <p className="text-sm leading-7 text-slate-600">{detail}</p> : null}
      </div>
    </div>
  );
}
