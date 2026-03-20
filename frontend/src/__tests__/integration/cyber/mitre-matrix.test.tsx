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

// Mock coverage response matching MITRECoverageResponseDTO from backend
const mockCoverage = {
  tactics: [
    { id: 'TA0002', name: 'Execution', short_name: 'execution', technique_count: 3, covered_count: 2 },
    { id: 'TA0001', name: 'Initial Access', short_name: 'initial-access', technique_count: 2, covered_count: 1 },
  ],
  techniques: [
    {
      technique_id: 'T1059', technique_name: 'PowerShell',
      tactic_ids: ['TA0002'], rule_count: 2, alert_count: 3, threat_count: 1,
      active_threat_count: 0, has_detection: true, coverage_state: 'covered',
      high_fp_rule_count: 0, description: 'Command interpreter', platforms: ['Windows'],
      rule_names: ['Rule A', 'Rule B'],
    },
    {
      technique_id: 'T1203', technique_name: 'Exploitation',
      tactic_ids: ['TA0002'], rule_count: 1, alert_count: 0, threat_count: 0,
      active_threat_count: 0, has_detection: true, coverage_state: 'covered',
      high_fp_rule_count: 0, description: 'Exploitation for client execution', platforms: ['Windows'],
      rule_names: ['Rule C'],
    },
    {
      technique_id: 'T1106', technique_name: 'Native API',
      tactic_ids: ['TA0002'], rule_count: 0, alert_count: 0, threat_count: 0,
      active_threat_count: 1, has_detection: false, coverage_state: 'gap',
      high_fp_rule_count: 0, description: 'Native API abuse', platforms: ['Windows'],
      rule_names: [],
    },
    {
      technique_id: 'T1190', technique_name: 'Exploit Public-Facing',
      tactic_ids: ['TA0001'], rule_count: 1, alert_count: 1, threat_count: 0,
      active_threat_count: 0, has_detection: true, coverage_state: 'covered',
      high_fp_rule_count: 0, description: 'Exploit public-facing app', platforms: ['Linux', 'Windows'],
      rule_names: ['Rule D'],
    },
    {
      technique_id: 'T1133', technique_name: 'External Remote Services',
      tactic_ids: ['TA0001'], rule_count: 0, alert_count: 0, threat_count: 0,
      active_threat_count: 0, has_detection: false, coverage_state: 'idle',
      high_fp_rule_count: 0, description: 'External remote services', platforms: ['Windows', 'Linux'],
      rule_names: [],
    },
  ],
  total_techniques: 5,
  covered_techniques: 3,
  coverage_percent: 60,
  active_techniques: 2,
  passive_techniques: 1,
  critical_gap_count: 1,
};

// Mock technique detail matching MITRETechniqueDetailDTO from backend
const mockTechniqueDetail = {
  id: 'T1059',
  name: 'PowerShell',
  description: 'Command and scripting interpreter used for execution.',
  tactic_ids: ['TA0002'],
  platforms: ['Windows'],
  data_sources: ['Process Creation'],
  coverage_state: 'covered',
  rule_count: 2,
  alert_count: 3,
  threat_count: 1,
  active_threat_count: 0,
  high_fp_rule_count: 0,
  linked_rules: [
    {
      id: 'r1', name: 'PowerShell Rule', rule_type: 'sigma', severity: 'high',
      enabled: true, trigger_count: 5, true_positive_count: 4, false_positive_count: 1,
    },
  ],
  linked_threats: [],
  recent_alerts: [],
};

const server = setupServer(
  http.get(`${API_URL}/api/v1/cyber/mitre/coverage`, () =>
    HttpResponse.json({ data: mockCoverage }),
  ),
  http.get(`${API_URL}/api/v1/cyber/mitre/techniques/:id`, () =>
    HttpResponse.json({ data: mockTechniqueDetail }),
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
  it('renders matrix with technique cells after loading', async () => {
    await renderMitrePage();
    await waitFor(() => {
      // Page header title is 'MITRE ATT&CK'
      expect(screen.getByText('MITRE ATT\u0026CK')).toBeTruthy();
    });
    await waitFor(() => {
      expect(screen.getByText('T1059')).toBeTruthy();
    });
  });

  it('displays coverage percentage from response', async () => {
    await renderMitrePage();
    await waitFor(() => {
      expect(screen.getByText(/60\.0% of techniques covered/)).toBeTruthy();
    });
  });

  it('filters to gap techniques when Gaps Only is clicked', async () => {
    await renderMitrePage();
    await waitFor(() => screen.getByText('T1059'));
    fireEvent.click(screen.getByText('Gaps Only ⚠'));
    await waitFor(() => {
      expect(screen.getByText('T1106')).toBeTruthy();
    });
  });

  it('filters to techniques with alerts when With Alerts is clicked', async () => {
    await renderMitrePage();
    await waitFor(() => screen.getByText('T1059'));
    fireEvent.click(screen.getByText('With Alerts'));
    await waitFor(() => {
      expect(screen.getByText('T1059')).toBeTruthy();
    });
  });

  it('filters techniques by search input', async () => {
    await renderMitrePage();
    await waitFor(() => screen.getByText('T1059'));
    const searchInput = screen.getByPlaceholderText(/Search T1059/i);
    fireEvent.change(searchInput, { target: { value: 'T1059' } });
    await waitFor(() => {
      expect(screen.getByText('T1059')).toBeTruthy();
    });
  });

  it('opens technique panel on cell click', async () => {
    await renderMitrePage();
    await waitFor(() => screen.getByText('T1059'));
    fireEvent.click(screen.getByText('T1059'));
    await waitFor(() => {
      // Panel loads technique detail and shows description unique to the detail response
      expect(screen.getByText(/Command and scripting interpreter used for execution/i)).toBeTruthy();
    });
  });

  it('shows linked rules in panel on cell click', async () => {
    await renderMitrePage();
    await waitFor(() => screen.getByText('T1059'));
    fireEvent.click(screen.getByText('T1059'));
    await waitFor(() => {
      expect(screen.getByText('PowerShell Rule')).toBeTruthy();
    });
  });

  it('shows all techniques with All filter (default)', async () => {
    await renderMitrePage();
    await waitFor(() => screen.getByText('T1059'));
    const allBtn = screen.getByText('All');
    expect(allBtn).toBeTruthy();
    expect(screen.getByText('T1059')).toBeTruthy();
    expect(screen.getByText('T1106')).toBeTruthy();
  });
});
