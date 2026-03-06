import { describe, it, expect, beforeAll, afterEach, afterAll, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { AuditLog } from '@/types/models';

const API_URL = 'http://localhost:8080';

// Mock next/navigation
vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  usePathname: () => '/admin/audit',
  useSearchParams: () => new URLSearchParams(),
}));

// Mock auth
vi.mock('@/hooks/use-auth', () => ({
  useAuth: () => ({
    user: { id: 'current-user-id', first_name: 'Admin', last_name: 'User', email: 'admin@test.com' },
    hasPermission: () => true,
    tenant: { id: 'tenant-1' },
  }),
}));

const mockLogs: AuditLog[] = [
  {
    id: 'log-1',
    tenant_id: 'tenant-1',
    user_id: 'user-1',
    user_email: 'alice@example.com',
    action: 'user.create',
    resource_type: 'user',
    resource_id: 'user-abc-1234',
    ip_address: '192.168.1.1',
    user_agent: 'Mozilla/5.0',
    metadata: { email: 'new@example.com' },
    created_at: '2024-03-01T10:00:00Z',
  },
  {
    id: 'log-2',
    tenant_id: 'tenant-1',
    user_id: null,
    user_email: null,
    action: 'login.failed',
    resource_type: 'auth',
    resource_id: null,
    ip_address: '10.0.0.1',
    user_agent: 'curl/7.68',
    metadata: { reason: 'invalid_password' },
    created_at: '2024-03-01T09:00:00Z',
  },
];

const server = setupServer(
  http.get(`${API_URL}/api/v1/audit/logs`, () =>
    HttpResponse.json({
      data: mockLogs,
      meta: { page: 1, per_page: 50, total: 2, total_pages: 1 },
    })
  )
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

async function renderAuditPage() {
  const { default: AuditLogsPage } = await import(
    '@/app/(dashboard)/admin/audit/page'
  );
  const queryClient = makeQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <AuditLogsPage />
    </QueryClientProvider>
  );
}

describe('Audit Logs Page', () => {
  it('test_pageHeaderVisible: renders the page header', async () => {
    await renderAuditPage();
    expect(screen.getByText('Audit Logs')).toBeInTheDocument();
  });

  it('test_loadsAndDisplaysLogs: renders audit log rows after load', async () => {
    await renderAuditPage();
    await waitFor(() => {
      expect(screen.getByText('user.create')).toBeInTheDocument();
      expect(screen.getByText('login.failed')).toBeInTheDocument();
    });
  });

  it('test_showsUserEmail: displays user email for logged-in actions', async () => {
    await renderAuditPage();
    await waitFor(() => {
      expect(screen.getByText('alice@example.com')).toBeInTheDocument();
    });
  });

  it('test_showsSystemForNullUser: shows System for null user_email', async () => {
    await renderAuditPage();
    await waitFor(() => {
      expect(screen.getByText('System')).toBeInTheDocument();
    });
  });

  it('test_verifyHashChainButtonVisible: verify hash chain button is shown', async () => {
    await renderAuditPage();
    expect(screen.getByText('Verify Hash Chain')).toBeInTheDocument();
  });

  it('test_errorState: shows error on API failure', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/audit/logs`, () =>
        HttpResponse.json({ message: 'Server error' }, { status: 500 })
      )
    );
    await renderAuditPage();
    await waitFor(() => {
      expect(
        screen.getByText(/failed to load/i) ||
        screen.getByText(/error/i) ||
        screen.queryByRole('alert')
      ).toBeTruthy();
    });
  });

  it('test_verifyChain_success: verify chain button calls API and shows result', async () => {
    server.use(
      http.post(`${API_URL}/api/v1/audit/verify`, () =>
        HttpResponse.json({ valid: true, count: 2 })
      )
    );
    const user = userEvent.setup();
    await renderAuditPage();
    const btn = screen.getByText('Verify Hash Chain');
    await user.click(btn);
    await waitFor(() => {
      expect(
        screen.getByText(/chain valid/i) ||
        screen.getByText(/2 records/i) ||
        screen.getByText(/valid/i)
      ).toBeTruthy();
    });
  });
});
