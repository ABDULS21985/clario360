'use client';

import Link from 'next/link';
import { Calendar, CheckSquare, Shield, Users } from 'lucide-react';
import { GaugeChart } from '@/components/shared/charts/gauge-chart';
import { KpiCard } from '@/components/shared/kpi-card';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import type { ActaKPIs } from '@/types/suites';

interface ActaDashboardKpisProps {
  kpis: ActaKPIs;
}

export function ActaDashboardKpis({ kpis }: ActaDashboardKpisProps) {
  return (
    <div className="grid grid-cols-1 gap-4 xl:grid-cols-4">
      <Link href="/acta/committees" className="block">
        <KpiCard
          title="Active Committees"
          value={kpis.active_committees}
          icon={Users}
          iconColor="text-sky-600"
          description="Board and governance committees"
          className="h-full transition hover:border-sky-300 hover:shadow-sm"
        />
      </Link>

      <Link href="/acta/meetings" className="block">
        <KpiCard
          title="Upcoming Meetings"
          value={kpis.upcoming_meetings_30d}
          icon={Calendar}
          iconColor="text-indigo-600"
          description="Scheduled within the next 30 days"
          className="h-full transition hover:border-indigo-300 hover:shadow-sm"
        />
      </Link>

      <Link href="/acta/action-items" className="block">
        <KpiCard
          title="Open Action Items"
          value={kpis.open_action_items}
          icon={CheckSquare}
          iconColor="text-amber-600"
          description={
            kpis.overdue_action_items > 0
              ? `${kpis.overdue_action_items} overdue`
              : 'No overdue items'
          }
          className="h-full transition hover:border-amber-300 hover:shadow-sm"
        />
      </Link>

      <Card className="h-full">
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            Compliance Score
          </CardTitle>
          <Shield className="h-5 w-5 text-emerald-600" />
        </CardHeader>
        <CardContent className="flex items-center justify-between gap-4">
          <div>
            <p className="text-2xl font-bold">
              {Math.round(kpis.compliance_score)}%
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              Minutes pending approval: {kpis.minutes_pending_approval}
            </p>
            <p className="text-xs text-muted-foreground">
              Avg attendance: {Math.round(kpis.attendance_rate_avg)}%
            </p>
            <Button variant="link" size="sm" className="mt-2 h-auto px-0" asChild>
              <Link href="/acta/compliance">Open compliance</Link>
            </Button>
          </div>
          <GaugeChart
            value={kpis.compliance_score}
            max={100}
            size={124}
            label="Governance"
            className="h-[82px] w-[124px]"
          />
        </CardContent>
      </Card>
    </div>
  );
}
