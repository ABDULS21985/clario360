'use client';

import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { ArrowRight } from 'lucide-react';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { Badge } from '@/components/ui/badge';
import { RelativeTime } from '@/components/shared/relative-time';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { StatusBadge } from '@/components/shared/status-badge';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS, ROUTES } from '@/lib/constants';
import { ALERT_STATUS_CONFIG, getAlertStatusVariant } from '@/lib/cyber-alerts';
import type { CyberAlert } from '@/types/cyber';

interface AlertRelatedProps {
  alert: CyberAlert;
}

export function AlertRelated({ alert }: AlertRelatedProps) {
  const relatedQuery = useQuery({
    queryKey: ['alert-related', alert.id],
    queryFn: () => apiGet<{ data: CyberAlert[] }>(API_ENDPOINTS.CYBER_ALERT_RELATED(alert.id)),
  });

  const relatedAlerts = relatedQuery.data?.data ?? [];

  if (relatedQuery.isLoading) {
    return <LoadingSkeleton variant="list-item" count={4} />;
  }

  if (relatedQuery.error) {
    return <ErrorState message="Failed to load related alerts" onRetry={() => void relatedQuery.refetch()} />;
  }

  if (relatedAlerts.length === 0) {
    return (
      <div className="rounded-[26px] border border-dashed bg-card p-8 text-center text-muted-foreground">
        No related alerts were found for this case.
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {relatedAlerts.map((item) => {
        const relations = inferRelations(alert, item);

        return (
          <Link
            key={item.id}
            href={`${ROUTES.CYBER_ALERTS}/${item.id}`}
            className="block rounded-[26px] border bg-card p-5 shadow-sm transition-transform hover:-translate-y-0.5"
          >
            <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
              <div className="min-w-0 space-y-3">
                <div className="flex flex-wrap items-center gap-2">
                  <SeverityIndicator severity={item.severity} showLabel />
                  <StatusBadge
                    status={item.status}
                    config={ALERT_STATUS_CONFIG}
                    variant={getAlertStatusVariant(item.status)}
                  />
                  {relations.map((relation) => (
                    <Badge key={relation} variant="secondary">
                      {relation}
                    </Badge>
                  ))}
                </div>
                <div>
                  <h3 className="text-lg font-semibold tracking-[-0.03em] text-slate-950">{item.title}</h3>
                  <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">
                    {item.description || 'No description was supplied.'}
                  </p>
                </div>
                <div className="flex flex-wrap gap-4 text-xs text-muted-foreground">
                  <span>Rule: {item.rule_name ?? 'Detection pipeline'}</span>
                  <span>Asset: {item.asset_name ?? item.asset_hostname ?? item.asset_ip_address ?? 'Unknown'}</span>
                  <span>Technique: {item.mitre_technique_id ?? 'Unmapped'}</span>
                </div>
              </div>
              <div className="flex items-center gap-3 text-sm text-muted-foreground">
                <RelativeTime date={item.created_at} />
                <ArrowRight className="h-4 w-4" />
              </div>
            </div>
          </Link>
        );
      })}
    </div>
  );
}

function inferRelations(source: CyberAlert, candidate: CyberAlert): string[] {
  const relations: string[] = [];

  if (source.rule_id && source.rule_id === candidate.rule_id) {
    relations.push('Same Rule');
  }
  if (source.asset_id && source.asset_id === candidate.asset_id) {
    relations.push('Same Asset');
  }
  if (source.mitre_technique_id && source.mitre_technique_id === candidate.mitre_technique_id) {
    relations.push('Same Technique');
  }
  if (source.source === candidate.source) {
    relations.push('Same Source');
  }

  return relations.length > 0 ? relations : ['Correlated'];
}
