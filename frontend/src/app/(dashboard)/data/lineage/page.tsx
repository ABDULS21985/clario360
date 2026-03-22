'use client';

import { useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { Button } from '@/components/ui/button';
import { LineageControls } from '@/app/(dashboard)/data/lineage/_components/lineage-controls';
import {
  LineageDag,
  type LineageDagApi,
  type LineageLayoutSnapshot,
  type LineageViewportState,
} from '@/app/(dashboard)/data/lineage/_components/lineage-dag';
import { LineageDetailPanel } from '@/app/(dashboard)/data/lineage/_components/lineage-detail-panel';
import { LineageImpactPanel } from '@/app/(dashboard)/data/lineage/_components/lineage-impact-panel';
import { LineageMinimap } from '@/app/(dashboard)/data/lineage/_components/lineage-minimap';
import { LineageSearch } from '@/app/(dashboard)/data/lineage/_components/lineage-search';
import { dataSuiteApi, type ImpactAnalysis, type LineageNode } from '@/lib/data-suite';
import { showApiError } from '@/lib/toast';

export default function DataLineagePage() {
  const searchParams = useSearchParams();
  const focusType = searchParams?.get('type');
  const focusId = searchParams?.get('id');
  const [direction, setDirection] = useState<'LR' | 'TB'>('LR');
  const [search, setSearch] = useState('');
  const [selectedNode, setSelectedNode] = useState<LineageNode | null>(null);
  const [impactMode, setImpactMode] = useState(false);
  const [impact, setImpact] = useState<ImpactAnalysis | null>(null);
  const [dagApi, setDagApi] = useState<LineageDagApi | null>(null);
  const [layout, setLayout] = useState<LineageLayoutSnapshot | null>(null);
  const [viewport, setViewport] = useState<LineageViewportState | null>(null);

  const graphQuery = useQuery({
    queryKey: ['data-lineage', focusType, focusId],
    queryFn: () => (focusType && focusId ? dataSuiteApi.getEntityLineageGraph(focusType, focusId) : dataSuiteApi.getLineageGraph()),
  });

  const handleSelectNode = async (node: LineageNode) => {
    setSelectedNode(node);
    if (!impactMode) {
      setImpact(null);
      return;
    }
    try {
      const result = await dataSuiteApi.getLineageImpact(node.type, node.entity_id);
      setImpact(result);
    } catch (error) {
      setImpact(null);
      showApiError(error);
    }
  };

  if (graphQuery.isLoading || !graphQuery.data) {
    return (
      <PermissionRedirect permission="data:read">
        <div className="space-y-6">
          <PageHeader title="Lineage" description="Loading lineage graph and relationship metadata." />
          <LoadingSkeleton variant="chart" />
        </div>
      </PermissionRedirect>
    );
  }

  if (graphQuery.error) {
    return (
      <PermissionRedirect permission="data:read">
        <ErrorState message={graphQuery.error instanceof Error ? graphQuery.error.message : 'Failed to load lineage.'} onRetry={() => void graphQuery.refetch()} />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader
          title="Lineage"
          description="End-to-end data flow from sources through pipelines and models to downstream consumers."
          actions={
            <div className="flex flex-wrap items-center gap-2">
              <LineageSearch value={search} onChange={setSearch} onSelectResult={(node) => void handleSelectNode(node)} />
              <LineageControls
                direction={direction}
                onDirectionChange={setDirection}
                onFit={() => dagApi?.fitToScreen()}
                onReset={() => {
                  setSearch('');
                  setSelectedNode(null);
                  setImpact(null);
                  setDirection('LR');
                  dagApi?.reset();
                }}
                onZoomIn={() => dagApi?.zoomIn()}
                onZoomOut={() => dagApi?.zoomOut()}
                onFullscreen={() => dagApi?.fullscreen()}
              />
              <Button type="button" variant={impactMode ? 'default' : 'outline'} onClick={() => {
                setImpactMode((current) => !current);
                setImpact(null);
              }}>
                Impact Analysis
              </Button>
            </div>
          }
        />

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-[1fr_340px]">
          <LineageDag
            graph={graphQuery.data}
            direction={direction}
            selectedNodeId={selectedNode?.id ?? null}
            search={search}
            impact={impact}
            onSelectNode={(node) => void handleSelectNode(node)}
            onReady={setDagApi}
            onLayoutChange={setLayout}
            onViewportChange={setViewport}
          />

          <div className="space-y-4">
            <LineageMinimap
              layout={layout}
              viewport={viewport}
              onNavigate={(x, y) => dagApi?.centerOn(x, y)}
            />
            <LineageDetailPanel node={selectedNode} />
            {impactMode ? <LineageImpactPanel impact={impact} /> : null}
          </div>
        </div>
      </div>
    </PermissionRedirect>
  );
}
