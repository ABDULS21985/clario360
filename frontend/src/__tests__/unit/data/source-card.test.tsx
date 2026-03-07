import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SourceCard } from '@/app/(dashboard)/data/sources/_components/source-card';
import { dataSources } from '@/__tests__/data-suite-fixtures';

describe('SourceCard', () => {
  it('test_activeSource: shows active status and source stats', () => {
    render(
      <SourceCard
        source={dataSources[0]}
        testing={false}
        onTest={vi.fn()}
        onSync={vi.fn()}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
      />,
    );

    expect(screen.getByText('Customer Database')).toBeInTheDocument();
    expect(screen.getByText('active')).toBeInTheDocument();
    expect(screen.getByText(/15 tables/i)).toBeInTheDocument();
    expect(screen.getByTestId('source-status-dot-active')).toBeInTheDocument();
  });

  it('test_errorSource: shows error status and message', () => {
    render(
      <SourceCard
        source={dataSources[1]}
        testing={false}
        onTest={vi.fn()}
        onSync={vi.fn()}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
      />,
    );

    expect(screen.getByText('error')).toBeInTheDocument();
    expect(screen.getByText(/Connection refused/i)).toBeInTheDocument();
  });

  it('test_syncingSource: shows syncing status', () => {
    render(
      <SourceCard
        source={dataSources[2]}
        testing={false}
        onTest={vi.fn()}
        onSync={vi.fn()}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
      />,
    );

    expect(screen.getByText('syncing')).toBeInTheDocument();
    expect(screen.getByTestId('source-status-dot-syncing')).toBeInTheDocument();
  });

  it('test_typeIcon: renders the correct type icon marker', () => {
    const { rerender } = render(
      <SourceCard
        source={dataSources[0]}
        testing={false}
        onTest={vi.fn()}
        onSync={vi.fn()}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
      />,
    );

    expect(screen.getByTestId('source-type-icon-postgresql')).toBeInTheDocument();

    rerender(
      <SourceCard
        source={dataSources[1]}
        testing={false}
        onTest={vi.fn()}
        onSync={vi.fn()}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
      />,
    );

    expect(screen.getByTestId('source-type-icon-api')).toBeInTheDocument();
  });

  it('test_testButton: clicking test invokes handler and shows inline result state', async () => {
    const user = userEvent.setup();
    const onTest = vi.fn();
    const { rerender } = render(
      <SourceCard
        source={dataSources[0]}
        testing={false}
        onTest={onTest}
        onSync={vi.fn()}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
      />,
    );

    await user.click(screen.getByRole('button', { name: /test/i }));
    expect(onTest).toHaveBeenCalledTimes(1);

    rerender(
      <SourceCard
        source={dataSources[0]}
        testing
        onTest={onTest}
        onSync={vi.fn()}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
      />,
    );
    expect(screen.getByText(/Testing connection/i)).toBeInTheDocument();

    rerender(
      <SourceCard
        source={dataSources[0]}
        testing={false}
        testResult={{
          success: true,
          latency_ms: 12,
          version: 'PostgreSQL 16.2',
          message: 'ok',
        }}
        onTest={onTest}
        onSync={vi.fn()}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
      />,
    );

    expect(screen.getByText(/Connected in 12ms/i)).toBeInTheDocument();
  });
});

