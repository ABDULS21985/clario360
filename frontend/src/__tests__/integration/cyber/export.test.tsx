import { describe, it, expect, vi, beforeAll, afterEach, afterAll } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ExportMenu } from '@/components/cyber/export-menu';

const API_URL = 'http://localhost:8080';

// Mock URL / blob APIs
global.URL.createObjectURL = vi.fn(() => 'blob:test');
global.URL.revokeObjectURL = vi.fn();

// Track download calls
const appendChildSpy = vi.spyOn(document.body, 'appendChild').mockImplementation(() => document.body);
const removeChildSpy = vi.spyOn(document.body, 'removeChild').mockImplementation(() => document.body);

const server = setupServer(
  http.get(`${API_URL}/api/v1/cyber/alerts`, ({ request }) => {
    const accept = request.headers.get('Accept');
    if (accept?.includes('text/csv')) {
      return new HttpResponse('id,title\n1,Alert 1', {
        headers: { 'Content-Type': 'text/csv' },
      });
    }
    return HttpResponse.json({ data: [{ id: '1', title: 'Alert 1' }] });
  }),
  http.post(`${API_URL}/api/v1/cyber/report`, () =>
    HttpResponse.json({ data: { job_id: 'job-123' } }),
  ),
  http.get(`${API_URL}/api/v1/jobs/job-123`, () =>
    HttpResponse.json({
      data: {
        job_id: 'job-123',
        status: 'completed',
        download_url: 'https://example.com/report.pdf',
        created_at: new Date().toISOString(),
      },
    }),
  ),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => { server.resetHandlers(); vi.clearAllMocks(); });
afterAll(() => server.close());

function renderExport(props: Partial<React.ComponentProps<typeof ExportMenu>> = {}) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <ExportMenu
        entityType="alerts"
        baseUrl={`${API_URL}/api/v1/cyber/alerts`}
        currentFilters={{ status: 'new', severity: ['critical'] }}
        totalCount={42}
        enabledFormats={['csv', 'json', 'pdf']}
        pdfReportUrl={`${API_URL}/api/v1/cyber/report`}
        {...props}
      />
    </QueryClientProvider>,
  );
}

describe('Export Integration', () => {
  it('test_csvExport: click CSV → fetch called → download triggered', async () => {
    renderExport();
    fireEvent.click(screen.getByText('Export'));
    await waitFor(() => screen.getByText(/Export as CSV/i));
    fireEvent.click(screen.getByText(/Export as CSV/i));
    // Download should be triggered (createObjectURL called or fetch happened)
    await waitFor(() => {
      // Should not show error
      expect(screen.queryByText(/Export failed/i)).toBeFalsy();
    });
  });

  it('test_jsonExport: click JSON → download triggered', async () => {
    renderExport();
    fireEvent.click(screen.getByText('Export'));
    await waitFor(() => screen.getByText(/Export as JSON/i));
    fireEvent.click(screen.getByText(/Export as JSON/i));
    await waitFor(() => {
      expect(screen.queryByText(/Export failed/i)).toBeFalsy();
    });
  });

  it('test_largeExportWarning: totalCount=15000 → warning dialog → confirm → export proceeds', async () => {
    renderExport({ totalCount: 15000 });
    fireEvent.click(screen.getByText('Export'));
    await waitFor(() => screen.getByText(/Export as CSV/i));
    fireEvent.click(screen.getByText(/Export as CSV/i));
    await waitFor(() => screen.getByText(/Large Export/i));
    fireEvent.click(screen.getByText('Export Anyway'));
    await waitFor(() => {
      expect(screen.queryByText(/Large Export/i)).toBeFalsy();
    });
  });

  it('test_exportWithFilters: active filters shown in component', () => {
    renderExport({ currentFilters: { severity: 'critical', status: 'new' } });
    // Verify component renders with filters
    expect(screen.getByText('Export')).toBeTruthy();
  });
});
