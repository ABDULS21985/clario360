'use client';

import { useCallback, useMemo, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { format, formatDistanceToNow } from 'date-fns';
import {
  ArrowLeft,
  Copy,
  ExternalLink,
  Link2,
  Plus,
  ShieldAlert,
  Trash2,
} from 'lucide-react';
import { toast } from 'sonner';
import { PageHeader } from '@/components/common/page-header';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { CTISeverityBadge } from '@/components/cyber/cti/severity-badge';
import { CTIStatusBadge } from '@/components/cyber/cti/status-badge';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import {
  addEventTags,
  deleteThreatEvent,
  fetchCampaigns,
  fetchRelatedCampaignsForEvent,
  linkEventToCampaign,
  removeEventTag,
  markEventFalsePositive,
  resolveEvent,
} from '@/lib/cti-api';
import { API_ENDPOINTS } from '@/lib/constants';
import { CTI_EVENT_TYPE_LABELS, type CTICampaign, type CTIEventTimelineItem, type CTIThreatEvent } from '@/types/cti';

function buildTimeline(event: CTIThreatEvent): CTIEventTimelineItem[] {
  const items: CTIEventTimelineItem[] = [
    {
      id: 'first-seen',
      label: 'First Observed',
      timestamp: event.first_seen_at,
      description: 'Initial intelligence sighting was recorded.',
    },
    {
      id: 'created',
      label: 'Ingested',
      timestamp: event.created_at,
      description: 'Event entered the CTI platform.',
    },
  ];

  if (event.last_seen_at && event.last_seen_at !== event.first_seen_at) {
    items.push({
      id: 'last-seen',
      label: 'Last Seen',
      timestamp: event.last_seen_at,
      description: 'Most recent observation from the source feed.',
    });
  }

  if (event.is_false_positive && event.updated_at) {
    items.push({
      id: 'false-positive',
      label: 'Marked False Positive',
      timestamp: event.updated_at,
      tone: 'warning',
      description: 'Analyst marked this event as noise.',
    });
  }

  if (event.resolved_at) {
    items.push({
      id: 'resolved',
      label: 'Resolved',
      timestamp: event.resolved_at,
      tone: 'success',
      description: 'Response workflow completed.',
    });
  }

  return items.sort((left, right) => Date.parse(left.timestamp) - Date.parse(right.timestamp));
}

function toSvgPosition(latitude: number, longitude: number) {
  return {
    x: ((longitude + 180) / 360) * 240,
    y: ((90 - latitude) / 180) * 120,
  };
}

function MiniOriginMap({ event }: { event: CTIThreatEvent }) {
  const latitude = event.origin_latitude ?? 0;
  const longitude = event.origin_longitude ?? 0;
  const { x, y } = toSvgPosition(latitude, longitude);

  return (
    <svg viewBox="0 0 240 120" className="w-full rounded-xl border border-white/10 bg-slate-950/60">
      <rect x="0" y="0" width="240" height="120" fill="rgba(15,23,42,0.72)" />
      {Array.from({ length: 7 }).map((_, index) => (
        <line
          key={`v-${index}`}
          x1={index * 40}
          y1={0}
          x2={index * 40}
          y2={120}
          stroke="rgba(255,255,255,0.06)"
        />
      ))}
      {Array.from({ length: 5 }).map((_, index) => (
        <line
          key={`h-${index}`}
          x1={0}
          y1={index * 30}
          x2={240}
          y2={index * 30}
          stroke="rgba(255,255,255,0.06)"
        />
      ))}
      <circle cx={x} cy={y} r="6" fill="#FF3B5C">
        <animate attributeName="r" values="6;11;6" dur="2s" repeatCount="indefinite" />
        <animate attributeName="opacity" values="1;0.4;1" dur="2s" repeatCount="indefinite" />
      </circle>
    </svg>
  );
}

export default function CTIEventDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const eventId = params?.id ?? '';
  const [tagInput, setTagInput] = useState('');
  const [linkDialogOpen, setLinkDialogOpen] = useState(false);
  const [selectedCampaignId, setSelectedCampaignId] = useState<string>('');

  const { data: envelope, isLoading, error, mutate } = useRealtimeData<{ data: CTIThreatEvent }>(
    API_ENDPOINTS.CTI_EVENT_DETAIL(eventId),
    { pollInterval: 30_000 },
  );

  const campaignsQuery = useQuery({
    queryKey: ['cti-campaign-options'],
    queryFn: async () => fetchCampaigns({ page: 1, per_page: 100, sort: 'first_seen_at', order: 'desc' }),
  });

  const relatedCampaignsQuery = useQuery({
    queryKey: ['cti-event-related-campaigns', eventId],
    queryFn: () => fetchRelatedCampaignsForEvent(eventId),
    enabled: Boolean(eventId),
  });

  const event = envelope?.data;
  const timeline = useMemo(() => (event ? buildTimeline(event) : []), [event]);

  const handleCopyIOC = useCallback(() => {
    if (!event?.ioc_value) {
      return;
    }
    void navigator.clipboard.writeText(event.ioc_value);
    toast.success('IOC copied to clipboard');
  }, [event?.ioc_value]);

  const handleResolve = useCallback(async () => {
    await resolveEvent(eventId);
    toast.success('Threat event resolved');
    await Promise.all([mutate(), relatedCampaignsQuery.refetch()]);
  }, [eventId, mutate, relatedCampaignsQuery]);

  const handleFalsePositive = useCallback(async () => {
    await markEventFalsePositive(eventId);
    toast.success('Threat event marked as false positive');
    await mutate();
  }, [eventId, mutate]);

  const handleDelete = useCallback(async () => {
    await deleteThreatEvent(eventId);
    toast.success('Threat event deleted');
    router.push('/cyber/cti/events');
  }, [eventId, router]);

  const handleAddTags = useCallback(async () => {
    const tags = tagInput
      .split(',')
      .map((value) => value.trim())
      .filter(Boolean);

    if (tags.length === 0) {
      return;
    }

    await addEventTags(eventId, tags);
    setTagInput('');
    toast.success('Tags added');
    await mutate();
  }, [eventId, mutate, tagInput]);

  const handleRemoveTag = useCallback(async (tag: string) => {
    await removeEventTag(eventId, tag);
    toast.success('Tag removed');
    await mutate();
  }, [eventId, mutate]);

  const handleLinkCampaign = useCallback(async () => {
    if (!selectedCampaignId) {
      return;
    }
    await linkEventToCampaign(selectedCampaignId, eventId);
    toast.success('Event linked to campaign');
    setLinkDialogOpen(false);
    setSelectedCampaignId('');
    await relatedCampaignsQuery.refetch();
  }, [eventId, relatedCampaignsQuery, selectedCampaignId]);

  if (isLoading) {
    return (
      <PermissionRedirect permission="cyber:read">
        <LoadingSkeleton variant="card" />
      </PermissionRedirect>
    );
  }

  if (!eventId || error || !event) {
    return (
      <PermissionRedirect permission="cyber:read">
        <ErrorState
          message={!eventId ? 'Threat event identifier is missing.' : 'Failed to load CTI threat event.'}
          onRetry={() => {
            void mutate();
          }}
        />
      </PermissionRedirect>
    );
  }

  const relatedCampaigns = relatedCampaignsQuery.data ?? [];
  const campaignOptions = campaignsQuery.data?.data ?? [];

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" onClick={() => router.push('/cyber/cti/events')}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <PageHeader
            title={event.title}
            description={`Threat event ${event.id.slice(0, 8)} • ${CTI_EVENT_TYPE_LABELS[event.event_type] ?? event.event_type}`}
          />
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <CTISeverityBadge severity={event.severity_code} />
          <Badge variant="outline">{CTI_EVENT_TYPE_LABELS[event.event_type] ?? event.event_type}</Badge>
          {event.resolved_at && <CTIStatusBadge status="resolved" type="campaign" />}
          {event.is_false_positive && <CTIStatusBadge status="false_positive" type="takedown" />}
          <span className="text-xs text-muted-foreground">
            First seen {formatDistanceToNow(new Date(event.first_seen_at), { addSuffix: true })}
          </span>
          <div className="ml-auto flex flex-wrap items-center gap-2">
            {!event.resolved_at && (
              <Button size="sm" variant="outline" onClick={() => void handleResolve()}>
                Resolve
              </Button>
            )}
            {!event.is_false_positive && (
              <Button size="sm" variant="outline" onClick={() => void handleFalsePositive()}>
                False Positive
              </Button>
            )}
            <Button size="sm" variant="outline" onClick={() => setLinkDialogOpen(true)}>
              <Link2 className="mr-1.5 h-3.5 w-3.5" />
              Link to Campaign
            </Button>
            <Button size="sm" variant="destructive" onClick={() => void handleDelete()}>
              <Trash2 className="mr-1.5 h-3.5 w-3.5" />
              Delete
            </Button>
          </div>
        </div>

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
          <Card className="xl:col-span-2">
            <CardHeader className="p-4 pb-2">
              <CardTitle className="text-sm">Event Details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4 p-4 pt-0 text-sm">
              {event.description && <p className="leading-6 text-muted-foreground">{event.description}</p>}
              <Separator />
              <DetailRow label="Confidence" value={`${(event.confidence_score * 100).toFixed(0)}%`} />
              <DetailRow label="Category" value={event.category_label || '—'} />
              <DetailRow label="Source" value={event.source_name || '—'} />
              <DetailRow label="Target Sector" value={event.target_sector_label || event.sector_label || '—'} />
              <DetailRow label="Target Organization" value={event.target_org_name || '—'} />
              <DetailRow label="Origin" value={`${event.origin_city || 'Unknown'}, ${event.origin_country_code?.toUpperCase() || '—'}`} />
              <DetailRow label="Target Country" value={event.target_country_code?.toUpperCase() || '—'} />
              <DetailRow label="Created" value={format(new Date(event.created_at), 'PPpp')} />
              <DetailRow label="Last Seen" value={format(new Date(event.last_seen_at), 'PPpp')} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="p-4 pb-2">
              <CardTitle className="text-sm">Indicator & Origin</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4 p-4 pt-0">
              <MiniOriginMap event={event} />

              {event.ioc_type && event.ioc_value && (
                <div className="rounded-xl border border-white/10 bg-slate-950/60 p-3">
                  <div className="flex items-center gap-2 text-xs uppercase tracking-[0.2em] text-muted-foreground">
                    <ShieldAlert className="h-3.5 w-3.5" />
                    {event.ioc_type}
                  </div>
                  <div className="mt-2 flex items-center gap-2">
                    <code className="min-w-0 flex-1 truncate text-xs">{event.ioc_value}</code>
                    <Button size="icon" variant="ghost" className="h-7 w-7" onClick={handleCopyIOC}>
                      <Copy className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
              )}

              {event.mitre_technique_ids.length > 0 && (
                <div className="space-y-2">
                  <p className="text-xs font-medium uppercase tracking-[0.2em] text-muted-foreground">
                    MITRE Techniques
                  </p>
                  <div className="flex flex-wrap gap-2">
                    {event.mitre_technique_ids.map((technique) => (
                      <Badge key={technique} variant="outline" className="gap-1 text-[11px]">
                        <a
                          href={`https://attack.mitre.org/techniques/${technique.replace('.', '/')}/`}
                          target="_blank"
                          rel="noreferrer"
                          className="flex items-center gap-1"
                        >
                          {technique}
                          <ExternalLink className="h-3 w-3" />
                        </a>
                      </Badge>
                    ))}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
          <Card>
            <CardHeader className="p-4 pb-2">
              <CardTitle className="text-sm">Tags</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 p-4 pt-0">
              <div className="flex gap-2">
                <Input
                  value={tagInput}
                  onChange={(eventValue) => setTagInput(eventValue.target.value)}
                  placeholder="Add tags, comma separated"
                />
                <Button onClick={() => void handleAddTags()}>
                  <Plus className="mr-1.5 h-4 w-4" />
                  Add
                </Button>
              </div>
              <div className="flex flex-wrap gap-2">
                {event.tags.length > 0 ? event.tags.map((tag) => (
                  <Badge key={tag} variant="secondary" className="gap-1 pr-1 text-[11px]">
                    {tag}
                    <button type="button" className="rounded-full px-1 hover:bg-black/10" onClick={() => void handleRemoveTag(tag)}>
                      ×
                    </button>
                  </Badge>
                )) : (
                  <p className="text-sm text-muted-foreground">No tags attached to this event.</p>
                )}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="p-4 pb-2">
              <CardTitle className="text-sm">Timeline</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 p-4 pt-0">
              {timeline.map((item) => (
                <div key={item.id} className="relative pl-5">
                  <span className="absolute left-0 top-1.5 h-2.5 w-2.5 rounded-full bg-emerald-400" />
                  <div className="flex items-center justify-between gap-2">
                    <p className="text-sm font-medium">{item.label}</p>
                    <p className="text-xs text-muted-foreground">{format(new Date(item.timestamp), 'PP p')}</p>
                  </div>
                  {item.description && <p className="text-xs text-muted-foreground">{item.description}</p>}
                </div>
              ))}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="p-4 pb-2">
              <CardTitle className="text-sm">Related Campaigns</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 p-4 pt-0">
              {relatedCampaigns.length > 0 ? relatedCampaigns.map((campaign) => (
                <button
                  key={campaign.id}
                  type="button"
                  onClick={() => router.push(`/cyber/cti/campaigns?campaign=${campaign.id}`)}
                  className="w-full rounded-xl border border-white/10 bg-slate-950/50 p-3 text-left hover:bg-slate-950/70"
                >
                  <div className="flex items-center justify-between gap-2">
                    <div>
                      <p className="text-sm font-medium">{campaign.name}</p>
                      <p className="text-xs text-muted-foreground">{campaign.campaign_code}</p>
                    </div>
                    <CTISeverityBadge severity={campaign.severity_code} size="sm" />
                  </div>
                </button>
              )) : (
                <p className="text-sm text-muted-foreground">No linked campaigns found for this event.</p>
              )}
            </CardContent>
          </Card>
        </div>
      </div>

      <Dialog open={linkDialogOpen} onOpenChange={setLinkDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Link Event to Campaign</DialogTitle>
            <DialogDescription>
              Choose an existing CTI campaign to associate with this threat event.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-2">
            <p className="text-sm font-medium">Campaign</p>
            <Select value={selectedCampaignId} onValueChange={setSelectedCampaignId}>
              <SelectTrigger>
                <SelectValue placeholder="Select a campaign" />
              </SelectTrigger>
              <SelectContent>
                {campaignOptions.map((campaign: CTICampaign) => (
                  <SelectItem key={campaign.id} value={campaign.id}>
                    {campaign.name} ({campaign.campaign_code})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setLinkDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={() => void handleLinkCampaign()} disabled={!selectedCampaignId}>
              Link Campaign
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PermissionRedirect>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-start justify-between gap-4">
      <span className="text-muted-foreground">{label}</span>
      <span className="text-right font-medium">{value}</span>
    </div>
  );
}
