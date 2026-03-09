'use client';

import type { MITRECoverage, MITRETechniqueCoverage } from '@/types/cyber';
import type { MitreFilter } from './mitre-filter-bar';
import { MitreTacticHeader } from './mitre-tactic-header';
import { MitreCell } from './mitre-cell';

interface MitreMatrixProps {
  coverage: MITRECoverage;
  activeFilter: MitreFilter;
  search: string;
  selectedTechnique: MITRETechniqueCoverage | null;
  onSelectTechnique: (technique: MITRETechniqueCoverage | null) => void;
}

function applyFilter(technique: MITRETechniqueCoverage, filter: MitreFilter): boolean {
  if (filter === 'covered') return technique.rule_count > 0;
  if (filter === 'gaps') return technique.rule_count === 0;
  if (filter === 'alerts') return technique.alert_count > 0;
  return true;
}

function applySearch(technique: MITRETechniqueCoverage, search: string): boolean {
  if (!search) return true;
  const q = search.toLowerCase();
  return (
    technique.technique_id.toLowerCase().includes(q) ||
    technique.technique_name.toLowerCase().includes(q)
  );
}

export function MitreMatrix({
  coverage,
  activeFilter,
  search,
  selectedTechnique,
  onSelectTechnique,
}: MitreMatrixProps) {
  // Group techniques by tactic
  const byTactic = (coverage.techniques ?? []).reduce<Record<string, MITRETechniqueCoverage[]>>(
    (acc, t) => {
      if (!acc[t.tactic_id]) acc[t.tactic_id] = [];
      acc[t.tactic_id].push(t);
      return acc;
    },
    {},
  );

  const tactics = coverage.tactics ?? [];
  const maxRows = tactics.reduce((max, tactic) => {
    const techniques = byTactic[tactic.id] ?? [];
    const filtered = techniques.filter(
      (t) => applyFilter(t, activeFilter) && applySearch(t, search),
    );
    return Math.max(max, filtered.length);
  }, 0);

  if (maxRows === 0) {
    return (
      <div className="flex h-32 items-center justify-center rounded-xl border text-muted-foreground">
        No techniques match the current filter.
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-xl border bg-card">
      {/* Sticky header row */}
      <div
        className="sticky top-0 z-10 grid border-b bg-card/95 backdrop-blur"
        style={{ gridTemplateColumns: `repeat(${tactics.length}, minmax(96px, 1fr))` }}
      >
        {tactics.map((tactic) => {
          const tacticTechniques = byTactic[tactic.id] ?? [];
          return (
            <div key={tactic.id} className="border-r last:border-r-0">
              <MitreTacticHeader
                id={tactic.id}
                name={tactic.name}
                shortName={tactic.short_name}
                covered={tactic.covered_count}
                total={tactic.technique_count}
              />
            </div>
          );
        })}
      </div>

      {/* Technique columns */}
      <div
        className="grid"
        style={{ gridTemplateColumns: `repeat(${tactics.length}, minmax(96px, 1fr))` }}
      >
        {tactics.map((tactic) => {
          const allTechniques = byTactic[tactic.id] ?? [];
          const filtered = allTechniques.filter(
            (t) => applyFilter(t, activeFilter) && applySearch(t, search),
          );

          return (
            <div key={tactic.id} className="border-r last:border-r-0">
              <div className="flex flex-col gap-0.5 p-1">
                {filtered.map((tech) => (
                  <MitreCell
                    key={tech.technique_id}
                    technique={tech}
                    selected={selectedTechnique?.technique_id === tech.technique_id}
                    highlighted={!!search && applySearch(tech, search)}
                    onSelect={(t) =>
                      onSelectTechnique(
                        selectedTechnique?.technique_id === t.technique_id ? null : t,
                      )
                    }
                  />
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
