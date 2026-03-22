'use client';

import { useState } from 'react';
import { type ColumnDef } from '@tanstack/react-table';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { File, MoreHorizontal, Pencil, Plus, Trash2 } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { RelativeTime } from '@/components/shared/relative-time';
import { StatusBadge } from '@/components/shared/status-badge';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SearchInput } from '@/components/shared/forms/search-input';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useDataTable } from '@/hooks/use-data-table';
import { useAuth } from '@/hooks/use-auth';
import { API_ENDPOINTS } from '@/lib/constants';
import { enterpriseApi } from '@/lib/enterprise';
import { fetchSuitePaginated } from '@/lib/suite-api';
import { documentStatusConfig } from '@/lib/status-configs';
import { showApiError, showSuccess } from '@/lib/toast';
import type { LexDocument } from '@/types/suites';
import { DocumentFormDialog } from './_components/document-form-dialog';

const DOCUMENT_FILTERS = [
  {
    key: 'type',
    label: 'Type',
    type: 'select' as const,
    options: [
      { label: 'Policy', value: 'policy' },
      { label: 'Regulation', value: 'regulation' },
      { label: 'Template', value: 'template' },
      { label: 'Memo', value: 'memo' },
      { label: 'Opinion', value: 'opinion' },
      { label: 'Filing', value: 'filing' },
      { label: 'Correspondence', value: 'correspondence' },
      { label: 'Resolution', value: 'resolution' },
      { label: 'Power of Attorney', value: 'power_of_attorney' },
      { label: 'Other', value: 'other' },
    ],
  },
  {
    key: 'status',
    label: 'Status',
    type: 'select' as const,
    options: [
      { label: 'Draft', value: 'draft' },
      { label: 'Active', value: 'active' },
      { label: 'Archived', value: 'archived' },
      { label: 'Superseded', value: 'superseded' },
    ],
  },
];

export default function LexDocumentsPage() {
  const queryClient = useQueryClient();
  const { hasPermission } = useAuth();
  const canWrite = hasPermission('lex:write');

  const [createOpen, setCreateOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<LexDocument | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<LexDocument | null>(null);

  const { tableProps, searchValue, setSearch } = useDataTable<LexDocument>({
    queryKey: 'lex-documents',
    fetchFn: (params) => fetchSuitePaginated<LexDocument>(API_ENDPOINTS.LEX_DOCUMENTS, params),
    defaultPageSize: 25,
    defaultSort: { column: 'updated_at', direction: 'desc' },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => enterpriseApi.lex.deleteDocument(id),
    onSuccess: async () => {
      showSuccess('Document deleted.', 'The legal document has been removed.');
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['lex-documents'] }),
        queryClient.invalidateQueries({ queryKey: ['lex-overview'] }),
      ]);
      setDeleteTarget(null);
    },
    onError: showApiError,
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
      id: 'confidentiality',
      header: 'Confidentiality',
      cell: ({ row }) => (
        <span className="text-sm capitalize text-muted-foreground">{row.original.confidentiality}</span>
      ),
    },
    {
      id: 'current_version',
      accessorKey: 'current_version',
      header: 'Version',
      enableSorting: true,
      cell: ({ row }) => <span className="text-sm text-muted-foreground">v{row.original.current_version}</span>,
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
    ...(canWrite
      ? [
          {
            id: 'actions',
            header: '',
            cell: ({ row }: { row: { original: LexDocument } }) => (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon" className="h-8 w-8">
                    <MoreHorizontal className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={() => setEditTarget(row.original)}>
                    <Pencil className="mr-2 h-4 w-4" />
                    Edit
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    className="text-destructive"
                    onClick={() => setDeleteTarget(row.original)}
                  >
                    <Trash2 className="mr-2 h-4 w-4" />
                    Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            ),
          } satisfies ColumnDef<LexDocument>,
        ]
      : []),
  ];

  return (
    <PermissionRedirect permission="lex:read">
      <div className="space-y-6">
        <PageHeader
          title="Documents"
          description="Legal document repository backed by the lex-service document APIs."
          actions={
            canWrite ? (
              <Button onClick={() => setCreateOpen(true)}>
                <Plus className="mr-1.5 h-4 w-4" />
                Create Document
              </Button>
            ) : undefined
          }
        />
        <DataTable
          {...tableProps}
          columns={columns}
          filters={DOCUMENT_FILTERS}
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
            description: 'No legal documents matched the current filters.',
          }}
        />
        <DocumentFormDialog
          open={createOpen}
          onOpenChange={setCreateOpen}
        />
        {editTarget ? (
          <DocumentFormDialog
            open
            document={editTarget}
            onOpenChange={(open) => { if (!open) setEditTarget(null); }}
          />
        ) : null}
        <ConfirmDialog
          open={deleteTarget !== null}
          onOpenChange={(open) => { if (!open) setDeleteTarget(null); }}
          title="Delete document"
          description={`Are you sure you want to delete "${deleteTarget?.title}"? This action cannot be undone.`}
          confirmLabel="Delete"
          variant="destructive"
          loading={deleteMutation.isPending}
          onConfirm={() => {
            if (deleteTarget) deleteMutation.mutate(deleteTarget.id);
          }}
        />
      </div>
    </PermissionRedirect>
  );
}
