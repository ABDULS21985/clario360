import { describe, it, expect, beforeAll, afterEach, afterAll, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const API_URL = 'http://localhost:8080';

vi.mock('@/hooks/use-auth', () => ({
  useAuth: () => ({
    user: { id: 'u1', first_name: 'Admin', permissions: ['cyber:read'] },
    hasPermission: () => true,
  }),
}));

vi.mock('@/hooks/use-websocket', () => ({
  useWebSocket: () => ({ isConnected: false }),
}));

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), back: vi.fn(), prefetch: vi.fn() }),
  usePathname: () => '/cyber',
  useSearchParams: () => ({ get: () => null }),
  redirect: vi.fn(),
}));

const mockDashboard = {
  kpis: {
    open_alerts: 42,
    critical_alerts: 5,
    open_vulnerabilities: 123,
    critical_vulnerabilities: 8,
    active_threats: 3,
    mttr_hours: 4.2,
    mean_resolve_hours: 6.1,
    risk_score: 67,
    risk_grade: 'C',
    alerts_delta: -5,
    vulns_delta: 2,
  },
  alert_timeline: { granularity: 'hour', points: [] },
  severity_distribution: { counts: { critical: 5, high: 15 }, total: 20 },
  alert_trend: [],
  vulnerability_trend: [],
  recent_alerts: [
    {
      id: 'alert-1',
      title: 'Brute Force Attack',
      severity: 'critical',
      status: 'new',
      created_at: '2024-01-01T00:00:00Z',
    },
  ],
  top_attacked_assets: [],
  analyst_workload: [],
  mitre_heatmap: { cells: [], max_count: 0 },
  calculated_at: '2024-01-01T00:00:00Z',
};

const server = setupServer(
  http.get(`${API_URL}/api/v1/cyber/dashboard`, () =>
    HttpResponse.json({ data: mockDashboard }),
  ),
  http.get(`${API_URL}/api/v1/cyber/vulnerabilities/aging`, () =>
    HttpResponse.json({ data: { buckets: [] } }),
  ),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

async function renderDashboard() {
  const { default: Page } = await import('@/app/(dashboard)/cyber/page');
  return render(
    <QueryClientProvider client={makeQueryClient()}>
      <Page />
    </QueryClientProvider>,
  );
}

describe('SOC Dashboard', () => {
  it('renders page header', async () => {
    await renderDashboard();
    await waitFor(() => {
      expect(screen.getByText('Security Operations Center')).toBeInTheDocument();
    });
  });

  it('displays KPI open alerts count', async () => {
    await renderDashboard();
    await waitFor(() => {
      expect(screen.getByText('42')).toBeInTheDocument();
    });
  });

  it('shows recent alert', async () => {
    await renderDashboard();
    await waitFor(() => {
      expect(screen.getByText('Brute Force Attack')).toBeInTheDocument();
    });
  });

  it('shows error state on API failure', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/dashboard`, () =>
        HttpResponse.json({ error: 'internal error' }, { status: 500 }),
      ),
    );
    await renderDashboard();
    await waitFor(() => {
      expect(screen.getByText(/failed to load/i)).toBeInTheDocument();
    });
  });
});
