'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { type ColumnDef } from '@tanstack/react-table';
import { FileText, Plus } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { RelativeTime } from '@/components/shared/relative-time';
import { StatusBadge } from '@/components/shared/status-badge';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SearchInput } from '@/components/shared/forms/search-input';
import { Button } from '@/components/ui/button';
import { useDataTable } from '@/hooks/use-data-table';
import { useAuth } from '@/hooks/use-auth';
import { API_ENDPOINTS } from '@/lib/constants';
import { fetchSuitePaginated } from '@/lib/suite-api';
import { contractStatusConfig } from '@/lib/status-configs';
import type { LexContract } from '@/types/suites';
import { ContractFormDialog } from './_components/contract-form-dialog';

const CONTRACT_FILTERS = [
  {
    key: 'status',
    label: 'Status',
    type: 'select' as const,
    options: [
      { label: 'Draft', value: 'draft' },
      { label: 'Internal Review', value: 'internal_review' },
      { label: 'Legal Review', value: 'legal_review' },
      { label: 'Negotiation', value: 'negotiation' },
      { label: 'Pending Signature', value: 'pending_signature' },
      { label: 'Active', value: 'active' },
      { label: 'Suspended', value: 'suspended' },
      { label: 'Expired', value: 'expired' },
      { label: 'Terminated', value: 'terminated' },
      { label: 'Renewed', value: 'renewed' },
      { label: 'Cancelled', value: 'cancelled' },
    ],
  },
];

export default function LexContractsPage() {
  const router = useRouter();
  const { hasPermission } = useAuth();
  const [createOpen, setCreateOpen] = useState(false);
  const canWrite = hasPermission('lex:write');

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
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {[row.original.party_a_name, row.original.party_b_name].filter(Boolean).join(', ') || '—'}
        </span>
      ),
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => <StatusBadge status={row.original.status} config={contractStatusConfig} size="sm" />,
    },
    {
      id: 'total_value',
      accessorKey: 'total_value',
      header: 'Value',
      enableSorting: true,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.total_value != null ? `${row.original.currency} ${row.original.total_value.toLocaleString()}` : 'Undisclosed'}
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
        <PageHeader
          title="Contracts"
          description="Contract portfolio across lifecycle state, counterparty coverage, and renewal timing."
          actions={
            canWrite ? (
              <Button onClick={() => setCreateOpen(true)}>
                <Plus className="mr-1.5 h-4 w-4" />
                Create Contract
              </Button>
            ) : undefined
          }
        />
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
        <ContractFormDialog
          open={createOpen}
          onOpenChange={setCreateOpen}
          onSaved={(contract) => router.push(`/lex/contracts/${contract.id}`)}
        />
      </div>
    </PermissionRedirect>
  );
}
