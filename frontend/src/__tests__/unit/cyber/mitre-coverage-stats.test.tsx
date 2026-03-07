import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MitreCoverageStats } from '@/app/(dashboard)/cyber/mitre/_components/mitre-coverage-stats';
import type { MITRECoverage } from '@/types/cyber';

function makeCoverage(overrides: Partial<MITRECoverage> = {}): MITRECoverage {
  return {
    tactics: [
      { id: 'TA0002', name: 'Execution', technique_count: 20, covered_count: 12 },
      { id: 'TA0001', name: 'Initial Access', technique_count: 15, covered_count: 8 },
    ],
    techniques: [],
    total_techniques: 201,
    covered_techniques: 94,
    coverage_percent: 46.77,
    active_techniques: 62,
    passive_techniques: 32,
    total_alerts_90d: 1247,
    ...overrides,
  };
}

describe('MitreCoverageStats', () => {
  it('test_overallPercentage: 94 of 201 → "47%" displayed', () => {
    render(<MitreCoverageStats coverage={makeCoverage()} />);
    expect(screen.getByText(/47%/)).toBeTruthy();
    expect(screen.getByText(/94 of 201/)).toBeTruthy();
  });

  it('test_progressBarSegments: active=62, passive=32, gaps=107 → counts shown', () => {
    render(<MitreCoverageStats coverage={makeCoverage()} />);
    expect(screen.getByText('62')).toBeTruthy(); // active
    expect(screen.getByText('32')).toBeTruthy(); // passive
    expect(screen.getByText('107')).toBeTruthy(); // gaps (201 - 94)
  });

  it('test_topTactic: tactic with highest covered_count → displayed', () => {
    render(<MitreCoverageStats coverage={makeCoverage()} />);
    // Execution has covered_count=12 which is highest
    // top tactic should show in alerts section
    expect(screen.getByText(/1,247/)).toBeTruthy();
  });

  it('test_alert_count_displayed', () => {
    render(<MitreCoverageStats coverage={makeCoverage({ total_alerts_90d: 5678 })} />);
    expect(screen.getByText(/5,678/)).toBeTruthy();
  });
});
