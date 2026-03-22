import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const API_URL = 'http://localhost:8080';

vi.mock('@/hooks/use-auth', () => ({
  useAuth: () => ({
    user: { id: 'u1', first_name: 'Admin', permissions: ['cyber:read', 'cyber:write'] },
    isHydrated: true,
    hasPermission: () => true,
  }),
}));

vi.mock('@/hooks/use-websocket', () => ({
  useWebSocket: () => ({ isConnected: false }),
}));

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), back: vi.fn(), prefetch: vi.fn() }),
  usePathname: () => '/cyber/assets',
  useSearchParams: () => ({ get: () => null, forEach: () => {} }),
  redirect: vi.fn(),
}));

type AssetRecord = {
  id: string;
  tenant_id: string;
  name: string;
  type: string;
  criticality: string;
  status: string;
  tags: string[];
  vulnerability_count: number;
  critical_vuln_count: number;
  high_vuln_count: number;
  alert_count: number;
  ip_address?: string;
  hostname?: string;
  os?: string;
  owner?: string;
  department?: string;
  location?: string;
  created_at: string;
  updated_at: string;
};

let assetStore: AssetRecord[] = [];
let createPayload: Record<string, unknown> | null = null;
let updateRequests: Array<{ id: string; body: Record<string, unknown> }> = [];
let bulkPayload: Record<string, unknown> | null = null;
let scanPayload: Record<string, unknown> | null = null;

const server = setupServer(
  http.get(`${API_URL}/api/v1/cyber/assets`, () =>
    HttpResponse.json({
      data: assetStore,
      meta: { page: 1, per_page: 25, total: assetStore.length, total_pages: 1 },
    }),
  ),
  http.get(`${API_URL}/api/v1/cyber/assets/stats`, () =>
    HttpResponse.json({ data: buildStats(assetStore) }),
  ),
  http.post(`${API_URL}/api/v1/cyber/assets`, async ({ request }) => {
    createPayload = (await request.json()) as Record<string, unknown>;
    const created = buildAsset(`asset-${assetStore.length + 1}`, {
      name: String(createPayload.name),
      type: String(createPayload.type ?? 'server'),
      criticality: String(createPayload.criticality ?? 'medium'),
      tags: parseTags(createPayload.tags),
      ip_address: optionalString(createPayload.ip_address),
      hostname: optionalString(createPayload.hostname),
      os: optionalString(createPayload.os),
      owner: optionalString(createPayload.owner),
      department: optionalString(createPayload.department),
    });
    assetStore = [created, ...assetStore];
    return HttpResponse.json({ data: created }, { status: 201 });
  }),
  http.put(`${API_URL}/api/v1/cyber/assets/:id`, async ({ params, request }) => {
    const body = (await request.json()) as Record<string, unknown>;
    const id = String(params.id);
    updateRequests.push({ id, body });
    assetStore = assetStore.map((asset) => (
      asset.id === id
        ? {
            ...asset,
            ...applyAssetUpdate(body),
            updated_at: '2024-01-03T00:00:00Z',
          }
        : asset
    ));
    const updated = assetStore.find((asset) => asset.id === id);
    if (!updated) {
      return HttpResponse.json({ error: 'not found' }, { status: 404 });
    }
    return HttpResponse.json({ data: updated });
  }),
  http.delete(`${API_URL}/api/v1/cyber/assets/:id`, ({ params }) => {
    const id = String(params.id);
    assetStore = assetStore.filter((asset) => asset.id !== id);
    return new HttpResponse(null, { status: 204 });
  }),
  http.post(`${API_URL}/api/v1/cyber/assets/bulk`, async ({ request }) => {
    bulkPayload = (await request.json()) as Record<string, unknown>;
    const assets = Array.isArray(bulkPayload.assets) ? (bulkPayload.assets as Array<Record<string, unknown>>) : [];
    const created = assets.map((asset, index) =>
      buildAsset(`bulk-${index + 1}`, {
        name: String(asset.name),
        type: String(asset.type ?? 'server'),
        criticality: String(asset.criticality ?? 'medium'),
        tags: parseTags(asset.tags),
        ip_address: optionalString(asset.ip_address),
        owner: optionalString(asset.owner),
      }),
    );
    assetStore = [...created, ...assetStore];
    return HttpResponse.json({
      data: {
        created: created.length,
        updated: 0,
        failed: 0,
      },
    }, { status: 201 });
  }),
  http.post(`${API_URL}/api/v1/cyber/assets/scan`, async ({ request }) => {
    scanPayload = (await request.json()) as Record<string, unknown>;
    return HttpResponse.json({
      data: {
        scan_id: 'scan-1',
        status: 'pending',
        message: 'scan started',
      },
    }, { status: 202 });
  }),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
beforeEach(() => {
  assetStore = [
    buildAsset('asset-1', {
      name: 'web-prod-01',
      type: 'server',
      criticality: 'high',
      tags: ['production'],
      vulnerability_count: 3,
      critical_vuln_count: 1,
      high_vuln_count: 2,
      alert_count: 1,
      owner: 'Infra Team',
    }),
    buildAsset('asset-2', {
      name: 'db-primary',
      type: 'database',
      criticality: 'critical',
      tags: [],
      vulnerability_count: 0,
      critical_vuln_count: 0,
      high_vuln_count: 0,
      alert_count: 0,
      owner: 'DBA Team',
    }),
  ];
  createPayload = null;
  updateRequests = [];
  bulkPayload = null;
  scanPayload = null;
});
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

function makeQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
}

async function renderPage() {
  const { default: Page } = await import('@/app/(dashboard)/cyber/assets/page');
  return render(
    <QueryClientProvider client={makeQueryClient()}>
      <Page />
    </QueryClientProvider>,
  );
}

describe('Asset Inventory Page', () => {
  it('renders page header and initial list', async () => {
    await renderPage();

    await waitFor(() => {
      expect(screen.getByText('Asset Inventory')).toBeInTheDocument();
      expect(screen.getByText('web-prod-01')).toBeInTheDocument();
      expect(screen.getByText('db-primary')).toBeInTheDocument();
    });
  });

  it('creates an asset from the page dialog and refreshes the table', async () => {
    const user = userEvent.setup();
    await renderPage();
    await screen.findByText('web-prod-01');

    await user.click(screen.getByRole('button', { name: /add asset/i }));

    const dialog = await screen.findByRole('dialog');
    await user.type(within(dialog).getByLabelText(/^name$/i), 'cache-prod-01');
    await user.click(within(dialog).getByRole('button', { name: /create asset/i }));

    await waitFor(() => {
      expect(createPayload).toMatchObject({
        name: 'cache-prod-01',
        type: 'server',
        criticality: 'medium',
      });
    });

    expect(await screen.findByText('cache-prod-01')).toBeInTheDocument();
  });

  it('edits an asset from row actions and shows the persisted result', async () => {
    const user = userEvent.setup();
    await renderPage();
    await screen.findByText('web-prod-01');

    await openAssetActions(user, 'web-prod-01');
    await user.click(await screen.findByRole('menuitem', { name: /edit/i }));

    const dialog = await screen.findByRole('dialog');
    const nameInput = within(dialog).getByLabelText(/^name$/i);
    await user.clear(nameInput);
    await user.type(nameInput, 'web-prod-01-renamed');
    await user.click(within(dialog).getByRole('button', { name: /save changes/i }));

    await waitFor(() => {
      expect(updateRequests.at(-1)).toMatchObject({
        id: 'asset-1',
        body: expect.objectContaining({
          name: 'web-prod-01-renamed',
          status: 'active',
        }),
      });
    });

    expect(await screen.findByText('web-prod-01-renamed')).toBeInTheDocument();
  });

  it('manages tags from row actions and refreshes the visible tags', async () => {
    const user = userEvent.setup();
    await renderPage();
    await screen.findByText('web-prod-01');

    await openAssetActions(user, 'web-prod-01');
    await user.click(await screen.findByRole('menuitem', { name: /manage tags/i }));

    const dialog = await screen.findByRole('dialog');
    const tagInput = within(dialog).getByPlaceholderText(/add tag/i);
    await user.type(tagInput, 'PCI{enter}');
    await user.click(within(dialog).getByRole('button', { name: /save tags/i }));

    await waitFor(() => {
      expect(updateRequests.at(-1)).toMatchObject({
        id: 'asset-1',
        body: {
          tags: ['production', 'pci'],
        },
      });
    });

    const row = findAssetRow('web-prod-01');
    expect(within(row).getByText('pci')).toBeInTheDocument();
  });

  it('deletes an asset after confirmation and removes it from the table', async () => {
    const user = userEvent.setup();
    await renderPage();
    await screen.findByText('db-primary');

    await openAssetActions(user, 'db-primary');
    await user.click(await screen.findByRole('menuitem', { name: /delete/i }));

    const dialog = await screen.findByRole('dialog');
    await user.type(within(dialog).getByLabelText(/type delete to confirm/i), 'DELETE');
    await user.click(within(dialog).getByRole('button', { name: /^delete asset$/i }));

    await waitFor(() => {
      expect(screen.queryByText('db-primary')).not.toBeInTheDocument();
    });
  });

  it('starts a scan from the toolbar with the backend request shape the handler expects', async () => {
    const user = userEvent.setup();
    await renderPage();
    await screen.findByText('web-prod-01');

    await user.click(screen.getByRole('button', { name: /^scan$/i }));

    const dialog = await screen.findByRole('dialog');
    await user.type(within(dialog).getByLabelText(/targets/i), '10.0.0.5');
    await user.type(within(dialog).getByLabelText(/ports/i), '80,443');
    await user.click(within(dialog).getByRole('button', { name: /start scan/i }));

    await waitFor(() => {
      expect(scanPayload).toEqual({
        scan_type: 'network',
        targets: ['10.0.0.5'],
        ports: [80, 443],
        options: {
          vuln_scan: true,
          config_audit: true,
        },
      });
    });
  });

  it('bulk imports assets from the toolbar and refreshes the list', async () => {
    const user = userEvent.setup();
    const payload = [
      {
        name: 'vpn-gateway-01',
        type: 'network_device',
        criticality: 'high',
        owner: 'Security Ops',
      },
    ];

    await renderPage();
    await screen.findByText('web-prod-01');

    await user.click(screen.getByRole('button', { name: /^import$/i }));

    const dialog = await screen.findByRole('dialog');
    const textarea = within(dialog).getByLabelText(/json input/i);
    await user.click(textarea);
    await user.paste(JSON.stringify(payload));
    await user.click(within(dialog).getByRole('button', { name: /validate & preview/i }));
    await screen.findByText(/preview \(1 assets\)/i);
    await user.click(within(dialog).getByRole('button', { name: /import 1 assets/i }));

    await waitFor(() => {
      expect(bulkPayload).toEqual({ assets: payload });
    });

    expect(await screen.findByText('vpn-gateway-01')).toBeInTheDocument();
  });

  it('shows an error state on list failure', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/cyber/assets`, () =>
        HttpResponse.json({ error: 'server error' }, { status: 500 }),
      ),
    );

    await renderPage();

    await waitFor(() => {
      expect(screen.getByText(/failed to load/i)).toBeInTheDocument();
    });
  });
});

function buildAsset(id: string, overrides: Partial<AssetRecord> = {}): AssetRecord {
  return {
    id,
    tenant_id: 't1',
    name: 'asset',
    type: 'server',
    criticality: 'medium',
    status: 'active',
    tags: [],
    vulnerability_count: 0,
    critical_vuln_count: 0,
    high_vuln_count: 0,
    alert_count: 0,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    ...overrides,
  };
}

function buildStats(assets: AssetRecord[]) {
  return {
    total: assets.length,
    by_type: assets.reduce<Record<string, number>>((acc, asset) => {
      acc[asset.type] = (acc[asset.type] ?? 0) + 1;
      return acc;
    }, {}),
    by_criticality: assets.reduce<Record<string, number>>((acc, asset) => {
      acc[asset.criticality] = (acc[asset.criticality] ?? 0) + 1;
      return acc;
    }, {}),
    by_status: assets.reduce<Record<string, number>>((acc, asset) => {
      acc[asset.status] = (acc[asset.status] ?? 0) + 1;
      return acc;
    }, {}),
    assets_with_vulns: assets.filter((asset) => asset.vulnerability_count > 0).length,
    assets_discovered_this_week: assets.length,
  };
}

function parseTags(value: unknown): string[] {
  return Array.isArray(value) ? value.map((tag) => String(tag)) : [];
}

function optionalString(value: unknown): string | undefined {
  return typeof value === 'string' && value.length > 0 ? value : undefined;
}

function applyAssetUpdate(body: Record<string, unknown>): Partial<AssetRecord> {
  const update: Partial<AssetRecord> = {};

  if ('name' in body) {
    update.name = String(body.name);
  }
  if ('type' in body) {
    update.type = String(body.type);
  }
  if ('criticality' in body) {
    update.criticality = String(body.criticality);
  }
  if ('status' in body) {
    update.status = String(body.status);
  }
  if ('tags' in body) {
    update.tags = parseTags(body.tags);
  }

  const optionalFields = ['ip_address', 'hostname', 'os', 'owner', 'department', 'location'] as const;
  for (const field of optionalFields) {
    if (field in body) {
      update[field] = optionalString(body[field]);
    }
  }

  return update;
}

function findAssetRow(name: string): HTMLElement {
  const row = screen.getByText(name).closest('tr');
  if (!row) {
    throw new Error(`Could not find table row for ${name}`);
  }
  return row;
}

async function openAssetActions(user: ReturnType<typeof userEvent.setup>, name: string) {
  const row = findAssetRow(name);
  await user.click(within(row).getByRole('button', { name: /actions/i }));
}
