import { describe, it, expect, beforeAll, afterEach, afterAll, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const API_URL = 'http://localhost:8080';

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
  usePathname: () => '/cyber/assets',
  useSearchParams: () => ({ get: () => null, forEach: () => {} }),
  redirect: vi.fn(),
}));

const mockAssets = [
  {
    id: 'asset-1',
    tenant_id: 't1',
    name: 'web-prod-01',
    type: 'server',
    criticality: 'high',
    status: 'active',
    tags: ['production'],
    vulnerability_count: 3,
    critical_vuln_count: 1,
    high_vuln_count: 2,
    alert_count: 1,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'asset-2',
    tenant_id: 't1',
    name: 'db-primary',
    type: 'database',
    criticality: 'critical',
    status: 'active',
    tags: [],
    vulnerability_count: 0,
    critical_vuln_count: 0,
    high_vuln_count: 0,
    alert_count: 0,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

const mockStats = {
  total: 2,
  by_type: { server: 1, database: 1 },
  by_criticality: { high: 1, critical: 1 },
  by_status: { active: 2 },
  assets_with_vulns: 1,
  assets_discovered_this_week: 1,
};

const server = setupServer(
  http.get(`${API_URL}/api/v1/cyber/assets`, () =>
    HttpResponse.json({
      data: mockAssets,
      meta: { page: 1, per_page: 25, total: 2, total_pages: 1 },
    }),
  ),
  http.get(`${API_URL}/api/v1/cyber/assets/stats`, () =>
    HttpResponse.json({ data: mockStats }),
  ),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

async function renderPage() {
  const { default: Page } = await import('@/app/(dashboard)/cyber/assets/page');
  return render(
    <QueryClientProvider client={makeQueryClient()}>
      <Page />
    </QueryClientProvider>,
  );
}

describe('Asset Inventory Page', () => {
  it('renders page header', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText('Asset Inventory')).toBeInTheDocument();
    });
  });

  it('shows asset names in table', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText('web-prod-01')).toBeInTheDocument();
      expect(screen.getByText('db-primary')).toBeInTheDocument();
    });
  });

  it('shows Add Asset button', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /add asset/i })).toBeInTheDocument();
    });
  });

  it('shows error state on API failure', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/assets`, () =>
        HttpResponse.json({ error: 'server error' }, { status: 500 }),
      ),
    );
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText(/failed to load/i)).toBeInTheDocument();
    });
  });
});
