'use client';

import { use, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { StatusBadge } from '@/components/shared/status-badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { RootCauseAnalysisPanel } from '@/components/cyber/root-cause-analysis-panel';
import { ArrowLeft, UserCheck, RefreshCw, ArrowUpCircle } from 'lucide-react';
import { AlertExplanationPanel } from './_components/alert-explanation-panel';
import { AlertContextPanel } from './_components/alert-context-panel';
import { AlertInvestigationTab } from './_components/alert-investigation-tab';
import { AlertEvidenceTab } from './_components/alert-evidence-tab';
import { AlertTimelineTab } from './_components/alert-timeline-tab';
import { AlertRemediationTab } from './_components/alert-remediation-tab';
import { AlertStatusDialog } from '../_components/alert-status-dialog';
import { AlertAssignDialog } from '../_components/alert-assign-dialog';
import { AlertEscalateDialog } from '../_components/alert-escalate-dialog';
import type { CyberAlert, RootCauseAnalysis } from '@/types/cyber';

interface Props {
  params: Promise<{ id: string }>;
}

export default function AlertDetailPage({ params }: Props) {
  const { id } = use(params);
  const router = useRouter();

  const [statusOpen, setStatusOpen] = useState(false);
  const [assignOpen, setAssignOpen] = useState(false);
  const [escalateOpen, setEscalateOpen] = useState(false);
  const [activeTab, setActiveTab] = useState('investigation');

  const { data: envelope, isLoading, error, refetch } = useQuery({
    queryKey: [`cyber-alert-${id}`],
    queryFn: () => apiGet<{ data: CyberAlert }>(`${API_ENDPOINTS.CYBER_ALERTS}/${id}`),
    refetchInterval: 30000,
  });
  const rootCauseQuery = useQuery({
    queryKey: ['cyber-alert-root-cause', id],
    queryFn: () => apiGet<{ data: RootCauseAnalysis }>(`/api/v1/rca/security_alert/${id}`),
    enabled: activeTab === 'root-cause' && Boolean(id),
    staleTime: 120000,
  });

  const alert = envelope?.data;

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        {isLoading ? (
          <>
            <div className="h-8 w-64 animate-pulse rounded bg-muted" />
            <LoadingSkeleton variant="card" />
          </>
        ) : error || !alert ? (
          <ErrorState message="Failed to load alert" onRetry={() => refetch()} />
        ) : (
          <>
            <PageHeader
              title={
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => router.back()}
                    className="flex h-8 w-8 items-center justify-center rounded-full border bg-background text-muted-foreground shadow-sm transition-colors hover:bg-accent"
                  >
                    <ArrowLeft className="h-4 w-4" />
                  </button>
                  <span className="truncate">{alert.title}</span>
                </div>
              }
              description={
                <div className="flex items-center gap-3 pl-11">
                  <SeverityIndicator severity={alert.severity} showLabel />
                  <StatusBadge status={alert.status} />
                  <span className="text-xs text-muted-foreground">Source: {alert.source}</span>
                </div>
              }
              actions={
                <div className="flex items-center gap-2">
                  <Button variant="outline" size="sm" onClick={() => void refetch()}>
                    <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
                    Refresh
                  </Button>
                  <Button variant="outline" size="sm" onClick={() => setAssignOpen(true)}>
                    <UserCheck className="mr-1.5 h-3.5 w-3.5" />
                    Assign
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    className="text-amber-600"
                    onClick={() => setEscalateOpen(true)}
                  >
                    <ArrowUpCircle className="mr-1.5 h-3.5 w-3.5" />
                    Escalate
                  </Button>
                  <Button size="sm" onClick={() => setStatusOpen(true)}>
                    Update Status
                  </Button>
                </div>
              }
            />

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
              {/* Left: tabs */}
              <div className="lg:col-span-2">
                <Tabs value={activeTab} onValueChange={setActiveTab}>
                  <TabsList className="w-full justify-start overflow-x-auto">
                    <TabsTrigger value="investigation">Investigation</TabsTrigger>
                    <TabsTrigger value="explanation">AI Analysis</TabsTrigger>
                    <TabsTrigger value="evidence">Evidence</TabsTrigger>
                    <TabsTrigger value="timeline">Timeline</TabsTrigger>
                    <TabsTrigger value="root-cause">Root Cause</TabsTrigger>
                    <TabsTrigger value="remediation">Remediation</TabsTrigger>
                  </TabsList>

                  <TabsContent value="investigation" className="mt-4">
                    <AlertInvestigationTab alertId={alert.id} />
                  </TabsContent>
                  <TabsContent value="explanation" className="mt-4">
                    <AlertExplanationPanel
                      explanation={alert.explanation}
                      confidenceScore={alert.confidence_score}
                    />
                  </TabsContent>
                  <TabsContent value="evidence" className="mt-4">
                    <AlertEvidenceTab explanation={alert.explanation} />
                  </TabsContent>
                  <TabsContent value="timeline" className="mt-4">
                    <AlertTimelineTab alertId={alert.id} />
                  </TabsContent>
                  <TabsContent value="root-cause" className="mt-4">
                    <RootCauseAnalysisPanel
                      analysis={rootCauseQuery.data?.data}
                      isLoading={rootCauseQuery.isLoading || rootCauseQuery.isFetching}
                      error={rootCauseQuery.error instanceof Error ? rootCauseQuery.error.message : null}
                      onAnalyze={() => void rootCauseQuery.refetch()}
                      analyzeLabel="Refresh Analysis"
                      emptyTitle="Analyze the underlying attack path"
                      emptyDescription="This view reconstructs the earliest correlated security event, the kill-chain progression, and the blast radius for this alert."
                    />
                  </TabsContent>
                  <TabsContent value="remediation" className="mt-4">
                    <AlertRemediationTab alertId={alert.id} explanation={alert.explanation} />
                  </TabsContent>
                </Tabs>
              </div>

              {/* Right: context sidebar */}
              <div className="space-y-4">
                <div className="rounded-xl border p-4">
                  <h3 className="mb-3 text-sm font-semibold">Alert Details</h3>
                  <AlertContextPanel alert={alert} />
                </div>
                <div className="rounded-xl border bg-muted/30 p-4">
                  <h3 className="mb-2 text-sm font-semibold">Description</h3>
                  <p className="text-sm leading-relaxed text-muted-foreground">{alert.description}</p>
                </div>
              </div>
            </div>

            <AlertStatusDialog
              open={statusOpen}
              onOpenChange={setStatusOpen}
              alert={alert}
              onSuccess={() => refetch()}
            />
            <AlertAssignDialog
              open={assignOpen}
              onOpenChange={setAssignOpen}
              alert={alert}
              onSuccess={() => refetch()}
            />
            <AlertEscalateDialog
              open={escalateOpen}
              onOpenChange={setEscalateOpen}
              alert={alert}
              onSuccess={() => refetch()}
            />
          </>
        )}
      </div>
    </PermissionRedirect>
  );
}
