'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Plus, Layout } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { PageHeader } from '@/components/common/page-header';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SearchInput } from '@/components/shared/forms/search-input';
import { ErrorState } from '@/components/common/error-state';
import { useDataTable } from '@/hooks/use-data-table';
import {
  useDeleteWorkflowDefinition,
  usePublishWorkflowDefinition,
  useArchiveWorkflowDefinition,
  useCloneWorkflowDefinition,
  useCreateWorkflowDefinition,
} from '@/hooks/use-workflow-definitions';
import { getDefinitionColumns } from './definition-columns';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { WorkflowDefinition } from '@/types/models';
import type { PaginatedResponse } from '@/types/api';
import type { FilterConfig } from '@/types/table';

const definitionFilters: FilterConfig[] = [
  {
    key: 'status',
    label: 'Status',
    type: 'multi-select',
    options: [
      { label: 'Draft', value: 'draft' },
      { label: 'Active', value: 'active' },
      { label: 'Archived', value: 'archived' },
    ],
  },
  {
    key: 'category',
    label: 'Category',
    type: 'multi-select',
    options: [
      { label: 'Approval', value: 'approval' },
      { label: 'Onboarding', value: 'onboarding' },
      { label: 'Review', value: 'review' },
      { label: 'Escalation', value: 'escalation' },
      { label: 'Notification', value: 'notification' },
      { label: 'Data Pipeline', value: 'data_pipeline' },
      { label: 'Compliance', value: 'compliance' },
      { label: 'Custom', value: 'custom' },
    ],
  },
];

export function DefinitionList() {
  const router = useRouter();
  const [deleteTarget, setDeleteTarget] = useState<WorkflowDefinition | null>(null);
  const [publishTarget, setPublishTarget] = useState<WorkflowDefinition | null>(null);
  const [archiveTarget, setArchiveTarget] = useState<WorkflowDefinition | null>(null);

  const deleteMutation = useDeleteWorkflowDefinition();
  const publishMutation = usePublishWorkflowDefinition();
  const archiveMutation = useArchiveWorkflowDefinition();
  const cloneMutation = useCloneWorkflowDefinition();
  const createMutation = useCreateWorkflowDefinition();

  const table = useDataTable<WorkflowDefinition>({
    queryKey: 'workflow-definitions',
    defaultPageSize: 25,
    defaultSort: { column: 'updated_at', direction: 'desc' },
    fetchFn: (params) =>
      apiGet<PaginatedResponse<WorkflowDefinition>>(
        API_ENDPOINTS.WORKFLOWS_DEFINITIONS,
        {
          page: params.page,
          per_page: params.per_page,
          sort: params.sort ?? 'updated_at',
          order: params.order ?? 'desc',
          search: params.search,
          ...(params.filters?.status
            ? {
                status: Array.isArray(params.filters.status)
                  ? params.filters.status.join(',')
                  : params.filters.status,
              }
            : {}),
          ...(params.filters?.category
            ? {
                category: Array.isArray(params.filters.category)
                  ? params.filters.category.join(',')
                  : params.filters.category,
              }
            : {}),
        },
      ),
  });

  const columns = getDefinitionColumns({
    onEdit: (def) =>
      router.push(`/admin/workflows/definitions/${def.id}/designer`),
    onView: (def) =>
      router.push(`/admin/workflows/definitions/${def.id}`),
    onPublish: (def) => setPublishTarget(def),
    onArchive: (def) => setArchiveTarget(def),
    onClone: (def) => cloneMutation.mutate(def.id),
    onDelete: (def) => setDeleteTarget(def),
  });

  function handleCreate() {
    createMutation.mutate(
      {
        name: 'Untitled Workflow',
        description: '',
        category: 'custom',
        trigger_config: { type: 'manual' },
        steps: [{ id: 'end_1', type: 'end', name: 'End', config: {}, transitions: [] }],
        variables: {},
      },
      {
        onSuccess: (newDef) => {
          router.push(`/admin/workflows/definitions/${newDef.id}/designer`);
        },
      },
    );
  }

  if (table.error) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Workflow Definitions"
          description="Design and manage workflow definitions"
        />
        <ErrorState
          message="Failed to load definitions"
          onRetry={() => table.refetch()}
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Workflow Definitions"
        description="Design and manage workflow definitions."
        actions={
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => router.push('/admin/workflows/templates')}
            >
              <Layout className="mr-1.5 h-3.5 w-3.5" />
              From Template
            </Button>
            <Button size="sm" onClick={handleCreate} disabled={createMutation.isPending}>
              <Plus className="mr-1.5 h-3.5 w-3.5" />
              Create Definition
            </Button>
          </div>
        }
      />

      <DataTable
        columns={columns}
        filters={definitionFilters}
        searchSlot={
          <SearchInput
            value={table.searchValue}
            onChange={table.setSearch}
            placeholder="Search definitions..."
          />
        }
        {...table.tableProps}
        onRowClick={(row) =>
          router.push(`/admin/workflows/definitions/${row.id}`)
        }
        emptyState={{
          icon: Layout,
          title: 'No workflow definitions',
          description: 'Create your first workflow definition to get started.',
          action: {
            label: 'Create Definition',
            onClick: handleCreate,
            icon: Plus,
          },
        }}
      />

      {/* Delete confirmation */}
      <AlertDialog
        open={!!deleteTarget}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Definition</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete &ldquo;{deleteTarget?.name}&rdquo;? This
              action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={() => {
                if (deleteTarget) {
                  deleteMutation.mutate(deleteTarget.id, {
                    onSuccess: () => setDeleteTarget(null),
                  });
                }
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Publish confirmation */}
      <AlertDialog
        open={!!publishTarget}
        onOpenChange={(open) => {
          if (!open) setPublishTarget(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Publish Definition</AlertDialogTitle>
            <AlertDialogDescription>
              Publishing &ldquo;{publishTarget?.name}&rdquo; will make it available for
              creating new workflow instances. Continue?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (publishTarget) {
                  publishMutation.mutate(publishTarget.id, {
                    onSuccess: () => setPublishTarget(null),
                  });
                }
              }}
            >
              Publish
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Archive confirmation */}
      <AlertDialog
        open={!!archiveTarget}
        onOpenChange={(open) => {
          if (!open) setArchiveTarget(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Archive Definition</AlertDialogTitle>
            <AlertDialogDescription>
              Archiving &ldquo;{archiveTarget?.name}&rdquo; will prevent new instances
              from being created. Existing instances will continue to run.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (archiveTarget) {
                  archiveMutation.mutate(archiveTarget.id, {
                    onSuccess: () => setArchiveTarget(null),
                  });
                }
              }}
            >
              Archive
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
