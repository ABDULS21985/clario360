'use client';

import { useMemo, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Pencil, Power, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { PageHeader } from '@/components/common/page-header';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { PermissionGate } from '@/components/auth/permission-gate';
import { ActorFormDialog } from '@/components/cyber/cti/actor-form-dialog';
import { IOCValueDisplay } from '@/components/cyber/cti/ioc-value-display';
import { CTIKPIStatCard } from '@/components/cyber/cti/kpi-stat-card';
import { MitreTechniqueBadges } from '@/components/cyber/cti/mitre-technique-badges';
import { CTISeverityBadge } from '@/components/cyber/cti/severity-badge';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { MitreMiniHeatmap } from '@/components/cyber/mitre-mini-heatmap';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { apiGet } from '@/lib/api';
import {
  deleteThreatActor,
  fetchCampaignIOCs,
  fetchCampaigns,
  fetchThreatActor,
  updateThreatActor,
} from '@/lib/cti-api';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';
import { countryCodeToFlag, formatNumber, formatRelativeTime } from '@/lib/cti-utils';
import { formatDateTime } from '@/lib/utils';
import {
  CTI_ACTOR_TYPE_LABELS,
  CTI_MOTIVATION_LABELS,
  CTI_SOPHISTICATION_LABELS,
} from '@/types/cti';
import type { MITREHeatmapData, MITRETactic, MITRETechniqueItem } from '@/types/cyber';

export default function CTIActorDetailPage() {
  const params = useParams<{ id: string }>();
  const actorId = params?.id ?? '';
  const router = useRouter();
  const queryClient = useQueryClient();
  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);

  const actorQuery = useQuery({
    queryKey: ['cti-actor', actorId],
    queryFn: () => fetchThreatActor(actorId),
    enabled: Boolean(actorId),
  });
  const campaignsQuery = useQuery({
    queryKey: ['cti-actor-campaigns', actorId],
    queryFn: () => fetchCampaigns({ actor_id: actorId, page: 1, per_page: 100, sort: 'first_seen_at', order: 'desc' }),
    enabled: Boolean(actorId),
  });
  const iocsQuery = useQuery({
    queryKey: ['cti-actor-iocs', actorId],
    queryFn: async () => {
      const campaigns = campaignsQuery.data?.data ?? [];
      const responses = await Promise.all(
        campaigns.map((campaign) => fetchCampaignIOCs(campaign.id, 1, 100)),
      );

      const iocMap = new Map<string, (typeof responses)[number]['data'][number]>();
      responses.forEach((response) => {
        response.data.forEach((ioc) => {
          const key = `${ioc.ioc_type}:${ioc.ioc_value}`;
          if (!iocMap.has(key)) {
            iocMap.set(key, ioc);
          }
        });
      });

      return Array.from(iocMap.values());
    },
    enabled: campaignsQuery.isSuccess,
  });
  const mitreMetaQuery = useQuery({
    queryKey: ['cti-actor-mitre-meta'],
    queryFn: async () => {
      const [techniques, tactics] = await Promise.all([
        apiGet<{ data: MITRETechniqueItem[] }>(API_ENDPOINTS.CYBER_MITRE_TECHNIQUES),
        apiGet<{ data: MITRETactic[] }>(API_ENDPOINTS.CYBER_MITRE_TACTICS),
      ]);

      return {
        techniques: techniques.data,
        tactics: tactics.data,
      };
    },
    staleTime: 5 * 60_000,
  });

  const actor = actorQuery.data;
  const campaigns = campaignsQuery.data?.data ?? [];
  const aggregatedIocs = iocsQuery.data ?? [];

  const techniqueIds = useMemo(
    () => Array.from(new Set(campaigns.flatMap((campaign) => campaign.mitre_technique_ids))),
    [campaigns],
  );

  const techniqueHeatmap = useMemo<MITREHeatmapData>(() => {
    const tacticNameMap = new Map((mitreMetaQuery.data?.tactics ?? []).map((item) => [item.id, item.name]));
    const techniqueMap = new Map((mitreMetaQuery.data?.techniques ?? []).map((item) => [item.id, item]));
    const counts = new Map<string, number>();

    campaigns.forEach((campaign) => {
      campaign.mitre_technique_ids.forEach((techniqueId) => {
        counts.set(techniqueId, (counts.get(techniqueId) ?? 0) + 1);
      });
    });

    const cells = Array.from(counts.entries()).flatMap(([techniqueId, count]) => {
      const technique = techniqueMap.get(techniqueId);
      const tacticIds = technique?.tactic_ids ?? ['unknown'];

      return tacticIds.map((tacticId) => ({
        tactic_id: tacticId,
        tactic_name: tacticNameMap.get(tacticId) ?? tacticId,
        technique_id: techniqueId,
        technique_name: technique?.name ?? techniqueId,
        alert_count: count,
        critical_count: 0,
        last_seen: campaigns[0]?.last_seen_at ?? new Date().toISOString(),
        has_detection: false,
      }));
    });

    return {
      cells,
      max_count: Math.max(...Array.from(counts.values()), 0),
    };
  }, [campaigns, mitreMetaQuery.data]);

  const toggleMutation = useMutation({
    mutationFn: async (isActive: boolean) => updateThreatActor(actorId, { is_active: isActive }),
    onMutate: async (isActive) => {
      const previousActor = queryClient.getQueryData(['cti-actor', actorId]);
      queryClient.setQueryData(['cti-actor', actorId], (current: typeof actor) => (
        current ? { ...current, is_active: isActive } : current
      ));
      return { previousActor };
    },
    onError: (_error, _isActive, context) => {
      queryClient.setQueryData(['cti-actor', actorId], context?.previousActor);
      toast.error('Failed to update actor status');
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['cti-actor', actorId] }),
        queryClient.invalidateQueries({ queryKey: ['cti-actors'] }),
      ]);
      toast.success('Actor status updated');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async () => deleteThreatActor(actorId),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['cti-actors'] });
      toast.success('Threat actor deleted');
      router.push(ROUTES.CYBER_CTI_ACTORS);
    },
    onError: () => toast.error('Failed to delete threat actor'),
  });

  if (actorQuery.isLoading) {
    return (
      <PermissionRedirect permission="cyber:read">
        <LoadingSkeleton variant="card" count={2} />
      </PermissionRedirect>
    );
  }

  if (!actor || actorQuery.error) {
    return (
      <PermissionRedirect permission="cyber:read">
        <ErrorState message="Failed to load threat actor" onRetry={() => void actorQuery.refetch()} />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title={(
            <div className="flex items-center gap-3">
              <button
                onClick={() => router.push(ROUTES.CYBER_CTI_ACTORS)}
                className="flex h-8 w-8 items-center justify-center rounded-full border bg-background text-muted-foreground shadow-sm transition-colors hover:bg-accent"
              >
                <ArrowLeft className="h-4 w-4" />
              </button>
              <span className="truncate">{actor.name}</span>
            </div>
          )}
          description={(
            <div className="flex flex-wrap items-center gap-3 pl-11 text-sm text-muted-foreground">
              <span className="rounded-full border px-3 py-1">{CTI_ACTOR_TYPE_LABELS[actor.actor_type]}</span>
              <span className="rounded-full border px-3 py-1">{actor.is_active ? 'Active' : 'Inactive'}</span>
              <span className="rounded-full border px-3 py-1">Risk {Math.round(actor.risk_score)}</span>
              <span>{countryCodeToFlag(actor.origin_country_code)} {actor.origin_country_code?.toUpperCase() ?? 'Unknown'}</span>
            </div>
          )}
          actions={(
            <div className="flex flex-wrap items-center gap-2">
              <PermissionGate permission="cyber:write">
                <Button variant="outline" size="sm" onClick={() => toggleMutation.mutate(!actor.is_active)}>
                  <Power className="mr-1.5 h-3.5 w-3.5" />
                  {actor.is_active ? 'Deactivate' : 'Activate'}
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
          )}
        />

        <Tabs defaultValue="profile">
          <TabsList className="w-full justify-start overflow-x-auto">
            <TabsTrigger value="profile">Profile</TabsTrigger>
            <TabsTrigger value="campaigns">Campaigns ({campaigns.length})</TabsTrigger>
            <TabsTrigger value="techniques">Techniques ({techniqueIds.length})</TabsTrigger>
            <TabsTrigger value="iocs">IOCs ({aggregatedIocs.length})</TabsTrigger>
          </TabsList>

          <TabsContent value="profile" className="space-y-4">
            <div className="grid gap-4 md:grid-cols-3">
              <CTIKPIStatCard label="Active Campaigns" value={campaigns.filter((campaign) => campaign.status === 'active').length} subtitle="Currently active" />
              <CTIKPIStatCard label="Total IOCs" value={aggregatedIocs.length} subtitle="Across linked campaigns" />
              <CTIKPIStatCard label="Observed Since" value={actor.first_observed_at ? new Date(actor.first_observed_at).getFullYear() : '—'} subtitle="Earliest record" />
            </div>

            <div className="grid gap-4 lg:grid-cols-2">
              <Card>
                <CardHeader>
                  <CardTitle>Actor Profile</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3 text-sm">
                  <DetailRow label="Origin Country" value={`${countryCodeToFlag(actor.origin_country_code)} ${actor.origin_country_code?.toUpperCase() ?? 'Unknown'}`} />
                  <DetailRow label="Sophistication" value={CTI_SOPHISTICATION_LABELS[actor.sophistication_level]} />
                  <DetailRow label="Motivation" value={CTI_MOTIVATION_LABELS[actor.primary_motivation]} />
                  <DetailRow label="First Observed" value={formatDateTime(actor.first_observed_at)} />
                  <DetailRow label="Last Activity" value={formatDateTime(actor.last_activity_at)} />
                  <DetailRow label="MITRE Group" value={actor.mitre_group_id ?? '—'} />
                </CardContent>
              </Card>
              <Card>
                <CardHeader>
                  <CardTitle>Aliases & Notes</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3 text-sm">
                  <div className="flex flex-wrap gap-2">
                    {(actor.aliases ?? []).length ? actor.aliases.map((alias) => (
                      <span key={alias} className="rounded-full border px-3 py-1 text-sm text-muted-foreground">{alias}</span>
                    )) : (
                      <span className="text-muted-foreground">No aliases recorded</span>
                    )}
                  </div>
                  <p className="rounded-2xl border bg-muted/20 p-4 text-muted-foreground">
                    {actor.description || 'No analyst notes recorded for this actor.'}
                  </p>
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          <TabsContent value="campaigns" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Associated Campaigns</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {campaigns.length ? campaigns.map((campaign) => (
                  <div key={campaign.id} className="grid gap-3 rounded-2xl border bg-background p-4 lg:grid-cols-[1.3fr,0.9fr,0.8fr,0.8fr] lg:items-center">
                    <div className="space-y-1">
                      <Link href={`${ROUTES.CYBER_CTI_CAMPAIGNS}/${campaign.id}`} className="font-medium text-foreground hover:underline">
                        {campaign.name}
                      </Link>
                      <p className="text-sm text-muted-foreground">{campaign.campaign_code}</p>
                    </div>
                    <CTISeverityBadge severity={campaign.severity_code} size="sm" />
                    <span className="text-sm text-muted-foreground">{campaign.status}</span>
                    <span className="text-sm text-muted-foreground">{formatRelativeTime(campaign.last_seen_at ?? campaign.first_seen_at)}</span>
                  </div>
                )) : (
                  <div className="rounded-2xl border border-dashed px-4 py-8 text-center text-sm text-muted-foreground">
                    No campaigns currently reference this threat actor.
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="techniques" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Observed Techniques</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <MitreMiniHeatmap data={techniqueHeatmap} maxTactics={6} maxTechniques={6} />
                <MitreTechniqueBadges techniqueIds={techniqueIds} maxVisible={16} />
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="iocs" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Aggregated IOCs</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {aggregatedIocs.length ? aggregatedIocs.map((ioc) => (
                  <div key={ioc.id} className="grid gap-3 rounded-2xl border bg-background p-4 lg:grid-cols-[1.4fr,0.7fr,0.7fr] lg:items-center">
                    <IOCValueDisplay type={ioc.ioc_type} value={ioc.ioc_value} className="border-0 bg-transparent p-0" />
                    <span className="text-sm text-muted-foreground">{formatNumber(Math.round(ioc.confidence_score * 100))}% confidence</span>
                    <span className="text-sm text-muted-foreground">{formatRelativeTime(ioc.last_seen_at)}</span>
                  </div>
                )) : (
                  <div className="rounded-2xl border border-dashed px-4 py-8 text-center text-sm text-muted-foreground">
                    No IOCs are linked through this actor’s campaigns yet.
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>

      <ActorFormDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        actor={actor}
        onSuccess={() => void actorQuery.refetch()}
      />

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete threat actor"
        description="This removes the actor profile from CTI views and detaches it from analyst pivots."
        confirmLabel="Delete Actor"
        variant="destructive"
        typeToConfirm={actor.name}
        loading={deleteMutation.isPending}
        onConfirm={async () => {
          await deleteMutation.mutateAsync();
        }}
      />
    </PermissionRedirect>
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
