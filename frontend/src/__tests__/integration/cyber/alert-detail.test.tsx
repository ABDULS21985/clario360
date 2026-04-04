import { describe, it, expect, beforeAll, afterEach, afterAll, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const API_URL = 'http://localhost:8080';
const ALERT_ID = 'alert-abc-123';

vi.mock('@/hooks/use-auth', () => ({
  useAuth: () => ({
    user: { id: 'u1', first_name: 'Admin', permissions: ['cyber:read'], roles: [] },
    isAuthenticated: true,
    isHydrated: true,
    hasPermission: () => true,
    hasAnyPermission: () => true,
    hasAllPermissions: () => true,
  }),
}));

vi.mock('@/hooks/use-websocket', () => ({
  useWebSocket: () => ({ isConnected: false }),
}));

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), back: vi.fn(), prefetch: vi.fn() }),
  usePathname: () => `/cyber/alerts/${ALERT_ID}`,
  useSearchParams: () => ({ get: () => null }),
  redirect: vi.fn(),
}));

// Mock React.use for params
vi.mock('react', async (importOriginal) => {
  const React = await importOriginal<typeof import('react')>();
  return {
    ...React,
    use: (promise: unknown) => {
      if (promise instanceof Promise) {
        return { id: ALERT_ID };
      }
      return promise;
    },
  };
});

const mockAlert = {
  id: ALERT_ID,
  tenant_id: 't1',
  title: 'Lateral Movement Detected',
  description: 'Suspicious lateral movement from 192.168.1.10',
  severity: 'critical',
  status: 'investigating',
  source: 'SIEM',
  explanation: {
    summary: 'AI detected suspicious lateral movement behavior.',
    reason: 'Multiple failed logins followed by successful access.',
    evidence: [],
    matched_conditions: ['Unusual login pattern'],
    confidence_factors: [],
    recommended_actions: ['Isolate host'],
    false_positive_indicators: [],
  },
  confidence_score: 89,
  event_count: 23,
  first_event_at: '2024-01-01T00:00:00Z',
  last_event_at: '2024-01-01T01:00:00Z',
  tags: ['lateral-movement'],
  asset_ids: [],
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

const server = setupServer(
  http.get(`${API_URL}/api/v1/cyber/alerts/${ALERT_ID}`, () =>
    HttpResponse.json({ data: mockAlert }),
  ),
  http.get(`${API_URL}/api/v1/cyber/alerts/${ALERT_ID}/comments`, () =>
    HttpResponse.json({ data: [] }),
  ),
  http.get(`${API_URL}/api/v1/cyber/alerts/${ALERT_ID}/timeline`, () =>
    HttpResponse.json({ data: [] }),
  ),
  http.get(`${API_URL}/api/v1/cyber/alerts/${ALERT_ID}/related`, () =>
    HttpResponse.json({ data: [] }),
  ),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

async function renderPage() {
  const { default: Page } = await import('@/app/(dashboard)/cyber/alerts/[id]/page');
  return render(
    <QueryClientProvider client={makeQueryClient()}>
      <Page params={{ id: ALERT_ID }} />
    </QueryClientProvider>,
  );
}

describe('Alert Detail Page', () => {
  it('renders alert title', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText('Lateral Movement Detected')).toBeInTheDocument();
    });
  });

  it('shows AI explanation tab', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByRole('tab', { name: /AI Explanation/i })).toBeInTheDocument();
    });
  });

  it('shows action buttons', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /change status/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /assign/i })).toBeInTheDocument();
    });
  });

  it('shows error state on API failure', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/alerts/${ALERT_ID}`, () =>
        HttpResponse.json({ error: 'not found' }, { status: 404 }),
      ),
    );
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText(/failed to load/i)).toBeInTheDocument();
    });
  });
});
