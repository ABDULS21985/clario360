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
  usePathname: () => '/cyber/remediation',
  useSearchParams: () => ({ get: () => null }),
  redirect: vi.fn(),
}));

const mockActions = [
  {
    id: 'action-1',
    tenant_id: 't1',
    title: 'Patch OpenSSL CVE-2024-0001',
    description: 'Apply security patch for critical OpenSSL vulnerability',
    type: 'patch',
    severity: 'critical',
    status: 'pending_approval',
    plan: { steps: [{ number: 1, action: 'Run apt-get upgrade openssl' }], reversible: true },
    affected_asset_ids: ['asset-1'],
    execution_mode: 'manual',
    tags: [],
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'action-2',
    tenant_id: 't1',
    title: 'Update Firewall Rules',
    description: 'Block suspicious outbound connections',
    type: 'firewall_rule',
    severity: 'high',
    status: 'approved',
    plan: { steps: [{ number: 1, action: 'Add iptables rule' }], reversible: true },
    affected_asset_ids: [],
    execution_mode: 'semi_automatic',
    tags: ['firewall'],
    created_at: '2024-01-02T00:00:00Z',
    updated_at: '2024-01-02T00:00:00Z',
  },
];

const mockStats = {
  total: 2,
  by_status: { pending_approval: 1, approved: 1 },
  by_severity: { critical: 1, high: 1 },
  by_type: { patch: 1, firewall_rule: 1 },
  pending_approval: 1,
  execution_pending: 0,
};

const server = setupServer(
  http.get(`${API_URL}/api/v1/cyber/remediation`, () =>
    HttpResponse.json({
      data: mockActions,
      meta: { page: 1, per_page: 25, total: 2, total_pages: 1 },
    }),
  ),
  http.get(`${API_URL}/api/v1/cyber/remediation/stats`, () =>
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
  const { default: Page } = await import('@/app/(dashboard)/cyber/remediation/page');
  return render(
    <QueryClientProvider client={makeQueryClient()}>
      <Page />
    </QueryClientProvider>,
  );
}

describe('Remediation Page', () => {
  it('renders page header', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText('Remediation')).toBeInTheDocument();
    });
  });

  it('shows remediation action titles', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText('Patch OpenSSL CVE-2024-0001')).toBeInTheDocument();
      expect(screen.getByText('Update Firewall Rules')).toBeInTheDocument();
    });
  });

  it('shows KPI card for pending approval count', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText('Pending Approval')).toBeInTheDocument();
      expect(screen.getByText('1')).toBeInTheDocument();
    });
  });

  it('shows New Action button', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /new action/i })).toBeInTheDocument();
    });
  });

  it('shows status badges', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText('Pending Approval')).toBeInTheDocument();
      expect(screen.getByText('Approved')).toBeInTheDocument();
    });
  });

  it('shows error state on API failure', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/remediation`, () =>
        HttpResponse.json({ error: 'server error' }, { status: 500 }),
      ),
    );
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText(/failed to load/i)).toBeInTheDocument();
    });
  });
});
