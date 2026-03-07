import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { WorkflowStepTimeline } from './workflow-step-timeline';
import type { StepDefinition, StepExecution } from '@/types/models';

const definitionSteps: StepDefinition[] = [
  { id: 'triage', name: 'Triage Alert', type: 'human_task' },
  { id: 'investigate', name: 'Investigate Threat', type: 'human_task' },
  { id: 'approve', name: 'Approve Remediation', type: 'condition' },
  { id: 'dry-run', name: 'Execute Dry Run', type: 'service_task' },
  { id: 'execute', name: 'Execute Remediation', type: 'service_task' },
  { id: 'verify', name: 'Verify', type: 'end' },
];

function buildExecution(overrides: Partial<StepExecution>): StepExecution {
  return {
    id: overrides.id ?? `${overrides.step_id ?? 'step'}-execution`,
    step_id: overrides.step_id ?? 'triage',
    step_name: overrides.step_name ?? 'Triage Alert',
    step_type: overrides.step_type ?? 'human_task',
    status: overrides.status ?? 'completed',
    started_at: overrides.started_at ?? '2026-03-07T10:00:00Z',
    completed_at: overrides.completed_at ?? '2026-03-07T10:15:00Z',
    duration_seconds: overrides.duration_seconds ?? 900,
    attempt: overrides.attempt ?? 1,
    input: overrides.input ?? null,
    output: overrides.output ?? null,
    error: overrides.error ?? null,
    assigned_to: overrides.assigned_to ?? null,
    completed_by: overrides.completed_by ?? 'John Doe',
  };
}

describe('WorkflowStepTimeline', () => {
  it('renders all steps from the definition', () => {
    render(
      <WorkflowStepTimeline
        steps={[]}
        currentStepId="investigate"
        definitionSteps={definitionSteps}
      />,
    );

    for (const step of definitionSteps) {
      expect(screen.getByText(step.name)).toBeInTheDocument();
    }
  });

  it('shows a green check icon for completed steps', () => {
    const { container } = render(
      <WorkflowStepTimeline
        steps={[buildExecution({ step_id: 'triage', step_name: 'Triage Alert' })]}
        currentStepId="investigate"
        definitionSteps={definitionSteps}
      />,
    );

    expect(container.querySelector('svg.text-green-500')).toBeTruthy();
  });

  it('shows the current step with a blue pulsing indicator', () => {
    const { container } = render(
      <WorkflowStepTimeline
        steps={[
          buildExecution({ step_id: 'investigate', step_name: 'Investigate Threat', status: 'running' }),
        ]}
        currentStepId="investigate"
        definitionSteps={definitionSteps}
      />,
    );

    expect(container.querySelector('.animate-ping.bg-blue-400')).toBeTruthy();
    expect(container.querySelector('.bg-blue-500')).toBeTruthy();
    expect(screen.getByText('In Progress')).toBeInTheDocument();
  });

  it('shows pending steps with gray outlined circles', () => {
    const { container } = render(
      <WorkflowStepTimeline
        steps={[]}
        currentStepId={null}
        definitionSteps={definitionSteps}
      />,
    );

    expect(container.querySelector('svg.text-gray-300')).toBeTruthy();
    expect(screen.getAllByText('Pending').length).toBeGreaterThan(0);
  });

  it('shows failed steps with a red icon and error text', () => {
    const { container } = render(
      <WorkflowStepTimeline
        steps={[
          buildExecution({
            step_id: 'approve',
            step_name: 'Approve Remediation',
            status: 'failed',
            completed_at: null,
            duration_seconds: null,
            error: 'Gateway timeout',
          }),
        ]}
        currentStepId={null}
        definitionSteps={definitionSteps}
      />,
    );

    expect(container.querySelector('svg.text-red-500')).toBeTruthy();
    expect(screen.getByText('Failed: Gateway timeout')).toBeInTheDocument();
  });

  it('shows completed step details including assignee and duration', () => {
    render(
      <WorkflowStepTimeline
        steps={[
          buildExecution({
            step_id: 'triage',
            step_name: 'Triage Alert',
            completed_by: 'John Doe',
            duration_seconds: 900,
          }),
        ]}
        currentStepId={null}
        definitionSteps={definitionSteps}
      />,
    );

    expect(screen.getByText(/Completed by John Doe/i)).toBeInTheDocument();
    expect(screen.getByText(/15min/i)).toBeInTheDocument();
  });
});
