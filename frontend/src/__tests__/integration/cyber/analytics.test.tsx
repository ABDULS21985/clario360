import { describe, it, expect, beforeAll, afterEach, afterAll } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { renderWithQuery } from '../../utils/render-with-query';

// Components under test
import { ThreatLandscape } from '@/app/(dashboard)/cyber/analytics/_components/threat-landscape';
import { ThreatForecast } from '@/app/(dashboard)/cyber/analytics/_components/threat-forecast';
import { AlertVolumeForecast } from '@/app/(dashboard)/cyber/analytics/_components/alert-volume-forecast';
import { TechniqueTrends } from '@/app/(dashboard)/cyber/analytics/_components/technique-trends';
import { CampaignDetection } from '@/app/(dashboard)/cyber/analytics/_components/campaign-detection';

const API_URL = 'http://localhost:8080';

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

const mockLandscape = {
  active_threat_count: 12,
  total_threats: 42,
  indicators_total: 256,
  top_threat_type: 'apt',
  by_type: [
    { name: 'apt', count: 8 },
    { name: 'malware', count: 4 },
  ],
  by_severity: [
    { name: 'critical', count: 5 },
    { name: 'high', count: 7 },
  ],
};

const mockThreatForecast = {
  prediction_type: 'attack_technique_trend',
  model_version: 'technique-trend-v1',
  confidence_score: 0.85,
  items: [
    {
      technique_id: 'T1566',
      technique_name: 'Phishing',
      trend: 'increasing' as const,
      growth_rate: 0.25,
      forecast: { p10: 10, p50: 15, p90: 20 },
    },
    {
      technique_id: 'T1059',
      technique_name: 'Command and Scripting Interpreter',
      trend: 'stable' as const,
      growth_rate: 0.0,
      forecast: { p10: 5, p50: 8, p90: 12 },
    },
  ],
};

const mockAlertForecast = {
  prediction_type: 'alert_volume_forecast',
  model_version: 'alert-volume-v1',
  confidence_score: 0.75,
  forecast: {
    horizon_days: 30,
    points: [
      { timestamp: '2026-03-23T00:00:00Z', value: 145.3, bounds: { p10: 130, p50: 145, p90: 160 } },
      { timestamp: '2026-03-24T00:00:00Z', value: 150.0, bounds: { p10: 135, p50: 150, p90: 165 } },
      { timestamp: '2026-03-25T00:00:00Z', value: 148.5, bounds: { p10: 133, p50: 148, p90: 163 } },
    ],
    anomaly_flag: false,
  },
};

const mockCampaigns = {
  prediction_type: 'campaign_detection',
  model_version: 'campaign-detector-v1',
  items: [
    {
      cluster_id: 'campaign-1',
      alert_ids: ['a1', 'a2', 'a3'],
      alert_titles: ['Phishing email detected', 'Malicious attachment', 'C2 callback'],
      start_at: '2026-03-20T10:00:00Z',
      end_at: '2026-03-22T14:00:00Z',
      stage: 'active_attack',
      mitre_techniques: ['T1566', 'T1059'],
      shared_iocs: ['192.168.1.100', 'evil.example.com'],
      confidence_interval: { p10: 0.7, p50: 0.8, p90: 0.9 },
    },
  ],
};

// ---------------------------------------------------------------------------
// MSW server
// ---------------------------------------------------------------------------

const server = setupServer(
  http.get(`${API_URL}/api/v1/cyber/analytics/landscape`, () =>
    HttpResponse.json({ data: mockLandscape }),
  ),
  http.get(`${API_URL}/api/v1/cyber/analytics/threat-forecast`, () =>
    HttpResponse.json({ data: mockThreatForecast }),
  ),
  http.get(`${API_URL}/api/v1/cyber/analytics/alert-forecast`, () =>
    HttpResponse.json({ data: mockAlertForecast }),
  ),
  http.get(`${API_URL}/api/v1/cyber/analytics/technique-trends`, () =>
    HttpResponse.json({ data: mockThreatForecast }),
  ),
  http.get(`${API_URL}/api/v1/cyber/analytics/campaigns`, () =>
    HttpResponse.json({ data: mockCampaigns }),
  ),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

// ---------------------------------------------------------------------------
// ThreatLandscape
// ---------------------------------------------------------------------------

describe('ThreatLandscape', () => {
  it('renders KPI cards and chart titles with data', async () => {
    renderWithQuery(<ThreatLandscape />);

    await waitFor(() => {
      expect(screen.getByText('Active Threats')).toBeInTheDocument();
    });
    expect(screen.getByText('12')).toBeInTheDocument();
    expect(screen.getByText('Total IOCs')).toBeInTheDocument();
    expect(screen.getByText('256')).toBeInTheDocument();
    expect(screen.getByText('Top Threat Type')).toBeInTheDocument();
    expect(screen.getByText('Threats by Type')).toBeInTheDocument();
    expect(screen.getByText('Threats by Severity')).toBeInTheDocument();
  });

  it('renders error state with retry button', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/analytics/landscape`, () =>
        HttpResponse.json({ error: 'fail' }, { status: 500 }),
      ),
    );

    renderWithQuery(<ThreatLandscape />);

    await waitFor(() => {
      expect(screen.getByText('Failed to load threat landscape data.')).toBeInTheDocument();
    });
    expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// ThreatForecast
// ---------------------------------------------------------------------------

describe('ThreatForecast', () => {
  it('renders emerging threats with increasing trend only', async () => {
    renderWithQuery(<ThreatForecast />);

    await waitFor(() => {
      expect(screen.getByText('Emerging Threats — 7-Day Forecast')).toBeInTheDocument();
    });
    // T1566 has trend=increasing, should be visible
    expect(screen.getByText('T1566')).toBeInTheDocument();
    expect(screen.getByText('Phishing')).toBeInTheDocument();
    expect(screen.getByText('+25.0%')).toBeInTheDocument();

    // T1059 has trend=stable, should be filtered out
    expect(screen.queryByText('T1059')).not.toBeInTheDocument();
  });

  it('shows empty message when no increasing techniques', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/analytics/threat-forecast`, () =>
        HttpResponse.json({
          data: {
            ...mockThreatForecast,
            items: [{ ...mockThreatForecast.items[1], trend: 'stable' }],
          },
        }),
      ),
    );

    renderWithQuery(<ThreatForecast />);

    await waitFor(() => {
      expect(
        screen.getByText('No techniques are forecasted to increase in the next 7 days.'),
      ).toBeInTheDocument();
    });
  });

  it('renders error state with retry', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/analytics/threat-forecast`, () =>
        HttpResponse.json({ error: 'fail' }, { status: 500 }),
      ),
    );

    renderWithQuery(<ThreatForecast />);

    await waitFor(() => {
      expect(screen.getByText('Failed to load threat forecast.')).toBeInTheDocument();
    });
    expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// AlertVolumeForecast
// ---------------------------------------------------------------------------

describe('AlertVolumeForecast', () => {
  it('renders chart when data is available', async () => {
    renderWithQuery(<AlertVolumeForecast />);

    await waitFor(() => {
      expect(screen.getByText('Alert Volume Forecast (30 Days)')).toBeInTheDocument();
    });
  });

  it('shows insufficient data message when points are empty', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/analytics/alert-forecast`, () =>
        HttpResponse.json({
          data: {
            ...mockAlertForecast,
            forecast: { ...mockAlertForecast.forecast, points: [] },
          },
        }),
      ),
    );

    renderWithQuery(<AlertVolumeForecast />);

    await waitFor(() => {
      expect(
        screen.getByText('Insufficient data to generate alert volume forecast.'),
      ).toBeInTheDocument();
    });
  });

  it('renders error state with retry', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/analytics/alert-forecast`, () =>
        HttpResponse.json({ error: 'fail' }, { status: 500 }),
      ),
    );

    renderWithQuery(<AlertVolumeForecast />);

    await waitFor(() => {
      expect(screen.getByText('Failed to load alert volume forecast.')).toBeInTheDocument();
    });
    expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// TechniqueTrends
// ---------------------------------------------------------------------------

describe('TechniqueTrends', () => {
  it('renders technique table with data', async () => {
    renderWithQuery(<TechniqueTrends />);

    await waitFor(() => {
      expect(screen.getByText('Attack Technique Trends (30 Days)')).toBeInTheDocument();
    });
    // Both techniques should appear (no filtering unlike ThreatForecast)
    expect(screen.getByText('T1566')).toBeInTheDocument();
    expect(screen.getByText('Phishing')).toBeInTheDocument();
    expect(screen.getByText('T1059')).toBeInTheDocument();
  });

  it('shows empty message when no technique data', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/analytics/technique-trends`, () =>
        HttpResponse.json({ data: { items: [] } }),
      ),
    );

    renderWithQuery(<TechniqueTrends />);

    await waitFor(() => {
      expect(screen.getByText('No technique trend data available yet.')).toBeInTheDocument();
    });
  });

  it('renders error state with retry', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/analytics/technique-trends`, () =>
        HttpResponse.json({ error: 'fail' }, { status: 500 }),
      ),
    );

    renderWithQuery(<TechniqueTrends />);

    await waitFor(() => {
      expect(screen.getByText('Failed to load technique trend data.')).toBeInTheDocument();
    });
    expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// CampaignDetection
// ---------------------------------------------------------------------------

describe('CampaignDetection', () => {
  it('renders campaign cards with correct data', async () => {
    renderWithQuery(<CampaignDetection />);

    await waitFor(() => {
      expect(screen.getByText('Campaign Detection')).toBeInTheDocument();
    });
    expect(screen.getByText('Campaign #campaign-1')).toBeInTheDocument();
    expect(screen.getByText('active attack')).toBeInTheDocument();
    expect(screen.getByText('3')).toBeInTheDocument(); // alert count
    expect(screen.getByText('80%')).toBeInTheDocument(); // confidence p50
    expect(screen.getByText('T1566')).toBeInTheDocument();
    expect(screen.getByText('T1059')).toBeInTheDocument();
  });

  it('builds correct Investigate link with alert_ids', async () => {
    renderWithQuery(<CampaignDetection />);

    await waitFor(() => {
      expect(screen.getByText('Investigate Alerts')).toBeInTheDocument();
    });

    const link = screen.getByRole('link', { name: /investigate alerts/i });
    expect(link).toHaveAttribute(
      'href',
      '/cyber/alerts?alert_ids=a1&alert_ids=a2&alert_ids=a3',
    );
  });

  it('renders shared IOCs', async () => {
    renderWithQuery(<CampaignDetection />);

    await waitFor(() => {
      expect(screen.getByText('192.168.1.100')).toBeInTheDocument();
    });
    expect(screen.getByText('evil.example.com')).toBeInTheDocument();
  });

  it('shows empty state when no campaigns detected', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/analytics/campaigns`, () =>
        HttpResponse.json({ data: { items: [] } }),
      ),
    );

    renderWithQuery(<CampaignDetection />);

    await waitFor(() => {
      expect(
        screen.getByText(/no active campaigns detected/i),
      ).toBeInTheDocument();
    });
  });

  it('renders error state with retry', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/analytics/campaigns`, () =>
        HttpResponse.json({ error: 'fail' }, { status: 500 }),
      ),
    );

    renderWithQuery(<CampaignDetection />);

    await waitFor(() => {
      expect(screen.getByText('Failed to load campaign data.')).toBeInTheDocument();
    });
    expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument();
  });

  it('retry button refetches data on click', async () => {
    const user = userEvent.setup();
    let callCount = 0;

    server.use(
      http.get(`${API_URL}/api/v1/cyber/analytics/campaigns`, () => {
        callCount++;
        if (callCount <= 1) {
          return HttpResponse.json({ error: 'fail' }, { status: 500 });
        }
        return HttpResponse.json({ data: mockCampaigns });
      }),
    );

    renderWithQuery(<CampaignDetection />);

    await waitFor(() => {
      expect(screen.getByText('Failed to load campaign data.')).toBeInTheDocument();
    });

    await user.click(screen.getByRole('button', { name: /retry/i }));

    await waitFor(() => {
      expect(screen.getByText('Campaign #campaign-1')).toBeInTheDocument();
    });
  });
});
