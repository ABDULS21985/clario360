import type { ComponentProps } from 'react';
import { render, screen } from '@testing-library/react';
import { subDays } from 'date-fns';
import { describe, expect, it, vi } from 'vitest';
import { NotificationList } from './notification-list';
import { TooltipProvider } from '@/components/ui/tooltip';
import type { Notification } from '@/types/models';

function buildNotification(overrides: Partial<Notification>): Notification {
  return {
    id: overrides.id ?? crypto.randomUUID(),
    title: overrides.title ?? 'Notification',
    body: overrides.body ?? 'Body',
    category: overrides.category ?? 'workflow',
    priority: overrides.priority ?? 'medium',
    read: overrides.read ?? false,
    read_at: overrides.read_at ?? null,
    action_url: overrides.action_url ?? '/workflows/tasks/task-1',
    created_at: overrides.created_at ?? new Date().toISOString(),
    type: overrides.type,
    data: overrides.data ?? null,
  };
}

function renderList(overrides: Partial<ComponentProps<typeof NotificationList>> = {}) {
  return render(
    <TooltipProvider>
      <NotificationList
        notifications={[]}
        isLoading={false}
        isLoadingMore={false}
        hasMore={false}
        onLoadMore={vi.fn()}
        onMarkRead={vi.fn()}
        onDelete={vi.fn()}
        sentinelRef={vi.fn()}
        {...overrides}
      />
    </TooltipProvider>,
  );
}

describe('NotificationList', () => {
  it('groups notifications by date headers', () => {
    const notifications = [
      buildNotification({ id: 'today', created_at: new Date().toISOString() }),
      buildNotification({ id: 'yesterday', created_at: subDays(new Date(), 1).toISOString() }),
      buildNotification({ id: 'older', created_at: subDays(new Date(), 40).toISOString() }),
    ];

    renderList({ notifications });

    expect(screen.getByText('Today')).toBeInTheDocument();
    expect(screen.getByText('Yesterday')).toBeInTheDocument();
    expect(screen.getByText('Older')).toBeInTheDocument();
  });

  it('shows the empty state when there are no notifications', () => {
    renderList({ category: 'unread' });

    expect(screen.getByText('No unread notifications.')).toBeInTheDocument();
  });

  it('shows a load more button when more pages are available', () => {
    renderList({
      notifications: [buildNotification({ id: 'n-1' })],
      hasMore: true,
    });

    expect(screen.getByRole('button', { name: /load more notifications/i })).toBeInTheDocument();
  });

  it('shows the no more notifications message when pagination ends', () => {
    renderList({
      notifications: [buildNotification({ id: 'n-1' })],
      hasMore: false,
    });

    expect(screen.getByText('No more notifications.')).toBeInTheDocument();
  });
});
