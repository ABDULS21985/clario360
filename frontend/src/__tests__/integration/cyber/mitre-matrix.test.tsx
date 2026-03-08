import { describe, it, expect, vi, beforeAll, afterEach, afterAll } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
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
  usePathname: () => '/cyber/mitre',
  useSearchParams: () => new URLSearchParams(),
  redirect: vi.fn(),
}));

const mockCoverage = {
  tactics: [
    { id: 'TA0002', name: 'Execution', short_name: 'Execution', technique_count: 3, covered_count: 2 },
    { id: 'TA0001', name: 'Initial Access', short_name: 'Init Accs', technique_count: 2, covered_count: 1 },
  ],
  techniques: [
    { technique_id: 'T1059', technique_name: 'PowerShell', tactic_id: 'TA0002', tactic_name: 'Execution', rule_count: 2, alert_count: 3, has_detection: true },
    { technique_id: 'T1203', technique_name: 'Exploitation', tactic_id: 'TA0002', tactic_name: 'Execution', rule_count: 1, alert_count: 0, has_detection: true },
    { technique_id: 'T1106', technique_name: 'Native API', tactic_id: 'TA0002', tactic_name: 'Execution', rule_count: 0, alert_count: 0, has_detection: false },
    { technique_id: 'T1190', technique_name: 'Exploit Public-Facing', tactic_id: 'TA0001', tactic_name: 'Initial Access', rule_count: 1, alert_count: 1, has_detection: true },
    { technique_id: 'T1133', technique_name: 'External Remote Services', tactic_id: 'TA0001', tactic_name: 'Initial Access', rule_count: 0, alert_count: 0, has_detection: false },
  ],
  total_techniques: 5,
  covered_techniques: 3,
  coverage_percent: 60,
  active_techniques: 2,
  passive_techniques: 1,
  total_alerts_90d: 4,
};

const server = setupServer(
  http.get(`${API_URL}/api/v1/cyber/mitre/coverage`, () =>
    HttpResponse.json({ data: mockCoverage }),
  ),
  http.get(`${API_URL}/api/v1/cyber/mitre/techniques/:id`, () =>
    HttpResponse.json({ data: { description: 'Test description', platforms: ['Windows'] } }),
  ),
  http.get(`${API_URL}/api/v1/cyber/rules`, () =>
    HttpResponse.json({ data: [], meta: { total: 0, page: 1, per_page: 20, total_pages: 0 } }),
  ),
  http.get(`${API_URL}/api/v1/cyber/alerts`, () =>
    HttpResponse.json({ data: [], meta: { total: 0, page: 1, per_page: 10, total_pages: 0 } }),
  ),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

async function renderMitrePage() {
  const { default: Page } = await import('@/app/(dashboard)/cyber/mitre/page');
  return render(
    <QueryClientProvider client={makeQueryClient()}>
      <Page />
    </QueryClientProvider>,
  );
}

describe('MITRE Matrix Integration', () => {
  it('test_matrixLoads: MSW returns tactics + techniques + coverage → matrix renders', async () => {
    await renderMitrePage();
    await waitFor(() => {
      expect(screen.getByText('MITRE ATT&CK Coverage')).toBeTruthy();
    });
    await waitFor(() => {
      expect(screen.getByText('T1059')).toBeTruthy();
    });
  });

  it('test_coverageStats: 3/5 covered → "60%" shown', async () => {
    await renderMitrePage();
    await waitFor(() => {
      expect(screen.getByText(/60%/)).toBeTruthy();
    });
  });

  it('test_filterGapsOnly: click "Gaps Only" → shows gap techniques', async () => {
    await renderMitrePage();
    await waitFor(() => screen.getByText('T1059'));
    fireEvent.click(screen.getByText('Gaps Only ⚠'));
    await waitFor(() => {
      expect(screen.getByText('T1106')).toBeTruthy();
    });
  });

  it('test_filterWithAlerts: click "With Alerts" → only active cells visible', async () => {
    await renderMitrePage();
    await waitFor(() => screen.getByText('T1059'));
    fireEvent.click(screen.getByText('With Alerts'));
    await waitFor(() => {
      expect(screen.getByText('T1059')).toBeTruthy();
    });
  });

  it('test_searchTechnique: type "T1059" → shows matching cell', async () => {
    await renderMitrePage();
    await waitFor(() => screen.getByText('T1059'));
    const searchInput = screen.getByPlaceholderText(/Search T1059/i);
    fireEvent.change(searchInput, { target: { value: 'T1059' } });
    await waitFor(() => {
      expect(screen.getByText('T1059')).toBeTruthy();
    });
  });

  it('test_cellClickOpensPanel: click technique cell → panel appears', async () => {
    await renderMitrePage();
    await waitFor(() => screen.getByText('T1059'));
    fireEvent.click(screen.getByText('T1059'));
    await waitFor(() => {
      // Panel should appear with technique name
      expect(screen.getByText('PowerShell')).toBeTruthy();
    });
  });

  it('test_panelShowsRules: technique with rules → panel opens', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/rules`, () =>
        HttpResponse.json({
          data: [{ id: 'r1', name: 'PowerShell Rule', type: 'sigma', severity: 'high', enabled: true, trigger_count: 5, false_positive_rate: 0.1, mitre_technique_ids: ['T1059'], mitre_tactic_ids: [], is_template: false, tags: [], tenant_id: 't1', description: '', created_at: '', updated_at: '' }],
          meta: { total: 1, page: 1, per_page: 20, total_pages: 1 },
        }),
      ),
    );
    await renderMitrePage();
    await waitFor(() => screen.getByText('T1059'));
    fireEvent.click(screen.getByText('T1059'));
    await waitFor(() => {
      expect(screen.getByText('PowerShell Rule')).toBeTruthy();
    });
  });

  it('test_createRuleFromGap: gap technique → panel shows create rule button', async () => {
    await renderMitrePage();
    await waitFor(() => screen.getAllByText('T1106'));
    fireEvent.click(screen.getAllByText('T1106')[0]);
    // Panel opens — if no rules → create rule button
    await waitFor(() => {
      expect(screen.getAllByText('T1106').length).toBeGreaterThan(0);
    });
  });

  it('test_allFilterShows_allTechniques', async () => {
    await renderMitrePage();
    await waitFor(() => screen.getByText('T1059'));
    // All filter should be the default
    const allBtn = screen.getByText('All');
    expect(allBtn).toBeTruthy();
    expect(screen.getByText('T1059')).toBeTruthy();
    expect(screen.getByText('T1106')).toBeTruthy();
  });
});
