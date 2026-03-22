'use client';

import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import { Badge } from '@/components/ui/badge';
import { ShieldAlert, ExternalLink } from 'lucide-react';
import { timeAgo } from '@/lib/utils';
import type { PaginatedResponse } from '@/types/api';
import type { Vulnerability } from '@/types/cyber';

interface AssetVulnerabilitiesTabProps {
  assetId: string;
}

const STATUS_COLORS: Record<string, string> = {
  open: 'bg-red-100 text-red-800',
  in_progress: 'bg-blue-100 text-blue-800',
  mitigated: 'bg-yellow-100 text-yellow-800',
  resolved: 'bg-green-100 text-green-800',
  accepted: 'bg-gray-100 text-gray-800',
  false_positive: 'bg-muted text-muted-foreground',
};

export function AssetVulnerabilitiesTab({ assetId }: AssetVulnerabilitiesTabProps) {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['asset-vulns', assetId],
    queryFn: () =>
      apiGet<PaginatedResponse<Vulnerability>>(
        `${API_ENDPOINTS.CYBER_ASSETS}/${assetId}/vulnerabilities`,
        { per_page: 50 },
      ),
  });

  if (isLoading) return <LoadingSkeleton variant="table-row" count={8} />;
  if (error) return <ErrorState message="Failed to load vulnerabilities" onRetry={() => refetch()} />;
  if (!data || data.data.length === 0) {
    return (
      <EmptyState
        icon={ShieldAlert}
        title="No vulnerabilities"
        description="This asset has no known vulnerabilities."
      />
    );
  }

  return (
    <div className="space-y-3">
      <p className="text-sm text-muted-foreground">{data.meta.total} vulnerabilities found</p>
      <div className="overflow-hidden rounded-lg border">
        <table className="w-full text-sm">
          <thead className="border-b bg-muted/30">
            <tr>
              <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Severity</th>
              <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">CVE / Title</th>
              <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">CVSS</th>
              <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground hidden md:table-cell">Status</th>
              <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground hidden lg:table-cell">Age</th>
            </tr>
          </thead>
          <tbody>
            {data.data.map((vuln) => (
              <tr key={vuln.id} className="border-b last:border-0 hover:bg-muted/20">
                <td className="px-4 py-3">
                  <SeverityIndicator severity={vuln.severity} showLabel />
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-2">
                    <div>
                      {vuln.cve_id && (
                        <a
                          href={`https://nvd.nist.gov/vuln/detail/${vuln.cve_id}`}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="flex items-center gap-1 font-mono text-xs text-blue-600 hover:underline"
                        >
                          {vuln.cve_id}
                          <ExternalLink className="h-3 w-3" />
                        </a>
                      )}
                      <p className="font-medium leading-tight">{vuln.title}</p>
                      {vuln.has_exploit && (
                        <span className="inline-flex items-center rounded-sm bg-red-100 px-1 py-0.5 text-xs text-red-700">
                          Exploit Available
                        </span>
                      )}
                    </div>
                  </div>
                </td>
                <td className="px-4 py-3">
                  {vuln.cvss_score != null ? (
                    <span className="font-mono text-sm">{vuln.cvss_score.toFixed(1)}</span>
                  ) : '—'}
                </td>
                <td className="px-4 py-3 hidden md:table-cell">
                  <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_COLORS[vuln.status] ?? ''}`}>
                    {vuln.status.replace('_', ' ')}
                  </span>
                </td>
                <td className="px-4 py-3 hidden lg:table-cell text-muted-foreground text-xs">
                  {vuln.age_days}d
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
