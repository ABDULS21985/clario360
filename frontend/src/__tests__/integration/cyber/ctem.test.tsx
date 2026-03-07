import { describe, it, expect, beforeAll, afterEach, afterAll, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const API_URL = 'http://localhost:8080';

vi.mock('@/hooks/use-auth', () => ({
  useAuth: () => ({
    user: { id: 'u1', first_name: 'Admin', permissions: ['cyber:read'] },
    isHydrated: true,
    hasPermission: () => true,
  }),
}));

vi.mock('@/hooks/use-websocket', () => ({
  useWebSocket: () => ({ isConnected: false }),
}));

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), back: vi.fn(), prefetch: vi.fn() }),
  usePathname: () => '/cyber/ctem',
  useSearchParams: () => ({ get: () => null, forEach: () => {} }),
  redirect: vi.fn(),
}));

const mockExposureScore = {
  score: 62,
  grade: 'C',
  trend: 'down',
  trend_delta: -3.2,
  calculated_at: '2024-01-01T00:00:00Z',
};

const mockAssessments = [
  {
    id: 'assessment-1',
    tenant_id: 't1',
    name: 'Q1 2024 Assessment',
    status: 'completed',
    phases: [],
    scope: { all_assets: true, include_external_exposure: false },
    findings_summary: { critical: 2, high: 5, medium: 8, low: 3, total: 18 },
    exposure_score: 58,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-15T00:00:00Z',
  },
];

const server = setupServer(
  http.get(`${API_URL}/api/v1/cyber/ctem/exposure-score`, () =>
    HttpResponse.json({ data: mockExposureScore }),
  ),
  http.get(`${API_URL}/api/v1/cyber/ctem/exposure-score/history`, () =>
    HttpResponse.json({ data: [] }),
  ),
  http.get(`${API_URL}/api/v1/cyber/ctem/assessments`, () =>
    HttpResponse.json({
      data: mockAssessments,
      meta: { page: 1, per_page: 25, total: 1, total_pages: 1 },
    }),
  ),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

async function renderPage() {
  const { default: Page } = await import('@/app/(dashboard)/cyber/ctem/page');
  return render(
    <QueryClientProvider client={makeQueryClient()}>
      <Page />
    </QueryClientProvider>,
  );
}

describe('CTEM Page', () => {
  it('renders page header', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText(/continuous threat exposure/i)).toBeInTheDocument();
    });
  });

  it('shows exposure score', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText('62')).toBeInTheDocument();
    });
  });

  it('shows assessment card', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText('Q1 2024 Assessment')).toBeInTheDocument();
    });
  });

  it('shows New Assessment button', async () => {
    await renderPage();
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /new assessment/i })).toBeInTheDocument();
    });
  });

  it('shows error state when assessments API fails', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/ctem/assessments`, () =>
        HttpResponse.json({ error: 'server error' }, { status: 500 }),
      ),
    );
    await renderPage();
    await waitFor(() => {
      expect(screen.getByText(/failed to load/i)).toBeInTheDocument();
    });
  });
});
