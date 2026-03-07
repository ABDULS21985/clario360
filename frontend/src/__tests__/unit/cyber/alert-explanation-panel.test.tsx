import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { AlertExplanationPanel } from '@/app/(dashboard)/cyber/alerts/[id]/_components/alert-explanation-panel';
import type { AlertExplanation } from '@/types/cyber';

const makeExplanation = (overrides: Partial<AlertExplanation> = {}): AlertExplanation => ({
  summary: 'Suspicious lateral movement detected from internal host.',
  reason: 'Multiple failed authentication attempts followed by successful login.',
  evidence: [],
  matched_conditions: ['Failed login count > 10', 'Unusual time of day'],
  confidence_factors: [
    { factor: 'Multiple failed logins', impact: 0.4, description: 'Strong indicator' },
    { factor: 'Off-hours access', impact: 0.3, description: 'Suspicious timing' },
  ],
  recommended_actions: ['Isolate host', 'Reset credentials'],
  false_positive_indicators: ['Known automated scan'],
  ...overrides,
});

describe('AlertExplanationPanel', () => {
  it('renders AI analysis summary', () => {
    render(<AlertExplanationPanel explanation={makeExplanation()} confidenceScore={80} />);
    expect(screen.getByText('Suspicious lateral movement detected from internal host.')).toBeInTheDocument();
  });

  it('renders detection reason', () => {
    render(<AlertExplanationPanel explanation={makeExplanation()} confidenceScore={70} />);
    expect(screen.getByText('Multiple failed authentication attempts followed by successful login.')).toBeInTheDocument();
  });

  it('renders matched conditions', () => {
    render(<AlertExplanationPanel explanation={makeExplanation()} confidenceScore={75} />);
    expect(screen.getByText('Failed login count > 10')).toBeInTheDocument();
    expect(screen.getByText('Unusual time of day')).toBeInTheDocument();
  });

  it('renders recommended actions', () => {
    render(<AlertExplanationPanel explanation={makeExplanation()} confidenceScore={75} />);
    expect(screen.getByText('Isolate host')).toBeInTheDocument();
    expect(screen.getByText('Reset credentials')).toBeInTheDocument();
  });

  it('renders false positive indicators section', () => {
    render(<AlertExplanationPanel explanation={makeExplanation()} confidenceScore={75} />);
    expect(screen.getByText('Known automated scan')).toBeInTheDocument();
  });

  it('does not render matched conditions section when empty', () => {
    render(<AlertExplanationPanel explanation={makeExplanation({ matched_conditions: [] })} confidenceScore={75} />);
    expect(screen.queryByText('Matched Conditions')).not.toBeInTheDocument();
  });

  it('renders the confidence gauge with correct score', () => {
    render(<AlertExplanationPanel explanation={makeExplanation()} confidenceScore={92} />);
    expect(screen.getByText('92%')).toBeInTheDocument();
  });
});
