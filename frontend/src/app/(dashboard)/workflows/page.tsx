'use client';

import { useState } from 'react';
import { Workflow } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { formatDateTime } from '@/lib/utils';
import { cn } from '@/lib/utils';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import { Badge } from '@/components/ui/badge';
import type { PaginatedResponse } from '@/types/api';

interface WorkflowInstance {
  id: string;
  name: string;
  status: 'active' | 'completed' | 'failed' | 'paused';
  created_at: string;
  updated_at: string;
}

function statusVariant(status: string): 'default' | 'success' | 'destructive' | 'warning' {
  const map: Record<string, 'default' | 'success' | 'destructive' | 'warning'> = {
    active: 'default',
    completed: 'success',
    failed: 'destructive',
    paused: 'warning',
  };
  return map[status] ?? 'default';
}

export default function WorkflowsPage() {
  const [page, setPage] = useState(1);

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['workflows', 'instances', page],
    queryFn: () =>
      apiGet<PaginatedResponse<WorkflowInstance>>('/api/v1/workflows/instances', {
        page,
        per_page: 20,
      }),
  });

  return (
    <div className="space-y-6">
      <PageHeader title="Workflows" description="Monitor workflow instances" />

      {isLoading ? (
        <LoadingSkeleton variant="table-row" count={8} />
      ) : isError ? (
        <ErrorState message="Failed to load workflows" onRetry={() => refetch()} />
      ) : !data || data.data.length === 0 ? (
        <EmptyState icon={Workflow} title="No workflows" description="No workflow instances found." />
      ) : (
        <div className="rounded-lg border bg-card overflow-hidden">
          <table className="w-full text-sm">
            <thead className="border-b bg-muted/30">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Workflow</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Status</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground hidden md:table-cell">Updated</th>
              </tr>
            </thead>
            <tbody>
              {data.data.map((wf) => (
                <tr key={wf.id} className="border-b last:border-0 hover:bg-muted/30">
                  <td className="px-4 py-3">
                    <p className="font-medium">{wf.name}</p>
                    <p className="text-xs text-muted-foreground font-mono">{wf.id.slice(0, 8)}</p>
                  </td>
                  <td className="px-4 py-3">
                    <Badge variant={statusVariant(wf.status)}>{wf.status}</Badge>
                  </td>
                  <td className="px-4 py-3 text-xs text-muted-foreground hidden md:table-cell">
                    {formatDateTime(wf.updated_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
