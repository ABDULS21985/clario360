'use client';

import { useQuery } from '@tanstack/react-query';
import { Boxes, FolderOpen, Package } from 'lucide-react';
import { KpiCard } from '@/components/shared/kpi-card';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { SectionCard } from '@/components/suites/section-card';
import { API_ENDPOINTS } from '@/lib/constants';
import { fetchSuitePaginated } from '@/lib/suite-api';
import { isEmptyObject } from '@/lib/suite-utils';
import { truncate } from '@/lib/utils';
import type { Dataset, DataSource } from '@/types/suites';

export default function DataDarkDataPage() {
  const sourcesQuery = useQuery({
    queryKey: ['data-dark-data', 'sources'],
    queryFn: () =>
      fetchSuitePaginated<DataSource>(API_ENDPOINTS.DATA_SOURCES, {
        page: 1,
        per_page: 200,
        sort: 'updated_at',
        order: 'desc',
      }),
  });
  const datasetsQuery = useQuery({
    queryKey: ['data-dark-data', 'datasets'],
    queryFn: () =>
      fetchSuitePaginated<Dataset>(API_ENDPOINTS.DATA_DATASETS, {
        page: 1,
        per_page: 200,
        sort: 'updated_at',
        order: 'desc',
      }),
  });

  if (sourcesQuery.isLoading && datasetsQuery.isLoading) {
    return (
      <PermissionRedirect permission="data:read">
        <div className="space-y-6">
          <PageHeader title="Dark Data" description="Derived view of undocumented sources and under-modeled datasets." />
          <LoadingSkeleton variant="card" count={4} />
        </div>
      </PermissionRedirect>
    );
  }

  if (sourcesQuery.error && datasetsQuery.error) {
    return (
      <PermissionRedirect permission="data:read">
        <ErrorState
          message="Failed to load dark data indicators."
          onRetry={() => {
            void sourcesQuery.refetch();
            void datasetsQuery.refetch();
          }}
        />
      </PermissionRedirect>
    );
  }

  const sources = sourcesQuery.data?.data ?? [];
  const datasets = datasetsQuery.data?.data ?? [];
  const undocumentedSources = sources.filter((source) => isEmptyObject(source.schema_metadata));
  const schemaLightDatasets = datasets.filter((dataset) => isEmptyObject(dataset.schema_definition));
  const lineageGaps = datasets.filter((dataset) => isEmptyObject(dataset.lineage));

  return (
    <PermissionRedirect permission="data:read">
      <div className="space-y-6">
        <PageHeader
          title="Dark Data"
          description="Derived governance view over sources and datasets with weak metadata, weak lineage, or low documentation coverage."
        />

        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <KpiCard title="Undocumented Sources" value={undocumentedSources.length} icon={FolderOpen} iconColor="text-orange-600" />
          <KpiCard title="Schema-Light Datasets" value={schemaLightDatasets.length} icon={Boxes} iconColor="text-violet-600" />
          <KpiCard title="Lineage Gaps" value={lineageGaps.length} icon={Package} iconColor="text-red-600" />
          <KpiCard title="Coverage Scope" value={(sources.length + datasets.length).toLocaleString()} icon={Package} iconColor="text-blue-600" description="Sources and datasets inspected from live APIs" />
        </div>

        <div className="grid gap-4 xl:grid-cols-3">
          <SectionCard title="Undocumented Sources" description="Sources missing schema metadata." contentClassName="space-y-3">
            {undocumentedSources.length === 0 ? (
              <p className="text-sm text-muted-foreground">No undocumented sources detected in the current window.</p>
            ) : (
              undocumentedSources.slice(0, 10).map((source) => (
                <div key={source.id} className="rounded-lg border px-4 py-3">
                  <p className="font-medium">{source.name}</p>
                  <p className="text-xs capitalize text-muted-foreground">{source.type.replace(/_/g, ' ')} • {source.status}</p>
                </div>
              ))
            )}
          </SectionCard>

          <SectionCard title="Schema-Light Datasets" description="Datasets with empty schema definitions." contentClassName="space-y-3">
            {schemaLightDatasets.length === 0 ? (
              <p className="text-sm text-muted-foreground">All datasets on this sample include schema definitions.</p>
            ) : (
              schemaLightDatasets.slice(0, 10).map((dataset) => (
                <div key={dataset.id} className="rounded-lg border px-4 py-3">
                  <p className="font-medium">{dataset.name}</p>
                  <p className="text-xs text-muted-foreground">v{dataset.version} • {dataset.source_name ?? 'Unknown source'}</p>
                  {dataset.description ? <p className="mt-2 text-xs text-muted-foreground">{truncate(dataset.description, 120)}</p> : null}
                </div>
              ))
            )}
          </SectionCard>

          <SectionCard title="Lineage Gaps" description="Datasets that are not yet mapped into lineage flows." contentClassName="space-y-3">
            {lineageGaps.length === 0 ? (
              <p className="text-sm text-muted-foreground">No lineage gaps were detected in the current sample.</p>
            ) : (
              lineageGaps.slice(0, 10).map((dataset) => (
                <div key={dataset.id} className="rounded-lg border px-4 py-3">
                  <p className="font-medium">{dataset.name}</p>
                  <p className="text-xs text-muted-foreground">{dataset.status} • {dataset.source_name ?? 'Unknown source'}</p>
                </div>
              ))
            )}
          </SectionCard>
        </div>
      </div>
    </PermissionRedirect>
  );
}
