import { describe, it, expect, beforeAll, afterEach, afterAll, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const API_URL = 'http://localhost:8080';
const ASSET_ID = 'asset-xyz-456';

vi.mock('@/hooks/use-auth', () => ({
  useAuth: () => ({
    user: { id: 'u1', first_name: 'Admin', permissions: ['cyber:read', 'cyber:write'] },
    isHydrated: true,
    hasPermission: () => true,
  }),
}));

vi.mock('@/hooks/use-websocket', () => ({
  useWebSocket: () => ({ isConnected: false }),
}));

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), back: vi.fn(), prefetch: vi.fn() }),
  usePathname: () => `/cyber/assets/${ASSET_ID}`,
  useSearchParams: () => ({ get: () => null }),
  redirect: vi.fn(),
}));

vi.mock('react', async (importOriginal) => {
  const React = await importOriginal<typeof import('react')>();
  return {
    ...React,
    use: (promise: unknown) => {
      if (promise instanceof Promise) return { id: ASSET_ID };
      return promise;
    },
  };
});

const mockAsset = {
  id: ASSET_ID,
  tenant_id: 't1',
  name: 'web-prod-01',
  type: 'server',
  ip_address: '10.0.0.1',
  hostname: 'web-prod-01.example.com',
  os: 'Ubuntu 22.04',
  owner: 'Platform Team',
  department: 'Engineering',
  criticality: 'high',
  status: 'active',
  tags: ['production', 'web'],
  vulnerability_count: 5,
  critical_vuln_count: 1,
  high_vuln_count: 3,
  alert_count: 2,
  metadata: { env: 'prod' },
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

const server = setupServer(
  http.get(`${API_URL}/api/v1/cyber/assets/${ASSET_ID}`, () =>
    HttpResponse.json({ data: mockAsset }),
  ),
  http.get(`${API_URL}/api/v1/cyber/assets/${ASSET_ID}/vulnerabilities`, () =>
    HttpResponse.json({ data: [], meta: { page: 1, per_page: 25, total: 0, total_pages: 0 } }),
  ),
  http.get(`${API_URL}/api/v1/cyber/assets/${ASSET_ID}/relationships`, () =>
    HttpResponse.json({ data: [] }),
  ),
  http.get(`${API_URL}/api/v1/cyber/assets/${ASSET_ID}/activity`, () =>
    HttpResponse.json({ data: [] }),
  ),
  http.get(`${API_URL}/api/v1/cyber/alerts`, () =>
    HttpResponse.json({ data: [], meta: { page: 1, per_page: 25, total: 0, total_pages: 0 } }),
  ),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

async function renderPage() {
  const { default: Page } = await import('@/app/(dashboard)/cyber/assets/[id]/page');
  return render(
    <QueryClientProvider client={makeQueryClient()}>
      <Page params={Promise.resolve({ id: ASSET_ID })} />
    </QueryClientProvider>,
  );
}

describe('Asset Detail Page', () => {
  it('renders asset name in header', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText('web-prod-01')).toBeInTheDocument();
    });
  });

  it('shows security summary bar with vuln counts', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getAllByText('Vulnerabilities').length).toBeGreaterThan(0);
      expect(screen.getAllByText('5').length).toBeGreaterThan(0);
    });
  });

  it('renders action buttons', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /scan/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /edit/i })).toBeInTheDocument();
    });
  });

  it('shows tab navigation', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByRole('tab', { name: 'Overview' })).toBeInTheDocument();
      expect(screen.getByRole('tab', { name: /vulnerabilities/i })).toBeInTheDocument();
    });
  });

  it('shows error state on API failure', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/assets/${ASSET_ID}`, () =>
        HttpResponse.json({ error: 'not found' }, { status: 404 }),
      ),
    );
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText(/failed to load/i)).toBeInTheDocument();
    });
  });
});
