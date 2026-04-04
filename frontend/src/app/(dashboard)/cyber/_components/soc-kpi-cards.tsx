'use client';

import { useRouter } from 'next/navigation';
import { AlertTriangle, ShieldAlert, Activity, Clock } from 'lucide-react';
import { KpiCard } from '@/components/shared/kpi-card';
import type { KPICards } from '@/types/cyber';

interface SocKpiCardsProps {
  kpis: KPICards;
  loading?: boolean;
}

function formatMTTR(hours: number): string {
  if (hours < 1) return `${Math.round(hours * 60)}m`;
  const h = Math.floor(hours);
  const m = Math.round((hours - h) * 60);
  return m > 0 ? `${h}h ${m}m` : `${h}h`;
}

function mttrColor(hours: number): string {
  if (hours < 4) return 'text-green-600';
  if (hours <= 8) return 'text-yellow-600';
  return 'text-red-600';
}

function riskGradeColor(grade: string): string {
  switch (grade) {
    case 'A': return 'text-green-600';
    case 'B': return 'text-blue-600';
    case 'C': return 'text-yellow-600';
    case 'D': return 'text-orange-600';
    default: return 'text-red-600';
  }
}

export function SocKpiCards({ kpis, loading }: SocKpiCardsProps) {
  const router = useRouter();

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <div
        className="cursor-pointer"
        onClick={() => router.push('/cyber/alerts?status=new,acknowledged,investigating')}
      >
        <KpiCard
          title="Open Alerts"
          value={kpis.open_alerts}
          change={kpis.alerts_delta !== 0 ? kpis.alerts_delta : undefined}
          changeLabel="vs yesterday"
          icon={AlertTriangle}
          iconColor={kpis.open_alerts > 0 ? 'text-orange-500' : 'text-muted-foreground'}
          loading={loading}
        />
      </div>

      <div
        className="cursor-pointer"
        onClick={() => router.push('/cyber/alerts?severity=critical')}
      >
        <KpiCard
          title="Critical Alerts"
          value={kpis.critical_alerts}
          icon={ShieldAlert}
          iconColor={kpis.critical_alerts > 0 ? 'text-red-600' : 'text-muted-foreground'}
          loading={loading}
        />
      </div>

      <div
        className="cursor-pointer"
        onClick={() => router.push('/cyber/vciso')}
      >
        <KpiCard
          title="Risk Score"
          value={
            kpis.risk_grade
              ? `${Math.round(kpis.risk_score)} (${kpis.risk_grade})`
              : Math.round(kpis.risk_score)
          }
          icon={Activity}
          iconColor={riskGradeColor(kpis.risk_grade)}
          loading={loading}
        />
      </div>

      <div className="cursor-pointer" onClick={() => undefined}>
        <KpiCard
          title="MTTR"
          value={loading ? '—' : formatMTTR(kpis.mttr_hours)}
          icon={Clock}
          iconColor={mttrColor(kpis.mttr_hours)}
          description="Mean time to respond"
          loading={loading}
        />
      </div>
    </div>
  );
}
