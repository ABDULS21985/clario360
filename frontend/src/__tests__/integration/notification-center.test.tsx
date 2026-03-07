import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithQuery } from '@/__tests__/utils/render-with-query';
import { NotificationsPageClient } from '@/app/(dashboard)/notifications/notifications-page-client';
import { useNotificationStore } from '@/stores/notification-store';
import { useRealtimeStore } from '@/stores/realtime-store';
import type { Notification } from '@/types/models';

const API_URL = 'http://localhost:8080';
const pushMock = vi.fn();
let searchParams = new URLSearchParams();

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: pushMock, replace: vi.fn() }),
  usePathname: () => '/notifications',
  useSearchParams: () => searchParams,
}));

vi.mock('@/components/realtime/new-data-toast', () => ({
  showNewDataToast: vi.fn(),
}));

let notifications: Notification[] = [];
let lastListQuery = new URLSearchParams();

function buildNotification(overrides: Partial<Notification>): Notification {
  return {
    id: overrides.id ?? crypto.randomUUID(),
    type: overrides.type ?? 'notification.new',
    title: overrides.title ?? 'New notification',
    body: overrides.body ?? 'Body',
    category: overrides.category ?? 'workflow',
    priority: overrides.priority ?? 'medium',
    data: overrides.data ?? null,
    action_url: overrides.action_url ?? '/workflows/tasks/task-1',
    read: overrides.read ?? false,
    read_at: overrides.read_at ?? null,
    created_at: overrides.created_at ?? new Date().toISOString(),
  };
}

function filterNotifications(query: URLSearchParams): Notification[] {
  return notifications.filter((notification) => {
    const category = query.get('category');
    const read = query.get('read');

    if (category && notification.category !== category) {
      return false;
    }

    if (read === 'false' && notification.read_at) {
      return false;
    }

    return true;
  });
}

const server = setupServer(
  http.get(`${API_URL}/api/v1/notifications`, ({ request }) => {
    const url = new URL(request.url);
    if (url.searchParams.get('per_page') === '20') {
      lastListQuery = new URLSearchParams(url.search);
    }
    const page = Number(url.searchParams.get('page') ?? '1');
    const perPage = Number(url.searchParams.get('per_page') ?? '20');
    const filtered = filterNotifications(url.searchParams);
    const start = (page - 1) * perPage;
    const data = filtered.slice(start, start + perPage);

    return HttpResponse.json({
      data,
      meta: {
        page,
        per_page: perPage,
        total: filtered.length,
        total_pages: Math.max(1, Math.ceil(filtered.length / perPage)),
      },
    });
  }),
  http.put(`${API_URL}/api/v1/notifications/read-all`, () => {
    notifications = notifications.map((notification) => ({
      ...notification,
      read: true,
      read_at: notification.read_at ?? new Date().toISOString(),
    }));
    return HttpResponse.json({});
  }),
  http.put(`${API_URL}/api/v1/notifications/:id/read`, ({ params }) => {
    notifications = notifications.map((notification) =>
      notification.id === params.id
        ? {
            ...notification,
            read: true,
            read_at: notification.read_at ?? new Date().toISOString(),
          }
        : notification,
    );
    return HttpResponse.json({});
  }),
  http.delete(`${API_URL}/api/v1/notifications/:id`, ({ params }) => {
    notifications = notifications.filter((notification) => notification.id !== params.id);
    return HttpResponse.json({});
  }),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => {
  server.resetHandlers();
  pushMock.mockReset();
  searchParams = new URLSearchParams();
  lastListQuery = new URLSearchParams();
  useNotificationStore.setState({
    unreadCount: 0,
    recentNotifications: [],
    connectionStatus: 'disconnected',
    reconnectAttempt: 0,
    nextRetryAt: null,
    reconnectToken: 0,
    isInitialized: false,
  });
  useRealtimeStore.setState({ subscriptions: {}, queryEvents: {}, topicEvents: {} });
});
afterAll(() => server.close());

beforeEach(() => {
  notifications = [
    buildNotification({
      id: 'n-1',
      title: 'Critical security alert',
      category: 'security',
      priority: 'critical',
      action_url: '/cyber/alerts/alert-1',
    }),
    buildNotification({
      id: 'n-2',
      title: 'Workflow task assigned',
      category: 'workflow',
      priority: 'medium',
      action_url: '/workflows/tasks/task-1',
    }),
    buildNotification({
      id: 'n-3',
      title: 'Older system notice',
      category: 'system',
      priority: 'low',
      read: true,
      read_at: new Date().toISOString(),
      action_url: null,
    }),
  ];

  useNotificationStore.setState({
    unreadCount: 2,
    recentNotifications: notifications.slice(0, 2),
    connectionStatus: 'connected',
    reconnectAttempt: 0,
    nextRetryAt: null,
    reconnectToken: 0,
    isInitialized: true,
  });
});

describe('Notification Center Page', () => {
  it('loads notifications and groups them in the list', async () => {
    renderWithQuery(<NotificationsPageClient />);

    expect(await screen.findByText('Critical security alert')).toBeInTheDocument();
    expect(screen.getByText('Workflow task assigned')).toBeInTheDocument();
    expect(screen.getByText('Today')).toBeInTheDocument();
  });

  it('applies the category filter from the URL', async () => {
    searchParams = new URLSearchParams('tab=security');

    renderWithQuery(<NotificationsPageClient />);

    expect(await screen.findByText('Critical security alert')).toBeInTheDocument();
    expect(lastListQuery.get('category')).toBe('security');
    expect(screen.queryByText('Workflow task assigned')).not.toBeInTheDocument();
  });

  it('marks a notification as read and navigates to its action URL', async () => {
    const user = userEvent.setup();

    renderWithQuery(<NotificationsPageClient />);

    await user.click(await screen.findByText('Critical security alert'));

    await waitFor(() => {
      expect(pushMock).toHaveBeenCalledWith('/cyber/alerts/alert-1');
      expect(screen.queryByLabelText(/Critical security alert \(unread\)/i)).not.toBeInTheDocument();
    });
  });

  it('marks all notifications as read through the confirmation dialog', async () => {
    const user = userEvent.setup();

    renderWithQuery(<NotificationsPageClient />);

    await user.click(await screen.findByRole('button', { name: /mark all read/i }));
    await user.click(await screen.findByRole('button', { name: /mark all read/i }));

    await waitFor(() => {
      expect(useNotificationStore.getState().unreadCount).toBe(0);
    });
  });

  it('deletes a notification from the list', async () => {
    const user = userEvent.setup();

    renderWithQuery(<NotificationsPageClient />);

    await user.click((await screen.findAllByLabelText('Delete notification'))[0]!);

    await waitFor(() => {
      expect(screen.queryByText('Critical security alert')).not.toBeInTheDocument();
    });
  });

  it('loads more notifications when requested', async () => {
    notifications = Array.from({ length: 21 }, (_, index) =>
      buildNotification({
        id: `n-${index + 1}`,
        title: `Notification ${index + 1}`,
        category: index % 2 === 0 ? 'workflow' : 'security',
      }),
    );
    useNotificationStore.setState({ unreadCount: 21 });

    const user = userEvent.setup();
    renderWithQuery(<NotificationsPageClient />);

    await screen.findByText('Notification 1');
    await user.click(screen.getByRole('button', { name: /load more notifications/i }));

    expect(await screen.findByText('Notification 21')).toBeInTheDocument();
  });

  it('prepends a new real-time notification when a websocket event arrives', async () => {
    renderWithQuery(<NotificationsPageClient />);

    await screen.findByText('Critical security alert');

    useRealtimeStore
      .getState()
      .publish(
        'notification.new',
        buildNotification({
          id: 'n-live',
          title: 'Live workflow update',
          category: 'workflow',
        }),
        new Date().toISOString(),
      );

    expect(await screen.findByText('Live workflow update')).toBeInTheDocument();
  });
});
