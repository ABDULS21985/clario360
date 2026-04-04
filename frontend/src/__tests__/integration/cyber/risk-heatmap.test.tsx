import { describe, it, expect, vi, beforeAll, afterEach, afterAll } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const API_URL = 'http://localhost:8080';

vi.mock('@/hooks/use-auth', () => ({
  useAuth: () => ({
    user: { id: 'u1', permissions: ['cyber:read'] },
    isHydrated: true,
    hasPermission: () => true,
  }),
}));
vi.mock('@/hooks/use-websocket', () => ({
  useWebSocket: () => ({ isConnected: false }),
}));
vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), back: vi.fn(), prefetch: vi.fn() }),
  usePathname: () => '/cyber/risk-heatmap',
  useSearchParams: () => new URLSearchParams(),
  redirect: vi.fn(),
}));

const mockHeatmap = {
  asset_types: ['server', 'endpoint', 'database'],
  total_vulnerabilities: 150,
  generated_at: '2024-01-01T00:00:00Z',
  cells: [
    { asset_type: 'server', severity: 'critical', count: 23, affected_asset_count: 5, total_assets_of_type: 10 },
    { asset_type: 'server', severity: 'high', count: 45, affected_asset_count: 8, total_assets_of_type: 10 },
    { asset_type: 'server', severity: 'medium', count: 67, affected_asset_count: 10, total_assets_of_type: 10 },
    { asset_type: 'server', severity: 'low', count: 15, affected_asset_count: 4, total_assets_of_type: 10 },
    { asset_type: 'endpoint', severity: 'critical', count: 0, affected_asset_count: 0, total_assets_of_type: 20 },
    { asset_type: 'endpoint', severity: 'high', count: 10, affected_asset_count: 3, total_assets_of_type: 20 },
    { asset_type: 'endpoint', severity: 'medium', count: 20, affected_asset_count: 5, total_assets_of_type: 20 },
    { asset_type: 'endpoint', severity: 'low', count: 25, affected_asset_count: 8, total_assets_of_type: 20 },
    { asset_type: 'database', severity: 'critical', count: 15, affected_asset_count: 2, total_assets_of_type: 5 },
    { asset_type: 'database', severity: 'high', count: 10, affected_asset_count: 2, total_assets_of_type: 5 },
    { asset_type: 'database', severity: 'medium', count: 10, affected_asset_count: 2, total_assets_of_type: 5 },
    { asset_type: 'database', severity: 'low', count: 10, affected_asset_count: 2, total_assets_of_type: 5 },
  ],
};

const server = setupServer(
  http.get(`${API_URL}/api/v1/cyber/risk/heatmap`, () =>
    HttpResponse.json({ data: mockHeatmap }),
  ),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

async function renderHeatmapPage() {
  const { default: Page } = await import('@/app/(dashboard)/cyber/risk-heatmap/page');
  return render(
    <QueryClientProvider client={makeQueryClient()}>
      <Page />
    </QueryClientProvider>,
  );
}

describe('Risk Heatmap Integration', () => {
  it('test_heatmapLoads: MSW returns heatmap data → grid renders with cell values', async () => {
    await renderHeatmapPage();
    await waitFor(() => {
      expect(screen.getByText('Risk Heatmap')).toBeTruthy();
    });
    await waitFor(() => {
      // Some cell value should appear as SVG text
      const svgTexts = Array.from(document.querySelectorAll('text')).map((t) => t.textContent);
      expect(svgTexts.some((t) => t === '23')).toBe(true);
    });
  });

  it('test_totals: grand total shown', async () => {
    await renderHeatmapPage();
    await waitFor(() => {
      expect(screen.getAllByText(String(mockHeatmap.total_vulnerabilities)).length).toBeGreaterThan(0);
    });
  });

  it('test_emptyHeatmap: all zeros → "No vulnerability data" message', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/risk/heatmap`, () =>
        HttpResponse.json({
          data: {
            ...mockHeatmap,
            total_vulnerabilities: 0,
            cells: mockHeatmap.cells.map((c) => ({ ...c, count: 0 })),
          },
        }),
      ),
    );
    await renderHeatmapPage();
    await waitFor(() => {
      expect(screen.getByText(/No vulnerability data/i)).toBeTruthy();
    });
  });

  it('test_summaryInsights_shown: key insights rendered', async () => {
    await renderHeatmapPage();
    await waitFor(() => {
      // The summary table should show insight text
      expect(screen.getByText(/Key Insights/i)).toBeTruthy();
    });
  });

  it('test_apiError: shows error state on failure', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/risk/heatmap`, () =>
        HttpResponse.json({ error: 'internal error' }, { status: 500 }),
      ),
    );
    await renderHeatmapPage();
    await waitFor(() => {
      expect(screen.getByText(/Failed to load/i)).toBeTruthy();
    });
  });
});
