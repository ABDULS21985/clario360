'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { enterpriseApi } from '@/lib/enterprise';
import { actaMonthString } from '@/lib/enterprise/utils';
import { ActaCalendarWidget } from './_components/acta-calendar-widget';
import { ActaComplianceBars } from './_components/acta-compliance-bars';
import { ActaDashboardKpis } from './_components/acta-dashboard-kpis';
import { ActaOverdueActions } from './_components/acta-overdue-actions';
import { ActaUpcomingMeetings } from './_components/acta-upcoming-meetings';

export default function ActaPage() {
  const [month, setMonth] = useState(actaMonthString(new Date()));
  const dashboardQuery = useQuery({
    queryKey: ['acta-dashboard'],
    queryFn: () => enterpriseApi.acta.getDashboard(),
  });
  const calendarQuery = useQuery({
    queryKey: ['acta-calendar', month],
    queryFn: () => enterpriseApi.acta.getCalendar(month),
  });

  if (dashboardQuery.isLoading || calendarQuery.isLoading) {
    return (
      <PermissionRedirect permission="acta:read">
        <div className="space-y-6">
          <PageHeader title="Board Governance" description="Board governance operations, meetings, and compliance" />
          <LoadingSkeleton variant="card" count={4} />
        </div>
      </PermissionRedirect>
    );
  }

  if (dashboardQuery.error || calendarQuery.error || !dashboardQuery.data) {
    return (
      <PermissionRedirect permission="acta:read">
        <ErrorState
          message="Failed to load board governance overview."
          onRetry={() => {
            void dashboardQuery.refetch();
            void calendarQuery.refetch();
          }}
        />
      </PermissionRedirect>
    );
  }

  const dashboard = dashboardQuery.data;

  return (
    <PermissionRedirect permission="acta:read">
      <div className="space-y-6">
        <PageHeader
          title="Board Governance"
          description="Committee operations, meeting readiness, action tracking, and compliance posture from live Acta APIs."
        />

        <ActaDashboardKpis kpis={dashboard.kpis} />

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-[1.4fr_1fr]">
          <ActaCalendarWidget
            month={month}
            days={calendarQuery.data ?? []}
            onMonthChange={setMonth}
          />
          <ActaUpcomingMeetings meetings={dashboard.upcoming_meetings} />
        </div>

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <ActaOverdueActions items={dashboard.overdue_action_items} />
          <ActaComplianceBars items={dashboard.compliance_by_committee} />
        </div>
      </div>
    </PermissionRedirect>
  );
}
