'use client';

import { useEffect, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Plus, RotateCw } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { useDataTable } from '@/hooks/use-data-table';
import { apiDelete, apiGet, apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { parseApiError } from '@/lib/format';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams } from '@/types/table';
import type { ThreatFeedConfig, ThreatFeedSyncSummary } from '@/types/cyber';
import { Button } from '@/components/ui/button';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';

import { AddFeedDialog } from './_components/add-feed-dialog';
import { FeedDetail } from './_components/feed-detail';
import { FeedList } from './_components/feed-list';

async function fetchThreatFeeds(params: FetchParams): Promise<PaginatedResponse<ThreatFeedConfig>> {
  return apiGet<PaginatedResponse<ThreatFeedConfig>>(API_ENDPOINTS.CYBER_THREAT_FEEDS, {
    page: params.page,
    per_page: params.per_page,
    search: params.search || undefined,
    sort: params.sort || undefined,
    order: params.order || undefined,
  });
}

export default function CyberThreatFeedsPage() {
  const [selectedFeed, setSelectedFeed] = useState<ThreatFeedConfig | null>(null);
  const [editorFeed, setEditorFeed] = useState<ThreatFeedConfig | null>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  const [syncingFeedId, setSyncingFeedId] = useState<string | null>(null);
  const [deletingFeed, setDeletingFeed] = useState<ThreatFeedConfig | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  const queryClient = useQueryClient();

  const { tableProps, refetch } = useDataTable<ThreatFeedConfig>({
    fetchFn: fetchThreatFeeds,
    queryKey: 'cyber-threat-feeds',
    defaultPageSize: 20,
    defaultSort: { column: 'updated_at', direction: 'desc' },
  });

  // Keep selectedFeed in sync with fresh list data after refetch
  useEffect(() => {
    if (!selectedFeed) return;
    const fresh = tableProps.data.find((f) => f.id === selectedFeed.id);
    if (fresh && fresh.updated_at !== selectedFeed.updated_at) {
      setSelectedFeed(fresh);
    }
  }, [tableProps.data, selectedFeed]);

  async function handleSync(feed: ThreatFeedConfig) {
    try {
      setSyncingFeedId(feed.id);
      const response = await apiPost<{ data: ThreatFeedSyncSummary }>(API_ENDPOINTS.CYBER_THREAT_FEED_SYNC(feed.id));
      toast.success(`${response.data.indicators_imported} indicators imported from ${feed.name}`);
      // Invalidate history queries so the "Imported" column and detail panel refresh
      void queryClient.invalidateQueries({ queryKey: ['cyber-threat-feed-last-history'] });
      void queryClient.invalidateQueries({ queryKey: ['cyber-threat-feed-history'] });
      await refetch();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Unable to sync feed');
    } finally {
      setSyncingFeedId(null);
    }
  }

  async function handleDelete() {
    if (!deletingFeed) return;
    try {
      setDeleteLoading(true);
      await apiDelete(`${API_ENDPOINTS.CYBER_THREAT_FEEDS}/${deletingFeed.id}`);
      toast.success(`Feed "${deletingFeed.name}" deleted`);
      if (selectedFeed?.id === deletingFeed.id) {
        setSelectedFeed(null);
      }
      setDeletingFeed(null);
      await refetch();
    } catch (error) {
      toast.error(parseApiError(error));
    } finally {
      setDeleteLoading(false);
    }
  }

  return (
    <PermissionRedirect permission="cyber:manage">
      <div className="space-y-6">
        <PageHeader
          title="Threat Intelligence Feeds"
          description="Configure external IOC sources, control sync cadence, and inspect the last ingest preview before those indicators enter your tenant."
          actions={(
            <div className="flex flex-wrap items-center gap-2">
              {syncingFeedId && (
                <Button variant="outline" disabled>
                  <RotateCw className="mr-2 h-4 w-4 animate-spin" />
                  Syncing…
                </Button>
              )}
              <Button
                onClick={() => {
                  setEditorFeed(null);
                  setEditorOpen(true);
                }}
              >
                <Plus className="mr-2 h-4 w-4" />
                Add Feed
              </Button>
            </div>
          )}
        />

        <div className="rounded-[28px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-2 shadow-[var(--card-shadow)]">
          <FeedList
            tableProps={tableProps}
            onSelect={setSelectedFeed}
            onEdit={(feed) => {
              setEditorFeed(feed);
              setEditorOpen(true);
            }}
            onSync={(feed) => void handleSync(feed)}
            onDelete={setDeletingFeed}
          />
        </div>
      </div>

      <AddFeedDialog
        open={editorOpen}
        onOpenChange={setEditorOpen}
        feed={editorFeed}
        onSuccess={(feed) => {
          setSelectedFeed(feed);
          void refetch();
        }}
      />

      <FeedDetail
        open={Boolean(selectedFeed)}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedFeed(null);
          }
        }}
        feed={selectedFeed}
        onEdit={(feed) => {
          setEditorFeed(feed);
          setEditorOpen(true);
        }}
        onSynced={() => {
          void queryClient.invalidateQueries({ queryKey: ['cyber-threat-feed-last-history'] });
          void queryClient.invalidateQueries({ queryKey: ['cyber-threat-feed-history'] });
          void refetch();
        }}
      />

      <ConfirmDialog
        open={Boolean(deletingFeed)}
        onOpenChange={(open) => { if (!open) setDeletingFeed(null); }}
        title="Delete threat feed"
        description={`Are you sure you want to delete "${deletingFeed?.name}"? Previously imported indicators will not be removed.`}
        confirmLabel={deleteLoading ? 'Deleting…' : 'Delete'}
        variant="destructive"
        loading={deleteLoading}
        onConfirm={() => void handleDelete()}
      />
    </PermissionRedirect>
  );
}
