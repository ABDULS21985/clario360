import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithQuery } from '@/__tests__/utils/render-with-query';
import { TaskDetailPageClient } from '@/app/(dashboard)/workflows/tasks/[id]/task-detail-page-client';
import type { HumanTask, StepDefinition, StepExecution, User } from '@/types/models';

const API_URL = 'http://localhost:8080';
const pushMock = vi.fn();

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: pushMock, replace: vi.fn() }),
  useParams: () => ({ id: 'task-1' }),
}));

const currentUser: User = {
  id: 'user-1',
  tenant_id: 'tenant-1',
  email: 'amina@example.com',
  first_name: 'Amina',
  last_name: 'Analyst',
  status: 'active',
  mfa_enabled: false,
  last_login_at: null,
  password_changed_at: '2026-03-07T10:00:00Z',
  roles: [
    {
      id: 'role-1',
      tenant_id: 'tenant-1',
      name: 'analyst',
      slug: 'analyst',
      description: '',
      permissions: [],
      is_system: false,
      created_at: '2026-03-07T10:00:00Z',
      updated_at: '2026-03-07T10:00:00Z',
    },
  ],
  created_at: '2026-03-07T10:00:00Z',
  updated_at: '2026-03-07T10:00:00Z',
};

vi.mock('@/hooks/use-auth', () => ({
  useAuth: () => ({
    user: currentUser,
  }),
}));

let task: HumanTask;
let completePayload: Record<string, unknown> | null = null;
let rejectPayload: Record<string, unknown> | null = null;
let delegatePayload: Record<string, unknown> | null = null;

const definitionSteps: StepDefinition[] = [
  { id: 'review', name: 'Review task', type: 'human_task' },
  { id: 'approve', name: 'Approve', type: 'condition' },
];

const history: StepExecution[] = [
  {
    id: 'history-1',
    step_id: 'review',
    step_name: 'Review task',
    step_type: 'human_task',
    status: 'running',
    started_at: '2026-03-07T10:00:00Z',
    completed_at: null,
    duration_seconds: null,
    attempt: 1,
    input: null,
    output: null,
    error: null,
    assigned_to: 'Amina Analyst',
    completed_by: null,
  },
];

function buildTask(overrides: Partial<HumanTask> = {}): HumanTask {
  return {
    id: 'task-1',
    name: 'Review remediation plan',
    description: 'Confirm the remediation plan is safe to run.',
    instance_id: 'instance-1',
    definition_name: 'Alert workflow',
    workflow_name: 'Alert workflow',
    step_id: 'review',
    status: 'claimed',
    priority: 1,
    form_schema: [
      { name: 'approved', type: 'boolean', label: 'Approved', required: true },
      {
        name: 'summary',
        type: 'text',
        label: 'Summary',
        required: true,
        placeholder: 'Add summary',
      },
    ],
    form_data: null,
    sla_deadline: null,
    sla_breached: false,
    claimed_by: currentUser.id,
    claimed_by_name: 'Amina Analyst',
    assignee_role: 'analyst',
    assignee_id: null,
    metadata: {},
    created_at: '2026-03-07T10:00:00Z',
    updated_at: '2026-03-07T10:00:00Z',
    ...overrides,
  };
}

const server = setupServer(
  http.get(`${API_URL}/api/v1/workflows/tasks/task-1`, () => HttpResponse.json(task)),
  http.get(`${API_URL}/api/v1/workflows/instances/instance-1`, () =>
    HttpResponse.json({
      id: 'instance-1',
      definition_id: 'workflow-1',
      definition_name: 'Alert workflow',
      tenant_id: 'tenant-1',
      status: 'running',
      current_step_id: 'review',
      current_step_name: 'Review task',
      total_steps: 2,
      completed_steps: 0,
      started_at: '2026-03-07T10:00:00Z',
      completed_at: null,
      started_by: null,
      started_by_name: null,
      variables: {},
      step_outputs: {},
      definition_steps: definitionSteps,
    }),
  ),
  http.get(`${API_URL}/api/v1/workflows/instances/instance-1/history`, () =>
    HttpResponse.json({ steps: history }),
  ),
  http.post(`${API_URL}/api/v1/workflows/tasks/task-1/complete`, async ({ request }) => {
    completePayload = (await request.json()) as Record<string, unknown>;
    task = { ...task, status: 'completed' };
    return HttpResponse.json({});
  }),
  http.post(`${API_URL}/api/v1/workflows/tasks/task-1/reject`, async ({ request }) => {
    rejectPayload = (await request.json()) as Record<string, unknown>;
    task = { ...task, status: 'rejected' };
    return HttpResponse.json({});
  }),
  http.get(`${API_URL}/api/v1/roles/analyst/users`, () =>
    HttpResponse.json({
      data: [
        {
          ...currentUser,
          id: 'user-2',
          first_name: 'Sarah',
          last_name: 'Ahmed',
          email: 'sarah@example.com',
        },
      ],
      meta: { page: 1, per_page: 100, total: 1, total_pages: 1 },
    }),
  ),
  http.post(`${API_URL}/api/v1/workflows/tasks/task-1/delegate`, async ({ request }) => {
    delegatePayload = (await request.json()) as Record<string, unknown>;
    return HttpResponse.json({});
  }),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => {
  server.resetHandlers();
  window.localStorage.removeItem('clario360_task_draft_task-1');
  pushMock.mockReset();
  completePayload = null;
  rejectPayload = null;
  delegatePayload = null;
  vi.useRealTimers();
});
afterAll(() => server.close());

beforeEach(() => {
  task = buildTask();
});

describe('Task Detail Page', () => {
  it('renders the dynamic task form from the backend schema', async () => {
    renderWithQuery(<TaskDetailPageClient />);

    expect(await screen.findByText('Task Form')).toBeInTheDocument();
    expect(screen.getByRole('radiogroup')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Add summary')).toBeInTheDocument();
  });

  it('completes a task and redirects back to the task list', async () => {
    const user = userEvent.setup();

    renderWithQuery(<TaskDetailPageClient />);

    await user.click(await screen.findByLabelText(/yes/i));
    await user.type(screen.getByPlaceholderText('Add summary'), 'Plan is safe to execute.');
    await user.click(screen.getByRole('button', { name: /complete/i }));
    await user.click(await screen.findByRole('button', { name: 'Complete Task' }));

    await waitFor(() => {
      expect(completePayload).toEqual({
        form_data: {
          approved: true,
          summary: 'Plan is safe to execute.',
        },
      });
      expect(pushMock).toHaveBeenCalledWith('/workflows/tasks');
    });
  });

  it('rejects a task with a reason', async () => {
    const user = userEvent.setup();

    renderWithQuery(<TaskDetailPageClient />);

    await user.click(await screen.findByRole('button', { name: 'Reject' }));
    await user.type(
      await screen.findByLabelText(/reason/i),
      'The attached evidence is incomplete.',
    );
    await user.click(screen.getByRole('button', { name: 'Reject Task' }));

    await waitFor(() => {
      expect(rejectPayload).toEqual({ reason: 'The attached evidence is incomplete.' });
      expect(pushMock).toHaveBeenCalledWith('/workflows/tasks');
    });
  });

  it('delegates a task to another eligible user', async () => {
    const user = userEvent.setup();

    renderWithQuery(<TaskDetailPageClient />);

    await user.click(await screen.findByRole('button', { name: 'Delegate' }));
    await user.click(await screen.findByRole('combobox', { name: '' }));
    await user.click(await screen.findByText(/Sarah Ahmed/i));
    await user.click(screen.getByRole('button', { name: 'Delegate Task' }));

    await waitFor(() => {
      expect(delegatePayload).toEqual({ delegate_to: 'user-2' });
      expect(pushMock).toHaveBeenCalledWith('/workflows/tasks');
    });
  });

  it('autosaves and restores a draft from localStorage', async () => {
    const firstRender = renderWithQuery(<TaskDetailPageClient />);

    const input = await screen.findByPlaceholderText('Add summary');
    fireEvent.change(input, { target: { value: 'Draft value' } });

    const originalHidden = document.hidden;
    Object.defineProperty(document, 'hidden', {
      configurable: true,
      value: true,
    });
    fireEvent(document, new Event('visibilitychange'));
    Object.defineProperty(document, 'hidden', {
      configurable: true,
      value: originalHidden,
    });

    const rawDraft = window.localStorage.getItem('clario360_task_draft_task-1');
    expect(rawDraft).toContain('Draft value');

    firstRender.unmount();
    renderWithQuery(<TaskDetailPageClient />);

    await waitFor(() => {
      expect(screen.getByText(/Draft restored from/i)).toBeInTheDocument();
    });
  });

  it('renders completed tasks in read-only mode', async () => {
    task = buildTask({
      status: 'completed',
      form_data: { approved: true, summary: 'Already submitted' },
    });

    renderWithQuery(<TaskDetailPageClient />);

    expect(await screen.findByText(/Showing submitted data\./i)).toBeInTheDocument();
    expect(screen.getByDisplayValue('Already submitted')).toBeDisabled();
  });
});
