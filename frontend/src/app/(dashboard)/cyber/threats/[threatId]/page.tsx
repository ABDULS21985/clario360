'use client';

import { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, ChevronDown, Pencil, RefreshCw, Trash2 } from 'lucide-react';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';
import { THREAT_STATUS_TRANSITIONS } from '@/lib/cyber-threats';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { PermissionGate } from '@/components/auth/permission-gate';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { StatusBadge } from '@/components/shared/status-badge';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useApiMutation } from '@/hooks/use-api-mutation';
import type { Threat, ThreatStatus } from '@/types/cyber';

import { CreateThreatDialog } from '../_components/create-threat-dialog';
import { ThreatOverview } from './_components/threat-overview';
import { ThreatIndicatorsTab } from './_components/threat-indicators-tab';
import { ThreatAlertsTab } from './_components/threat-alerts-tab';
import { ThreatTimelineTab } from './_components/threat-timeline-tab';
import { ThreatMitreTab } from './_components/threat-mitre-tab';

interface Props {
  params: { threatId: string };
}

export default function ThreatDetailPage({ params }: Props) {
  const { threatId } = params;
  const router = useRouter();
  const [activeTab, setActiveTab] = useState('overview');
  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [pendingStatus, setPendingStatus] = useState<ThreatStatus | null>(null);

  const threatQuery = useQuery({
    queryKey: [`cyber-threat-${threatId}`],
    queryFn: () => apiGet<{ data: Threat }>(API_ENDPOINTS.CYBER_THREAT_DETAIL(threatId)),
  });

  const threat = threatQuery.data?.data;
  const allowedStatuses = useMemo(
    () => (threat ? THREAT_STATUS_TRANSITIONS[threat.status] : []),
    [threat],
  );

  const statusMutation = useApiMutation<Threat, { status: ThreatStatus }>(
    'put',
    API_ENDPOINTS.CYBER_THREAT_STATUS(threatId),
    {
      invalidateKeys: ['cyber-threats', `cyber-threat-${threatId}`],
      successMessage: 'Threat status updated',
      onSuccess: () => {
        setPendingStatus(null);
        void threatQuery.refetch();
      },
    },
  );

  const deleteMutation = useApiMutation<{ deleted: boolean }, { id: string }>(
    'delete',
    ({ id }) => API_ENDPOINTS.CYBER_THREAT_DETAIL(id),
    {
      invalidateKeys: ['cyber-threats'],
      successMessage: 'Threat deleted',
      onSuccess: () => {
        router.push(ROUTES.CYBER_THREATS);
      },
    },
  );

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        {threatQuery.isLoading ? (
          <>
            <div className="h-8 w-64 animate-pulse rounded bg-muted" />
            <LoadingSkeleton variant="card" count={2} />
          </>
        ) : threatQuery.error || !threat ? (
          <ErrorState message="Failed to load threat" onRetry={() => void threatQuery.refetch()} />
        ) : (
          <>
            <PageHeader
              title={
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => router.push(ROUTES.CYBER_THREATS)}
                    className="flex h-8 w-8 items-center justify-center rounded-full border bg-background text-muted-foreground shadow-sm transition-colors hover:bg-accent"
                  >
                    <ArrowLeft className="h-4 w-4" />
                  </button>
                  <span className="truncate">{threat.name}</span>
                </div>
              }
              description={
                <div className="flex flex-wrap items-center gap-3 pl-11">
                  <SeverityIndicator severity={threat.severity} showLabel />
                  <StatusBadge status={threat.status} />
                  <Badge variant="outline" className="capitalize">{threat.type.replaceAll('_', ' ')}</Badge>
                  {threat.threat_actor && <Badge variant="secondary">{threat.threat_actor}</Badge>}
                  {threat.campaign && <Badge variant="secondary">{threat.campaign}</Badge>}
                </div>
              }
              actions={
                <div className="flex items-center gap-2">
                  <Button variant="outline" size="sm" onClick={() => void threatQuery.refetch()}>
                    <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
                    Refresh
                  </Button>
                  <PermissionGate permission="cyber:write">
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="outline" size="sm" disabled={allowedStatuses.length === 0}>
                          Update Status
                          <ChevronDown className="ml-1.5 h-3.5 w-3.5" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        {allowedStatuses.map((status) => (
                          <DropdownMenuItem key={status} onClick={() => setPendingStatus(status)}>
                            Move to {status.replaceAll('_', ' ')}
                          </DropdownMenuItem>
                        ))}
                      </DropdownMenuContent>
                    </DropdownMenu>
                    <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
                      <Pencil className="mr-1.5 h-3.5 w-3.5" />
                      Edit Threat
                    </Button>
                    <Button variant="outline" size="sm" className="text-destructive" onClick={() => setDeleteOpen(true)}>
                      <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                      Delete Threat
                    </Button>
                  </PermissionGate>
                </div>
              }
            />

            <Tabs value={activeTab} onValueChange={setActiveTab}>
              <TabsList className="w-full justify-start overflow-x-auto">
                <TabsTrigger value="overview">Overview</TabsTrigger>
                <TabsTrigger value="indicators">Indicators</TabsTrigger>
                <TabsTrigger value="alerts">Related Alerts</TabsTrigger>
                <TabsTrigger value="timeline">Activity Timeline</TabsTrigger>
                <TabsTrigger value="mitre">MITRE Mapping</TabsTrigger>
              </TabsList>

              <TabsContent value="overview">
                <ThreatOverview threat={threat} />
              </TabsContent>
              <TabsContent value="indicators">
                <ThreatIndicatorsTab threatId={threat.id} />
              </TabsContent>
              <TabsContent value="alerts">
                <ThreatAlertsTab threatId={threat.id} />
              </TabsContent>
              <TabsContent value="timeline">
                <ThreatTimelineTab threatId={threat.id} />
              </TabsContent>
              <TabsContent value="mitre">
                <ThreatMitreTab threat={threat} />
              </TabsContent>
            </Tabs>

            <CreateThreatDialog
              open={editOpen}
              onOpenChange={setEditOpen}
              threat={threat}
              onSuccess={() => {
                setEditOpen(false);
                void threatQuery.refetch();
              }}
            />
            <ConfirmDialog
              open={Boolean(pendingStatus)}
              onOpenChange={(open) => {
                if (!open) {
                  setPendingStatus(null);
                }
              }}
              title="Update threat status"
              description={pendingStatus ? `Move this threat from ${threat.status} to ${pendingStatus}?` : ''}
              confirmLabel="Confirm"
              onConfirm={async () => {
                if (!pendingStatus) {
                  return;
                }
                await statusMutation.mutateAsync({ status: pendingStatus });
              }}
              loading={statusMutation.isPending}
            />
            <ConfirmDialog
              open={deleteOpen}
              onOpenChange={setDeleteOpen}
              title="Delete threat"
              description="This will remove the threat from active views while preserving historical records."
              confirmLabel="Delete Threat"
              variant="destructive"
              typeToConfirm={threat.name}
              onConfirm={async () => {
                await deleteMutation.mutateAsync({ id: threat.id });
              }}
              loading={deleteMutation.isPending}
            />
          </>
        )}
      </div>
    </PermissionRedirect>
  );
}
