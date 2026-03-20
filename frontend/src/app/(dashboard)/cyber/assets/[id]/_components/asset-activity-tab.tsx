'use client';

import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import { Timeline } from '@/components/shared/timeline';
import { Activity } from 'lucide-react';
import type { AlertTimelineEntry } from '@/types/cyber';

interface AssetActivityTabProps {
  assetId: string;
}

export function AssetActivityTab({ assetId }: AssetActivityTabProps) {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['asset-activity', assetId],
    queryFn: () =>
      apiGet<{ data: AlertTimelineEntry[] }>(
        `${API_ENDPOINTS.CYBER_ASSETS}/${assetId}/activity`,
      ),
  });

  if (isLoading) return <LoadingSkeleton variant="list-item" count={6} />;
  if (error) return <ErrorState message="Failed to load activity" onRetry={() => refetch()} />;

  const entries = data?.data ?? [];

  if (entries.length === 0) {
    return (
      <EmptyState
        icon={Activity}
        title="No activity"
        description="No activity has been recorded for this asset yet."
      />
    );
  }

  const items = entries.map((entry) => ({
    id: entry.id,
    title: entry.action.replace(/_/g, ' '),
    description: entry.description,
    timestamp: entry.created_at,
    user: entry.actor_name,
  }));

  return <Timeline items={items} />;
}
