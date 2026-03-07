'use client';

import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { ArrowRight, Eye, FileBarChart, Grid3X3, LayoutDashboard } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { KpiCard } from '@/components/shared/kpi-card';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { RelativeTime } from '@/components/shared/relative-time';
import { SectionCard } from '@/components/suites/section-card';
import { API_ENDPOINTS } from '@/lib/constants';
import { fetchSuitePaginated } from '@/lib/suite-api';
import type { VisusDashboard, VisusReport, VisusWidget } from '@/types/suites';
import { Badge } from '@/components/ui/badge';

export default function VisusPage() {
  const dashboardsQuery = useQuery({
    queryKey: ['visus-overview', 'dashboards'],
    queryFn: () => fetchSuitePaginated<VisusDashboard>(API_ENDPOINTS.VISUS_DASHBOARDS, { page: 1, per_page: 10, order: 'desc' }),
  });
  const reportsQuery = useQuery({
    queryKey: ['visus-overview', 'reports'],
    queryFn: () => fetchSuitePaginated<VisusReport>(API_ENDPOINTS.VISUS_REPORTS, { page: 1, per_page: 10, order: 'desc' }),
  });
  const widgetsQuery = useQuery({
    queryKey: ['visus-overview', 'widgets'],
    queryFn: () => fetchSuitePaginated<VisusWidget>(API_ENDPOINTS.VISUS_WIDGETS, { page: 1, per_page: 100, order: 'desc' }),
  });

  if (dashboardsQuery.isLoading && reportsQuery.isLoading && widgetsQuery.isLoading) {
    return (
      <PermissionRedirect permission="visus:read">
        <div className="space-y-6">
          <PageHeader title="Executive Intelligence" description="Executive dashboards and reports" />
          <LoadingSkeleton variant="card" count={4} />
        </div>
      </PermissionRedirect>
    );
  }

  if (dashboardsQuery.error && reportsQuery.error && widgetsQuery.error) {
    return (
      <PermissionRedirect permission="visus:read">
        <ErrorState
          message="Failed to load executive intelligence views."
          onRetry={() => {
            void dashboardsQuery.refetch();
            void reportsQuery.refetch();
            void widgetsQuery.refetch();
          }}
        />
      </PermissionRedirect>
    );
  }

  const dashboards = dashboardsQuery.data?.data ?? [];
  const reports = reportsQuery.data?.data ?? [];
  const widgets = widgetsQuery.data?.data ?? [];
  const widgetTypeCounts = countBy(widgets.map((widget) => widget.type));

  return (
    <PermissionRedirect permission="visus:read">
      <div className="space-y-6">
        <PageHeader
          title="Executive Intelligence"
          description="Live executive reporting inventory across dashboards, widgets, and scheduled reports."
          actions={
            <Button size="sm" asChild>
              <Link href="/visus/reports">Open reports</Link>
            </Button>
          }
        />

        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <KpiCard title="Dashboards" value={dashboardsQuery.data?.meta.total ?? 0} icon={LayoutDashboard} iconColor="text-blue-600" />
          <KpiCard title="Reports" value={reportsQuery.data?.meta.total ?? 0} icon={FileBarChart} iconColor="text-violet-600" />
          <KpiCard title="Widgets" value={widgetsQuery.data?.meta.total ?? 0} icon={Grid3X3} iconColor="text-green-600" />
          <KpiCard title="Default Dashboards" value={dashboards.filter((dashboard) => dashboard.is_default).length} icon={Eye} iconColor="text-orange-600" />
        </div>

        <div className="grid gap-4 xl:grid-cols-[1fr_1fr]">
          <SectionCard
            title="Dashboards"
            description="Dashboard definitions currently available to the tenant."
            actions={
              <Button variant="ghost" size="sm" asChild>
                <Link href="/visus/reports">
                  Reports
                  <ArrowRight className="ml-1 h-3.5 w-3.5" />
                </Link>
              </Button>
            }
          >
            <div className="space-y-3">
              {dashboards.length === 0 ? (
                <p className="text-sm text-muted-foreground">No executive dashboards are configured.</p>
              ) : (
                dashboards.map((dashboard) => (
                  <div key={dashboard.id} className="rounded-lg border px-4 py-3">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="font-medium">{dashboard.name}</p>
                        <p className="text-xs text-muted-foreground">{dashboard.description || 'No description provided'}</p>
                      </div>
                      {dashboard.is_default ? <Badge variant="success">Default</Badge> : null}
                    </div>
                    <div className="mt-2 text-xs text-muted-foreground">
                      {dashboard.widget_count} widget{dashboard.widget_count === 1 ? '' : 's'}
                    </div>
                  </div>
                ))
              )}
            </div>
          </SectionCard>

          <SectionCard title="Widget Mix" description="Distribution of widget types across dashboard inventory.">
            <div className="space-y-3">
              {Object.keys(widgetTypeCounts).length === 0 ? (
                <p className="text-sm text-muted-foreground">No dashboard widgets have been configured.</p>
              ) : (
                Object.entries(widgetTypeCounts)
                  .sort((left, right) => right[1] - left[1])
                  .map(([type, count]) => (
                    <div key={type}>
                      <div className="mb-1 flex items-center justify-between text-sm">
                        <span className="capitalize text-muted-foreground">{type.replace(/_/g, ' ')}</span>
                        <span className="font-medium">{count}</span>
                      </div>
                      <div className="h-2 overflow-hidden rounded-full bg-muted">
                        <div className="h-full rounded-full bg-primary" style={{ width: `${Math.min(count * 14, 100)}%` }} />
                      </div>
                    </div>
                  ))
              )}
            </div>
          </SectionCard>
        </div>

        <SectionCard title="Recent Reports" description="Most recently updated report definitions.">
          <div className="space-y-3">
            {reports.length === 0 ? (
              <p className="text-sm text-muted-foreground">No executive reports are currently configured.</p>
            ) : (
              reports.map((report) => (
                <div key={report.id} className="rounded-lg border px-4 py-3">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="font-medium">{report.name}</p>
                      <p className="text-xs text-muted-foreground capitalize">{report.type.replace(/_/g, ' ')}</p>
                    </div>
                    {report.schedule ? <Badge variant="outline">{report.schedule}</Badge> : null}
                  </div>
                  <div className="mt-2 flex items-center justify-between text-xs text-muted-foreground">
                    <span>{report.file_url ? 'Last output available' : 'No output generated yet'}</span>
                    {report.last_generated_at ? <RelativeTime date={report.last_generated_at} /> : <span>Never generated</span>}
                  </div>
                </div>
              ))
            )}
          </div>
        </SectionCard>
      </div>
    </PermissionRedirect>
  );
}

function countBy(values: string[]): Record<string, number> {
  return values.reduce<Record<string, number>>((acc, value) => {
    acc[value] = (acc[value] ?? 0) + 1;
    return acc;
  }, {});
}
