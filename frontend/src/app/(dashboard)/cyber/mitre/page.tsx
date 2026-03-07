'use client';

import { useState } from 'react';
import { Shield, RefreshCw, Search } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { API_ENDPOINTS } from '@/lib/constants';
import { timeAgo } from '@/lib/utils';
import type { MITRECoverage, MITRETechniqueCoverage } from '@/types/cyber';

function CoverageBar({ covered, total }: { covered: number; total: number }) {
  const pct = total > 0 ? (covered / total) * 100 : 0;
  const color = pct >= 80 ? 'bg-green-500' : pct >= 50 ? 'bg-amber-500' : 'bg-red-500';
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-20 overflow-hidden rounded-full bg-muted">
        <div className={`h-full rounded-full transition-all ${color}`} style={{ width: `${pct}%` }} />
      </div>
      <span className="text-xs tabular-nums text-muted-foreground">{covered}/{total}</span>
    </div>
  );
}

function TechniqueCard({
  technique,
  selected,
  onClick,
}: {
  technique: MITRETechniqueCoverage;
  selected: boolean;
  onClick: () => void;
}) {
  const alertColor =
    technique.alert_count === 0
      ? 'bg-muted/20'
      : technique.alert_count >= 5
      ? 'bg-red-100 dark:bg-red-950/30'
      : 'bg-amber-100 dark:bg-amber-950/30';

  return (
    <button
      onClick={onClick}
      className={`w-full rounded-lg border p-2.5 text-left transition-all hover:shadow-sm ${
        selected ? 'border-primary ring-1 ring-primary' : 'hover:border-muted-foreground/40'
      } ${alertColor}`}
    >
      <div className="mb-1 flex items-start justify-between gap-1">
        <span className="font-mono text-xs font-bold text-muted-foreground">{technique.technique_id}</span>
        {technique.has_detection ? (
          <span className="h-2 w-2 shrink-0 rounded-full bg-green-500" title="Has detection rule" />
        ) : (
          <span className="h-2 w-2 shrink-0 rounded-full bg-muted-foreground/30" title="No detection" />
        )}
      </div>
      <p className="line-clamp-2 text-xs leading-tight">{technique.technique_name}</p>
      {technique.alert_count > 0 && (
        <div className="mt-1.5 flex items-center gap-1">
          <Badge variant="outline" className="h-4 px-1 py-0 text-[10px]">
            {technique.alert_count} alert{technique.alert_count !== 1 ? 's' : ''}
          </Badge>
        </div>
      )}
    </button>
  );
}

export default function MitrePage() {
  const [search, setSearch] = useState('');
  const [selectedTactic, setSelectedTactic] = useState<string | null>(null);
  const [selectedTech, setSelectedTech] = useState<MITRETechniqueCoverage | null>(null);

  const {
    data: envelope,
    isLoading,
    error,
    mutate: refetch,
  } = useRealtimeData<{ data: MITRECoverage }>(API_ENDPOINTS.CYBER_MITRE_COVERAGE, {
    pollInterval: 120000,
  });

  const coverage = envelope?.data;

  const filteredTactics = coverage?.tactics.filter((t) =>
    !selectedTactic || t.id === selectedTactic,
  );

  const filteredTechniques = coverage?.techniques.filter((t) => {
    const matchesTactic = !selectedTactic || t.tactic_id === selectedTactic;
    const matchesSearch = !search || t.technique_name.toLowerCase().includes(search.toLowerCase()) || t.technique_id.toLowerCase().includes(search.toLowerCase());
    return matchesTactic && matchesSearch;
  });

  const groupedByTactic = filteredTechniques?.reduce<Record<string, MITRETechniqueCoverage[]>>((acc, t) => {
    if (!acc[t.tactic_id]) acc[t.tactic_id] = [];
    acc[t.tactic_id].push(t);
    return acc;
  }, {});

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="MITRE ATT&CK Coverage"
          description="Visual map of detection coverage across the MITRE ATT&CK framework tactics and techniques"
          actions={
            <Button variant="outline" size="sm" onClick={() => void refetch()}>
              <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
              Refresh
            </Button>
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
            {/* Coverage summary */}
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              {[
                { label: 'Total Techniques', value: coverage.total_techniques, color: 'text-muted-foreground' },
                { label: 'Covered', value: coverage.covered_techniques, color: 'text-green-600' },
                { label: 'Uncovered', value: coverage.total_techniques - coverage.covered_techniques, color: 'text-red-600' },
                { label: 'Coverage', value: `${coverage.coverage_percent.toFixed(0)}%`, color: coverage.coverage_percent >= 70 ? 'text-green-600' : coverage.coverage_percent >= 40 ? 'text-amber-600' : 'text-red-600' },
              ].map(({ label, value, color }) => (
                <div key={label} className="rounded-xl border bg-card p-4 text-center">
                  <p className={`text-2xl font-bold tabular-nums ${color}`}>{value}</p>
                  <p className="mt-0.5 text-xs text-muted-foreground">{label}</p>
                </div>
              ))}
            </div>

            {/* Tactic selector */}
            <div className="rounded-xl border bg-card p-4">
              <div className="mb-3 flex items-center justify-between">
                <p className="text-sm font-semibold">Tactics</p>
                {selectedTactic && (
                  <button
                    className="text-xs text-primary hover:underline"
                    onClick={() => { setSelectedTactic(null); setSelectedTech(null); }}
                  >
                    Show all
                  </button>
                )}
              </div>
              <div className="flex flex-wrap gap-2">
                {coverage.tactics.map((tactic) => {
                  const pct = tactic.technique_count > 0 ? Math.round((tactic.covered_count / tactic.technique_count) * 100) : 0;
                  const isSelected = selectedTactic === tactic.id;
                  return (
                    <button
                      key={tactic.id}
                      onClick={() => { setSelectedTactic(isSelected ? null : tactic.id); setSelectedTech(null); }}
                      className={`flex items-center gap-2 rounded-lg border px-3 py-2 text-left transition-colors ${
                        isSelected ? 'border-primary bg-primary/10' : 'hover:bg-muted/30'
                      }`}
                    >
                      <div>
                        <p className="text-xs font-semibold">{tactic.name}</p>
                        <CoverageBar covered={tactic.covered_count} total={tactic.technique_count} />
                      </div>
                      <span className={`text-xs font-bold ${pct >= 80 ? 'text-green-600' : pct >= 50 ? 'text-amber-600' : 'text-red-600'}`}>
                        {pct}%
                      </span>
                    </button>
                  );
                })}
              </div>
            </div>

            {/* Legend */}
            <div className="flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
              <div className="flex items-center gap-1.5">
                <span className="h-2.5 w-2.5 rounded-full bg-green-500" /> Has detection rule
              </div>
              <div className="flex items-center gap-1.5">
                <span className="h-2.5 w-2.5 rounded-full bg-muted-foreground/30" /> No detection
              </div>
              <div className="flex items-center gap-1.5">
                <span className="h-3 w-3 rounded bg-red-100 dark:bg-red-950/30 border border-red-200" /> 5+ alerts
              </div>
              <div className="flex items-center gap-1.5">
                <span className="h-3 w-3 rounded bg-amber-100 dark:bg-amber-950/30 border border-amber-200" /> 1-4 alerts
              </div>
            </div>

            {/* Search */}
            <div className="relative max-w-sm">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="Search techniques…"
                className="pl-9"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>

            {/* Heatmap grid + detail panel */}
            <div className={`grid grid-cols-1 gap-6 ${selectedTech ? 'lg:grid-cols-3' : ''}`}>
              <div className={selectedTech ? 'lg:col-span-2' : ''}>
                {groupedByTactic && Object.entries(groupedByTactic).map(([tacticId, techniques]) => {
                  const tacticInfo = coverage.tactics.find((t) => t.id === tacticId);
                  return (
                    <div key={tacticId} className="mb-6">
                      <div className="mb-2 flex items-center gap-3">
                        <h3 className="text-sm font-semibold">{tacticInfo?.name ?? tacticId}</h3>
                        <span className="text-xs text-muted-foreground">
                          {techniques.filter((t) => t.has_detection).length}/{techniques.length} covered
                        </span>
                        <div className="flex-1">
                          <div className="h-1 overflow-hidden rounded-full bg-muted">
                            <div
                              className="h-full rounded-full bg-green-500"
                              style={{
                                width: `${techniques.length > 0 ? (techniques.filter((t) => t.has_detection).length / techniques.length) * 100 : 0}%`,
                              }}
                            />
                          </div>
                        </div>
                      </div>
                      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
                        {techniques.map((tech) => (
                          <TechniqueCard
                            key={tech.technique_id}
                            technique={tech}
                            selected={selectedTech?.technique_id === tech.technique_id}
                            onClick={() => setSelectedTech(selectedTech?.technique_id === tech.technique_id ? null : tech)}
                          />
                        ))}
                      </div>
                    </div>
                  );
                })}
              </div>

              {/* Detail panel */}
              {selectedTech && (
                <div className="sticky top-4 h-fit rounded-xl border bg-card p-5 shadow-sm">
                  <div className="mb-4 flex items-start justify-between">
                    <div>
                      <span className="font-mono text-sm font-bold text-primary">{selectedTech.technique_id}</span>
                      <h3 className="mt-1 font-semibold">{selectedTech.technique_name}</h3>
                      <p className="text-xs text-muted-foreground">Tactic: {selectedTech.tactic_name}</p>
                    </div>
                    <button
                      onClick={() => setSelectedTech(null)}
                      className="text-muted-foreground hover:text-foreground"
                    >
                      ×
                    </button>
                  </div>

                  <div className="space-y-3">
                    <div className="flex items-center justify-between rounded-lg border p-3">
                      <span className="text-sm text-muted-foreground">Detection</span>
                      {selectedTech.has_detection ? (
                        <div className="flex items-center gap-1.5 text-green-600">
                          <Shield className="h-4 w-4" />
                          <span className="text-sm font-medium">{selectedTech.rule_count} rule{selectedTech.rule_count !== 1 ? 's' : ''}</span>
                        </div>
                      ) : (
                        <span className="text-sm font-medium text-red-600">No rules</span>
                      )}
                    </div>
                    <div className="flex items-center justify-between rounded-lg border p-3">
                      <span className="text-sm text-muted-foreground">Alerts</span>
                      <span className={`text-sm font-bold ${selectedTech.alert_count > 0 ? 'text-orange-600' : 'text-green-600'}`}>
                        {selectedTech.alert_count}
                      </span>
                    </div>
                    {selectedTech.last_alert && (
                      <div className="flex items-center justify-between rounded-lg border p-3">
                        <span className="text-sm text-muted-foreground">Last Alert</span>
                        <span className="text-sm">{timeAgo(selectedTech.last_alert)}</span>
                      </div>
                    )}
                  </div>

                  {!selectedTech.has_detection && (
                    <div className="mt-4 rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950/20">
                      <p className="text-xs font-medium text-amber-800 dark:text-amber-400">Coverage Gap</p>
                      <p className="mt-1 text-xs text-amber-700 dark:text-amber-500">
                        No detection rules configured for this technique. Consider adding a rule to improve coverage.
                      </p>
                    </div>
                  )}

                  <a
                    href={`https://attack.mitre.org/techniques/${selectedTech.technique_id}/`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="mt-4 block text-center text-xs text-primary hover:underline"
                  >
                    View on MITRE ATT&CK →
                  </a>
                </div>
              )}
            </div>
          </>
        )}
      </div>
    </PermissionRedirect>
  );
}
