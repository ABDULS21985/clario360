import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { TaskClaimButton } from './task-claim-button';
import type { HumanTask, Role, User } from '@/types/models';

const { apiPostMock, showSuccessMock, showApiErrorMock } = vi.hoisted(() => ({
  apiPostMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showApiErrorMock: vi.fn(),
}));

let mockUser: User | null = null;

vi.mock('@/lib/api', () => ({
  apiPost: apiPostMock,
}));

vi.mock('@/lib/toast', () => ({
  showSuccess: showSuccessMock,
  showApiError: showApiErrorMock,
}));

vi.mock('@/hooks/use-auth', () => ({
  useAuth: () => ({
    user: mockUser,
  }),
}));

function buildRole(slug: string): Role {
  return {
    id: `${slug}-id`,
    tenant_id: 'tenant-1',
    name: slug,
    slug,
    description: '',
    permissions: [],
    is_system: false,
    created_at: '2026-03-07T10:00:00Z',
    updated_at: '2026-03-07T10:00:00Z',
  };
}

function buildUser(roleSlugs: string[]): User {
  return {
    id: 'user-1',
    tenant_id: 'tenant-1',
    email: 'analyst@example.com',
    first_name: 'Amina',
    last_name: 'Analyst',
    status: 'active',
    mfa_enabled: false,
    last_login_at: null,
    roles: roleSlugs.map(buildRole),
    created_at: '2026-03-07T10:00:00Z',
    updated_at: '2026-03-07T10:00:00Z',
  };
}

function buildTask(overrides: Partial<HumanTask> = {}): HumanTask {
  return {
    id: 'task-1',
    name: 'Review remediation plan',
    description: 'Confirm the remediation plan is safe to run.',
    instance_id: 'instance-1',
    definition_name: 'Alert workflow',
    workflow_name: 'Alert workflow',
    step_id: 'review',
    status: 'pending',
    priority: 1,
    form_schema: [],
    form_data: null,
    sla_deadline: null,
    sla_breached: false,
    claimed_by: null,
    claimed_by_name: null,
    assignee_role: 'analyst',
    assignee_id: null,
    metadata: {},
    created_at: '2026-03-07T10:00:00Z',
    updated_at: '2026-03-07T10:00:00Z',
    ...overrides,
  };
}

describe('TaskClaimButton', () => {
  beforeEach(() => {
    mockUser = buildUser(['analyst']);
    apiPostMock.mockReset();
    showSuccessMock.mockReset();
    showApiErrorMock.mockReset();
  });

  afterEach(() => {
    mockUser = null;
  });

  it('shows the claim button for an eligible unclaimed task', () => {
    render(<TaskClaimButton task={buildTask()} onSuccess={vi.fn()} />);

    expect(screen.getByRole('button', { name: /claim this task/i })).toBeInTheDocument();
  });

  it('hides the button when the task is already claimed', () => {
    render(
      <TaskClaimButton
        task={buildTask({ claimed_by: 'other-user' })}
        onSuccess={vi.fn()}
      />,
    );

    expect(screen.queryByRole('button', { name: /claim this task/i })).not.toBeInTheDocument();
  });

  it('hides the button when the user lacks the required role', () => {
    mockUser = buildUser(['reviewer']);

    render(<TaskClaimButton task={buildTask()} onSuccess={vi.fn()} />);

    expect(screen.queryByRole('button', { name: /claim this task/i })).not.toBeInTheDocument();
  });

  it('claims the task successfully and triggers onSuccess', async () => {
    const user = userEvent.setup();
    const onSuccess = vi.fn();
    apiPostMock.mockResolvedValue(undefined);

    render(<TaskClaimButton task={buildTask()} onSuccess={onSuccess} />);

    await user.click(screen.getByRole('button', { name: /claim this task/i }));

    await waitFor(() => {
      expect(apiPostMock).toHaveBeenCalledWith('/api/v1/workflows/tasks/task-1/claim');
      expect(showSuccessMock).toHaveBeenCalledWith('Task claimed.');
      expect(onSuccess).toHaveBeenCalledTimes(1);
    });
  });

  it('shows a conflict error when another user already claimed the task', async () => {
    const user = userEvent.setup();
    const onSuccess = vi.fn();
    apiPostMock.mockRejectedValue({ status: 409 });

    render(<TaskClaimButton task={buildTask()} onSuccess={onSuccess} />);

    await user.click(screen.getByRole('button', { name: /claim this task/i }));

    await waitFor(() => {
      expect(showApiErrorMock).toHaveBeenCalledTimes(1);
      expect(
        (showApiErrorMock.mock.calls[0]?.[0] as Error).message,
      ).toContain('claimed by someone else');
      expect(onSuccess).toHaveBeenCalledTimes(1);
    });
  });
});
