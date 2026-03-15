'use client';


import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, ScanLine, AlertCircle } from 'lucide-react';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { formatDateTime, timeAgo } from '@/lib/utils';
import type { AssetScan } from '@/types/cyber';
import type { PaginatedResponse } from '@/types/api';

interface Props {
  params: { id: string };
}

interface ScannedAsset {
  id: string;
  name: string;
  type: string;
  ip_address?: string;
  status: string;
  criticality: string;
}

const STATUS_CONFIG: Record<
  AssetScan['status'],
  { label: string; className: string; dot?: boolean }
> = {
  pending: {
    label: 'Pending',
    className:
      'bg-secondary text-secondary-foreground hover:bg-secondary',
  },
  running: {
    label: 'Running',
    className:
      'bg-blue-100 text-blue-800 hover:bg-blue-100 dark:bg-blue-900/30 dark:text-blue-300',
    dot: true,
  },
  completed: {
    label: 'Completed',
    className:
      'bg-green-100 text-green-800 hover:bg-green-100 dark:bg-green-900/30 dark:text-green-300',
  },
  failed: {
    label: 'Failed',
    className: 'bg-destructive text-destructive-foreground hover:bg-destructive/90',
  },
  cancelled: {
    label: 'Cancelled',
    className: 'border border-border bg-transparent text-muted-foreground',
  },
};

const CRITICALITY_COLORS: Record<string, string> = {
  critical: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
  high: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300',
  medium: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300',
  low: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
};

function computeDuration(scan: AssetScan): string {
  if (!scan.started_at) return '—';
  const end = scan.completed_at ? new Date(scan.completed_at) : new Date();
  const start = new Date(scan.started_at);
  const diffMs = end.getTime() - start.getTime();
  if (diffMs < 0) return '—';
  const mins = Math.floor(diffMs / 60000);
  const secs = Math.floor((diffMs % 60000) / 1000);
  if (mins === 0) return `${secs}s`;
  return `${mins}m ${secs}s`;
}

export default function ScanDetailPage({ params }: Props) {
  const { id } = params;
  const router = useRouter();

  const { data: scanEnvelope, isLoading, error, refetch } = useQuery({
    queryKey: [`cyber-asset-scan-${id}`],
    queryFn: () => apiGet<{ data: AssetScan }>(`${API_ENDPOINTS.CYBER_ASSETS_SCANS}/${id}`),
    refetchInterval: (q) =>
      q.state.data?.data.status === 'running' ? 5000 : false,
  });

  const { data: assetsEnvelope, isLoading: assetsLoading } = useQuery({
    queryKey: [`cyber-asset-scan-assets-${id}`],
    queryFn: () =>
      apiGet<PaginatedResponse<ScannedAsset>>(API_ENDPOINTS.CYBER_ASSETS, {
        scan_id: id,
        per_page: 100,
      }),
    enabled: !!scanEnvelope?.data,
  });

  const scan = scanEnvelope?.data;
  const assets = assetsEnvelope?.data ?? [];
  const statusConfig = scan ? (STATUS_CONFIG[scan.status] ?? STATUS_CONFIG.cancelled) : null;

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        {isLoading ? (
          <>
            <div className="h-8 w-48 animate-pulse rounded bg-muted" />
            <LoadingSkeleton variant="card" />
          </>
        ) : error || !scan || !statusConfig ? (
          <ErrorState message="Failed to load scan details" onRetry={() => refetch()} />
        ) : (
          <>
            {/* Header */}
            <PageHeader
              title={
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => router.push('/cyber/assets/scans')}
                    className="flex h-8 w-8 items-center justify-center rounded-full border bg-background text-muted-foreground shadow-sm transition-colors hover:bg-accent"
                  >
                    <ArrowLeft className="h-4 w-4" />
                  </button>
                  <ScanLine className="h-5 w-5 text-muted-foreground" />
                  <span className="capitalize">{scan.scan_type} Scan</span>
                  <Badge
                    className={`${statusConfig.className} flex items-center gap-1.5 text-xs capitalize`}
                  >
                    {statusConfig.dot && (
                      <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-blue-500" />
                    )}
                    {statusConfig.label}
                  </Badge>
                </div>
              }
              description={
                <span className="pl-11 text-xs text-muted-foreground">
                  Started {scan.started_at ? timeAgo(scan.started_at) : '—'}
                </span>
              }
            />

            {/* Stats */}
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
              <div className="rounded-xl border bg-card p-4 text-center">
                <p className="text-2xl font-bold tabular-nums text-green-600 dark:text-green-400">
                  {scan.assets_found.toLocaleString()}
                </p>
                <p className="mt-0.5 text-xs text-muted-foreground">Assets Found</p>
              </div>
              <div className="rounded-xl border bg-card p-4 text-center">
                <p className="text-2xl font-bold tabular-nums text-blue-600 dark:text-blue-400">
                  {scan.assets_updated.toLocaleString()}
                </p>
                <p className="mt-0.5 text-xs text-muted-foreground">Assets Updated</p>
              </div>
              <div className="rounded-xl border bg-card p-4 text-center">
                <p className="text-2xl font-bold tabular-nums text-foreground">
                  {computeDuration(scan)}
                </p>
                <p className="mt-0.5 text-xs text-muted-foreground">Duration</p>
              </div>
              <div className="rounded-xl border bg-card p-4 text-center">
                <p className="truncate text-sm font-semibold text-foreground">
                  {scan.target ?? '—'}
                </p>
                <p className="mt-0.5 text-xs text-muted-foreground">Target</p>
              </div>
            </div>

            {/* Details card */}
            <div className="rounded-xl border bg-card p-5">
              <h3 className="mb-4 text-sm font-semibold">Scan Details</h3>
              <dl className="grid grid-cols-1 gap-x-8 gap-y-3 sm:grid-cols-2 lg:grid-cols-3">
                <div>
                  <dt className="text-xs font-medium text-muted-foreground">Scan Type</dt>
                  <dd className="mt-0.5 text-sm capitalize">{scan.scan_type}</dd>
                </div>
                <div>
                  <dt className="text-xs font-medium text-muted-foreground">Status</dt>
                  <dd className="mt-0.5 text-sm capitalize">{scan.status}</dd>
                </div>
                <div>
                  <dt className="text-xs font-medium text-muted-foreground">Target (CIDR)</dt>
                  <dd className="mt-0.5 font-mono text-sm">{scan.target ?? '—'}</dd>
                </div>
                <div>
                  <dt className="text-xs font-medium text-muted-foreground">Started At</dt>
                  <dd className="mt-0.5 text-sm">
                    {scan.started_at ? formatDateTime(scan.started_at) : '—'}
                  </dd>
                </div>
                <div>
                  <dt className="text-xs font-medium text-muted-foreground">Completed At</dt>
                  <dd className="mt-0.5 text-sm">
                    {scan.completed_at ? formatDateTime(scan.completed_at) : '—'}
                  </dd>
                </div>
                <div>
                  <dt className="text-xs font-medium text-muted-foreground">Duration</dt>
                  <dd className="mt-0.5 text-sm">{computeDuration(scan)}</dd>
                </div>
                {scan.error && (
                  <div className="col-span-full">
                    <dt className="text-xs font-medium text-destructive">Error</dt>
                    <dd className="mt-0.5 rounded-md bg-destructive/10 px-3 py-2 font-mono text-xs text-destructive">
                      {scan.error}
                    </dd>
                  </div>
                )}
              </dl>
            </div>

            {/* Discovered assets */}
            <div>
              <h3 className="mb-3 text-sm font-semibold">
                Discovered Assets
                {assets.length > 0 && (
                  <span className="ml-2 rounded-full bg-muted px-1.5 py-0.5 text-xs font-medium text-muted-foreground">
                    {assets.length}
                  </span>
                )}
              </h3>

              {assetsLoading ? (
                <LoadingSkeleton variant="table-row" />
              ) : assets.length === 0 ? (
                <EmptyState
                  icon={AlertCircle}
                  title="No assets discovered"
                  description="This scan did not discover any assets, or they have not been loaded yet."
                />
              ) : (
                <div className="rounded-xl border bg-card">
                  <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b bg-muted/50">
                          <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                            Name
                          </th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                            Type
                          </th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                            IP Address
                          </th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                            Status
                          </th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
                            Criticality
                          </th>
                        </tr>
                      </thead>
                      <tbody className="divide-y">
                        {assets.map((asset) => (
                          <tr
                            key={asset.id}
                            className="transition-colors hover:bg-muted/30"
                          >
                            <td className="px-4 py-3 font-medium">{asset.name}</td>
                            <td className="px-4 py-3">
                              <span className="rounded-full bg-muted px-2 py-0.5 text-xs capitalize">
                                {asset.type.replace(/_/g, ' ')}
                              </span>
                            </td>
                            <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                              {asset.ip_address ?? '—'}
                            </td>
                            <td className="px-4 py-3">
                              <span className="rounded-full bg-muted px-2 py-0.5 text-xs capitalize">
                                {asset.status}
                              </span>
                            </td>
                            <td className="px-4 py-3">
                              <span
                                className={`rounded-full px-2 py-0.5 text-xs font-medium capitalize ${CRITICALITY_COLORS[asset.criticality] ?? 'bg-muted text-muted-foreground'}`}
                              >
                                {asset.criticality}
                              </span>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </div>
          </>
        )}
      </div>
    </PermissionRedirect>
  );
}
