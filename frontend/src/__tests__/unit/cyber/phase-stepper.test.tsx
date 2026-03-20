import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { PhaseStepper } from '@/app/(dashboard)/cyber/ctem/[id]/_components/phase-stepper';
import type { CTEMPhaseInfo } from '@/types/cyber';

const makePhases = (overrides: Partial<CTEMPhaseInfo>[] = []): CTEMPhaseInfo[] => [
  { phase: 'scoping', status: 'completed', ...overrides[0] },
  { phase: 'discovery', status: 'running', progress_percent: 60, ...overrides[1] },
  { phase: 'prioritization', status: 'pending', ...overrides[2] },
  { phase: 'validation', status: 'pending', ...overrides[3] },
  { phase: 'mobilization', status: 'pending', ...overrides[4] },
];

describe('PhaseStepper', () => {
  it('renders all 5 phase labels', () => {
    render(<PhaseStepper phases={makePhases()} />);
    expect(screen.getByText('Scoping')).toBeInTheDocument();
    expect(screen.getByText('Discovery')).toBeInTheDocument();
    expect(screen.getByText('Prioritization')).toBeInTheDocument();
    expect(screen.getByText('Validation')).toBeInTheDocument();
    expect(screen.getByText('Mobilization')).toBeInTheDocument();
  });

  it('shows spinner for running phase', () => {
    const { container } = render(<PhaseStepper phases={makePhases()} currentPhase="discovery" />);
    // Loader2 with animate-spin should be rendered
    const spinner = container.querySelector('.animate-spin');
    expect(spinner).toBeInTheDocument();
  });

  it('renders progress bar for running phase with progress_percent', () => {
    const { container } = render(<PhaseStepper phases={makePhases()} currentPhase="discovery" />);
    const progressBar = container.querySelector('[style*="width: 60%"]');
    expect(progressBar).toBeInTheDocument();
  });

  it('handles empty phases array', () => {
    render(<PhaseStepper phases={[]} />);
    // all phases show as pending - just check labels are there
    expect(screen.getByText('Scoping')).toBeInTheDocument();
  });

  it('applies failed styling for failed phase', () => {
    const phases: CTEMPhaseInfo[] = [
      { phase: 'scoping', status: 'failed' },
      { phase: 'discovery', status: 'pending' },
      { phase: 'prioritization', status: 'pending' },
      { phase: 'validation', status: 'pending' },
      { phase: 'mobilization', status: 'pending' },
    ];
    render(<PhaseStepper phases={phases} />);
    const failedLabel = screen.getByText('Scoping');
    expect(failedLabel.className).toContain('red');
  });
});
