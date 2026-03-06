'use client';

import { useState } from 'react';
import { File } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { formatDateTime, cn } from '@/lib/utils';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import { Badge } from '@/components/ui/badge';
import type { PaginatedResponse } from '@/types/api';
import type { FileItem } from '@/types/models';

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function statusVariant(status: string): 'default' | 'success' | 'destructive' | 'warning' | 'outline' {
  const map: Record<string, 'default' | 'success' | 'destructive' | 'warning' | 'outline'> = {
    available: 'success',
    processing: 'default',
    quarantined: 'destructive',
    pending: 'warning',
    deleted: 'outline',
  };
  return map[status] ?? 'outline';
}

export default function FilesPage() {
  const [page, setPage] = useState(1);

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['files', page],
    queryFn: () =>
      apiGet<PaginatedResponse<FileItem>>('/api/v1/files', { page, per_page: 25 }),
  });

  return (
    <div className="space-y-6">
      <PageHeader title="Files" description="Uploaded files and documents" />

      {isLoading ? (
        <LoadingSkeleton variant="table-row" count={10} />
      ) : isError ? (
        <ErrorState message="Failed to load files" onRetry={() => refetch()} />
      ) : !data || data.data.length === 0 ? (
        <EmptyState icon={File} title="No files" description="No files uploaded yet." />
      ) : (
        <div className="rounded-lg border bg-card overflow-hidden">
          <table className="w-full text-sm">
            <thead className="border-b bg-muted/30">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Name</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground hidden md:table-cell">Type</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground hidden md:table-cell">Size</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Status</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground hidden lg:table-cell">Uploaded</th>
              </tr>
            </thead>
            <tbody>
              {data.data.map((file) => (
                <tr key={file.id} className="border-b last:border-0 hover:bg-muted/30">
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <File className="h-4 w-4 shrink-0 text-muted-foreground" />
                      <span className="font-medium truncate max-w-[200px]">{file.original_name}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-xs text-muted-foreground hidden md:table-cell">{file.content_type}</td>
                  <td className="px-4 py-3 text-xs text-muted-foreground hidden md:table-cell">{formatBytes(file.size)}</td>
                  <td className="px-4 py-3">
                    <Badge variant={statusVariant(file.status)}>{file.status}</Badge>
                  </td>
                  <td className="px-4 py-3 text-xs text-muted-foreground hidden lg:table-cell">
                    {formatDateTime(file.created_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {data.meta.total_pages > 1 && (
            <div className="flex items-center justify-between border-t px-4 py-3">
              <p className="text-xs text-muted-foreground">Page {page} of {data.meta.total_pages}</p>
              <div className="flex gap-2">
                <button disabled={page <= 1} onClick={() => setPage((p) => p - 1)}
                  className="rounded border px-3 py-1 text-xs disabled:opacity-50 hover:bg-accent">Previous</button>
                <button disabled={page >= data.meta.total_pages} onClick={() => setPage((p) => p + 1)}
                  className="rounded border px-3 py-1 text-xs disabled:opacity-50 hover:bg-accent">Next</button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
