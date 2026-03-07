'use client';

import { useQuery } from '@tanstack/react-query';
import { Scale } from 'lucide-react';
import { KpiCard } from '@/components/shared/kpi-card';
import { RelativeTime } from '@/components/shared/relative-time';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { SectionCard } from '@/components/suites/section-card';
import { API_ENDPOINTS } from '@/lib/constants';
import { fetchSuiteData, fetchSuitePaginated } from '@/lib/suite-api';
import type { ComplianceDashboard, ComplianceRule } from '@/types/suites';

export default function LexCompliancePage() {
  const dashboardQuery = useQuery({
    queryKey: ['lex-compliance-dashboard'],
    queryFn: () => fetchSuiteData<ComplianceDashboard>(API_ENDPOINTS.LEX_COMPLIANCE),
  });
  const rulesQuery = useQuery({
    queryKey: ['lex-compliance-rules'],
    queryFn: () => fetchSuitePaginated<ComplianceRule>(API_ENDPOINTS.LEX_REGULATIONS, { page: 1, per_page: 25, order: 'desc' }),
  });

  if (dashboardQuery.isLoading && rulesQuery.isLoading) {
    return (
      <PermissionRedirect permission="lex:read">
        <div className="space-y-6">
          <PageHeader title="Compliance" description="Regulatory compliance tracking" />
          <LoadingSkeleton variant="card" count={4} />
        </div>
      </PermissionRedirect>
    );
  }

  if (dashboardQuery.error && rulesQuery.error) {
    return (
      <PermissionRedirect permission="lex:read">
        <ErrorState
          message="Failed to load compliance posture."
          onRetry={() => {
            void dashboardQuery.refetch();
            void rulesQuery.refetch();
          }}
        />
      </PermissionRedirect>
    );
  }

  const dashboard = dashboardQuery.data;
  const rules = rulesQuery.data?.data ?? [];

  return (
    <PermissionRedirect permission="lex:read">
      <div className="space-y-6">
        <PageHeader title="Compliance" description="Rule coverage, recent alerting, and contract exposure from the live lex-service dashboard." />

        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <KpiCard title="Total Rules" value={dashboard?.total_rules ?? 0} icon={Scale} iconColor="text-blue-600" />
          <KpiCard title="Enabled Rules" value={dashboard?.enabled_rules ?? 0} icon={Scale} iconColor="text-green-600" />
          <KpiCard title="Open Alerts" value={dashboard?.open_alerts ?? 0} icon={Scale} iconColor="text-red-600" />
          <KpiCard title="Contracts Expiring" value={dashboard?.contracts_expiring_30d ?? 0} icon={Scale} iconColor="text-orange-600" />
        </div>

        <div className="grid gap-4 xl:grid-cols-[0.95fr_1.05fr]">
          <SectionCard title="Recent Compliance Alerts" description="Latest alerts from legal compliance evaluations.">
            <div className="space-y-3">
              {dashboard?.recent_alerts.length ? (
                dashboard.recent_alerts.map((alert) => (
                  <div key={alert.id} className="rounded-lg border px-4 py-3">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="font-medium">{alert.title}</p>
                        <p className="text-xs text-muted-foreground">{alert.entity_type}</p>
                      </div>
                      <SeverityIndicator severity={normalizeSeverity(alert.severity)} size="sm" />
                    </div>
                    <div className="mt-2 flex items-center justify-between text-xs text-muted-foreground">
                      <span className="capitalize">{alert.status.replace(/_/g, ' ')}</span>
                      <RelativeTime date={alert.created_at} />
                    </div>
                  </div>
                ))
              ) : (
                <p className="text-sm text-muted-foreground">No open compliance alerts are present.</p>
              )}
            </div>
          </SectionCard>

          <SectionCard title="Regulation Library" description="Current rules available for compliance checks.">
            <div className="space-y-3">
              {rules.length === 0 ? (
                <p className="text-sm text-muted-foreground">No compliance rules have been configured.</p>
              ) : (
                rules.map((rule) => (
                  <div key={rule.id} className="rounded-lg border px-4 py-3">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="font-medium">{rule.name}</p>
                        <p className="text-xs text-muted-foreground">{rule.regulation_reference ?? rule.jurisdiction ?? 'Unspecified reference'}</p>
                      </div>
                      <SeverityIndicator severity={normalizeSeverity(rule.severity)} size="sm" />
                    </div>
                    <p className="mt-2 text-sm text-muted-foreground">{rule.description}</p>
                  </div>
                ))
              )}
            </div>
          </SectionCard>
        </div>
      </div>
    </PermissionRedirect>
  );
}

function normalizeSeverity(value: string): 'critical' | 'high' | 'medium' | 'low' | 'info' {
  switch (value) {
    case 'critical':
    case 'high':
    case 'medium':
    case 'low':
      return value;
    default:
      return 'info';
  }
}
