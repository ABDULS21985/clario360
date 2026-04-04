import { describe, it, expect, vi, beforeAll, afterEach, afterAll } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ExportMenu } from '@/components/cyber/export-menu';

const API_URL = 'http://localhost:8080';

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn() }),
}));

// Mock URL.createObjectURL
global.URL.createObjectURL = vi.fn(() => 'blob:test');
global.URL.revokeObjectURL = vi.fn();

const server = setupServer(
  http.get(`${API_URL}/api/v1/cyber/assets`, ({ request }) => {
    const accept = request.headers.get('Accept');
    if (accept?.includes('text/csv')) {
      return new HttpResponse('id,name\n1,Server A', {
        headers: { 'Content-Type': 'text/csv' },
      });
    }
    return HttpResponse.json({ data: [{ id: '1', name: 'Server A' }] });
  }),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => { server.resetHandlers(); vi.clearAllMocks(); });
afterAll(() => server.close());

function renderExportMenu(totalCount = 100) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <ExportMenu
        entityType="assets"
        baseUrl={`${API_URL}/api/v1/cyber/assets`}
        currentFilters={{}}
        totalCount={totalCount}
        enabledFormats={['csv', 'json', 'pdf']}
        pdfReportUrl={`${API_URL}/api/v1/cyber/report`}
      />
    </QueryClientProvider>,
  );
}

describe('ExportMenu', () => {
  it('test_rendersFormats: 3 menu items (CSV, JSON, PDF)', async () => {
    const user = userEvent.setup();
    renderExportMenu();
    await user.click(screen.getByRole('button', { name: /export/i }));
    await waitFor(() => {
      expect(screen.getByText(/Export as CSV/i)).toBeTruthy();
      expect(screen.getByText(/Export as JSON/i)).toBeTruthy();
      expect(screen.getByText(/Generate PDF Report/i)).toBeTruthy();
    });
  });

  it('test_largeExportWarning: totalCount=15000 → warning dialog shown before export', async () => {
    const user = userEvent.setup();
    renderExportMenu(15000);
    await user.click(screen.getByRole('button', { name: /export/i }));
    await waitFor(() => screen.getByText(/Export as CSV/i));
    await user.click(screen.getByText(/Export as CSV/i));
    await waitFor(() => {
      expect(screen.getByText(/Large Export/i)).toBeTruthy();
      expect(screen.getByText(/15,000/)).toBeTruthy();
    });
  });

  it('test_blocksOver50K: totalCount=60000 → export blocked with message', async () => {
    const user = userEvent.setup();
    renderExportMenu(60000);
    await user.click(screen.getByRole('button', { name: /export/i }));
    await waitFor(() => screen.getByText(/Export as CSV/i));
    await user.click(screen.getByText(/Export as CSV/i));
    await waitFor(() => {
      expect(screen.getByText(/Export Limit Exceeded/i)).toBeTruthy();
      expect(screen.getByText(/50,000/)).toBeTruthy();
    });
  });

  it('test_selectedExport: selectedCount=5 → "Export Selected (5 records)" visible', async () => {
    const user = userEvent.setup();
    render(
      <QueryClientProvider client={new QueryClient()}>
        <ExportMenu
          entityType="assets"
          baseUrl={`${API_URL}/api/v1/cyber/assets`}
          currentFilters={{}}
          totalCount={100}
          selectedCount={5}
          enabledFormats={['csv', 'json']}
        />
      </QueryClientProvider>,
    );
    await user.click(screen.getByRole('button', { name: /export/i }));
    await waitFor(() => {
      expect(screen.getByText(/Export Selected \(5 records\)/i)).toBeTruthy();
    });
  });

  it('test_exportWithFilters: renders without error with filters', () => {
    const { container } = render(
      <QueryClientProvider client={new QueryClient()}>
        <ExportMenu
          entityType="alerts"
          baseUrl={`${API_URL}/api/v1/cyber/alerts`}
          currentFilters={{ severity: ['critical', 'high'], status: 'new' }}
          totalCount={42}
          enabledFormats={['csv', 'json']}
        />
      </QueryClientProvider>,
    );
    expect(container.querySelector('button')).toBeTruthy();
  });
});
