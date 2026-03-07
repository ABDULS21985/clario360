'use client';

import Link from 'next/link';
import { type ColumnDef } from '@tanstack/react-table';
import { FileText } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { RelativeTime } from '@/components/shared/relative-time';
import { StatusBadge } from '@/components/shared/status-badge';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SearchInput } from '@/components/shared/forms/search-input';
import { useDataTable } from '@/hooks/use-data-table';
import { API_ENDPOINTS } from '@/lib/constants';
import { fetchSuitePaginated } from '@/lib/suite-api';
import { contractStatusConfig } from '@/lib/status-configs';
import { summarizeNamedRecords } from '@/lib/suite-utils';
import type { LexContract } from '@/types/suites';

const CONTRACT_FILTERS = [
  {
    key: 'status',
    label: 'Status',
    type: 'select' as const,
    options: [
      { label: 'Draft', value: 'draft' },
      { label: 'Review', value: 'review' },
      { label: 'Negotiation', value: 'negotiation' },
      { label: 'Active', value: 'active' },
      { label: 'Expired', value: 'expired' },
      { label: 'Terminated', value: 'terminated' },
    ],
  },
];

export default function LexContractsPage() {
  const { tableProps, searchValue, setSearch } = useDataTable<LexContract>({
    queryKey: 'lex-contracts',
    fetchFn: (params) => fetchSuitePaginated<LexContract>(API_ENDPOINTS.LEX_CONTRACTS, params),
    defaultPageSize: 25,
    defaultSort: { column: 'updated_at', direction: 'desc' },
  });

  const columns: ColumnDef<LexContract>[] = [
    {
      id: 'title',
      accessorKey: 'title',
      header: 'Contract',
      enableSorting: true,
      cell: ({ row }) => (
        <div>
          <Link href={`/lex/contracts/${row.original.id}`} className="font-medium hover:underline">
            {row.original.title}
          </Link>
          <p className="text-xs capitalize text-muted-foreground">{row.original.type.replace(/_/g, ' ')}</p>
        </div>
      ),
    },
    {
      id: 'parties',
      header: 'Parties',
      cell: ({ row }) => <span className="text-sm text-muted-foreground">{summarizeNamedRecords(row.original.parties, 2)}</span>,
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => <StatusBadge status={row.original.status} config={contractStatusConfig} size="sm" />,
    },
    {
      id: 'value',
      accessorKey: 'value',
      header: 'Value',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.value != null ? `${row.original.currency} ${row.original.value.toLocaleString()}` : 'Undisclosed'}
        </span>
      ),
    },
    {
      id: 'expiry_date',
      accessorKey: 'expiry_date',
      header: 'Expiry',
      enableSorting: true,
      cell: ({ row }) =>
        row.original.expiry_date ? (
          <RelativeTime date={row.original.expiry_date} />
        ) : (
          <span className="text-sm text-muted-foreground">No expiry</span>
        ),
    },
  ];

  return (
    <PermissionRedirect permission="lex:read">
      <div className="space-y-6">
        <PageHeader title="Contracts" description="Contract portfolio across lifecycle state, counterparty coverage, and renewal timing." />
        <DataTable
          {...tableProps}
          columns={columns}
          filters={CONTRACT_FILTERS}
          searchSlot={
            <SearchInput
              value={searchValue}
              onChange={setSearch}
              placeholder="Search contracts..."
              loading={tableProps.isLoading}
            />
          }
          emptyState={{
            icon: FileText,
            title: 'No contracts found',
            description: 'No contracts matched the current filters.',
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
