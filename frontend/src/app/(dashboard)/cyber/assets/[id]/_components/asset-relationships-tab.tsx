'use client';

import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import { GitBranch } from 'lucide-react';
import { RelationshipGraph } from './relationship-graph';
import type { CyberAsset, AssetRelationship } from '@/types/cyber';

interface AssetRelationshipsTabProps {
  asset: CyberAsset;
}

export function AssetRelationshipsTab({ asset }: AssetRelationshipsTabProps) {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['asset-relationships', asset.id],
    queryFn: () =>
      apiGet<{ data: AssetRelationship[] }>(
        `${API_ENDPOINTS.CYBER_ASSETS}/${asset.id}/relationships`,
      ),
  });

  if (isLoading) return <LoadingSkeleton variant="card" />;
  if (error) return <ErrorState message="Failed to load relationships" onRetry={() => refetch()} />;

  const relationships = data?.data ?? [];

  if (relationships.length === 0) {
    return (
      <EmptyState
        icon={GitBranch}
        title="No relationships"
        description="No asset relationships have been discovered. Run a network scan to detect connections between assets."
      />
    );
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        {relationships.length} relationship{relationships.length !== 1 ? 's' : ''} found.
        Drag nodes to explore. Scroll to zoom.
      </p>
      <RelationshipGraph
        assetId={asset.id}
        assetName={asset.name}
        assetType={asset.type}
        relationships={relationships}
        height={500}
      />
    </div>
  );
}
