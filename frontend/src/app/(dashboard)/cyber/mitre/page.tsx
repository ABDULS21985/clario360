'use client';

import { useState, useMemo } from 'react';
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

/** Shape the API actually returns for each technique in the coverage array */
interface APICoverageTechnique {
  technique_id: string;
  technique_name: string;
  tactic_ids: string[];
  has_detection: boolean;
  rule_count: number;
  rule_names: string[];
}

/** Shape of the full API response from the coverage endpoint */
interface APICoverageResponse {
  tactics: Array<{
    id: string;
    name: string;
    short_name?: string;
    technique_count: number;
    covered_count: number;
  }>;
  techniques: APICoverageTechnique[];
  total_techniques: number;
  covered_techniques: number;
  coverage_percent: number;
  active_techniques: number;
  passive_techniques: number;
}

/** Handles the legacy flat-array format from the old API. */
function transformLegacyCoverage(items: APICoverageTechnique[]): MITRECoverage {
  const techniques: MITRETechniqueCoverage[] = [];
  const tacticMap = new Map<string, { count: number; covered: number }>();

  for (const tech of items) {
    for (const tid of tech.tactic_ids ?? []) {
      techniques.push({
        technique_id: tech.technique_id,
        technique_name: tech.technique_name,
        tactic_id: tid,
        tactic_name: tid,
        rule_count: tech.rule_count,
        alert_count: 0,
        has_detection: tech.has_detection,
      });
      const entry = tacticMap.get(tid) ?? { count: 0, covered: 0 };
      entry.count++;
      if (tech.has_detection) entry.covered++;
      tacticMap.set(tid, entry);
    }
  }

  const total = items.length;
  const covered = items.filter((t) => t.has_detection).length;
  const active = items.filter((t) => t.has_detection && t.rule_count > 1).length;
  const tactics = Array.from(tacticMap.entries()).map(([id, c]) => ({
    id,
    name: id,
    technique_count: c.count,
    covered_count: c.covered,
  }));

  return {
    tactics,
    techniques,
    total_techniques: total,
    covered_techniques: covered,
    coverage_percent: total > 0 ? (covered / total) * 100 : 0,
    active_techniques: active,
    passive_techniques: covered - active,
  };
}

/**
 * Transforms the API response into the shape the matrix component expects.
 * Handles both the new aggregated format and the legacy flat-array format.
 */
function transformCoverage(raw: APICoverageResponse | APICoverageTechnique[]): MITRECoverage {
  // Legacy format: API returns a flat array of techniques instead of the structured object
  if (Array.isArray(raw)) {
    return transformLegacyCoverage(raw);
  }

  const rawTactics = raw.tactics ?? [];
  const rawTechniques = raw.techniques ?? [];

  const tacticNameMap = new Map(rawTactics.map((t) => [t.id, t.name]));

  // Expand each technique into per-tactic entries for the matrix component
  const techniques: MITRETechniqueCoverage[] = [];
  for (const tech of rawTechniques) {
    const tacticIds = tech.tactic_ids ?? [];
    for (const tacticId of tacticIds) {
      techniques.push({
        technique_id: tech.technique_id,
        technique_name: tech.technique_name,
        tactic_id: tacticId,
        tactic_name: tacticNameMap.get(tacticId) ?? tacticId,
        rule_count: tech.rule_count,
        alert_count: 0,
        has_detection: tech.has_detection,
      });
    }
  }

  return {
    tactics: rawTactics,
    techniques,
    total_techniques: raw.total_techniques ?? 0,
    covered_techniques: raw.covered_techniques ?? 0,
    coverage_percent: raw.coverage_percent ?? 0,
    active_techniques: raw.active_techniques ?? 0,
    passive_techniques: raw.passive_techniques ?? 0,
  };
}

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
  } = useRealtimeData<{ data: APICoverageResponse | APICoverageTechnique[] }>(API_ENDPOINTS.CYBER_MITRE_COVERAGE, {
    pollInterval: 120000,
  });

  const coverage = useMemo(
    () => (envelope?.data ? transformCoverage(envelope.data) : undefined),
    [envelope?.data],
  );

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
