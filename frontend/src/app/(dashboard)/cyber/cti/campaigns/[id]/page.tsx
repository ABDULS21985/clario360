'use client';

import { useMemo, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, ChevronDown, Link2, Pencil, Plus, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { PageHeader } from '@/components/common/page-header';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { PermissionGate } from '@/components/auth/permission-gate';
import { CampaignFormDialog } from '@/components/cyber/cti/campaign-form-dialog';
import { CTISeverityBadge } from '@/components/cyber/cti/severity-badge';
import { CTIStatusBadge } from '@/components/cyber/cti/status-badge';
import { EntityLinkDialog } from '@/components/cyber/cti/entity-link-dialog';
import { IOCValueDisplay } from '@/components/cyber/cti/ioc-value-display';
import { MitreTechniqueBadges } from '@/components/cyber/cti/mitre-technique-badges';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs';
import {
  createCampaignIOC,
  deleteCampaign,
  deleteCampaignIOC,
  fetchCampaign,
  fetchCampaignEvents,
  fetchCampaignIOCs,
  fetchRegions,
  fetchSectors,
  unlinkEventFromCampaign,
  updateCampaignStatus,
} from '@/lib/cti-api';
import { CTI_CAMPAIGN_STATUS_OPTIONS, formatConfidenceScore } from '@/lib/cti-utils';
import { formatDateTime, timeAgo } from '@/lib/utils';
import { ROUTES } from '@/lib/constants';
import { CTI_EVENT_TYPE_LABELS, type CTICampaignIOC } from '@/types/cti';

export default function CTICampaignDetailPage() {
  const params = useParams<{ id: string }>();
  const campaignId = params?.id ?? '';
  const router = useRouter();
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState('overview');
  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [linkOpen, setLinkOpen] = useState(false);
  const [iocDialogOpen, setIocDialogOpen] = useState(false);

  const campaignQuery = useQuery({
    queryKey: ['cti-campaign', campaignId],
    queryFn: () => fetchCampaign(campaignId),
    enabled: Boolean(campaignId),
  });
  const eventsQuery = useQuery({
    queryKey: ['cti-campaign-events', campaignId],
    queryFn: () => fetchCampaignEvents(campaignId, 1, 50),
    enabled: Boolean(campaignId),
  });
  const iocsQuery = useQuery({
    queryKey: ['cti-campaign-iocs', campaignId],
    queryFn: () => fetchCampaignIOCs(campaignId, 1, 100),
    enabled: Boolean(campaignId),
  });
  const sectorsQuery = useQuery({
    queryKey: ['cti-campaign-detail-sectors'],
    queryFn: fetchSectors,
  });
  const regionsQuery = useQuery({
    queryKey: ['cti-campaign-detail-regions'],
    queryFn: () => fetchRegions(),
  });

  const campaign = campaignQuery.data;
  const linkedEvents = eventsQuery.data?.data ?? [];
  const campaignIOCs = iocsQuery.data?.data ?? [];

  const sectorLabels = useMemo(() => {
    const labels = new Map((sectorsQuery.data ?? []).map((sector) => [sector.id, sector.label]));
    return (campaign?.target_sectors ?? []).map((sectorId) => labels.get(sectorId) ?? sectorId);
  }, [campaign?.target_sectors, sectorsQuery.data]);

  const regionLabels = useMemo(() => {
    const labels = new Map((regionsQuery.data ?? []).map((region) => [region.id, region.label]));
    return (campaign?.target_regions ?? []).map((regionId) => labels.get(regionId) ?? regionId);
  }, [campaign?.target_regions, regionsQuery.data]);

  const statusMutation = useMutation({
    mutationFn: async (status: string) => updateCampaignStatus(campaignId, status),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['cti-campaign', campaignId] }),
        queryClient.invalidateQueries({ queryKey: ['cti-campaigns'] }),
      ]);
      toast.success('Campaign status updated');
    },
    onError: () => toast.error('Failed to update campaign status'),
  });

  const unlinkMutation = useMutation({
    mutationFn: async (eventId: string) => unlinkEventFromCampaign(campaignId, eventId),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['cti-campaign-events', campaignId] }),
        queryClient.invalidateQueries({ queryKey: ['cti-campaign', campaignId] }),
      ]);
      toast.success('Threat event unlinked');
    },
    onError: () => toast.error('Failed to unlink threat event'),
  });

  const deleteMutation = useMutation({
    mutationFn: async () => deleteCampaign(campaignId),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['cti-campaigns'] });
      toast.success('Campaign deleted');
      router.push(ROUTES.CYBER_CTI_CAMPAIGNS);
    },
    onError: () => toast.error('Failed to delete campaign'),
  });

  const deleteIocMutation = useMutation({
    mutationFn: async (iocId: string) => deleteCampaignIOC(campaignId, iocId),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['cti-campaign-iocs', campaignId] }),
        queryClient.invalidateQueries({ queryKey: ['cti-campaign', campaignId] }),
      ]);
      toast.success('Campaign IOC removed');
    },
    onError: () => toast.error('Failed to remove campaign IOC'),
  });

  if (campaignQuery.isLoading) {
    return (
      <PermissionRedirect permission="cyber:read">
        <LoadingSkeleton variant="card" count={2} />
      </PermissionRedirect>
    );
  }

  if (!campaign || campaignQuery.error) {
    return (
      <PermissionRedirect permission="cyber:read">
        <ErrorState message="Failed to load campaign" onRetry={() => void campaignQuery.refetch()} />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title={
            <div className="flex items-center gap-3">
              <button
                onClick={() => router.push(ROUTES.CYBER_CTI_CAMPAIGNS)}
                className="flex h-8 w-8 items-center justify-center rounded-full border bg-background text-muted-foreground shadow-sm transition-colors hover:bg-accent"
              >
                <ArrowLeft className="h-4 w-4" />
              </button>
              <span className="truncate">{campaign.name}</span>
            </div>
          }
          description={
            <div className="flex flex-wrap items-center gap-3 pl-11">
              <CTISeverityBadge severity={campaign.severity_code} />
              <CTIStatusBadge status={campaign.status} type="campaign" />
              <span className="rounded-full border px-3 py-1 text-xs font-medium text-muted-foreground">
                {campaign.campaign_code}
              </span>
              {campaign.primary_actor_id && campaign.actor_name && (
                <Link className="text-sm font-medium text-emerald-700 hover:underline" href={`${ROUTES.CYBER_CTI_ACTORS}/${campaign.primary_actor_id}`}>
                  {campaign.actor_name}
                </Link>
              )}
            </div>
          }
          actions={
            <div className="flex flex-wrap items-center gap-2">
              <PermissionGate permission="cyber:write">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="outline" size="sm">
                      Update Status
                      <ChevronDown className="ml-1.5 h-3.5 w-3.5" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    {CTI_CAMPAIGN_STATUS_OPTIONS.filter((option) => option.value !== campaign.status).map((option) => (
                      <DropdownMenuItem key={option.value} onClick={() => statusMutation.mutate(option.value)}>
                        Move to {option.label}
                      </DropdownMenuItem>
                    ))}
                  </DropdownMenuContent>
                </DropdownMenu>
                <Button variant="outline" size="sm" onClick={() => setLinkOpen(true)}>
                  <Link2 className="mr-1.5 h-3.5 w-3.5" />
                  Link Events
                </Button>
                <Button variant="outline" size="sm" onClick={() => setIocDialogOpen(true)}>
                  <Plus className="mr-1.5 h-3.5 w-3.5" />
                  Add IOC
                </Button>
                <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
                  <Pencil className="mr-1.5 h-3.5 w-3.5" />
                  Edit
                </Button>
                <Button variant="outline" size="sm" className="text-destructive" onClick={() => setDeleteOpen(true)}>
                  <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                  Delete
                </Button>
              </PermissionGate>
            </div>
          }
        />

        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList className="w-full justify-start overflow-x-auto">
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="events">Linked Events</TabsTrigger>
            <TabsTrigger value="iocs">IOCs</TabsTrigger>
            <TabsTrigger value="targeting">Targeting</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-4">
            <div className="grid gap-4 lg:grid-cols-2">
              <Card>
                <CardHeader>
                  <CardTitle>Campaign Overview</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3 text-sm">
                  <DetailRow label="First Seen" value={formatDateTime(campaign.first_seen_at)} />
                  <DetailRow label="Last Seen" value={formatDateTime(campaign.last_seen_at)} />
                  <DetailRow label="Events Linked" value={String(campaign.event_count)} />
                  <DetailRow label="IOCs Tracked" value={String(campaign.ioc_count)} />
                  <DetailRow label="Primary Actor" value={campaign.actor_name ?? 'Unassigned'} />
                  {campaign.description ? (
                    <p className="rounded-2xl border bg-muted/20 p-4 text-muted-foreground">{campaign.description}</p>
                  ) : null}
                </CardContent>
              </Card>
              <Card>
                <CardHeader>
                  <CardTitle>TTP Coverage</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4 text-sm">
                  <MitreTechniqueBadges techniqueIds={campaign.mitre_technique_ids} />
                  <div>
                    <p className="mb-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                      TTP Summary
                    </p>
                    <p className="rounded-2xl border bg-muted/20 p-4 text-muted-foreground">
                      {campaign.ttps_summary || 'No TTP summary recorded.'}
                    </p>
                  </div>
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          <TabsContent value="events" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Linked Threat Events</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {linkedEvents.length ? linkedEvents.map((event) => (
                  <div key={event.id} className="grid gap-3 rounded-2xl border bg-background p-4 lg:grid-cols-[1.3fr,0.8fr,0.8fr,auto] lg:items-center">
                    <div className="space-y-1">
                      <Link href={`${ROUTES.CYBER_CTI_EVENTS}/${event.id}`} className="font-medium text-foreground hover:underline">
                        {event.title}
                      </Link>
                      <p className="text-sm text-muted-foreground">{event.ioc_value || event.target_org_name || 'No IOC or target metadata'}</p>
                    </div>
                    <div className="flex items-center gap-2">
                      <CTISeverityBadge severity={event.severity_code} />
                      <span className="text-sm text-muted-foreground">{CTI_EVENT_TYPE_LABELS[event.event_type] ?? event.event_type}</span>
                    </div>
                    <span className="text-sm text-muted-foreground">{timeAgo(event.first_seen_at)}</span>
                    <PermissionGate permission="cyber:write">
                      <Button variant="outline" size="sm" onClick={() => unlinkMutation.mutate(event.id)}>
                        Unlink
                      </Button>
                    </PermissionGate>
                  </div>
                )) : (
                  <EmptyMessage message="No threat events linked yet." />
                )}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="iocs" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Campaign Indicators</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {campaignIOCs.length ? campaignIOCs.map((ioc) => (
                  <CampaignIOCRow
                    key={ioc.id}
                    ioc={ioc}
                    onDelete={() => deleteIocMutation.mutate(ioc.id)}
                  />
                )) : (
                  <EmptyMessage message="No campaign IOCs added yet." />
                )}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="targeting" className="space-y-4">
            <div className="grid gap-4 lg:grid-cols-2">
              <Card>
                <CardHeader>
                  <CardTitle>Target Sectors</CardTitle>
                </CardHeader>
                <CardContent>
                  {sectorLabels.length ? (
                    <div className="flex flex-wrap gap-2">
                      {sectorLabels.map((label) => (
                        <span key={label} className="rounded-full border px-3 py-1 text-sm text-muted-foreground">{label}</span>
                      ))}
                    </div>
                  ) : (
                    <EmptyMessage message="No target sectors recorded." compact />
                  )}
                </CardContent>
              </Card>
              <Card>
                <CardHeader>
                  <CardTitle>Target Regions</CardTitle>
                </CardHeader>
                <CardContent>
                  {regionLabels.length ? (
                    <div className="flex flex-wrap gap-2">
                      {regionLabels.map((label) => (
                        <span key={label} className="rounded-full border px-3 py-1 text-sm text-muted-foreground">{label}</span>
                      ))}
                    </div>
                  ) : (
                    <EmptyMessage message="No target regions recorded." compact />
                  )}
                </CardContent>
              </Card>
            </div>
            <Card>
              <CardHeader>
                <CardTitle>Targeting Notes</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">
                  {campaign.target_description || 'No targeting narrative captured for this campaign.'}
                </p>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>

      <CampaignFormDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        campaign={campaign}
        onSuccess={() => void campaignQuery.refetch()}
      />
      <EntityLinkDialog
        open={linkOpen}
        onOpenChange={setLinkOpen}
        campaignId={campaignId}
        linkedEventIds={linkedEvents.map((event) => event.id)}
        onLinked={() => void eventsQuery.refetch()}
      />
      <AddCampaignIOCDialog
        open={iocDialogOpen}
        onOpenChange={setIocDialogOpen}
        campaignId={campaignId}
        onSuccess={() => void iocsQuery.refetch()}
      />
      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete campaign"
        description="This deletes the campaign record and removes it from the active CTI graph."
        confirmLabel="Delete Campaign"
        variant="destructive"
        typeToConfirm={campaign.name}
        loading={deleteMutation.isPending}
        onConfirm={async () => {
          await deleteMutation.mutateAsync();
        }}
      />
    </PermissionRedirect>
  );
}

function CampaignIOCRow({
  ioc,
  onDelete,
}: {
  ioc: CTICampaignIOC;
  onDelete: () => void;
}) {
  return (
    <div className="grid gap-3 rounded-2xl border bg-background p-4 lg:grid-cols-[1.4fr,0.8fr,0.8fr,auto] lg:items-center">
      <IOCValueDisplay type={ioc.ioc_type} value={ioc.ioc_value} className="border-0 bg-transparent p-0" />
      <span className="text-sm text-muted-foreground">Confidence {formatConfidenceScore(ioc.confidence_score)}</span>
      <span className="text-sm text-muted-foreground">Last seen {timeAgo(ioc.last_seen_at)}</span>
      <PermissionGate permission="cyber:write">
        <Button variant="outline" size="sm" onClick={onDelete}>Remove</Button>
      </PermissionGate>
    </div>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="text-muted-foreground">{label}</span>
      <span className="text-right font-medium text-foreground">{value || '—'}</span>
    </div>
  );
}

function EmptyMessage({ message, compact = false }: { message: string; compact?: boolean }) {
  return (
    <div className={compact ? 'text-sm text-muted-foreground' : 'rounded-2xl border border-dashed px-4 py-8 text-center text-sm text-muted-foreground'}>
      {message}
    </div>
  );
}

function AddCampaignIOCDialog({
  open,
  onOpenChange,
  campaignId,
  onSuccess,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  campaignId: string;
  onSuccess?: () => void;
}) {
  const queryClient = useQueryClient();
  const [iocType, setIocType] = useState('domain');
  const [iocValue, setIocValue] = useState('');
  const [confidence, setConfidence] = useState('80');
  const [sourceName, setSourceName] = useState('');

  const mutation = useMutation({
    mutationFn: async () => createCampaignIOC(campaignId, {
      ioc_type: iocType,
      ioc_value: iocValue.trim(),
      confidence_score: Number(confidence) / 100,
      source_name: sourceName.trim() || undefined,
    }),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['cti-campaign-iocs', campaignId] }),
        queryClient.invalidateQueries({ queryKey: ['cti-campaign', campaignId] }),
      ]);
      toast.success('Campaign IOC added');
      setIocType('domain');
      setIocValue('');
      setConfidence('80');
      setSourceName('');
      onOpenChange(false);
      onSuccess?.();
    },
    onError: () => toast.error('Failed to add campaign IOC'),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Add Campaign IOC</DialogTitle>
          <DialogDescription>
            Attach a new indicator of compromise to this campaign and surface it in campaign pivots.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-1.5">
              <label htmlFor="ioc_type" className="text-sm font-medium">IOC Type</label>
              <Input id="ioc_type" value={iocType} onChange={(event) => setIocType(event.target.value)} />
            </div>
            <div className="space-y-1.5">
              <label htmlFor="confidence" className="text-sm font-medium">Confidence %</label>
              <Input id="confidence" type="number" min={0} max={100} value={confidence} onChange={(event) => setConfidence(event.target.value)} />
            </div>
          </div>
          <div className="space-y-1.5">
            <label htmlFor="ioc_value" className="text-sm font-medium">IOC Value</label>
            <Input id="ioc_value" value={iocValue} onChange={(event) => setIocValue(event.target.value)} />
          </div>
          <div className="space-y-1.5">
            <label htmlFor="source_name" className="text-sm font-medium">Source Name</label>
            <Input id="source_name" value={sourceName} onChange={(event) => setSourceName(event.target.value)} />
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" disabled={mutation.isPending || !iocValue.trim()} onClick={() => mutation.mutate()}>
            {mutation.isPending ? 'Saving...' : 'Add IOC'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}