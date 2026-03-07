import { describe, it, expect, vi, beforeAll, afterEach, afterAll } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MitreTechniquePanel } from '@/app/(dashboard)/cyber/mitre/_components/mitre-technique-panel';
import type { MITRETechniqueCoverage, DetectionRule } from '@/types/cyber';

const API_URL = 'http://localhost:8080';

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn() }),
}));

const mockTech: MITRETechniqueCoverage = {
  technique_id: 'T1059',
  technique_name: 'Command and Scripting Interpreter',
  tactic_id: 'TA0002',
  tactic_name: 'Execution',
  rule_count: 2,
  alert_count: 3,
  has_detection: true,
};

const mockRules: DetectionRule[] = [
  {
    id: 'rule-1',
    tenant_id: 't1',
    name: 'PowerShell Rule',
    description: 'Detects PowerShell',
    type: 'sigma',
    severity: 'high',
    enabled: true,
    mitre_technique_ids: ['T1059'],
    mitre_tactic_ids: ['TA0002'],
    trigger_count: 5,
    false_positive_rate: 0.1,
    is_template: false,
    tags: [],
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

const server = setupServer(
  http.get(`${API_URL}/api/v1/cyber/mitre/techniques/T1059`, () =>
    HttpResponse.json({ data: { ...mockTech, description: 'Adversaries may abuse scripting.', platforms: ['Windows', 'Linux'] } }),
  ),
  http.get(`${API_URL}/api/v1/cyber/rules`, () =>
    HttpResponse.json({ data: mockRules, meta: { total: 1, page: 1, per_page: 20, total_pages: 1 } }),
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

function renderPanel(tech: MITRETechniqueCoverage | null = mockTech) {
  return render(
    <QueryClientProvider client={makeQueryClient()}>
      <MitreTechniquePanel technique={tech} onClose={vi.fn()} onCreateRule={vi.fn()} />
    </QueryClientProvider>,
  );
}

describe('MitreTechniquePanel', () => {
  it('test_rendersDescription: technique → description shown', async () => {
    renderPanel();
    await waitFor(() => {
      expect(screen.getByText(/Adversaries may abuse scripting/i)).toBeTruthy();
    });
  });

  it('test_detectionRulesList: 2 rules (from mockRules) → rule name shown', async () => {
    renderPanel();
    await waitFor(() => {
      expect(screen.getByText('PowerShell Rule')).toBeTruthy();
    });
  });

  it('test_gapRecommendation: no rules → "detection gap" message shown', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/rules`, () =>
        HttpResponse.json({ data: [], meta: { total: 0, page: 1, per_page: 20, total_pages: 0 } }),
      ),
    );
    const gapTech: MITRETechniqueCoverage = { ...mockTech, rule_count: 0, alert_count: 0, has_detection: false };
    renderPanel(gapTech);
    await waitFor(() => {
      expect(screen.getByText(/Detection Gap/i)).toBeTruthy();
    });
  });

  it('test_techniqueId_displayed', () => {
    renderPanel();
    expect(screen.getByText('T1059')).toBeTruthy();
  });

  it('test_tacticName_displayed', () => {
    renderPanel();
    expect(screen.getByText('Execution')).toBeTruthy();
  });
});
