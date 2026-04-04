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
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  fetchCampaigns,
  fetchThreatActor,
  deleteThreatActor,
  updateThreatActor,
} from '@/lib/cti-api';
import { formatCountryCode } from '@/lib/cti-utils';
import { formatDateTime } from '@/lib/utils';
import { ROUTES } from '@/lib/constants';
import {
  CTI_ACTOR_TYPE_LABELS,
  CTI_MOTIVATION_LABELS,
  CTI_SOPHISTICATION_LABELS,
} from '@/types/cti';

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
    queryFn: () => fetchCampaigns({ actor_id: actorId, page: 1, per_page: 50, sort: 'first_seen_at', order: 'desc' }),
    enabled: Boolean(actorId),
  });

  const actor = actorQuery.data;
  const campaigns = campaignsQuery.data?.data ?? [];

  const toggleMutation = useMutation({
    mutationFn: async (isActive: boolean) => updateThreatActor(actorId, { is_active: isActive }),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['cti-actor', actorId] }),
        queryClient.invalidateQueries({ queryKey: ['cti-actors'] }),
      ]);
      toast.success('Actor status updated');
    },
    onError: () => toast.error('Failed to update actor status'),
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
          title={
            <div className="flex items-center gap-3">
              <button
                onClick={() => router.push(ROUTES.CYBER_CTI_ACTORS)}
                className="flex h-8 w-8 items-center justify-center rounded-full border bg-background text-muted-foreground shadow-sm transition-colors hover:bg-accent"
              >
                <ArrowLeft className="h-4 w-4" />
              </button>
              <span className="truncate">{actor.name}</span>
            </div>
          }
          description={
            <div className="flex flex-wrap items-center gap-3 pl-11 text-sm text-muted-foreground">
              <span className="rounded-full border px-3 py-1">{CTI_ACTOR_TYPE_LABELS[actor.actor_type]}</span>
              <span className="rounded-full border px-3 py-1">{actor.is_active ? 'Active' : 'Inactive'}</span>
              <span className="rounded-full border px-3 py-1">Risk {Math.round(actor.risk_score)}</span>
            </div>
          }
          actions={
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
          }
        />

        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Actor Profile</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <DetailRow label="Origin Country" value={formatCountryCode(actor.origin_country_code)} />
              <DetailRow label="Sophistication" value={CTI_SOPHISTICATION_LABELS[actor.sophistication_level]} />
              <DetailRow label="Motivation" value={CTI_MOTIVATION_LABELS[actor.primary_motivation]} />
              <DetailRow label="First Observed" value={formatDateTime(actor.first_observed_at)} />
              <DetailRow label="Last Activity" value={formatDateTime(actor.last_activity_at)} />
              <DetailRow label="MITRE Group" value={actor.mitre_group_id ?? '—'} />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Analyst Notes</CardTitle>
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

        <Card>
          <CardHeader>
            <CardTitle>Associated Campaigns</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {campaigns.length ? campaigns.map((campaign) => (
              <div key={campaign.id} className="grid gap-3 rounded-2xl border bg-background p-4 lg:grid-cols-[1.3fr,0.8fr,0.8fr] lg:items-center">
                <div className="space-y-1">
                  <Link href={`${ROUTES.CYBER_CTI_CAMPAIGNS}/${campaign.id}`} className="font-medium text-foreground hover:underline">
                    {campaign.name}
                  </Link>
                  <p className="text-sm text-muted-foreground">{campaign.campaign_code}</p>
                </div>
                <span className="text-sm text-muted-foreground">{campaign.status}</span>
                <span className="text-sm text-muted-foreground">{campaign.event_count} events linked</span>
              </div>
            )) : (
              <div className="rounded-2xl border border-dashed px-4 py-8 text-center text-sm text-muted-foreground">
                No campaigns currently reference this threat actor.
              </div>
            )}
          </CardContent>
        </Card>
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