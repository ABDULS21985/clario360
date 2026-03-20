import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { QueryBuilder, type QueryOrderRowState } from '@/app/(dashboard)/data/analytics/_components/query-builder';
import { QueryResultsTable } from '@/app/(dashboard)/data/analytics/_components/query-results-table';
import type { QueryAggregationRowState } from '@/app/(dashboard)/data/analytics/_components/query-aggregation-builder';
import type { QueryFilterRowState } from '@/app/(dashboard)/data/analytics/_components/query-filter-builder';
import { analyticsResult, dataModels } from '@/__tests__/data-suite-fixtures';

function QueryBuilderHarness() {
  const [selectedModelId, setSelectedModelId] = useState<string | null>(null);
  const [selectedColumns, setSelectedColumns] = useState<string[]>([]);
  const [filters, setFilters] = useState<QueryFilterRowState[]>([]);
  const [aggregations, setAggregations] = useState<QueryAggregationRowState[]>([]);
  const [groupBy, setGroupBy] = useState<string[]>([]);
  const [orders, setOrders] = useState<QueryOrderRowState[]>([]);
  const [limit, setLimit] = useState(100);

  return (
    <QueryBuilder
      models={dataModels}
      selectedModelId={selectedModelId}
      selectedColumns={selectedColumns}
      filters={filters}
      aggregations={aggregations}
      groupBy={groupBy}
      orders={orders}
      limit={limit}
      running={false}
      onSelectModel={setSelectedModelId}
      onToggleColumn={(column) =>
        setSelectedColumns((current) =>
          current.includes(column) ? current.filter((item) => item !== column) : [...current, column],
        )
      }
      onSelectAllColumns={() =>
        setSelectedColumns(dataModels[0].schema_definition.map((field) => field.name))
      }
      onClearColumns={() => setSelectedColumns([])}
      onChangeFilters={setFilters}
      onChangeAggregations={setAggregations}
      onChangeGroupBy={setGroupBy}
      onChangeOrders={setOrders}
      onChangeLimit={setLimit}
      onRun={vi.fn()}
      onSave={vi.fn()}
      onClear={() => {
        setSelectedModelId(null);
        setSelectedColumns([]);
        setFilters([]);
        setAggregations([]);
        setGroupBy([]);
        setOrders([]);
        setLimit(100);
      }}
    />
  );
}

describe('QueryBuilder', () => {
  it('test_selectModel: populates columns after selecting a model', async () => {
    const user = userEvent.setup();
    render(<QueryBuilderHarness />);

    await user.click(screen.getByRole('combobox'));
    await user.click(screen.getByText('Customer Master'));

    expect(screen.getByText('email')).toBeInTheDocument();
    expect(screen.getByText('status')).toBeInTheDocument();
  });

  it('test_addFilter: adds a filter row', async () => {
    const user = userEvent.setup();
    render(<QueryBuilderHarness />);

    await user.click(screen.getByRole('combobox'));
    await user.click(screen.getByText('Customer Master'));
    await user.click(screen.getByRole('button', { name: /add filter/i }));

    expect(screen.getByText('Remove')).toBeInTheDocument();
  });

  it('test_addAggregation: shows the group by section after adding an aggregation', async () => {
    const user = userEvent.setup();
    render(<QueryBuilderHarness />);

    await user.click(screen.getByRole('combobox'));
    await user.click(screen.getByText('Customer Master'));
    await user.click(screen.getByRole('button', { name: /add aggregation/i }));

    expect(screen.getByText('Group By')).toBeInTheDocument();
  });

  it('test_runQuery: invokes the run callback when clicking run', async () => {
    const onRun = vi.fn();
    const user = userEvent.setup();
    render(
      <QueryBuilder
        models={dataModels}
        selectedModelId={dataModels[0].id}
        selectedColumns={['email']}
        filters={[]}
        aggregations={[]}
        groupBy={[]}
        orders={[]}
        limit={100}
        running={false}
        onSelectModel={vi.fn()}
        onToggleColumn={vi.fn()}
        onSelectAllColumns={vi.fn()}
        onClearColumns={vi.fn()}
        onChangeFilters={vi.fn()}
        onChangeAggregations={vi.fn()}
        onChangeGroupBy={vi.fn()}
        onChangeOrders={vi.fn()}
        onChangeLimit={vi.fn()}
        onRun={onRun}
        onSave={vi.fn()}
        onClear={vi.fn()}
      />,
    );

    await user.click(screen.getByRole('button', { name: /run query/i }));
    expect(onRun).toHaveBeenCalledTimes(1);
  });

  it('test_piiMaskedResults: shows lock indicators and masked banner for masked results', () => {
    render(<QueryResultsTable result={analyticsResult} />);

    expect(screen.getByText(/PII columns masked/i)).toBeInTheDocument();
    expect(screen.getAllByText(/🔒/i)[0]).toBeInTheDocument();
  });

  it('test_truncatedBanner: shows truncation warning when results are truncated', () => {
    render(<QueryResultsTable result={analyticsResult} />);

    expect(screen.getByText(/Results truncated/i)).toBeInTheDocument();
  });
});

