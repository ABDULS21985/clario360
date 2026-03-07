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
  usePathname: () => '/cyber/alerts',
  useSearchParams: () => ({ get: () => null }),
  redirect: vi.fn(),
}));

const mockAlerts = [
  {
    id: 'alert-1',
    tenant_id: 't1',
    title: 'Brute Force Detected',
    description: 'Multiple failed logins',
    severity: 'critical',
    status: 'new',
    source: 'SIEM',
    explanation: { summary: '', reason: '', evidence: [], matched_conditions: [], confidence_factors: [], recommended_actions: [], false_positive_indicators: [] },
    confidence_score: 87,
    event_count: 15,
    first_event_at: '2024-01-01T00:00:00Z',
    last_event_at: '2024-01-01T01:00:00Z',
    tags: [],
    asset_ids: [],
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'alert-2',
    tenant_id: 't1',
    title: 'Suspicious Outbound Traffic',
    description: 'Unusual data exfiltration attempt',
    severity: 'high',
    status: 'investigating',
    source: 'IDS',
    explanation: { summary: '', reason: '', evidence: [], matched_conditions: [], confidence_factors: [], recommended_actions: [], false_positive_indicators: [] },
    confidence_score: 72,
    event_count: 3,
    first_event_at: '2024-01-02T00:00:00Z',
    last_event_at: '2024-01-02T00:30:00Z',
    tags: [],
    asset_ids: [],
    created_at: '2024-01-02T00:00:00Z',
    updated_at: '2024-01-02T00:00:00Z',
  },
];

const mockStats = {
  by_severity: [{ name: 'critical', count: 1 }, { name: 'high', count: 1 }],
  by_status: [{ name: 'new', count: 1 }, { name: 'investigating', count: 1 }],
  by_rule: [],
  by_technique: [],
  open_count: 2,
  resolved_count: 0,
};

const server = setupServer(
  http.get(`${API_URL}/api/v1/cyber/alerts`, () =>
    HttpResponse.json({
      data: mockAlerts,
      meta: { page: 1, per_page: 25, total: 2, total_pages: 1 },
    }),
  ),
  http.get(`${API_URL}/api/v1/cyber/alerts/stats`, () =>
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
  const { default: Page } = await import('@/app/(dashboard)/cyber/alerts/page');
  return render(
    <QueryClientProvider client={makeQueryClient()}>
      <Page />
    </QueryClientProvider>,
  );
}

describe('Alert List Page', () => {
  it('renders page header', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText('Security Alerts')).toBeInTheDocument();
    });
  });

  it('displays alert titles', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText('Brute Force Detected')).toBeInTheDocument();
      expect(screen.getByText('Suspicious Outbound Traffic')).toBeInTheDocument();
    });
  });

  it('shows error state on failure', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/alerts`, () =>
        HttpResponse.json({ error: 'server error' }, { status: 500 }),
      ),
    );
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText(/failed to load/i)).toBeInTheDocument();
    });
  });
});
