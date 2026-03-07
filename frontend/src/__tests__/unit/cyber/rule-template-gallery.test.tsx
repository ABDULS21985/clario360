import { describe, it, expect, vi, beforeAll, afterEach, afterAll } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { RuleTemplateGallery } from '@/app/(dashboard)/cyber/rules/_components/rule-template-gallery';
import type { RuleTemplate } from '@/types/cyber';

const API_URL = 'http://localhost:8080';

function makeTemplates(count = 15): RuleTemplate[] {
  return Array.from({ length: count }, (_, i) => ({
    id: `tmpl-${i}`,
    name: `Template ${i + 1}`,
    description: `Description for template ${i + 1}`,
    type: i % 2 === 0 ? 'sigma' : 'threshold',
    severity: i < 5 ? 'critical' : i < 10 ? 'high' : 'medium',
    mitre_technique_ids: [`T10${String(i).padStart(2, '0')}`],
    category: 'threat',
  }));
}

const server = setupServer(
  http.get(`${API_URL}/api/v1/cyber/rules/templates`, () =>
    HttpResponse.json({ data: makeTemplates(15) }),
  ),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

describe('RuleTemplateGallery', () => {
  it('test_renders15Templates: 15 template cards displayed', async () => {
    const onActivate = vi.fn();
    render(
      <QueryClientProvider client={makeQueryClient()}>
        <RuleTemplateGallery
          open={true}
          onOpenChange={vi.fn()}
          activatedTemplateIds={[]}
          onActivate={onActivate}
        />
      </QueryClientProvider>,
    );
    await waitFor(() => {
      expect(screen.getByText('Template 1')).toBeTruthy();
      expect(screen.getByText('Template 15')).toBeTruthy();
    });
  });

  it('test_activateCreatesRule: click Activate → onActivate called', async () => {
    const onActivate = vi.fn();
    render(
      <QueryClientProvider client={makeQueryClient()}>
        <RuleTemplateGallery
          open={true}
          onOpenChange={vi.fn()}
          activatedTemplateIds={[]}
          onActivate={onActivate}
        />
      </QueryClientProvider>,
    );
    await waitFor(() => screen.getByText('Template 1'));
    const activateBtns = screen.getAllByText('Activate');
    fireEvent.click(activateBtns[0]);
    expect(onActivate).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'Template 1' }),
    );
  });

  it('test_alreadyActivated: activated template shows "Active ✓"', async () => {
    render(
      <QueryClientProvider client={makeQueryClient()}>
        <RuleTemplateGallery
          open={true}
          onOpenChange={vi.fn()}
          activatedTemplateIds={['tmpl-0']}
          onActivate={vi.fn()}
        />
      </QueryClientProvider>,
    );
    await waitFor(() => {
      expect(screen.getByText(/Active ✓/)).toBeTruthy();
    });
  });
});
