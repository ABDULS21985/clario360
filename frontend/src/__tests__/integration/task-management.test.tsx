import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithQuery } from '@/__tests__/utils/render-with-query';
import { useRealtimeStore } from '@/stores/realtime-store';
import { WorkflowTasksPageClient } from '@/app/(dashboard)/workflows/tasks/tasks-page-client';
import type { HumanTask, Role, User } from '@/types/models';

const API_URL = 'http://localhost:8080';

let searchParams = new URLSearchParams();
const pushMock = vi.fn();

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: pushMock, replace: vi.fn() }),
  usePathname: () => '/workflows/tasks',
  useSearchParams: () => searchParams,
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

const roles: Role[] = [
  {
    id: 'role-1',
    tenant_id: 'tenant-1',
    name: 'Analyst',
    slug: 'analyst',
    description: '',
    permissions: [],
    is_system: false,
    created_at: '2026-03-07T10:00:00Z',
    updated_at: '2026-03-07T10:00:00Z',
  },
];

let tasks: HumanTask[] = [];
let lastTaskQuery = new URLSearchParams();

function buildTask(overrides: Partial<HumanTask>): HumanTask {
  return {
    id: overrides.id ?? crypto.randomUUID(),
    name: overrides.name ?? 'Review task',
    description: overrides.description ?? 'Review the task details.',
    instance_id: overrides.instance_id ?? 'instance-1',
    definition_name: overrides.definition_name ?? 'Threat workflow',
    workflow_name: overrides.workflow_name ?? 'Threat workflow',
    step_id: overrides.step_id ?? 'triage',
    status: overrides.status ?? 'pending',
    priority: overrides.priority ?? 1,
    form_schema: overrides.form_schema ?? [],
    form_data: overrides.form_data ?? null,
    sla_deadline: overrides.sla_deadline ?? null,
    sla_breached: overrides.sla_breached ?? false,
    claimed_by: overrides.claimed_by ?? null,
    claimed_by_name: overrides.claimed_by_name ?? null,
    assignee_role: overrides.assignee_role ?? 'analyst',
    assignee_id: overrides.assignee_id ?? null,
    metadata: overrides.metadata ?? {},
    created_at: overrides.created_at ?? '2026-03-07T10:00:00Z',
    updated_at: overrides.updated_at ?? '2026-03-07T10:00:00Z',
  };
}

function computeCounts() {
  const pending = tasks.filter((task) => task.status === 'pending' && !task.sla_breached).length;
  const claimedByMe = tasks.filter((task) => task.status === 'claimed' && task.claimed_by === currentUser.id).length;
  const completed = tasks.filter((task) => task.status === 'completed').length;
  const overdue = tasks.filter((task) => task.sla_breached).length;
  const escalated = tasks.filter((task) => task.status === 'escalated').length;

  return {
    pending,
    claimed_by_me: claimedByMe,
    completed,
    overdue,
    escalated,
  };
}

function filterTasks(query: URLSearchParams): HumanTask[] {
  const status = query.get('status');
  const slaBreached = query.get('sla_breached');

  return tasks.filter((task) => {
    if (status) {
      const allowed = status.split(',');
      if (!allowed.includes(task.status)) {
        return false;
      }
    }

    if (slaBreached === 'true' && !task.sla_breached) {
      return false;
    }

    return true;
  });
}

const server = setupServer(
  http.get(`${API_URL}/api/v1/workflows/tasks`, ({ request }) => {
    const url = new URL(request.url);
    lastTaskQuery = new URLSearchParams(url.search);
    const data = filterTasks(url.searchParams);

    return HttpResponse.json({
      data,
      meta: {
        page: Number(url.searchParams.get('page') ?? '1'),
        per_page: Number(url.searchParams.get('per_page') ?? '25'),
        total: data.length,
        total_pages: 1,
      },
    });
  }),
  http.get(`${API_URL}/api/v1/workflows/tasks/count`, () => HttpResponse.json(computeCounts())),
  http.get(`${API_URL}/api/v1/roles`, () => HttpResponse.json(roles)),
  http.post(`${API_URL}/api/v1/workflows/tasks/:id/claim`, ({ params }) => {
    tasks = tasks.map((task) =>
      task.id === params.id
        ? {
            ...task,
            status: 'claimed',
            claimed_by: currentUser.id,
            claimed_by_name: `${currentUser.first_name} ${currentUser.last_name}`,
          }
        : task,
    );

    return HttpResponse.json({});
  }),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => {
  server.resetHandlers();
  pushMock.mockReset();
  searchParams = new URLSearchParams();
  useRealtimeStore.setState({ subscriptions: {}, queryEvents: {}, topicEvents: {} });
});
afterAll(() => server.close());

beforeEach(() => {
  tasks = [
    buildTask({ id: 'task-1', name: 'Review task', status: 'pending' }),
    buildTask({
      id: 'task-2',
      name: 'Complete report',
      status: 'claimed',
      claimed_by: currentUser.id,
      claimed_by_name: 'Amina Analyst',
    }),
    buildTask({
      id: 'task-3',
      name: 'Overdue review',
      status: 'pending',
      sla_breached: true,
    }),
  ];
  lastTaskQuery = new URLSearchParams();
});

describe('Task Management Page', () => {
  it('loads tasks and renders the table rows', async () => {
    renderWithQuery(<WorkflowTasksPageClient />);

    expect(await screen.findByText('Review task')).toBeInTheDocument();
    expect(screen.getByText('Complete report')).toBeInTheDocument();
    expect(screen.getByText('Overdue review')).toBeInTheDocument();
  });

  it('renders tab counts from the real count API', async () => {
    renderWithQuery(<WorkflowTasksPageClient />);

    await waitFor(() => {
      expect(screen.getByRole('tab', { name: /pending/i })).toHaveTextContent('1');
      expect(screen.getByRole('tab', { name: /claimed/i })).toHaveTextContent('1');
      expect(screen.getByRole('tab', { name: /overdue/i })).toHaveTextContent('1');
    });
  });

  it('applies tab filtering from the URL parameter', async () => {
    searchParams = new URLSearchParams('tab=pending');

    renderWithQuery(<WorkflowTasksPageClient />);

    await screen.findByText('Review task');

    expect(lastTaskQuery.get('status')).toBe('pending');
    expect(screen.queryByText('Complete report')).not.toBeInTheDocument();
  });

  it('claims an unassigned task directly from the table', async () => {
    const user = userEvent.setup();

    renderWithQuery(<WorkflowTasksPageClient />);

    const claimButtons = await screen.findAllByRole('button', { name: 'Claim' });
    await user.click(claimButtons[0]!);

    await waitFor(() => {
      expect(screen.getAllByText('You').length).toBe(2);
      expect(screen.getByRole('tab', { name: /claimed/i })).toHaveTextContent('2');
    });
  });

  it('opens task detail when the user clicks a task row', async () => {
    const user = userEvent.setup();

    renderWithQuery(<WorkflowTasksPageClient />);

    await user.click(await screen.findByText('Review task'));

    expect(pushMock).toHaveBeenCalledWith('/workflows/tasks/task-1');
  });
});
