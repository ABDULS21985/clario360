'use client';

import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CheckSquare, KanbanSquare, ListTodo } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { SearchInput } from '@/components/shared/forms/search-input';
import { Button } from '@/components/ui/button';
import { enterpriseApi } from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import type { ActaActionItem } from '@/types/suites';
import { actionItemColumns } from './_components/action-item-columns';
import { ActionItemKanban } from './_components/action-item-kanban';
import { CompleteActionDialog } from './_components/complete-action-dialog';
import { CreateActionItemDialog } from './_components/create-action-item-dialog';
import { ExtendDueDateDialog } from './_components/extend-due-date-dialog';

export default function ActaActionItemsPage() {
  const queryClient = useQueryClient();
  const [view, setView] = useState<'table' | 'kanban'>('table');
  const [scope, setScope] = useState<'all' | 'my' | 'overdue' | 'completed'>('all');
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [sortColumn, setSortColumn] = useState<'due_date' | 'updated_at'>('updated_at');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('desc');
  const [createOpen, setCreateOpen] = useState(false);
  const [completeItem, setCompleteItem] = useState<ActaActionItem | null>(null);
  const [extendItem, setExtendItem] = useState<ActaActionItem | null>(null);

  const actionsQuery = useQuery({
    queryKey: ['acta-action-items', scope],
    queryFn: async () => {
      switch (scope) {
        case 'my':
          return enterpriseApi.acta.listMyActionItems();
        case 'overdue':
          return enterpriseApi.acta.listOverdueActionItems();
        case 'completed': {
          const response = await enterpriseApi.acta.listActionItems({
            page: 1,
            per_page: 200,
            order: 'desc',
            filters: { status: 'completed' },
          });
          return response.data;
        }
        default: {
          const response = await enterpriseApi.acta.listActionItems({
            page: 1,
            per_page: 200,
            order: 'desc',
          });
          return response.data;
        }
      }
    },
  });
  const committeesQuery = useQuery({
    queryKey: ['acta-action-item-committees'],
    queryFn: () => enterpriseApi.acta.listCommittees({ page: 1, per_page: 100, order: 'asc' }),
  });
  const meetingsQuery = useQuery({
    queryKey: ['acta-action-item-meetings'],
    queryFn: () => enterpriseApi.acta.listMeetings({ page: 1, per_page: 200, order: 'desc' }),
  });

  const statusMutation = useMutation({
    mutationFn: ({ item, status }: { item: ActaActionItem; status: ActaActionItem['status'] }) =>
      enterpriseApi.acta.updateActionItemStatus(item.id, { status }),
    onSuccess: async () => {
      showSuccess('Action item updated.', 'The action item status has been changed.');
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['acta-action-items'] }),
        queryClient.invalidateQueries({ queryKey: ['acta-dashboard'] }),
      ]);
    },
    onError: showApiError,
  });

  const filtered = useMemo(() => {
    const items = actionsQuery.data ?? [];
    const query = search.trim().toLowerCase();
    const sorted = [...items].sort((left, right) => {
      const leftValue = left[sortColumn];
      const rightValue = right[sortColumn];
      if (leftValue === rightValue) {
        return 0;
      }
      if (sortDirection === 'asc') {
        return String(leftValue).localeCompare(String(rightValue));
      }
      return String(rightValue).localeCompare(String(leftValue));
    });
    if (!query) {
      return sorted;
    }
    return sorted.filter((item) =>
      [item.title, item.description, item.assignee_name, item.meeting_title ?? '']
        .join(' ')
        .toLowerCase()
        .includes(query),
    );
  }, [actionsQuery.data, search, sortColumn, sortDirection]);

  const paged = filtered.slice((page - 1) * pageSize, page * pageSize);

  return (
    <PermissionRedirect permission="acta:read">
      <div className="space-y-6">
        <PageHeader
          title="Action Items"
          description="Track governance follow-ups in table or kanban view."
          actions={
            <>
              <div className="flex rounded-lg border p-1">
                <Button variant={view === 'table' ? 'default' : 'ghost'} size="sm" onClick={() => setView('table')}>
                  <ListTodo className="mr-1.5 h-4 w-4" />
                  Table
                </Button>
                <Button variant={view === 'kanban' ? 'default' : 'ghost'} size="sm" onClick={() => setView('kanban')}>
                  <KanbanSquare className="mr-1.5 h-4 w-4" />
                  Kanban
                </Button>
              </div>
              <Button onClick={() => setCreateOpen(true)}>
                <CheckSquare className="mr-1.5 h-4 w-4" />
                Create Action
              </Button>
            </>
          }
        />

        <div className="flex flex-wrap gap-2">
          {(['all', 'my', 'overdue', 'completed'] as const).map((tab) => (
            <Button
              key={tab}
              variant={scope === tab ? 'default' : 'outline'}
              size="sm"
              onClick={() => setScope(tab)}
            >
              {tab === 'my' ? 'My Items' : tab.replace(/_/g, ' ')}
            </Button>
          ))}
        </div>

        <SearchInput
          value={search}
          onChange={setSearch}
          placeholder="Search action items..."
          loading={actionsQuery.isLoading}
        />

        {view === 'table' ? (
          <DataTable
            columns={actionItemColumns({
              onComplete: (item) => setCompleteItem(item),
              onExtend: (item) => setExtendItem(item),
            })}
            data={paged}
            totalRows={filtered.length}
            page={page}
            pageSize={pageSize}
            onPageChange={setPage}
            onPageSizeChange={setPageSize}
            sortColumn={sortColumn}
            sortDirection={sortDirection}
            onSortChange={(column, direction) => {
              if (column === 'due_date' || column === 'updated_at') {
                setSortColumn(column);
                setSortDirection(direction);
              }
            }}
            searchValue={search}
            onSearchChange={setSearch}
            isLoading={actionsQuery.isLoading}
            error={actionsQuery.error ? 'Failed to load action items.' : null}
            onRetry={() => void actionsQuery.refetch()}
            emptyState={{
              icon: CheckSquare,
              title: 'No action items',
              description: 'No action items matched the current scope.',
            }}
          />
        ) : (
          <ActionItemKanban
            items={filtered}
            onMove={(item, nextStatus) => {
              if (nextStatus === 'completed') {
                setCompleteItem(item);
                return;
              }
              statusMutation.mutate({ item, status: nextStatus });
            }}
          />
        )}

        <CreateActionItemDialog
          open={createOpen}
          onOpenChange={setCreateOpen}
          committees={committeesQuery.data?.data ?? []}
          meetings={meetingsQuery.data?.data ?? []}
        />
        <CompleteActionDialog
          open={Boolean(completeItem)}
          onOpenChange={(open) => !open && setCompleteItem(null)}
          item={completeItem}
        />
        <ExtendDueDateDialog
          open={Boolean(extendItem)}
          onOpenChange={(open) => !open && setExtendItem(null)}
          item={extendItem}
        />
      </div>
    </PermissionRedirect>
  );
}
