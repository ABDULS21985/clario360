'use client';

import { GaugeChart } from '@/components/shared/charts/gauge-chart';
import { BarChart } from '@/components/shared/charts/bar-chart';
import { KpiCard } from '@/components/shared/kpi-card';
import type { MITRECoverage } from '@/types/cyber';

export function MitreCoverageStats({ coverage }: { coverage: MITRECoverage }) {
  const tacticData = coverage.tactics.map((tactic) => ({
    tactic: tactic.short_name ?? tactic.name,
    covered: tactic.covered_count,
    total: tactic.technique_count,
  }));

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <KpiCard title="Coverage" value={`${coverage.covered_techniques}/${coverage.total_techniques}`} description={`${coverage.coverage_percent.toFixed(1)}% of techniques covered`} />
        <KpiCard title="Active Techniques" value={coverage.active_techniques} description="Covered techniques with recent alert activity" />
        <KpiCard title="Passive Techniques" value={coverage.passive_techniques} description="Covered techniques without recent alert activity" />
        <KpiCard title="Critical Gaps" value={coverage.critical_gap_count} description="Active threat techniques with no rule coverage" />
      </div>

      <div className="grid grid-cols-1 gap-6 xl:grid-cols-[280px_1fr]">
        <div className="rounded-[26px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-5 shadow-[var(--card-shadow)]">
          <p className="text-sm font-medium text-slate-900">Overall Coverage</p>
          <p className="mt-1 text-sm text-muted-foreground">Percentage of ATT&CK techniques currently covered by active detection content.</p>
          <div className="mt-4">
            <GaugeChart value={coverage.coverage_percent} label="Coverage" max={100} />
          </div>
        </div>
        <BarChart
          data={tacticData}
          xKey="tactic"
          yKeys={[
            { key: 'covered', label: 'Covered', color: '#0f766e' },
            { key: 'total', label: 'Total', color: '#cbd5e1' },
          ]}
          title="Coverage By Tactic"
          height={280}
        />
      </div>
    </div>
  );
}
