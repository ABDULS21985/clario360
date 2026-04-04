'use client';

import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { SearchInput } from '@/components/shared/forms/search-input';
import { CTISeverityBadge } from '@/components/cyber/cti/severity-badge';
import { fetchThreatEvents, linkEventToCampaign } from '@/lib/cti-api';
import { timeAgo } from '@/lib/utils';
import { CTI_EVENT_TYPE_LABELS } from '@/types/cti';

interface EntityLinkDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  campaignId: string;
  linkedEventIds?: string[];
  onLinked?: () => void;
}

export function EntityLinkDialog({
  open,
  onOpenChange,
  campaignId,
  linkedEventIds = [],
  onLinked,
}: EntityLinkDialogProps) {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const resultsQuery = useQuery({
    queryKey: ['cti-linkable-events', search],
    queryFn: () => fetchThreatEvents({
      page: 1,
      per_page: 25,
      sort: 'first_seen_at',
      order: 'desc',
      search: search || undefined,
    }),
    enabled: open,
  });

  const linkMutation = useMutation({
    mutationFn: async (eventId: string) => linkEventToCampaign(campaignId, eventId),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['cti-campaign-events', campaignId] }),
        queryClient.invalidateQueries({ queryKey: ['cti-campaign', campaignId] }),
      ]);
      toast.success('Threat event linked to campaign');
      onLinked?.();
    },
    onError: () => {
      toast.error('Failed to link threat event');
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl">
        <DialogHeader>
          <DialogTitle>Link Threat Events</DialogTitle>
          <DialogDescription>
            Search the live CTI event stream and attach relevant observations to this campaign.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <SearchInput
            value={search}
            onChange={setSearch}
            placeholder="Search threat events by title, IOC, or target"
            loading={resultsQuery.isFetching}
          />
          <div className="overflow-hidden rounded-[22px] border border-[color:var(--card-border)] bg-[var(--card-bg)]">
            <div className="grid grid-cols-[1.4fr,0.7fr,0.9fr,0.7fr] gap-3 border-b px-4 py-3 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
              <span>Event</span>
              <span>Type</span>
              <span>First Seen</span>
              <span className="text-right">Action</span>
            </div>
            <div className="max-h-[420px] overflow-y-auto">
              {resultsQuery.data?.data.length ? (
                resultsQuery.data.data.map((event) => {
                  const isLinked = linkedEventIds.includes(event.id);
                  return (
                    <div key={event.id} className="grid grid-cols-[1.4fr,0.7fr,0.9fr,0.7fr] gap-3 border-b px-4 py-3 last:border-b-0">
                      <div className="min-w-0 space-y-1">
                        <div className="flex items-center gap-2">
                          <CTISeverityBadge severity={event.severity_code} />
                          <p className="truncate text-sm font-medium text-foreground">{event.title}</p>
                        </div>
                        <p className="truncate text-xs text-muted-foreground">
                          {event.ioc_value || event.target_org_name || event.origin_country_code || 'No IOC or target metadata'}
                        </p>
                      </div>
                      <span className="text-sm text-muted-foreground">
                        {CTI_EVENT_TYPE_LABELS[event.event_type] ?? event.event_type}
                      </span>
                      <span className="text-sm text-muted-foreground">{timeAgo(event.first_seen_at)}</span>
                      <div className="text-right">
                        <Button
                          size="sm"
                          variant={isLinked ? 'outline' : 'default'}
                          disabled={isLinked || linkMutation.isPending}
                          onClick={() => linkMutation.mutate(event.id)}
                        >
                          {isLinked ? 'Linked' : 'Link'}
                        </Button>
                      </div>
                    </div>
                  );
                })
              ) : (
                <div className="px-4 py-8 text-center text-sm text-muted-foreground">
                  {resultsQuery.isLoading ? 'Loading threat events…' : 'No threat events match the current search.'}
                </div>
              )}
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}