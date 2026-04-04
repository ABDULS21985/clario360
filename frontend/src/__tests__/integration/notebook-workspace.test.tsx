import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { TooltipProvider } from '@/components/ui/tooltip';
import NotebookWorkspacePage from '@/app/(dashboard)/notebooks/page';
import type { NotebookProfile, NotebookServer, NotebookTemplate } from '@/lib/notebooks';

const API_URL = 'http://localhost:8080';

// ---------------------------------------------------------------------------
// Global mocks required by the page
// ---------------------------------------------------------------------------

vi.mock('@/hooks/use-auth', () => ({
  useAuth: () => ({
    user: { id: 'u1', first_name: 'Analyst', permissions: ['*:read'] },
    isHydrated: true,
    hasPermission: () => true,
  }),
}));

vi.mock('@/hooks/use-websocket', () => ({
  useWebSocket: () => ({ isConnected: false }),
}));

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), back: vi.fn(), prefetch: vi.fn() }),
  usePathname: () => '/notebooks',
  useSearchParams: () => ({ get: () => null, forEach: () => {} }),
  redirect: vi.fn(),
}));

// ---------------------------------------------------------------------------
// Mock data — exact backend contract
// ---------------------------------------------------------------------------

const mockProfiles: NotebookProfile[] = [
  {
    slug: 'soc-analyst',
    display_name: 'SOC Analyst',
    description: 'Security analysis and threat hunting.',
    cpu: '2 CPU',
    memory: '4 GB',
    storage: '5 GiB',
    spark_enabled: false,
    default: true,
  },
  {
    slug: 'data-scientist',
    display_name: 'Data Scientist',
    description: 'Model development and evaluation.',
    cpu: '4 CPU',
    memory: '8 GB',
    storage: '20 GiB',
    spark_enabled: false,
    default: false,
  },
];

const mockTemplates: NotebookTemplate[] = [
  {
    id: '01_threat_detection_quickstart',
    title: 'Threat Detection Quickstart',
    description: 'Pull recent alerts, visualize trends.',
    difficulty: 'beginner',
    tags: ['security', 'alerts'],
    filename: '01_threat_detection_quickstart.ipynb',
  },
  {
    id: '05_model_validation_framework',
    title: 'Model Validation Framework',
    description: 'Evaluate model precision and recall.',
    difficulty: 'advanced',
    tags: ['ai', 'governance'],
    filename: '05_model_validation_framework.ipynb',
  },
];

const mockRunningServer: NotebookServer = {
  id: 'default',
  profile: 'soc-analyst',
  status: 'running',
  url: 'https://notebooks.example.com/user/analyst/lab',
  started_at: new Date(Date.now() - 3_600_000).toISOString(),
  last_activity: new Date(Date.now() - 300_000).toISOString(),
  cpu_percent: 25.0,
  memory_mb: 1024,
  memory_limit_mb: 4096,
};

const mockStartingServer: NotebookServer = {
  ...mockRunningServer,
  status: 'starting',
  started_at: undefined,
  last_activity: undefined,
  cpu_percent: 0,
  memory_mb: 0,
};

// ---------------------------------------------------------------------------
// MSW server
// ---------------------------------------------------------------------------

const mswServer = setupServer(
  http.get(`${API_URL}/api/v1/notebooks/profiles`, () => HttpResponse.json(mockProfiles)),
  http.get(`${API_URL}/api/v1/notebooks/templates`, () => HttpResponse.json(mockTemplates)),
  http.get(`${API_URL}/api/v1/notebooks/servers`, () => HttpResponse.json([])),
);

beforeAll(() => mswServer.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => mswServer.resetHandlers());
afterAll(() => mswServer.close());

// ---------------------------------------------------------------------------
// Render helper
// ---------------------------------------------------------------------------

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <TooltipProvider>
        <NotebookWorkspacePage />
      </TooltipProvider>
    </QueryClientProvider>,
  );
}

// ---------------------------------------------------------------------------
// Initial render
// ---------------------------------------------------------------------------

describe('NotebookWorkspacePage — initial render', () => {
  it('renders page title', async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getByText('Notebook Workspace')).toBeInTheDocument();
    });
  });

  it('renders Active Servers heading', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText('Active Servers')).toBeInTheDocument());
  });

  it('renders empty server state', async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByText(/no active notebook server/i)).toBeInTheDocument(),
    );
  });

  it('renders template cards from API', async () => {
    renderPage();
    await waitFor(() => {
      expect(screen.getByText('Threat Detection Quickstart')).toBeInTheDocument();
      expect(screen.getByText('Model Validation Framework')).toBeInTheDocument();
    });
  });

  it('renders Launch Notebook button', async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /launch notebook/i })).toBeInTheDocument(),
    );
  });

  it('Launch Notebook button is enabled when no server', async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /launch notebook/i })).not.toBeDisabled(),
    );
  });
});

// ---------------------------------------------------------------------------
// Launch server flow
// ---------------------------------------------------------------------------

describe('NotebookWorkspacePage — launch server flow', () => {
  it('opens ProfileSelector dialog on Launch Notebook click', async () => {
    const user = userEvent.setup();
    renderPage();

    await waitFor(() =>
      expect(screen.getByRole('button', { name: /launch notebook/i })).toBeInTheDocument(),
    );
    await user.click(screen.getByRole('button', { name: /launch notebook/i }));

    await waitFor(() =>
      expect(screen.getByText('Launch Notebook Workspace')).toBeInTheDocument(),
    );
  });

  it('shows profile cards in selector', async () => {
    const user = userEvent.setup();
    renderPage();

    await waitFor(() =>
      expect(screen.getByRole('button', { name: /launch notebook/i })).toBeInTheDocument(),
    );
    await user.click(screen.getByRole('button', { name: /launch notebook/i }));

    await waitFor(() => {
      expect(screen.getByText('SOC Analyst')).toBeInTheDocument();
      expect(screen.getByText('Data Scientist')).toBeInTheDocument();
    });
  });

  it('POSTs correct profile slug on launch and closes dialog', async () => {
    const user = userEvent.setup();
    let capturedBody: Record<string, unknown> | null = null;

    mswServer.use(
      http.post(`${API_URL}/api/v1/notebooks/servers`, async ({ request }) => {
        capturedBody = (await request.json()) as Record<string, unknown>;
        return HttpResponse.json(mockStartingServer, { status: 201 });
      }),
    );

    renderPage();

    await waitFor(() =>
      expect(screen.getByRole('button', { name: /launch notebook/i })).toBeInTheDocument(),
    );
    await user.click(screen.getByRole('button', { name: /launch notebook/i }));

    await waitFor(() =>
      expect(screen.getByText('Launch Notebook Workspace')).toBeInTheDocument(),
    );
    await user.click(screen.getByRole('button', { name: /launch soc analyst/i }));

    await waitFor(() => {
      expect(capturedBody).toEqual({ profile: 'soc-analyst' });
    });
    // Dialog closes after success
    await waitFor(() =>
      expect(screen.queryByText('Launch Notebook Workspace')).not.toBeInTheDocument(),
    );
  });
});

// ---------------------------------------------------------------------------
// Running server — stop flow
// ---------------------------------------------------------------------------

describe('NotebookWorkspacePage — stop server', () => {
  it('disables Launch Notebook when a server is running', async () => {
    mswServer.use(
      http.get(`${API_URL}/api/v1/notebooks/servers`, () =>
        HttpResponse.json([mockRunningServer]),
      ),
    );
    renderPage();
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /launch notebook/i })).toBeDisabled(),
    );
  });

  it('shows Open JupyterLab link with server URL', async () => {
    mswServer.use(
      http.get(`${API_URL}/api/v1/notebooks/servers`, () =>
        HttpResponse.json([mockRunningServer]),
      ),
    );
    renderPage();
    await waitFor(() => {
      const link = screen.getByRole('link', { name: /open jupyterlab/i });
      expect(link).toHaveAttribute('href', mockRunningServer.url);
    });
  });

  it('DELETEs correct server ID on Stop click', async () => {
    const user = userEvent.setup();
    let deletedId = '';

    mswServer.use(
      http.get(`${API_URL}/api/v1/notebooks/servers`, () =>
        HttpResponse.json([mockRunningServer]),
      ),
      http.delete(`${API_URL}/api/v1/notebooks/servers/:id`, ({ params }) => {
        deletedId = params.id as string;
        return HttpResponse.json({ message: 'notebook server stopped' });
      }),
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /stop server/i })).toBeInTheDocument(),
    );
    await user.click(screen.getByRole('button', { name: /stop server/i }));

    await waitFor(() => {
      expect(deletedId).toBe('default');
    });
  });
});

// ---------------------------------------------------------------------------
// Template copy flow
// ---------------------------------------------------------------------------

describe('NotebookWorkspacePage — copy template', () => {
  it('enables Open Template button when server is running', async () => {
    mswServer.use(
      http.get(`${API_URL}/api/v1/notebooks/servers`, () =>
        HttpResponse.json([mockRunningServer]),
      ),
    );
    renderPage();
    await waitFor(() => {
      const buttons = screen.getAllByRole('button', { name: /open template/i });
      expect(buttons.length).toBeGreaterThan(0);
      buttons.forEach((btn) => expect(btn).not.toBeDisabled());
    });
  });

  it('POSTs template_id and opens result open_url', async () => {
    const user = userEvent.setup();
    const openMock = vi.fn();
    vi.stubGlobal('open', openMock);

    const copiedResult = {
      template_id: '01_threat_detection_quickstart',
      path: '01_threat_detection_quickstart.ipynb',
      open_url:
        'https://notebooks.example.com/user/analyst/lab/tree/01_threat_detection_quickstart.ipynb',
    };
    let capturedBody: Record<string, unknown> | null = null;

    mswServer.use(
      http.get(`${API_URL}/api/v1/notebooks/servers`, () =>
        HttpResponse.json([mockRunningServer]),
      ),
      http.post(
        `${API_URL}/api/v1/notebooks/servers/:id/copy-template`,
        async ({ request }) => {
          capturedBody = (await request.json()) as Record<string, unknown>;
          return HttpResponse.json(copiedResult);
        },
      ),
    );

    renderPage();

    await waitFor(() => {
      const btns = screen.getAllByRole('button', { name: /open template/i });
      expect(btns[0]).not.toBeDisabled();
    });

    await user.click(screen.getAllByRole('button', { name: /open template/i })[0]);

    await waitFor(() => {
      // Request must send template_id (not id)
      expect(capturedBody).toEqual({ template_id: '01_threat_detection_quickstart' });
    });
    await waitFor(() => {
      expect(openMock).toHaveBeenCalledWith(
        copiedResult.open_url,
        '_blank',
        'noopener,noreferrer',
      );
    });

    vi.unstubAllGlobals();
  });
});

// ---------------------------------------------------------------------------
// Error handling
// ---------------------------------------------------------------------------

describe('NotebookWorkspacePage — error states', () => {
  it('renders gracefully when server list API fails', async () => {
    mswServer.use(
      http.get(`${API_URL}/api/v1/notebooks/servers`, () =>
        HttpResponse.json({ code: 'BAD_GATEWAY', message: 'JupyterHub unavailable' }, { status: 502 }),
      ),
    );
    renderPage();
    await waitFor(() => {
      expect(screen.getByText('Active Servers')).toBeInTheDocument();
    });
  });

  it('renders gracefully when templates API fails', async () => {
    mswServer.use(
      http.get(`${API_URL}/api/v1/notebooks/templates`, () =>
        HttpResponse.json({ code: 'INTERNAL_ERROR', message: 'fail' }, { status: 500 }),
      ),
    );
    renderPage();
    await waitFor(() => {
      expect(screen.getByText('Notebook Workspace')).toBeInTheDocument();
    });
  });
});
