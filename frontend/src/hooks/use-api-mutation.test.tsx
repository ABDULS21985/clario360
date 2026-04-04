import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { useApiMutation } from './use-api-mutation';

vi.mock('@/lib/api', () => ({
  default: {
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}));

const { default: api } = await import('@/lib/api');

describe('useApiMutation', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('unwraps standard data envelopes returned by mutation endpoints', async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: {
        data: {
          id: 'asset-3',
          name: 'api-prod-03',
        },
      },
    });

    const { result } = renderHook(
      () => useApiMutation<{ id: string; name: string }, { name: string }>('post', '/api/v1/cyber/assets'),
      { wrapper: createWrapper() },
    );

    await act(async () => {
      const response = await result.current.mutateAsync({ name: 'api-prod-03' });
      expect(response).toEqual({ id: 'asset-3', name: 'api-prod-03' });
    });

    expect(api.post).toHaveBeenCalledWith('/api/v1/cyber/assets', { name: 'api-prod-03' });
  });

  it('preserves non-canonical payloads that only happen to include a data field', async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: {
        data: 'scan-1',
        status: 'accepted',
      },
    });

    const { result } = renderHook(
      () => useApiMutation<{ data: string; status: string }, { target: string }>('post', '/api/v1/cyber/assets/scan'),
      { wrapper: createWrapper() },
    );

    await act(async () => {
      const response = await result.current.mutateAsync({ target: '10.0.0.8' });
      expect(response).toEqual({ data: 'scan-1', status: 'accepted' });
    });
  });
});

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
      mutations: {
        retry: false,
      },
    },
  });

  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}
