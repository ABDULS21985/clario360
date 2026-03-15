'use client';

import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { ArrowRight, FileText, Gavel, Scale, ShieldCheck } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { KpiCard } from '@/components/shared/kpi-card';
import { RelativeTime } from '@/components/shared/relative-time';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { SectionCard } from '@/components/suites/section-card';
import { contractStatusConfig } from '@/lib/status-configs';
import { enterpriseApi } from '@/lib/enterprise';
import type { LexComplianceAlert, LexComplianceRule, LexContract } from '@/types/suites';
import { StatusBadge } from '@/components/shared/status-badge';

export default function LexPage() {
  const contractsQuery = useQuery({
    queryKey: ['lex-overview', 'contracts'],
    queryFn: () => enterpriseApi.lex.listContracts({ page: 1, per_page: 6, order: 'desc' }),
  });
  const documentsQuery = useQuery({
    queryKey: ['lex-overview', 'documents'],
    queryFn: () => enterpriseApi.lex.listDocuments({ page: 1, per_page: 6, order: 'desc' }),
  });
  const regulationsQuery = useQuery({
    queryKey: ['lex-overview', 'regulations'],
    queryFn: () => enterpriseApi.lex.listComplianceRules({ page: 1, per_page: 6, order: 'desc' }),
  });
  const complianceQuery = useQuery({
    queryKey: ['lex-overview', 'compliance'],
    queryFn: () => enterpriseApi.lex.getComplianceDashboard(),
  });
  const alertsQuery = useQuery({
    queryKey: ['lex-overview', 'alerts'],
    queryFn: () => enterpriseApi.lex.listComplianceAlerts({ page: 1, per_page: 6, order: 'desc' }),
  });

  if (contractsQuery.isLoading && documentsQuery.isLoading && regulationsQuery.isLoading && complianceQuery.isLoading && alertsQuery.isLoading) {
    return (
      <PermissionRedirect permission="lex:read">
        <div className="space-y-6">
          <PageHeader title="Legal" description="Contract management, documents, and compliance" />
          <LoadingSkeleton variant="card" count={4} />
        </div>
      </PermissionRedirect>
    );
  }

  if (contractsQuery.error && documentsQuery.error && regulationsQuery.error && complianceQuery.error && alertsQuery.error) {
    return (
      <PermissionRedirect permission="lex:read">
        <ErrorState
          message="Failed to load legal operations overview."
          onRetry={() => {
            void contractsQuery.refetch();
            void documentsQuery.refetch();
            void regulationsQuery.refetch();
            void complianceQuery.refetch();
            void alertsQuery.refetch();
          }}
        />
      </PermissionRedirect>
    );
  }

  const compliance = complianceQuery.data;
  const recentContracts = contractsQuery.data?.data ?? [];
  const recentAlerts = alertsQuery.data?.data ?? [];
  const regulations = regulationsQuery.data?.data ?? [];

  return (
    <PermissionRedirect permission="lex:read">
      <div className="space-y-6">
        <PageHeader
          title="Legal"
          description="Live legal operations view across contracts, document lifecycle, and compliance posture."
          actions={
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" asChild>
                <Link href="/lex/contracts">Contracts</Link>
              </Button>
              <Button size="sm" asChild>
                <Link href="/lex/compliance">Compliance</Link>
              </Button>
            </div>
          }
        />

        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <KpiCard title="Contracts" value={contractsQuery.data?.meta.total ?? 0} icon={FileText} iconColor="text-blue-600" />
          <KpiCard title="Documents" value={documentsQuery.data?.meta.total ?? 0} icon={Gavel} iconColor="text-violet-600" />
          <KpiCard title="Open Compliance Alerts" value={compliance?.open_alerts ?? 0} icon={ShieldCheck} iconColor="text-red-600" />
          <KpiCard title="Compliance Score" value={`${Math.round(compliance?.compliance_score ?? 0)}%`} icon={Scale} iconColor="text-orange-600" />
        </div>

        <div className="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
          <SectionCard
            title="Recent Contracts"
            description="Latest contract records and current lifecycle state."
            actions={
              <Button variant="ghost" size="sm" asChild>
                <Link href="/lex/contracts">
                  View all
                  <ArrowRight className="ml-1 h-3.5 w-3.5" />
                </Link>
              </Button>
            }
          >
            <div className="space-y-3">
              {recentContracts.length === 0 ? (
                <p className="text-sm text-muted-foreground">No contracts are available for this tenant.</p>
              ) : (
                recentContracts.map((contract) => (
                  <div key={contract.id} className="rounded-lg border px-4 py-3">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <Link href={`/lex/contracts/${contract.id}`} className="font-medium hover:underline">
                          {contract.title}
                        </Link>
                        <p className="text-xs text-muted-foreground capitalize">{contract.type.replace(/_/g, ' ')}</p>
                      </div>
                      <StatusBadge status={contract.status} config={contractStatusConfig} size="sm" />
                    </div>
                    <div className="mt-2 flex flex-wrap gap-3 text-xs text-muted-foreground">
                      <span>Value: {contract.total_value != null ? `${contract.currency} ${contract.total_value.toLocaleString()}` : 'Undisclosed'}</span>
                      <span>{contract.expiry_date ? `Expires ${new Date(contract.expiry_date).toLocaleDateString()}` : 'No expiry'}</span>
                    </div>
                  </div>
                ))
              )}
            </div>
          </SectionCard>

          <SectionCard title="Compliance Alerts" description="Latest non-compliance findings from the compliance dashboard.">
            <div className="space-y-3">
              {recentAlerts.length === 0 ? (
                <p className="text-sm text-muted-foreground">No active compliance alerts are currently open.</p>
              ) : (
                recentAlerts.map((alert: LexComplianceAlert) => (
                  <div key={alert.id} className="rounded-lg border px-4 py-3">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="font-medium">{alert.title}</p>
                        <p className="text-xs text-muted-foreground line-clamp-1">{alert.description}</p>
                      </div>
                      <SeverityIndicator severity={normalizeSeverity(alert.severity)} size="sm" />
                    </div>
                    <div className="mt-2 flex items-center justify-between text-xs text-muted-foreground">
                      <span className="capitalize">{alert.status.replace(/_/g, ' ')}</span>
                      <RelativeTime date={alert.created_at} />
                    </div>
                  </div>
                ))
              )}
            </div>
          </SectionCard>
        </div>

        <SectionCard title="Active Regulations" description="Enabled regulatory controls and rule definitions.">
          <div className="space-y-3">
            {regulations.length === 0 ? (
              <p className="text-sm text-muted-foreground">No regulations are configured for this tenant.</p>
            ) : (
              regulations.map((rule: LexComplianceRule) => (
                <div key={rule.id} className="rounded-lg border px-4 py-3">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="font-medium">{rule.name}</p>
                      <p className="text-xs text-muted-foreground capitalize">{rule.rule_type.replace(/_/g, ' ')}</p>
                    </div>
                    <div className="flex items-center gap-2">
                      <SeverityIndicator severity={normalizeSeverity(rule.severity)} size="sm" />
                      {rule.enabled ? <StatusBadge status="active" config={{ active: { label: 'Enabled', color: 'green', icon: ShieldCheck } }} size="sm" /> : null}
                    </div>
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
