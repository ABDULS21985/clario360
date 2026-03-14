'use client';

import { Users, ShieldAlert, AlertTriangle, Clock, Zap, ShieldX } from 'lucide-react';
import type { AccessDashboard } from '@/types/cyber';

interface AccessKpiCardsProps {
  dashboard: AccessDashboard;
}

export function AccessKpiCards({ dashboard }: AccessKpiCardsProps) {
  const cards = [
    {
      label: 'Total Identities',
      value: dashboard.total_identities,
      icon: Users,
      color: 'text-blue-600',
      bg: 'bg-blue-50 dark:bg-blue-950/20',
    },
    {
      label: 'High-Risk Identities',
      value: dashboard.high_risk_identities,
      icon: ShieldAlert,
      color: dashboard.high_risk_identities > 0 ? 'text-red-600' : 'text-green-600',
      bg: dashboard.high_risk_identities > 0 ? 'bg-red-50 dark:bg-red-950/20' : 'bg-green-50 dark:bg-green-950/20',
    },
    {
      label: 'Overprivileged',
      value: dashboard.overprivileged_mappings,
      icon: AlertTriangle,
      color: dashboard.overprivileged_mappings > 0 ? 'text-orange-600' : 'text-green-600',
      bg: dashboard.overprivileged_mappings > 0 ? 'bg-orange-50 dark:bg-orange-950/20' : 'bg-green-50 dark:bg-green-950/20',
    },
    {
      label: 'Stale Permissions',
      value: dashboard.stale_permissions,
      icon: Clock,
      color: dashboard.stale_permissions > 0 ? 'text-amber-600' : 'text-green-600',
      bg: dashboard.stale_permissions > 0 ? 'bg-amber-50 dark:bg-amber-950/20' : 'bg-green-50 dark:bg-green-950/20',
    },
    {
      label: 'Avg Blast Radius',
      value: `${Math.round(dashboard.avg_blast_radius)}`,
      suffix: '/100',
      icon: Zap,
      color: dashboard.avg_blast_radius >= 75 ? 'text-red-600' : dashboard.avg_blast_radius >= 50 ? 'text-amber-600' : 'text-green-600',
      bg: dashboard.avg_blast_radius >= 75 ? 'bg-red-50 dark:bg-red-950/20' : dashboard.avg_blast_radius >= 50 ? 'bg-amber-50 dark:bg-amber-950/20' : 'bg-green-50 dark:bg-green-950/20',
    },
    {
      label: 'Policy Violations',
      value: dashboard.policy_violations,
      icon: ShieldX,
      color: dashboard.policy_violations > 0 ? 'text-red-600' : 'text-green-600',
      bg: dashboard.policy_violations > 0 ? 'bg-red-50 dark:bg-red-950/20' : 'bg-green-50 dark:bg-green-950/20',
    },
  ];

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
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
