'use client';

import { WelcomeHeader } from '@/components/dashboard/welcome-header';
import { CriticalAlertsBanner } from '@/components/dashboard/critical-alerts-banner';
import { KpiGrid } from '@/components/dashboard/kpi-grid';
import { SecondaryMetricsStrip } from '@/components/dashboard/secondary-metrics-strip';
import { RecentAlertsTable } from '@/components/dashboard/recent-alerts-table';
import { MyTasksList } from '@/components/dashboard/my-tasks-list';
import { ActivityTimeline } from '@/components/dashboard/activity-timeline';
import { useAuth } from '@/hooks/use-auth';

export default function DashboardHome() {
  const { hasPermission } = useAuth();
  const hasCyber = hasPermission('cyber:read');

  return (
    <div className="space-y-6">
      {/* Critical alerts banner — only shows when there are critical items */}
      {hasCyber && <CriticalAlertsBanner />}

      {/* Welcome header with personalized greeting */}
      <WelcomeHeader />

      {/* Primary KPI cards with sparklines */}
      <KpiGrid />

      {/* Secondary metrics strip — compact horizontal indicators */}
      <SecondaryMetricsStrip />

      {/* Main content grid — alerts + tasks side by side */}
      <div className="grid gap-6 lg:grid-cols-2">
        {hasCyber && <RecentAlertsTable />}
        <MyTasksList />
      </div>

      {/* Live activity timeline */}
      <ActivityTimeline />
    </div>
  );
}
