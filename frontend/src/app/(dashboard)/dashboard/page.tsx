'use client';

import { WelcomeHeader } from '@/components/dashboard/welcome-header';
import { KpiGrid } from '@/components/dashboard/kpi-grid';
import { RecentAlertsTable } from '@/components/dashboard/recent-alerts-table';
import { MyTasksList } from '@/components/dashboard/my-tasks-list';
import { ActivityTimeline } from '@/components/dashboard/activity-timeline';
import { useAuth } from '@/hooks/use-auth';

export default function DashboardHome() {
  const { hasPermission } = useAuth();
  const hasCyber = hasPermission('cyber:read');

  return (
    <div className="space-y-6">
      <WelcomeHeader />
      <KpiGrid />

      <div className="grid gap-6 lg:grid-cols-2">
        {hasCyber && <RecentAlertsTable />}
        <MyTasksList />
      </div>

      <ActivityTimeline />
    </div>
  );
}
