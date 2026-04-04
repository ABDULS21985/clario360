'use client';

import { Database, ShieldAlert, Globe, Lock, Wifi } from 'lucide-react';
import type { DSPMDashboard } from '@/types/cyber';

interface DSPMKpiCardsProps {
  dashboard: DSPMDashboard;
}

export function DSPMKpiCards({ dashboard }: DSPMKpiCardsProps) {
  const cards = [
    {
      label: 'Data Assets',
      value: dashboard.total_data_assets,
      icon: Database,
      color: 'text-blue-600',
      bg: 'bg-blue-50 dark:bg-blue-950/20',
    },
    {
      label: 'Unencrypted',
      value: dashboard.unencrypted_count,
      icon: Lock,
      color: dashboard.unencrypted_count > 0 ? 'text-red-600' : 'text-green-600',
      bg: dashboard.unencrypted_count > 0 ? 'bg-red-50 dark:bg-red-950/20' : 'bg-green-50 dark:bg-green-950/20',
    },
    {
      label: 'No Access Control',
      value: dashboard.no_access_control_count,
      icon: ShieldAlert,
      color: dashboard.no_access_control_count > 0 ? 'text-orange-600' : 'text-green-600',
      bg: dashboard.no_access_control_count > 0 ? 'bg-orange-50 dark:bg-orange-950/20' : 'bg-green-50 dark:bg-green-950/20',
    },
    {
      label: 'Internet Facing',
      value: dashboard.internet_facing_count,
      icon: Globe,
      color: dashboard.internet_facing_count > 0 ? 'text-amber-600' : 'text-green-600',
      bg: dashboard.internet_facing_count > 0 ? 'bg-amber-50 dark:bg-amber-950/20' : 'bg-green-50 dark:bg-green-950/20',
    },
    {
      label: 'Posture Score',
      value: `${dashboard.avg_posture_score.toFixed(0)}`,
      suffix: '/100',
      icon: Wifi,
      color: dashboard.avg_posture_score >= 80 ? 'text-green-600' : dashboard.avg_posture_score >= 60 ? 'text-amber-600' : 'text-red-600',
      bg: dashboard.avg_posture_score >= 80 ? 'bg-green-50 dark:bg-green-950/20' : dashboard.avg_posture_score >= 60 ? 'bg-amber-50 dark:bg-amber-950/20' : 'bg-red-50 dark:bg-red-950/20',
    },
    {
      label: 'Risk Score',
      value: `${dashboard.avg_risk_score.toFixed(0)}`,
      suffix: '/100',
      icon: ShieldAlert,
      color: dashboard.avg_risk_score <= 30 ? 'text-green-600' : dashboard.avg_risk_score <= 60 ? 'text-amber-600' : 'text-red-600',
      bg: dashboard.avg_risk_score <= 30 ? 'bg-green-50 dark:bg-green-950/20' : dashboard.avg_risk_score <= 60 ? 'bg-amber-50 dark:bg-amber-950/20' : 'bg-red-50 dark:bg-red-950/20',
    },
  ];

  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-6">
      {cards.map(({ label, value, suffix, icon: Icon, color, bg }) => (
        <div key={label} className={`flex flex-col items-center rounded-xl border p-4 text-center ${bg}`}>
          <div className={`mb-2 flex h-10 w-10 items-center justify-center rounded-full bg-background border ${color}`}>
            <Icon className="h-5 w-5" />
          </div>
          <p className={`text-2xl font-bold tabular-nums ${color}`}>
            {value}
            {suffix && <span className="text-sm font-normal text-muted-foreground">{suffix}</span>}
          </p>
          <p className="mt-0.5 text-xs text-muted-foreground">{label}</p>
        </div>
      ))}
    </div>
  );
}
