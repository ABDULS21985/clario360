import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MitreCell } from '@/app/(dashboard)/cyber/mitre/_components/mitre-cell';
import type { MITRETechniqueCoverage } from '@/types/cyber';

// TooltipProvider requires wrapping
import { TooltipProvider } from '@/components/ui/tooltip';

function wrap(ui: React.ReactNode) {
  return render(<TooltipProvider>{ui}</TooltipProvider>);
}

function makeTech(overrides: Partial<MITRETechniqueCoverage> = {}): MITRETechniqueCoverage {
  return {
    technique_id: 'T1059',
    technique_name: 'Command and Scripting Interpreter',
    tactic_id: 'TA0002',
    tactic_name: 'Execution',
    rule_count: 0,
    alert_count: 0,
    has_detection: false,
    ...overrides,
  };
}

describe('MitreCell', () => {
  it('test_activeCoverage_greenBackground: rule_count>0, alert_count>0 → green classes', () => {
    const tech = makeTech({ rule_count: 2, alert_count: 5, has_detection: true });
    const { container } = wrap(
      <MitreCell technique={tech} selected={false} highlighted={false} onSelect={vi.fn()} />,
    );
    const btn = container.querySelector('button');
    expect(btn?.className).toContain('bg-green-100');
  });

  it('test_passiveCoverage_yellowBackground: rule_count>0, alert_count=0 → yellow classes', () => {
    const tech = makeTech({ rule_count: 1, alert_count: 0, has_detection: true });
    const { container } = wrap(
      <MitreCell technique={tech} selected={false} highlighted={false} onSelect={vi.fn()} />,
    );
    const btn = container.querySelector('button');
    expect(btn?.className).toContain('bg-yellow-50');
  });

  it('test_gap_redBackground: rule_count=0 → red classes + warning icon', () => {
    const tech = makeTech({ rule_count: 0, alert_count: 0, has_detection: false });
    const { container } = wrap(
      <MitreCell technique={tech} selected={false} highlighted={false} onSelect={vi.fn()} />,
    );
    const btn = container.querySelector('button');
    expect(btn?.className).toContain('bg-red-50');
    // Warning icon (AlertTriangle) should be present
    expect(container.querySelector('svg')).toBeTruthy();
  });

  it('test_ruleDots: rule_count=3 → 3 dots (●●●)', () => {
    const tech = makeTech({ rule_count: 3, alert_count: 3, has_detection: true });
    wrap(
      <MitreCell technique={tech} selected={false} highlighted={false} onSelect={vi.fn()} />,
    );
    // The dots span should contain ●●●
    const dots = screen.queryByText('●●●');
    expect(dots).toBeTruthy();
  });

  it('test_ruleDots_overflow: rule_count=5 → ●3+', () => {
    const tech = makeTech({ rule_count: 5, alert_count: 5, has_detection: true });
    wrap(
      <MitreCell technique={tech} selected={false} highlighted={false} onSelect={vi.fn()} />,
    );
    expect(screen.getByText('●3+')).toBeTruthy();
  });

  it('test_clickOpensPanel: click cell → onSelect callback fired', () => {
    const onSelect = vi.fn();
    const tech = makeTech({ rule_count: 1, alert_count: 1, has_detection: true });
    const { container } = wrap(
      <MitreCell technique={tech} selected={false} highlighted={false} onSelect={onSelect} />,
    );
    const btn = container.querySelector('button');
    fireEvent.click(btn!);
    expect(onSelect).toHaveBeenCalledWith(tech);
  });

  it('test_techniqueId_rendered', () => {
    const tech = makeTech({ technique_id: 'T1059' });
    wrap(<MitreCell technique={tech} selected={false} highlighted={false} onSelect={vi.fn()} />);
    expect(screen.getByText('T1059')).toBeTruthy();
  });
});
