import { describe, it, expect, vi, beforeAll, afterEach, afterAll } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MitreTechniquePanel } from '@/app/(dashboard)/cyber/mitre/_components/mitre-technique-panel';
import type { MITRETechniqueCoverage, MITRETechniqueDetail } from '@/types/cyber';

const API_URL = 'http://localhost:8080';

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn() }),
}));

const mockTech: MITRETechniqueCoverage = {
  technique_id: 'T1059',
  technique_name: 'Command and Scripting Interpreter',
  tactic_ids: ['TA0002'],
  tactic_id: 'TA0002',
  tactic_name: 'Execution',
  rule_count: 2,
  alert_count: 3,
  threat_count: 1,
  active_threat_count: 1,
  has_detection: true,
  coverage_state: 'covered',
  high_fp_rule_count: 0,
  description: 'Adversaries may abuse scripting interpreters.',
  platforms: ['Windows', 'Linux'],
};

// Mock detail response matching MITRETechniqueDetailDTO from backend
const mockDetail: MITRETechniqueDetail = {
  id: 'T1059',
  name: 'Command and Scripting Interpreter',
  description: 'Adversaries may abuse scripting interpreters to execute commands.',
  tactic_ids: ['TA0002'],
  platforms: ['Windows', 'Linux'],
  data_sources: ['Process Creation'],
  coverage_state: 'covered',
  rule_count: 2,
  alert_count: 3,
  threat_count: 1,
  active_threat_count: 1,
  high_fp_rule_count: 0,
  linked_rules: [
    {
      id: 'rule-1',
      name: 'PowerShell Rule',
      rule_type: 'sigma',
      severity: 'high',
      enabled: true,
      trigger_count: 5,
      true_positive_count: 4,
      false_positive_count: 1,
    },
  ],
  linked_threats: [],
  recent_alerts: [],
};

const server = setupServer(
  http.get(`${API_URL}/api/v1/cyber/mitre/techniques/T1059`, () =>
    HttpResponse.json({ data: mockDetail }),
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
      <MitreTechniquePanel technique={tech} onClose={vi.fn()} />
    </QueryClientProvider>,
  );
}

describe('MitreTechniquePanel', () => {
  it('renders technique description from detail endpoint', async () => {
    renderPanel();
    await waitFor(() => {
      expect(screen.getByText(/Adversaries may abuse scripting interpreters/i)).toBeTruthy();
    });
  });

  it('renders linked detection rules', async () => {
    renderPanel();
    await waitFor(() => {
      expect(screen.getByText('PowerShell Rule')).toBeTruthy();
    });
  });

  it('shows empty state when no rules cover the technique', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/mitre/techniques/T1059`, () =>
        HttpResponse.json({
          data: {
            ...mockDetail,
            coverage_state: 'gap',
            rule_count: 0,
            linked_rules: [],
          },
        }),
      ),
    );
    const gapTech: MITRETechniqueCoverage = { ...mockTech, rule_count: 0, alert_count: 0, has_detection: false, coverage_state: 'gap' };
    renderPanel(gapTech);
    await waitFor(() => {
      expect(screen.getByText(/No detection rules cover this technique yet/i)).toBeTruthy();
    });
  });

  it('renders the technique ID badge', async () => {
    renderPanel();
    await waitFor(() => {
      expect(screen.getByText('T1059')).toBeTruthy();
    });
  });

  it('renders coverage state badge', async () => {
    renderPanel();
    await waitFor(() => {
      expect(screen.getByText('covered')).toBeTruthy();
    });
  });

  it('renders metric section headers', async () => {
    renderPanel();
    await waitFor(() => {
      expect(screen.getByText('Rules')).toBeTruthy();
      expect(screen.getByText('Alerts')).toBeTruthy();
      expect(screen.getByText('Active Threats')).toBeTruthy();
    });
  });
});
