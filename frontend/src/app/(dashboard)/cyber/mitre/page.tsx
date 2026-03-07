'use client';

import { useState } from 'react';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { API_ENDPOINTS } from '@/lib/constants';
import { ExportMenu } from '@/components/cyber/export-menu';
import type { MITRECoverage, MITRETechniqueCoverage } from '@/types/cyber';

import { MitreCoverageStats } from './_components/mitre-coverage-stats';
import { MitreFilterBar, type MitreFilter } from './_components/mitre-filter-bar';
import { MitreLegend } from './_components/mitre-legend';
import { MitreMatrix } from './_components/mitre-matrix';
import { MitreTechniquePanel } from './_components/mitre-technique-panel';

export default function MitrePage() {
  const [activeFilter, setActiveFilter] = useState<MitreFilter>('all');
  const [search, setSearch] = useState('');
  const [selectedTechnique, setSelectedTechnique] = useState<MITRETechniqueCoverage | null>(null);
  const [createRuleTechniqueId, setCreateRuleTechniqueId] = useState<string | null>(null);

  const {
    data: envelope,
    isLoading,
    error,
    mutate: refetch,
  } = useRealtimeData<{ data: MITRECoverage }>(API_ENDPOINTS.CYBER_MITRE_COVERAGE, {
    pollInterval: 120000,
  });

  const coverage = envelope?.data;

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-5">
        <PageHeader
          title="MITRE ATT&CK Coverage"
          description="Detection coverage across the MITRE ATT&CK framework"
          actions={
            coverage ? (
              <ExportMenu
                entityType="mitre-coverage"
                baseUrl={API_ENDPOINTS.CYBER_MITRE_COVERAGE}
                currentFilters={{}}
                totalCount={coverage.total_techniques}
                enabledFormats={['csv', 'json']}
              />
            ) : undefined
          }
        />

        {isLoading ? (
          <div className="space-y-4">
            <LoadingSkeleton variant="card" />
            <LoadingSkeleton variant="table-row" count={6} />
          </div>
        ) : error || !coverage ? (
          <ErrorState message="Failed to load MITRE coverage" onRetry={() => void refetch()} />
        ) : (
          <>
            <MitreCoverageStats coverage={coverage} />

            <div className="flex flex-wrap items-center justify-between gap-3">
              <MitreFilterBar
                activeFilter={activeFilter}
                onFilterChange={setActiveFilter}
                search={search}
                onSearchChange={setSearch}
              />
            </div>

            <MitreLegend />

            <MitreMatrix
              coverage={coverage}
              activeFilter={activeFilter}
              search={search}
              selectedTechnique={selectedTechnique}
              onSelectTechnique={setSelectedTechnique}
            />
          </>
        )}

        <MitreTechniquePanel
          technique={selectedTechnique}
          onClose={() => setSelectedTechnique(null)}
          onCreateRule={(techniqueId) => {
            setCreateRuleTechniqueId(techniqueId);
            // Navigate to rules page with pre-filled technique
            window.location.href = `/cyber/rules?create=1&mitre_technique_id=${techniqueId}`;
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
