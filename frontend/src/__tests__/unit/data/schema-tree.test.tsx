import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SchemaTree } from '@/app/(dashboard)/data/sources/[id]/_components/schema-tree';
import { sourceSchema } from '@/__tests__/data-suite-fixtures';

describe('SchemaTree', () => {
  it('test_rendersAllTables: renders all top-level tables', () => {
    render(<SchemaTree schema={sourceSchema} />);

    expect(screen.getByTestId('schema-table-customers')).toBeInTheDocument();
    expect(screen.getByTestId('schema-table-orders')).toBeInTheDocument();
  });

  it('test_expandsColumns: clicking a table reveals its columns', async () => {
    const user = userEvent.setup();
    render(<SchemaTree schema={sourceSchema} />);

    await user.click(screen.getByTestId('schema-table-customers'));

    expect(screen.getAllByText('email').length).toBeGreaterThan(0);
    expect(screen.getByText('first_name')).toBeInTheDocument();
  });

  it('test_piiWarning: shows PII icon and badge for PII columns', async () => {
    const user = userEvent.setup();
    render(<SchemaTree schema={sourceSchema} />);

    await user.click(screen.getByTestId('schema-table-customers'));

    expect(screen.getByTestId('schema-icon-pii-email')).toBeInTheDocument();
    expect(screen.getAllByText('email').length).toBeGreaterThan(0);
  });

  it('test_classificationBadge: shows the classification badge for restricted tables', () => {
    const restrictedSchema = {
      ...sourceSchema,
      tables: [
        {
          ...sourceSchema.tables[0],
          name: 'restricted_customers',
          inferred_classification: 'restricted' as const,
        },
      ],
      table_count: 1,
    };

    render(<SchemaTree schema={restrictedSchema} />);

    expect(screen.getByText('Restricted')).toBeInTheDocument();
  });

  it('test_pkIcon: shows a primary key icon for PK columns', async () => {
    const user = userEvent.setup();
    render(<SchemaTree schema={sourceSchema} />);

    await user.click(screen.getByTestId('schema-table-customers'));

    expect(screen.getByTestId('schema-icon-pk-id')).toBeInTheDocument();
  });

  it('test_searchFilters: filters tables by search text', () => {
    render(<SchemaTree schema={sourceSchema} filter="customer" />);

    expect(screen.getByTestId('schema-table-customers')).toBeInTheDocument();
    expect(screen.queryByTestId('schema-table-orders')).not.toBeInTheDocument();
  });
});

