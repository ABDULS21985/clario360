'use client';

import { ArrowRight, ShieldAlert } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { CTISeverityBadge } from '@/components/cyber/cti/severity-badge';
import { formatNumber, formatRelativeTime } from '@/lib/cti-utils';
import type { CTIBrandAbuseSummary } from '@/types/visus-cti';

interface CTIBrandAbuseWidgetProps {
  incidents: CTIBrandAbuseSummary[];
  isLoading: boolean;
  onViewAll: () => void;
  onIncidentClick: (id: string) => void;
}

export function CTIBrandAbuseWidget({
  incidents,
  isLoading,
  onViewAll,
  onIncidentClick,
}: CTIBrandAbuseWidgetProps) {
  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-4 w-36" />
          <Skeleton className="h-3 w-48" />
        </CardHeader>
        <CardContent className="space-y-3">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-14 w-full rounded-lg" />
          ))}
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-4 space-y-0">
        <div>
          <CardTitle className="text-base">Critical Brand Abuse</CardTitle>
          <CardDescription>Executive watchlist of active impersonation and phishing incidents.</CardDescription>
        </div>
        <ShieldAlert className="h-5 w-5 text-muted-foreground" />
      </CardHeader>
      <CardContent className="space-y-3">
        {incidents.length === 0 ? (
          <div className="rounded-lg border border-dashed px-4 py-8 text-center text-sm text-muted-foreground">
            No critical brand abuse incidents are active.
          </div>
        ) : (
          incidents.slice(0, 5).map((incident) => (
            <button
              key={incident.id}
              type="button"
              onClick={() => onIncidentClick(incident.id)}
              className="w-full rounded-lg border px-3 py-3 text-left transition hover:bg-muted/30"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <p className="font-medium">{incident.brand_name}</p>
                  <p className="truncate font-mono text-xs text-rose-500">{incident.malicious_domain}</p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {incident.abuse_type} · {formatNumber(incident.detection_count)} detections · {formatRelativeTime(incident.last_detected_at)}
                  </p>
                </div>
                <CTISeverityBadge severity={incident.risk_level} size="sm" />
              </div>
            </button>
          ))
        )}

        <Button variant="ghost" size="sm" className="px-0" onClick={onViewAll}>
          View All
          <ArrowRight className="ml-1 h-4 w-4" />
        </Button>
      </CardContent>
    </Card>
  );
}
