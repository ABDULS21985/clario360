import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MitreCoverageStats } from '@/app/(dashboard)/cyber/mitre/_components/mitre-coverage-stats';
import type { MITRECoverage } from '@/types/cyber';

function makeCoverage(overrides: Partial<MITRECoverage> = {}): MITRECoverage {
  return {
    tactics: [
      { id: 'TA0002', name: 'Execution', short_name: 'execution', technique_count: 20, covered_count: 12 },
      { id: 'TA0001', name: 'Initial Access', short_name: 'initial-access', technique_count: 15, covered_count: 8 },
    ],
    techniques: [],
    total_techniques: 201,
    covered_techniques: 94,
    coverage_percent: 46.77,
    active_techniques: 62,
    passive_techniques: 32,
    critical_gap_count: 7,
    ...overrides,
  };
}

describe('MitreCoverageStats', () => {
  it('displays coverage fraction in KPI card', () => {
    render(<MitreCoverageStats coverage={makeCoverage()} />);
    // KpiCard value is `94/201`
    expect(screen.getByText('94/201')).toBeTruthy();
  });

  it('displays coverage percentage in description', () => {
    render(<MitreCoverageStats coverage={makeCoverage()} />);
    // Description shows "46.8% of techniques covered"
    expect(screen.getByText(/46\.8% of techniques covered/)).toBeTruthy();
  });

  it('displays active techniques count', () => {
    render(<MitreCoverageStats coverage={makeCoverage()} />);
    expect(screen.getByText('62')).toBeTruthy();
  });

  it('displays passive techniques count', () => {
    render(<MitreCoverageStats coverage={makeCoverage()} />);
    expect(screen.getByText('32')).toBeTruthy();
  });

  it('displays critical gap count', () => {
    render(<MitreCoverageStats coverage={makeCoverage()} />);
    expect(screen.getByText('7')).toBeTruthy();
  });

  it('displays overridden critical gap count', () => {
    render(<MitreCoverageStats coverage={makeCoverage({ critical_gap_count: 15 })} />);
    expect(screen.getByText('15')).toBeTruthy();
  });
});
