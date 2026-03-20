import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithQuery } from '@/__tests__/utils/render-with-query';
import AIModelDetailPage from '@/app/(dashboard)/admin/ai-governance/[modelId]/page';

const {
  authState,
  replaceMock,
  routeState,
  showApiErrorMock,
  showSuccessMock,
} = vi.hoisted(() => ({
  authState: { user: { id: 'user-1' } },
  replaceMock: vi.fn(),
  routeState: { modelId: 'model-1' },
  showApiErrorMock: vi.fn(),
  showSuccessMock: vi.fn(),
}));

vi.mock('next/navigation', () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: replaceMock,
    back: vi.fn(),
    prefetch: vi.fn(),
  }),
  usePathname: () => `/admin/ai-governance/${routeState.modelId}`,
  useSearchParams: () => new URLSearchParams(),
  useParams: () => ({ modelId: routeState.modelId }),
}));

vi.mock('@/hooks/use-auth', () => ({
  useAuth: () => ({
    user: authState.user,
    hasPermission: () => true,
    isHydrated: true,
  }),
}));

vi.mock('@/lib/toast', () => ({
  showSuccess: showSuccessMock,
  showApiError: showApiErrorMock,
}));

vi.mock('@/components/shared/charts/line-chart', () => ({
  LineChart: ({ data }: { data: Array<unknown> }) => (
    <div data-testid="line-chart">{data.length}</div>
  ),
}));

const API_URL = 'http://localhost:8080';

let requestCounts: Record<string, number>;
let startShadowPayload: Record<string, unknown> | null;
let modelResponse: Record<string, unknown> | null;
let versions: Array<Record<string, unknown>>;
let history: Array<Record<string, unknown>>;
let latestComparison: Record<string, unknown>;
let comparisonHistory: Array<Record<string, unknown>>;
let divergences: Array<Record<string, unknown>>;
let latestDrift: Record<string, unknown>;
let driftHistory: Array<Record<string, unknown>>;
let performancePoints: Array<Record<string, unknown>>;
let predictionStats: Array<Record<string, unknown>>;
let predictions: Array<Record<string, unknown>>;

const server = setupServer(
  http.get(`${API_URL}/api/v1/ai/models/:modelId`, () => {
    requestCounts.model += 1;

    if (!modelResponse) {
      return HttpResponse.json({ message: 'Model not found' }, { status: 404 });
    }

    return HttpResponse.json({ data: modelResponse });
  }),
  http.get(`${API_URL}/api/v1/ai/models/:modelId/versions`, () => {
    requestCounts.versions += 1;
    return HttpResponse.json({ data: versions });
  }),
  http.get(`${API_URL}/api/v1/ai/models/:modelId/lifecycle-history`, () => {
    requestCounts.history += 1;
    return HttpResponse.json({ data: history });
  }),
  http.post(`${API_URL}/api/v1/ai/models/:modelId/shadow/start`, async ({ request }) => {
    requestCounts.startShadow += 1;
    startShadowPayload = (await request.json()) as Record<string, unknown>;

    versions = versions.map((version) =>
      version.id === startShadowPayload?.version_id
        ? {
            ...version,
            status: 'shadow',
            promoted_to_shadow_at: '2026-03-10T10:00:00Z',
          }
        : version,
    );

    history = [
      {
        version_id: 'version-2',
        version_number: 2,
        from_status: 'staging',
        to_status: 'shadow',
        changed_by: 'governance-bot',
        reason: 'Shadow mode started from the detail page.',
        changed_at: '2026-03-10T10:00:00Z',
      },
      ...history,
    ];

    return HttpResponse.json({
      data: versions.find((version) => version.id === startShadowPayload?.version_id),
    });
  }),
  http.get(`${API_URL}/api/v1/ai/models/:modelId/shadow/comparison`, () => {
    requestCounts.shadowLatest += 1;
    return HttpResponse.json({ data: latestComparison });
  }),
  http.get(`${API_URL}/api/v1/ai/models/:modelId/shadow/comparison/history`, () => {
    requestCounts.shadowHistory += 1;
    return HttpResponse.json({ data: comparisonHistory });
  }),
  http.get(`${API_URL}/api/v1/ai/models/:modelId/shadow/divergences`, ({ request }) => {
    requestCounts.divergences += 1;
    const url = new URL(request.url);
    const page = Number(url.searchParams.get('page') ?? '1');
    const perPage = Number(url.searchParams.get('per_page') ?? '10');

    return HttpResponse.json({
      data: divergences,
      meta: {
        page,
        per_page: perPage,
        total: divergences.length,
        total_pages: 1,
      },
    });
  }),
  http.get(`${API_URL}/api/v1/ai/models/:modelId/drift`, () => {
    requestCounts.driftLatest += 1;
    return HttpResponse.json({ data: latestDrift });
  }),
  http.get(`${API_URL}/api/v1/ai/models/:modelId/drift/history`, () => {
    requestCounts.driftHistory += 1;
    return HttpResponse.json({ data: driftHistory });
  }),
  http.get(`${API_URL}/api/v1/ai/models/:modelId/performance`, () => {
    requestCounts.performance += 1;
    return HttpResponse.json({ data: performancePoints });
  }),
  http.get(`${API_URL}/api/v1/ai/predictions/stats`, () => {
    requestCounts.predictionStats += 1;
    return HttpResponse.json({ data: predictionStats });
  }),
  http.get(`${API_URL}/api/v1/ai/predictions`, ({ request }) => {
    requestCounts.predictions += 1;
    const url = new URL(request.url);
    const page = Number(url.searchParams.get('page') ?? '1');
    const perPage = Number(url.searchParams.get('per_page') ?? '10');
    const modelId = url.searchParams.get('model_id');
    const filtered = predictions.filter((prediction) => prediction.model_id === modelId);

    return HttpResponse.json({
      data: filtered.slice((page - 1) * perPage, page * perPage),
      meta: {
        page,
        per_page: perPage,
        total: filtered.length,
        total_pages: Math.max(1, Math.ceil(filtered.length / perPage)),
      },
    });
  }),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => {
  server.resetHandlers();
  replaceMock.mockReset();
  showSuccessMock.mockReset();
  showApiErrorMock.mockReset();
  routeState.modelId = 'model-1';
  startShadowPayload = null;
});
afterAll(() => server.close());

beforeEach(() => {
  requestCounts = {
    divergences: 0,
    driftHistory: 0,
    driftLatest: 0,
    history: 0,
    model: 0,
    performance: 0,
    predictions: 0,
    predictionStats: 0,
    shadowHistory: 0,
    shadowLatest: 0,
    startShadow: 0,
    versions: 0,
  };

  modelResponse = {
    model: {
      id: 'model-1',
      tenant_id: 'tenant-1',
      name: 'Malware Detector',
      slug: 'malware-detector',
      description: 'Detects malware activity across tenant telemetry.',
      model_type: 'ml_classifier',
      suite: 'cyber',
      owner_user_id: null,
      owner_team: 'Threat Research',
      risk_tier: 'high',
      status: 'active',
      tags: ['malware', 'triage'],
      metadata: {},
      created_by: 'user-1',
      created_at: '2026-03-01T10:00:00Z',
      updated_at: '2026-03-08T10:00:00Z',
    },
    production_version: {
      id: 'version-1',
      tenant_id: 'tenant-1',
      model_id: 'model-1',
      version_number: 1,
      status: 'production',
      description: 'Stable production baseline.',
      artifact_type: 'serialized_model',
      artifact_config: {},
      artifact_hash: 'prodhash1234567890',
      explainability_type: 'feature_importance',
      training_metrics: {},
      prediction_count: 2400,
      avg_latency_ms: 94,
      avg_confidence: 0.94,
      accuracy_metric: 0.92,
      false_positive_rate: 0.03,
      false_negative_rate: 0.04,
      feedback_count: 42,
      created_by: 'user-1',
      created_at: '2026-03-01T10:00:00Z',
      updated_at: '2026-03-08T10:00:00Z',
    },
    shadow_version: null,
  };

  versions = [
    {
      id: 'version-2',
      tenant_id: 'tenant-1',
      model_id: 'model-1',
      version_number: 2,
      status: 'staging',
      description: 'Candidate tuned for higher recall.',
      artifact_type: 'serialized_model',
      artifact_config: {},
      artifact_hash: 'staginghash123456',
      explainability_type: 'feature_importance',
      training_metrics: {},
      prediction_count: 180,
      avg_latency_ms: 88,
      avg_confidence: 0.89,
      accuracy_metric: 0.9,
      false_positive_rate: 0.05,
      false_negative_rate: 0.03,
      feedback_count: 7,
      created_by: 'user-1',
      created_at: '2026-03-08T10:00:00Z',
      updated_at: '2026-03-09T10:00:00Z',
    },
    {
      id: 'version-1',
      tenant_id: 'tenant-1',
      model_id: 'model-1',
      version_number: 1,
      status: 'production',
      description: 'Stable production baseline.',
      artifact_type: 'serialized_model',
      artifact_config: {},
      artifact_hash: 'prodhash1234567890',
      explainability_type: 'feature_importance',
      training_metrics: {},
      prediction_count: 2400,
      avg_latency_ms: 94,
      avg_confidence: 0.94,
      accuracy_metric: 0.92,
      false_positive_rate: 0.03,
      false_negative_rate: 0.04,
      feedback_count: 42,
      created_by: 'user-1',
      created_at: '2026-03-01T10:00:00Z',
      updated_at: '2026-03-08T10:00:00Z',
    },
  ];

  history = [
    {
      version_id: 'version-1',
      version_number: 1,
      from_status: 'shadow',
      to_status: 'production',
      changed_by: 'user-1',
      reason: 'Passed shadow evaluation.',
      changed_at: '2026-03-05T09:00:00Z',
    },
  ];

  latestComparison = {
    id: 'comparison-1',
    tenant_id: 'tenant-1',
    model_id: 'model-1',
    production_version_id: 'version-1',
    shadow_version_id: 'version-2',
    period_start: '2026-03-09T00:00:00Z',
    period_end: '2026-03-10T00:00:00Z',
    total_predictions: 120,
    agreement_count: 109,
    disagreement_count: 11,
    agreement_rate: 0.91,
    production_metrics: {},
    shadow_metrics: {},
    metrics_delta: {},
    divergence_samples: [
      {
        prediction_id: 'pred-1',
        input_hash: 'input-1',
        use_case: 'Malware triage',
        production_output: { label: 'benign' },
        shadow_output: { label: 'threat' },
        reason: 'Higher recall threshold raised the threat score.',
        created_at: '2026-03-10T08:00:00Z',
      },
    ],
    divergence_by_use_case: {},
    recommendation: 'promote',
    recommendation_reason: 'Agreement is high and the candidate catches missed threats.',
    recommendation_factors: {},
    created_at: '2026-03-10T00:05:00Z',
  };

  comparisonHistory = [latestComparison];

  divergences = [
    {
      prediction_id: 'pred-1',
      input_hash: 'input-1',
      use_case: 'Malware triage',
      production_output: { label: 'benign' },
      shadow_output: { label: 'threat' },
      reason: 'Higher recall threshold raised the threat score.',
      created_at: '2026-03-10T08:00:00Z',
    },
  ];

  latestDrift = {
    id: 'drift-1',
    tenant_id: 'tenant-1',
    model_id: 'model-1',
    model_version_id: 'version-1',
    period: '24h',
    period_start: '2026-03-09T00:00:00Z',
    period_end: '2026-03-10T00:00:00Z',
    output_psi: 0.11,
    output_drift_level: 'low',
    confidence_psi: 0.08,
    confidence_drift_level: 'low',
    current_volume: 2400,
    reference_volume: 2200,
    volume_change_pct: 9.1,
    current_p95_latency_ms: 128,
    reference_p95_latency_ms: 120,
    latency_change_pct: 6.7,
    current_accuracy: 0.92,
    reference_accuracy: 0.91,
    accuracy_change: 0.01,
    alerts: [],
    alert_count: 0,
    created_at: '2026-03-10T00:05:00Z',
  };

  driftHistory = [latestDrift];

  performancePoints = [
    {
      period_start: '2026-03-09T00:00:00Z',
      volume: 2400,
      avg_latency_ms: 94,
      accuracy: 0.92,
    },
  ];

  predictionStats = [
    {
      model_id: 'model-1',
      model_slug: 'malware-detector',
      suite: 'cyber',
      use_case: 'Malware triage',
      total: 300,
      shadow_total: 45,
      correct_feedback: 30,
      wrong_feedback: 10,
    },
    {
      model_id: 'model-1',
      model_slug: 'malware-detector',
      suite: 'cyber',
      use_case: 'Sandbox verdict review',
      total: 180,
      shadow_total: 20,
      correct_feedback: 6,
      wrong_feedback: 2,
    },
    {
      model_id: 'other-model',
      model_slug: 'phishing-detector',
      suite: 'cyber',
      use_case: 'Ignore me',
      total: 100,
      shadow_total: 10,
      correct_feedback: 50,
      wrong_feedback: 5,
    },
  ];

  predictions = [
    {
      id: 'prediction-1',
      tenant_id: 'tenant-1',
      model_id: 'model-1',
      model_version_id: 'version-1',
      model_version_number: 1,
      input_hash: 'input-1',
      prediction: { label: 'threat' },
      confidence: 0.97,
      explanation_structured: {},
      explanation_text: 'High-confidence malware verdict.',
      explanation_factors: [],
      suite: 'cyber',
      use_case: 'Malware triage',
      is_shadow: false,
      latency_ms: 112,
      feedback_correct: true,
      created_at: '2026-03-10T08:00:00Z',
    },
  ];
});

describe('AI governance detail page', () => {
  it('loads the detail page and lazily fetches tab-specific data', async () => {
    const user = userEvent.setup();

    renderWithQuery(<AIModelDetailPage />);

    expect(await screen.findByRole('heading', { name: 'Malware Detector' })).toBeInTheDocument();

    await waitFor(() => {
      expect(requestCounts.versions).toBe(1);
      expect(requestCounts.history).toBe(1);
    });

    expect(requestCounts.shadowLatest).toBe(0);
    expect(requestCounts.predictionStats).toBe(0);

    await user.click(screen.getByRole('tab', { name: 'Shadow' }));

    expect(await screen.findByText('Latest Recommendation')).toBeInTheDocument();

    await waitFor(() => {
      expect(requestCounts.shadowLatest).toBe(1);
      expect(requestCounts.shadowHistory).toBe(1);
      expect(requestCounts.divergences).toBe(1);
    });

    expect(screen.getByText('Agreement 91%')).toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: 'Feedback' }));

    expect(await screen.findByText('Use Cases')).toBeInTheDocument();

    await waitFor(() => {
      expect(requestCounts.predictionStats).toBe(1);
    });

    expect(screen.getByText('75%')).toBeInTheDocument();
    expect(screen.getByText('Malware triage')).toBeInTheDocument();
    expect(screen.queryByText('Ignore me')).not.toBeInTheDocument();
  });

  it('starts shadow mode and refreshes the version timeline', async () => {
    const user = userEvent.setup();

    renderWithQuery(<AIModelDetailPage />);

    expect(await screen.findByText('Candidate tuned for higher recall.')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Start Shadow' }));

    await waitFor(() => {
      expect(startShadowPayload).toEqual({ version_id: 'version-2' });
      expect(showSuccessMock).toHaveBeenCalledWith(
        'Shadow mode started.',
        'malware-detector v2 is now in shadow.',
      );
    });

    expect(await screen.findByRole('button', { name: 'Stop Shadow' })).toBeInTheDocument();

    await waitFor(() => {
      expect(requestCounts.startShadow).toBe(1);
      expect(requestCounts.versions).toBeGreaterThanOrEqual(2);
      expect(requestCounts.history).toBeGreaterThanOrEqual(2);
    });
  });

  it('refreshes the prediction log from the page header while on the predictions tab', async () => {
    const user = userEvent.setup();

    renderWithQuery(<AIModelDetailPage />);

    expect(await screen.findByRole('heading', { name: 'Malware Detector' })).toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: 'Predictions' }));

    expect(await screen.findByText('Prediction Log')).toBeInTheDocument();
    expect(await screen.findByText('Malware triage')).toBeInTheDocument();

    await waitFor(() => {
      expect(requestCounts.predictions).toBe(1);
    });

    await user.click(screen.getByRole('button', { name: 'Refresh' }));

    await waitFor(() => {
      expect(requestCounts.model).toBe(2);
      expect(requestCounts.versions).toBe(2);
      expect(requestCounts.history).toBe(2);
      expect(requestCounts.predictions).toBe(2);
    });
  });

  it('renders a not-found state when the model cannot be loaded', async () => {
    modelResponse = null;

    renderWithQuery(<AIModelDetailPage />);

    expect(await screen.findByText('Model not found')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Back to registry' })).toHaveAttribute(
      'href',
      '/admin/ai-governance',
    );
    expect(requestCounts.versions).toBe(0);
    expect(requestCounts.history).toBe(0);
  });
});
