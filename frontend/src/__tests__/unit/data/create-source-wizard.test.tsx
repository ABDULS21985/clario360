import { describe, expect, it, beforeEach, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CreateSourceWizard } from '@/app/(dashboard)/data/sources/_components/create-source-wizard';
import { sourceSchema } from '@/__tests__/data-suite-fixtures';
import { dataSuiteApi } from '@/lib/data-suite';

vi.mock('@/hooks/use-auth', () => ({
  useAuth: () => ({
    hasPermission: () => true,
    isHydrated: true,
  }),
}));

describe('CreateSourceWizard', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(window, 'confirm').mockReturnValue(true);
  });

  async function selectPostgresAndContinue(user: ReturnType<typeof userEvent.setup>) {
    const postgresCard = screen.getByText('PostgreSQL').closest('[class*="border"]');
    expect(postgresCard).not.toBeNull();
    await user.click(within(postgresCard as HTMLElement).getByRole('button', { name: /select/i }));
  }

  async function fillPostgresForm(user: ReturnType<typeof userEvent.setup>) {
    await user.type(screen.getByLabelText(/Host/i), 'db-prod');
    await user.clear(screen.getByLabelText(/Port/i));
    await user.type(screen.getByLabelText(/Port/i), '5432');
    await user.type(screen.getByLabelText(/Database/i), 'customers');
    await user.type(screen.getByLabelText(/Username/i), 'readonly');
    await user.type(screen.getByLabelText(/Password/i), 'secret');
  }

  it('test_step1_typeSelection: choosing PostgreSQL advances to the connection form', async () => {
    const user = userEvent.setup();
    render(<CreateSourceWizard open onOpenChange={vi.fn()} />);

    await selectPostgresAndContinue(user);

    expect(screen.getByLabelText(/Host/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Database/i)).toBeInTheDocument();
  });

  it('test_step2_validation: blocks continue until required fields are filled', async () => {
    const user = userEvent.setup();
    render(<CreateSourceWizard open onOpenChange={vi.fn()} />);

    await selectPostgresAndContinue(user);

    expect(screen.getByRole('button', { name: /continue/i })).toBeDisabled();

    await fillPostgresForm(user);
    expect(screen.getByRole('button', { name: /continue/i })).toBeEnabled();
  });

  it('test_step2_backPreservesState: preserves connection fields after navigating back', async () => {
    const user = userEvent.setup();
    render(<CreateSourceWizard open onOpenChange={vi.fn()} />);

    await selectPostgresAndContinue(user);
    await fillPostgresForm(user);
    await user.click(screen.getByRole('button', { name: /continue/i }));
    await user.click(screen.getByRole('button', { name: /^Type$/i }));
    await selectPostgresAndContinue(user);

    expect(screen.getByLabelText(/Host/i)).toHaveValue('db-prod');
    expect(screen.getByLabelText(/Database/i)).toHaveValue('customers');
  });

  it('test_step3_autoTest: entering step 3 triggers connection test automatically', async () => {
    vi.spyOn(dataSuiteApi, 'testSourceConfig').mockResolvedValue({
      success: true,
      latency_ms: 12,
      version: 'PostgreSQL 16.2',
      message: 'ok',
    });

    const user = userEvent.setup();
    render(<CreateSourceWizard open onOpenChange={vi.fn()} />);

    await selectPostgresAndContinue(user);
    await fillPostgresForm(user);
    await user.click(screen.getByRole('button', { name: /continue/i }));

    await waitFor(() => {
      expect(dataSuiteApi.testSourceConfig).toHaveBeenCalledTimes(1);
    });
    expect(screen.getByText(/Connected successfully/i)).toBeInTheDocument();
  });

  it('test_step3_skipTest: can skip the connection test and continue', async () => {
    vi.spyOn(dataSuiteApi, 'testSourceConfig').mockRejectedValue(new Error('timeout'));
    vi.spyOn(dataSuiteApi, 'createSource').mockResolvedValue({
      id: 'new-source-id',
      ...sourceSchema,
    } as never);
    vi.spyOn(dataSuiteApi, 'discoverSource').mockResolvedValue(sourceSchema);

    const user = userEvent.setup();
    render(<CreateSourceWizard open onOpenChange={vi.fn()} />);

    await selectPostgresAndContinue(user);
    await fillPostgresForm(user);
    await user.click(screen.getByRole('button', { name: /continue/i }));

    await waitFor(() => expect(screen.getByText(/Connection failed/i)).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /skip test/i }));

    expect(screen.getByText(/Discovering schema/i)).toBeInTheDocument();
  });

  it('test_step4_schemaTree: renders discovered schema in step 4', async () => {
    vi.spyOn(dataSuiteApi, 'testSourceConfig').mockResolvedValue({
      success: true,
      latency_ms: 12,
      version: 'PostgreSQL 16.2',
      message: 'ok',
    });
    vi.spyOn(dataSuiteApi, 'createSource').mockResolvedValue({
      ...dataSuiteApi,
      id: '11111111-1111-1111-1111-111111111111',
      name: 'postgresql_db-prod_customers',
      description: '',
      type: 'postgresql',
      status: 'pending_test',
      tags: [],
      metadata: {},
      tenant_id: 'tenant-1',
      created_by: 'user-1',
      created_at: '2026-03-07T00:00:00Z',
      updated_at: '2026-03-07T00:00:00Z',
    } as never);
    vi.spyOn(dataSuiteApi, 'discoverSource').mockResolvedValue(sourceSchema);

    const user = userEvent.setup();
    render(<CreateSourceWizard open onOpenChange={vi.fn()} />);

    await selectPostgresAndContinue(user);
    await fillPostgresForm(user);
    await user.click(screen.getByRole('button', { name: /continue/i }));

    await waitFor(() => expect(screen.getByText(/Connected successfully/i)).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /^Continue$/i }));

    await waitFor(() => {
      expect(screen.getByText(/Tables discovered/i)).toBeInTheDocument();
      expect(screen.getByText('public.customers')).toBeInTheDocument();
      expect(screen.getByText('email')).toBeInTheDocument();
    });
  });

  it('test_step5_submit: completes the wizard and calls the create/update APIs', async () => {
    vi.spyOn(dataSuiteApi, 'testSourceConfig').mockResolvedValue({
      success: true,
      latency_ms: 12,
      version: 'PostgreSQL 16.2',
      message: 'ok',
    });
    vi.spyOn(dataSuiteApi, 'createSource').mockResolvedValue({
      id: '11111111-1111-1111-1111-111111111111',
      tenant_id: 'tenant-1',
      name: 'postgresql_db-prod_customers',
      description: '',
      type: 'postgresql',
      status: 'pending_test',
      tags: [],
      metadata: {},
      created_by: 'user-1',
      created_at: '2026-03-07T00:00:00Z',
      updated_at: '2026-03-07T00:00:00Z',
    } as never);
    vi.spyOn(dataSuiteApi, 'discoverSource').mockResolvedValue(sourceSchema);
    vi.spyOn(dataSuiteApi, 'updateSource').mockResolvedValue({
      id: '11111111-1111-1111-1111-111111111111',
      tenant_id: 'tenant-1',
      name: 'customer_db_prod',
      description: 'Primary customer source',
      type: 'postgresql',
      status: 'active',
      tags: ['critical'],
      metadata: {},
      created_by: 'user-1',
      created_at: '2026-03-07T00:00:00Z',
      updated_at: '2026-03-07T00:05:00Z',
    } as never);

    const onCreated = vi.fn();
    const user = userEvent.setup();
    render(<CreateSourceWizard open onOpenChange={vi.fn()} onCreated={onCreated} />);

    await selectPostgresAndContinue(user);
    await fillPostgresForm(user);
    await user.click(screen.getByRole('button', { name: /continue/i }));
    await waitFor(() => expect(screen.getByText(/Connected successfully/i)).toBeInTheDocument());
    await user.click(screen.getByRole('button', { name: /^Continue$/i }));
    await waitFor(() => expect(screen.getByText(/I've reviewed the schema/i)).toBeInTheDocument());
    await user.click(screen.getByRole('checkbox', { name: /I've reviewed the schema/i }));
    await user.click(screen.getByRole('button', { name: /^Continue$/i }));

    await user.clear(screen.getByLabelText(/Source name/i));
    await user.type(screen.getByLabelText(/Source name/i), 'customer_db_prod');
    await user.type(screen.getByLabelText(/Description/i), 'Primary customer source');
    await user.click(screen.getByRole('button', { name: /Create Source/i }));

    await waitFor(() => {
      expect(dataSuiteApi.updateSource).toHaveBeenCalledTimes(1);
      expect(onCreated).toHaveBeenCalledTimes(1);
    });
  });
});

