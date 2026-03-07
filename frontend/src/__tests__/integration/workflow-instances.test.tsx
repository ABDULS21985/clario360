import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithQuery } from '@/__tests__/utils/render-with-query';
import { WorkflowsPageClient } from '@/app/(dashboard)/workflows/workflows-page-client';
import { WorkflowInstancePageClient } from '@/app/(dashboard)/workflows/[id]/workflow-instance-page-client';
import type { StepDefinition, StepExecution, WorkflowInstance } from '@/types/models';

const API_URL = 'http://localhost:8080';
const pushMock = vi.fn();

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: pushMock, replace: vi.fn() }),
  usePathname: () => '/workflows',
  useSearchParams: () => new URLSearchParams(),
  useParams: () => ({ id: 'instance-1' }),
}));

let instances: WorkflowInstance[] = [];
let instanceDetail: WorkflowInstance;
let history: StepExecution[] = [];

const definitionSteps: StepDefinition[] = [
  { id: 'triage', name: 'Triage Alert', type: 'human_task' },
  { id: 'approve', name: 'Approve Remediation', type: 'condition' },
];

function buildInstance(overrides: Partial<WorkflowInstance>): WorkflowInstance {
  return {
    id: overrides.id ?? crypto.randomUUID(),
    definition_id: overrides.definition_id ?? 'workflow-1',
    definition_name: overrides.definition_name ?? 'Alert workflow',
    tenant_id: overrides.tenant_id ?? 'tenant-1',
    status: overrides.status ?? 'running',
    current_step_id: overrides.current_step_id ?? 'triage',
    current_step_name: overrides.current_step_name ?? 'Triage Alert',
    total_steps: overrides.total_steps ?? 2,
    completed_steps: overrides.completed_steps ?? 0,
    started_at: overrides.started_at ?? '2026-03-07T10:00:00Z',
    completed_at: overrides.completed_at ?? null,
    started_by: overrides.started_by ?? null,
    started_by_name: overrides.started_by_name ?? null,
    variables: overrides.variables ?? { alert_id: 'ALT-1234' },
    step_outputs: overrides.step_outputs ?? {},
    definition_steps: overrides.definition_steps ?? definitionSteps,
    error_message: overrides.error_message ?? null,
    updated_at: overrides.updated_at ?? '2026-03-07T10:00:00Z',
  };
}

const server = setupServer(
  http.get(`${API_URL}/api/v1/workflows/instances`, () =>
    HttpResponse.json({
      data: instances,
      meta: { page: 1, per_page: 25, total: instances.length, total_pages: 1 },
    }),
  ),
  http.get(`${API_URL}/api/v1/workflows/instances/instance-1`, () =>
    HttpResponse.json(instanceDetail),
  ),
  http.get(`${API_URL}/api/v1/workflows/instances/instance-1/history`, () =>
    HttpResponse.json({ steps: history }),
  ),
  http.post(`${API_URL}/api/v1/workflows/instances/instance-1/cancel`, () => {
    instanceDetail = { ...instanceDetail, status: 'cancelled', current_step_id: null };
    return HttpResponse.json({});
  }),
  http.post(`${API_URL}/api/v1/workflows/instances/instance-1/retry`, () => {
    instanceDetail = { ...instanceDetail, status: 'running', current_step_id: 'triage' };
    return HttpResponse.json({});
  }),
  http.post(`${API_URL}/api/v1/workflows/instances/instance-1/suspend`, () => {
    instanceDetail = { ...instanceDetail, status: 'suspended' };
    return HttpResponse.json({});
  }),
  http.post(`${API_URL}/api/v1/workflows/instances/instance-1/resume`, () => {
    instanceDetail = { ...instanceDetail, status: 'running' };
    return HttpResponse.json({});
  }),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => {
  server.resetHandlers();
  pushMock.mockReset();
});
afterAll(() => server.close());

beforeEach(() => {
  instances = [
    buildInstance({ id: 'instance-1', definition_name: 'Alert workflow', status: 'running' }),
    buildInstance({ id: 'instance-2', definition_name: 'Contract approval', status: 'completed' }),
    buildInstance({ id: 'instance-3', definition_name: 'Meeting prep', status: 'failed' }),
  ];

  instanceDetail = buildInstance({ id: 'instance-1', definition_name: 'Alert workflow' });
  history = [
    {
      id: 'exec-1',
      step_id: 'triage',
      step_name: 'Triage Alert',
      step_type: 'human_task',
      status: 'running',
      started_at: '2026-03-07T10:00:00Z',
      completed_at: null,
      duration_seconds: null,
      attempt: 1,
      input: null,
      output: { valid: true },
      error: null,
      assigned_to: 'Amina Analyst',
      completed_by: null,
    },
  ];
});

describe('Workflow Instances', () => {
  it('loads workflow instances in the table', async () => {
    renderWithQuery(<WorkflowsPageClient />);

    expect(await screen.findByText('Alert workflow')).toBeInTheDocument();
    expect(screen.getByText('Contract approval')).toBeInTheDocument();
    expect(screen.getByText('Meeting prep')).toBeInTheDocument();
  });

  it('opens workflow detail when a row is clicked', async () => {
    const user = userEvent.setup();

    renderWithQuery(<WorkflowsPageClient />);

    await user.click(await screen.findByText('Alert workflow'));

    expect(pushMock).toHaveBeenCalledWith('/workflows/instance-1');
  });

  it('renders workflow instance detail with timeline and variables', async () => {
    renderWithQuery(<WorkflowInstancePageClient />);

    expect(await screen.findByText('Workflow Steps')).toBeInTheDocument();
    expect(screen.getAllByText('Triage Alert').length).toBeGreaterThan(0);
    expect(screen.getByText('Variables')).toBeInTheDocument();
    expect(screen.getByText('ALT-1234')).toBeInTheDocument();
  });

  it('cancels a running workflow after type confirmation', async () => {
    const user = userEvent.setup();

    renderWithQuery(<WorkflowInstancePageClient />);

    await user.click(await screen.findByRole('button', { name: /cancel workflow/i }));
    await user.type(await screen.findByPlaceholderText('CANCEL'), 'CANCEL');
    await user.click(screen.getByRole('button', { name: /cancel workflow/i }));

    await waitFor(() => {
      expect(screen.queryByRole('button', { name: /cancel workflow/i })).not.toBeInTheDocument();
    });
  });

  it('retries a failed workflow instance', async () => {
    instanceDetail = buildInstance({
      id: 'instance-1',
      definition_name: 'Alert workflow',
      status: 'failed',
      current_step_id: null,
    });

    const user = userEvent.setup();

    renderWithQuery(<WorkflowInstancePageClient />);

    await user.click(await screen.findByRole('button', { name: 'Retry' }));

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Suspend' })).toBeInTheDocument();
    });
  });
});
