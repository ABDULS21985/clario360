'use client';

import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';

import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { ExportMenu } from '@/components/cyber/export-menu';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { MITRECoverage, MITREFrameworkMeta, MITRETechniqueCoverage } from '@/types/cyber';

import { MitreCoverageStats } from './_components/mitre-coverage-stats';
import { MitreFilterBar, type MitreFilter } from './_components/mitre-filter-bar';
import { MitreLegend } from './_components/mitre-legend';
import { MitreMatrix } from './_components/mitre-matrix';
import { MitreTechniquePanel } from './_components/mitre-technique-panel';

type ExpandedTechniqueCoverage = MITRETechniqueCoverage & {
  tactic_id: string;
  tactic_name: string;
};

function expandCoverage(raw: MITRECoverage): MITRECoverage {
  const tacticNameMap = new Map(raw.tactics.map((tactic) => [tactic.id, tactic.name]));
  const techniques: ExpandedTechniqueCoverage[] = [];

  raw.techniques.forEach((technique) => {
    technique.tactic_ids.forEach((tacticId) => {
      techniques.push({
        ...technique,
        tactic_id: tacticId,
        tactic_name: tacticNameMap.get(tacticId) ?? tacticId,
      });
    });
  });

  return {
    ...raw,
    techniques: techniques as MITRECoverage['techniques'],
  };
}

export default function MitreCoveragePage() {
  const [activeFilter, setActiveFilter] = useState<MitreFilter>('all');
  const [search, setSearch] = useState('');
  const [selectedTechnique, setSelectedTechnique] = useState<MITRETechniqueCoverage | null>(null);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['cyber-mitre-coverage'],
    queryFn: () => apiGet<{ data: MITRECoverage }>(API_ENDPOINTS.CYBER_MITRE_COVERAGE),
    staleTime: 120_000,
  });

  const { data: metaData } = useQuery({
    queryKey: ['cyber-mitre-framework-meta'],
    queryFn: () => apiGet<{ data: MITREFrameworkMeta }>(API_ENDPOINTS.CYBER_MITRE_FRAMEWORK_META),
    staleTime: 3_600_000,
  });
  const frameworkMeta = metaData?.data;

  const coverage = useMemo(() => (data?.data ? expandCoverage(data.data) : null), [data?.data]);

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="MITRE ATT&CK"
          description="Track detection coverage, noisy techniques, and active gaps across the ATT&CK matrix."
          actions={
            coverage ? (
              <ExportMenu
                entityType="mitre-coverage"
                baseUrl={API_ENDPOINTS.CYBER_MITRE_COVERAGE}
                currentFilters={{}}
                totalCount={coverage.total_techniques}
                enabledFormats={['csv', 'json']}
                csvDataKey="techniques"
              />
            ) : undefined
          }
        />

        {isLoading ? (
          <div className="space-y-4">
            <LoadingSkeleton variant="card" />
            <LoadingSkeleton variant="card" />
          </div>
        ) : error || !coverage ? (
          <ErrorState message="Failed to load MITRE coverage." onRetry={() => void refetch()} />
        ) : (
          <>
            {frameworkMeta?.is_stale && (
              <div className="rounded-xl border border-yellow-200 bg-yellow-50 px-4 py-3 text-sm text-yellow-800">
                The embedded MITRE ATT&CK catalog (v{frameworkMeta.version}, updated {frameworkMeta.updated_at}) is {frameworkMeta.stale_days} days old. New techniques may be missing.
              </div>
            )}
            <MitreCoverageStats coverage={coverage} />
            <MitreFilterBar
              activeFilter={activeFilter}
              onFilterChange={setActiveFilter}
              search={search}
              onSearchChange={setSearch}
            />
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
        />
      </div>
    </PermissionRedirect>
  );
}
