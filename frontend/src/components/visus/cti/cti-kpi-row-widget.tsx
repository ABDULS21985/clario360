'use client';

import { useEffect, useState } from 'react';
import type { ComponentType } from 'react';
import { Clock3, Crosshair, Radar, ShieldAlert, ShieldCheck, Zap } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import type { CTIExecutiveSnapshot } from '@/types/cti';

interface CTIKPIRowWidgetProps {
  snapshot: CTIExecutiveSnapshot | null;
  isLoading: boolean;
}

interface KPIStatCardProps {
  label: string;
  value: number;
  suffix?: string;
  icon: ComponentType<{ className?: string }>;
  tone?: string;
}

function useAnimatedNumber(value: number): number {
  const [displayValue, setDisplayValue] = useState(0);

  useEffect(() => {
    let frame = 0;
    const start = performance.now();
    const duration = 700;

    const animate = (now: number) => {
      const progress = Math.min((now - start) / duration, 1);
      setDisplayValue(value * progress);
      if (progress < 1) {
        frame = window.requestAnimationFrame(animate);
      }
    };

    frame = window.requestAnimationFrame(animate);
    return () => window.cancelAnimationFrame(frame);
  }, [value]);

  return displayValue;
}

function KPIStatCard({
  label,
  value,
  suffix,
  icon: Icon,
  tone = 'from-slate-500/10 to-slate-900/5 border-slate-500/20',
}: KPIStatCardProps) {
  const animated = useAnimatedNumber(value);
  const formatted = suffix === 'h' ? animated.toFixed(1) : Math.round(animated).toLocaleString();

  return (
    <div className={cn('rounded-xl border bg-gradient-to-br px-4 py-3', tone)}>
      <div className="flex items-center justify-between gap-3">
        <span className="text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">{label}</span>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </div>
      <p className="mt-3 text-2xl font-semibold tracking-tight text-foreground">
        {formatted}
        {suffix}
      </p>
    </div>
  );
}

export function CTIKPIRowWidget({
  snapshot,
  isLoading,
}: CTIKPIRowWidgetProps) {
  if (isLoading) {
    return (
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-6">
        {Array.from({ length: 6 }).map((_, index) => (
          <Skeleton key={index} className="h-[106px] w-full rounded-xl" />
        ))}
      </div>
    );
  }

  if (!snapshot) {
    return (
      <div className="rounded-xl border border-dashed px-4 py-8 text-center text-sm text-muted-foreground">
        CTI KPI snapshots are currently unavailable.
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-6">
      <KPIStatCard label="Events 24h" value={snapshot.total_events_24h} icon={Zap} tone="from-sky-500/10 to-sky-900/5 border-sky-500/20" />
      <KPIStatCard label="Active Campaigns" value={snapshot.active_campaigns_count} icon={Radar} tone="from-violet-500/10 to-violet-900/5 border-violet-500/20" />
      <KPIStatCard label="Total IOCs" value={snapshot.total_iocs} icon={Crosshair} tone="from-orange-500/10 to-orange-900/5 border-orange-500/20" />
      <KPIStatCard label="Brand Abuse" value={snapshot.brand_abuse_critical_count} icon={ShieldAlert} tone="from-rose-500/10 to-rose-900/5 border-rose-500/20" />
      <KPIStatCard label="MTTD" value={snapshot.mean_time_to_detect_hours ?? 0} suffix="h" icon={Clock3} tone="from-amber-500/10 to-amber-900/5 border-amber-500/20" />
      <KPIStatCard label="MTTR" value={snapshot.mean_time_to_respond_hours ?? 0} suffix="h" icon={ShieldCheck} tone="from-emerald-500/10 to-emerald-900/5 border-emerald-500/20" />
    </div>
  );
}
