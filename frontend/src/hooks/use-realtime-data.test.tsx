import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { useRealtimeData } from './use-realtime-data';
import { useRealtimeStore } from '@/stores/realtime-store';

vi.mock('@/lib/api', () => ({
  apiGet: vi.fn(),
}));

const { apiGet } = await import('@/lib/api');

describe('useRealtimeData', () => {
  beforeEach(() => {
    useRealtimeStore.setState({
      subscriptions: {},
      queryEvents: {},
      topicEvents: {},
    });
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('fetches data on mount', async () => {
    vi.mocked(apiGet).mockResolvedValue({ count: 3 });
    const wrapper = createWrapper();

    const { result } = renderHook(
      () => useRealtimeData<{ count: number }>('/api/v1/example'),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.data).toEqual({ count: 3 });
    });
  });

  it('triggers revalidation when a matching topic is published', async () => {
    vi.mocked(apiGet).mockResolvedValue({ count: 3 });
    const wrapper = createWrapper();

    renderHook(
      () => useRealtimeData<{ count: number }>('/api/v1/example', { wsTopics: ['alert.created'] }),
      { wrapper },
    );

    await waitFor(() => expect(apiGet).toHaveBeenCalledTimes(1));

    useRealtimeStore.getState().publish('alert.created', { id: '1' }, new Date().toISOString());
    await delay(600);

    await waitFor(() => expect(apiGet).toHaveBeenCalledTimes(2));
  });

  it('debounces rapid burst messages into one revalidation', async () => {
    vi.mocked(apiGet).mockResolvedValue({ count: 3 });
    const wrapper = createWrapper();

    renderHook(
      () => useRealtimeData<{ count: number }>('/api/v1/example', { wsTopics: ['alert.created'] }),
      { wrapper },
    );

    await waitFor(() => expect(apiGet).toHaveBeenCalledTimes(1));

    useRealtimeStore.getState().publish('alert.created', { id: '1' }, new Date().toISOString());
    useRealtimeStore.getState().publish('alert.created', { id: '2' }, new Date().toISOString());
    useRealtimeStore.getState().publish('alert.created', { id: '3' }, new Date().toISOString());
    await delay(600);

    await waitFor(() => expect(apiGet).toHaveBeenCalledTimes(2));
  });

  it('fires onNewItem for notification payloads', async () => {
    vi.mocked(apiGet).mockResolvedValue({ count: 3 });
    const onNewItem = vi.fn();
    const wrapper = createWrapper();

    renderHook(
      () =>
        useRealtimeData<{ count: number }>('/api/v1/example', {
          wsTopics: ['alert.created'],
          onNewItem,
        }),
      { wrapper },
    );

    await waitFor(() => expect(apiGet).toHaveBeenCalledTimes(1));

    useRealtimeStore.getState().publish(
      'alert.created',
      {
        id: 'notification-1',
        title: 'Alert',
        body: 'New alert',
        category: 'security',
        priority: 'critical',
        read: false,
        created_at: new Date().toISOString(),
      },
      new Date().toISOString(),
    );
    await delay(600);

    await waitFor(() => expect(onNewItem).toHaveBeenCalledTimes(1));
  });

  it('unregisters subscriptions on unmount', async () => {
    vi.mocked(apiGet).mockResolvedValue({ count: 3 });
    const wrapper = createWrapper();

    const { unmount } = renderHook(
      () => useRealtimeData<{ count: number }>('/api/v1/example', { wsTopics: ['alert.created'] }),
      { wrapper },
    );

    await waitFor(() => expect(apiGet).toHaveBeenCalledTimes(1));
    expect(useRealtimeStore.getState().getKeysForTopic('alert.created').length).toBe(1);

    unmount();

    expect(useRealtimeStore.getState().getKeysForTopic('alert.created').length).toBe(0);
  });
});

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}
