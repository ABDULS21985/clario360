'use client';

import { type ColumnDef } from '@tanstack/react-table';
import { File } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { RelativeTime } from '@/components/shared/relative-time';
import { StatusBadge } from '@/components/shared/status-badge';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SearchInput } from '@/components/shared/forms/search-input';
import { useDataTable } from '@/hooks/use-data-table';
import { API_ENDPOINTS } from '@/lib/constants';
import { fetchSuitePaginated } from '@/lib/suite-api';
import { documentStatusConfig } from '@/lib/status-configs';
import type { LexDocument } from '@/types/suites';

export default function LexDocumentsPage() {
  const { tableProps, searchValue, setSearch } = useDataTable<LexDocument>({
    queryKey: 'lex-documents',
    fetchFn: (params) => fetchSuitePaginated<LexDocument>(API_ENDPOINTS.LEX_DOCUMENTS, params),
    defaultPageSize: 25,
    defaultSort: { column: 'updated_at', direction: 'desc' },
  });

  const columns: ColumnDef<LexDocument>[] = [
    {
      id: 'title',
      accessorKey: 'title',
      header: 'Document',
      enableSorting: true,
      cell: ({ row }) => (
        <div>
          <p className="font-medium">{row.original.title}</p>
          <p className="text-xs text-muted-foreground capitalize">{row.original.type.replace(/_/g, ' ')}</p>
        </div>
      ),
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      enableSorting: true,
      cell: ({ row }) => <StatusBadge status={row.original.status} config={documentStatusConfig} size="sm" />,
    },
    {
      id: 'version',
      accessorKey: 'version',
      header: 'Version',
      enableSorting: true,
      cell: ({ row }) => <span className="text-sm text-muted-foreground">v{row.original.version}</span>,
    },
    {
      id: 'tags',
      header: 'Tags',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">{row.original.tags.length > 0 ? row.original.tags.join(', ') : '—'}</span>
      ),
    },
    {
      id: 'updated_at',
      accessorKey: 'updated_at',
      header: 'Updated',
      enableSorting: true,
      cell: ({ row }) => <RelativeTime date={row.original.updated_at} />,
    },
  ];

  return (
    <PermissionRedirect permission="lex:read">
      <div className="space-y-6">
        <PageHeader title="Documents" description="Legal document repository backed by the lex-service document APIs." />
        <DataTable
          {...tableProps}
          columns={columns}
          searchSlot={
            <SearchInput
              value={searchValue}
              onChange={setSearch}
              placeholder="Search legal documents..."
              loading={tableProps.isLoading}
            />
          }
          emptyState={{
            icon: File,
            title: 'No documents found',
            description: 'No legal documents matched the current search.',
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
