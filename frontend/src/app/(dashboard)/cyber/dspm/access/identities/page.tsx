'use client';

import { useMemo } from 'react';
import { useRouter } from 'next/navigation';
import { format } from 'date-fns';
import { ArrowLeft, Users } from 'lucide-react';
import { type ColumnDef } from '@tanstack/react-table';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { IdentityProfile } from '@/types/cyber';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams } from '@/types/table';

function flattenParams(params: FetchParams): Record<string, unknown> {
  const flat: Record<string, unknown> = {
    page: params.page,
    per_page: params.per_page,
    sort: params.sort,
    order: params.order,
    search: params.search,
  };

  for (const [key, value] of Object.entries(params.filters ?? {})) {
    flat[key] = value;
  }

  return flat;
}

function fetchIdentities(params: FetchParams): Promise<PaginatedResponse<IdentityProfile>> {
  return apiGet<PaginatedResponse<IdentityProfile>>(
    API_ENDPOINTS.CYBER_DSPM_ACCESS_IDENTITIES,
    flattenParams(params),
  );
}

function scoreColor(score: number): string {
  if (score >= 75) return 'bg-red-500';
  if (score >= 50) return 'bg-orange-500';
  if (score >= 25) return 'bg-amber-500';
  return 'bg-green-500';
}

function scoreTextColor(score: number): string {
  if (score >= 75) return 'text-red-700';
  if (score >= 50) return 'text-orange-700';
  if (score >= 25) return 'text-amber-700';
  return 'text-green-700';
}

const STATUS_VARIANT: Record<string, string> = {
  active: 'bg-green-100 text-green-800 border-green-200',
  inactive: 'bg-gray-100 text-gray-800 border-gray-200',
  under_review: 'bg-amber-100 text-amber-800 border-amber-200',
  remediated: 'bg-blue-100 text-blue-800 border-blue-200',
};

function identityTypeLabel(type: string): string {
  return type
    .split('_')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

function statusLabel(status: string): string {
  return status
    .split('_')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

export default function DspmAccessIdentitiesPage() {
  const router = useRouter();

  const { tableProps, refetch } = useDataTable<IdentityProfile>({
    queryKey: 'dspm-access-identities',
    fetchFn: fetchIdentities,
    defaultSort: { column: 'access_risk_score', direction: 'desc' },
  });

  const columns: ColumnDef<IdentityProfile>[] = useMemo(
    () => [
      {
        accessorKey: 'identity_name',
        header: 'Name',
        cell: ({ row }) => (
          <div className="flex items-center gap-2">
            <span className="font-medium">{row.original.identity_name}</span>
            <Badge variant="outline" className="text-[10px]">
              {identityTypeLabel(row.original.identity_type)}
            </Badge>
          </div>
        ),
      },
      {
        accessorKey: 'access_risk_score',
        header: 'Risk Score',
        cell: ({ row }) => {
          const score = row.original.access_risk_score;
          return (
            <div className="flex items-center gap-2">
              <div className="h-2 w-16 overflow-hidden rounded-full bg-muted">
                <div
                  className={`h-full rounded-full ${scoreColor(score)}`}
                  style={{ width: `${Math.min(score, 100)}%` }}
                />
              </div>
              <span className={`text-xs font-semibold ${scoreTextColor(score)}`}>
                {score}
              </span>
            </div>
          );
        },
      },
      {
        accessorKey: 'blast_radius_score',
        header: 'Blast Radius',
        cell: ({ row }) => {
          const score = row.original.blast_radius_score;
          return (
            <div className="flex items-center gap-2">
              <div className="h-2 w-16 overflow-hidden rounded-full bg-muted">
                <div
                  className={`h-full rounded-full ${scoreColor(score)}`}
                  style={{ width: `${Math.min(score, 100)}%` }}
                />
              </div>
              <span className={`text-xs font-semibold ${scoreTextColor(score)}`}>
                {score}
              </span>
            </div>
          );
        },
      },
      {
        accessorKey: 'overprivileged_count',
        header: 'Overprivileged',
        cell: ({ row }) => {
          const count = row.original.overprivileged_count;
          return (
            <Badge variant={count > 0 ? 'destructive' : 'secondary'} className="text-xs">
              {count}
            </Badge>
          );
        },
      },
      {
        accessorKey: 'stale_permission_count',
        header: 'Stale Permissions',
        cell: ({ row }) => {
          const count = row.original.stale_permission_count;
          return (
            <Badge variant={count > 0 ? 'default' : 'secondary'} className="text-xs">
              {count}
            </Badge>
          );
        },
      },
      {
        accessorKey: 'total_assets_accessible',
        header: 'Assets Accessible',
        cell: ({ row }) => (
          <span className="text-sm">{row.original.total_assets_accessible}</span>
        ),
      },
      {
        accessorKey: 'status',
        header: 'Status',
        cell: ({ row }) => {
          const status = row.original.status;
          return (
            <Badge
              variant="outline"
              className={STATUS_VARIANT[status] ?? 'bg-gray-100 text-gray-800'}
            >
              {statusLabel(status)}
            </Badge>
          );
        },
      },
      {
        accessorKey: 'last_activity_at',
        header: 'Last Activity',
        cell: ({ row }) => {
          const dt = row.original.last_activity_at;
          if (!dt) {
            return <span className="text-xs text-muted-foreground">Never</span>;
          }
          return (
            <span className="text-xs text-muted-foreground">
              {format(new Date(dt), 'MMM d, yyyy HH:mm')}
            </span>
          );
        },
      },
    ],
    [],
  );

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Identity Risk Ranking"
          description="Identities sorted by access risk score"
          actions={
            <Button
              variant="outline"
              size="sm"
              onClick={() => router.push('/cyber/dspm/access')}
            >
              <ArrowLeft className="mr-1.5 h-3.5 w-3.5" />
              Back
            </Button>
          }
        />

        <DataTable
          columns={columns}
          searchPlaceholder="Search identities..."
          emptyState={{
            icon: Users,
            title: 'No identities found',
            description: 'No identity profiles match the current filters.',
          }}
          getRowId={(row) => row.id}
          onRowClick={(row) =>
            router.push(`/cyber/dspm/access/identities/${row.identity_id}`)
          }
          {...tableProps}
        />
      </div>
    </PermissionRedirect>
  );
}
