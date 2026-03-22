'use client';

import { useMemo, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Clock3, Play, RefreshCcw, TriangleAlert } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { DetailPanel } from '@/components/shared/detail-panel';
import { RelativeTime } from '@/components/shared/relative-time';
import { apiGet, apiPost } from '@/lib/api';
import {
  getThreatFeedIntervalLabel,
  getThreatFeedTypeLabel,
} from '@/lib/cyber-indicators';
import { API_ENDPOINTS } from '@/lib/constants';
import { parseApiError } from '@/lib/format';
import { formatDateTime } from '@/lib/utils';
import type {
  ThreatFeedConfig,
  ThreatFeedSyncHistory,
  ThreatFeedSyncSummary,
} from '@/types/cyber';

interface FeedDetailProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  feed: ThreatFeedConfig | null;
  onEdit?: (feed: ThreatFeedConfig) => void;
  onSynced?: () => void;
}

export function FeedDetail({
  open,
  onOpenChange,
  feed,
  onEdit,
  onSynced,
}: FeedDetailProps) {
  const [syncing, setSyncing] = useState(false);
  const queryClient = useQueryClient();

  // Fetch fresh feed data directly from the backend so the detail panel is self-sufficient
  const feedQuery = useQuery({
    queryKey: ['cyber-threat-feed-detail', feed?.id],
    queryFn: () => apiGet<{ data: ThreatFeedConfig }>(API_ENDPOINTS.CYBER_THREAT_FEED_DETAIL(feed!.id)),
    enabled: open && Boolean(feed?.id),
    // Use the parent-supplied feed as initial data to avoid a loading flash
    initialData: feed ? { data: feed } : undefined,
  });

  // Use the freshly fetched feed when available, fall back to the prop
  const liveFeed = feedQuery.data?.data ?? feed;

  const historyQuery = useQuery({
    queryKey: ['cyber-threat-feed-history', feed?.id],
    queryFn: () => apiGet<{ data: ThreatFeedSyncHistory[] }>(API_ENDPOINTS.CYBER_THREAT_FEED_HISTORY(feed!.id)),
    enabled: open && Boolean(feed?.id),
  });

  const latestHistory = historyQuery.data?.data?.[0];
  const previewIndicators = useMemo(() => {
    const metadata = latestHistory?.metadata as Record<string, unknown> | undefined;
    return Array.isArray(metadata?.preview_indicators)
      ? (metadata?.preview_indicators as Array<Record<string, unknown>>)
      : [];
  }, [latestHistory?.metadata]);

  async function handleSync() {
    if (!feed) {
      return;
    }
    try {
      setSyncing(true);
      const response = await apiPost<{ data: ThreatFeedSyncSummary }>(API_ENDPOINTS.CYBER_THREAT_FEED_SYNC(feed.id));
      toast.success(`${response.data.indicators_imported} indicators imported`);
      // Refetch both the feed detail and history immediately so the panel updates
      void queryClient.invalidateQueries({ queryKey: ['cyber-threat-feed-detail', feed.id] });
      void queryClient.invalidateQueries({ queryKey: ['cyber-threat-feed-history', feed.id] });
      void queryClient.invalidateQueries({ queryKey: ['cyber-threat-feed-last-history'] });
      onSynced?.();
    } catch (error) {
      toast.error(parseApiError(error));
    } finally {
      setSyncing(false);
    }
  }

  return (
    <DetailPanel
      open={open}
      onOpenChange={onOpenChange}
      title={liveFeed?.name ?? 'Threat Feed'}
      description={liveFeed ? `${getThreatFeedTypeLabel(liveFeed.type)} feed configuration` : 'Threat feed detail'}
      width="xl"
    >
      {!liveFeed ? (
        <p className="text-sm text-muted-foreground">Select a feed to inspect its sync history.</p>
      ) : (
        <div className="space-y-6">
          <div className="flex flex-wrap items-start justify-between gap-3 rounded-2xl border border-border/70 bg-slate-50/70 p-4">
            <div className="space-y-3">
              <div className="flex flex-wrap items-center gap-2">
                <Badge variant="outline">{getThreatFeedTypeLabel(liveFeed.type)}</Badge>
                <Badge variant={liveFeed.enabled ? 'default' : 'secondary'}>
                  {liveFeed.enabled ? 'Enabled' : 'Paused'}
                </Badge>
                <Badge variant="outline">{liveFeed.status}</Badge>
              </div>
              <p className="max-w-2xl break-all text-sm text-slate-700">{liveFeed.url ?? 'Manual feed without remote URL'}</p>
            </div>

            <div className="flex flex-wrap gap-2">
              {onEdit && (
                <Button variant="outline" size="sm" onClick={() => onEdit(liveFeed)}>
                  Edit Feed
                </Button>
              )}
              <Button size="sm" onClick={() => void handleSync()} disabled={syncing}>
                <Play className="mr-2 h-4 w-4" />
                {syncing ? 'Syncing…' : 'Sync Now'}
              </Button>
            </div>
          </div>

          <section className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <InfoCard title="Configuration">
              <InfoRow label="Sync Interval">{getThreatFeedIntervalLabel(liveFeed.sync_interval)}</InfoRow>
              <InfoRow label="Default Severity">{liveFeed.default_severity}</InfoRow>
              <InfoRow label="Default Confidence">{Math.round(liveFeed.default_confidence * 100)}%</InfoRow>
              <InfoRow label="Indicator Filter">
                {liveFeed.indicator_types.length > 0 ? liveFeed.indicator_types.join(', ') : 'All types'}
              </InfoRow>
              <InfoRow label="Tags">
                {liveFeed.default_tags.length > 0 ? liveFeed.default_tags.join(', ') : 'No defaults'}
              </InfoRow>
            </InfoCard>

            <InfoCard title="Sync State">
              <InfoRow label="Last Sync">
                {liveFeed.last_sync_at ? <RelativeTime date={liveFeed.last_sync_at} /> : 'Never'}
              </InfoRow>
              <InfoRow label="Last Status">{liveFeed.last_sync_status ?? 'Not synced yet'}</InfoRow>
              <InfoRow label="Next Sync">
                {liveFeed.next_sync_at ? formatDateTime(liveFeed.next_sync_at) : 'Manual only'}
              </InfoRow>
              <InfoRow label="Auth Type">{liveFeed.auth_type}</InfoRow>
            </InfoCard>
          </section>

          {liveFeed.last_error && (
            <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-sm text-red-800">
              <div className="mb-2 flex items-center gap-2 font-medium">
                <TriangleAlert className="h-4 w-4" />
                Last Sync Error
              </div>
              <p>{liveFeed.last_error}</p>
            </div>
          )}

          <section className="space-y-3 rounded-2xl border border-border/70 bg-background p-4">
            <div className="flex items-center gap-2">
              <RefreshCcw className="h-4 w-4 text-slate-700" />
              <h3 className="text-sm font-semibold text-slate-900">Import History</h3>
            </div>
            {historyQuery.isLoading ? (
              <p className="text-sm text-muted-foreground">Loading sync history…</p>
            ) : historyQuery.data?.data?.length ? (
              <div className="overflow-hidden rounded-2xl border border-border/70">
                <table className="min-w-full text-sm">
                  <thead className="bg-slate-50">
                    <tr className="border-b border-border/70">
                      <th className="px-4 py-2 text-left font-medium">Started</th>
                      <th className="px-4 py-2 text-left font-medium">Status</th>
                      <th className="px-4 py-2 text-left font-medium">Imported</th>
                      <th className="px-4 py-2 text-left font-medium">Duration</th>
                    </tr>
                  </thead>
                  <tbody>
                    {historyQuery.data.data.map((entry) => (
                      <tr key={entry.id} className="border-b border-border/60">
                        <td className="px-4 py-2">{formatDateTime(entry.started_at)}</td>
                        <td className="px-4 py-2">
                          <Badge variant="outline">{entry.status}</Badge>
                        </td>
                        <td className="px-4 py-2">{entry.indicators_imported}</td>
                        <td className="px-4 py-2">{formatDuration(entry.duration_ms)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">No sync executions recorded yet.</p>
            )}
          </section>

          <section className="space-y-3 rounded-2xl border border-border/70 bg-background p-4">
            <div className="flex items-center gap-2">
              <Clock3 className="h-4 w-4 text-slate-700" />
              <h3 className="text-sm font-semibold text-slate-900">Last Import Preview</h3>
            </div>
            {previewIndicators.length > 0 ? (
              <div className="overflow-hidden rounded-2xl border border-border/70">
                <table className="min-w-full text-sm">
                  <thead className="bg-slate-50">
                    <tr className="border-b border-border/70">
                      <th className="px-4 py-2 text-left font-medium">Type</th>
                      <th className="px-4 py-2 text-left font-medium">Value</th>
                      <th className="px-4 py-2 text-left font-medium">Severity</th>
                    </tr>
                  </thead>
                  <tbody>
                    {previewIndicators.map((indicator, index) => (
                      <tr key={`${indicator.id ?? index}`} className="border-b border-border/60">
                        <td className="px-4 py-2">{String(indicator.type ?? '—')}</td>
                        <td className="px-4 py-2 font-mono text-xs">{String(indicator.value ?? '—')}</td>
                        <td className="px-4 py-2">{String(indicator.severity ?? '—')}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">The last sync did not record a preview payload.</p>
            )}
          </section>
        </div>
      )}
    </DetailPanel>
  );
}

function InfoCard({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-3 rounded-2xl border border-border/70 bg-background p-4">
      <h3 className="text-sm font-semibold text-slate-900">{title}</h3>
      <div className="space-y-2 text-sm">{children}</div>
    </div>
  );
}

function InfoRow({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-muted-foreground">{label}</span>
      <span className="text-right font-medium text-slate-900">{children}</span>
    </div>
  );
}

function formatDuration(durationMs: number): string {
  if (durationMs < 1000) {
    return `${durationMs} ms`;
  }
  return `${(durationMs / 1000).toFixed(1)} s`;
}
