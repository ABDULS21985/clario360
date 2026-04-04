'use client';

import { ArrowRight, Radar } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { CTISeverityBadge } from '@/components/cyber/cti/severity-badge';
import { formatNumber, formatRelativeTime } from '@/lib/cti-utils';
import type { CTICampaignSummary } from '@/types/visus-cti';

interface CTICampaignsWidgetProps {
  campaigns: CTICampaignSummary[];
  isLoading: boolean;
  onViewAll: () => void;
  onCampaignClick: (id: string) => void;
}

export function CTICampaignsWidget({
  campaigns,
  isLoading,
  onViewAll,
  onCampaignClick,
}: CTICampaignsWidgetProps) {
  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-4 w-32" />
          <Skeleton className="h-3 w-40" />
        </CardHeader>
        <CardContent className="space-y-3">
          {Array.from({ length: 5 }).map((_, index) => (
            <Skeleton key={index} className="h-12 w-full rounded-lg" />
          ))}
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-4 space-y-0">
        <div>
          <CardTitle className="text-base">Active Campaigns</CardTitle>
          <CardDescription>Highest pressure campaigns requiring executive awareness.</CardDescription>
        </div>
        <Radar className="h-5 w-5 text-muted-foreground" />
      </CardHeader>
      <CardContent className="space-y-3">
        {campaigns.length === 0 ? (
          <div className="rounded-lg border border-dashed px-4 py-8 text-center text-sm text-muted-foreground">
            No active campaigns are available.
          </div>
        ) : (
          <div className="overflow-hidden rounded-lg border">
            <table className="w-full text-sm">
              <thead className="bg-muted/40 text-left text-xs uppercase tracking-[0.14em] text-muted-foreground">
                <tr>
                  <th className="px-3 py-2 font-medium">Campaign</th>
                  <th className="px-3 py-2 font-medium">Actor</th>
                  <th className="px-3 py-2 font-medium">Severity</th>
                  <th className="px-3 py-2 font-medium">IOCs</th>
                </tr>
              </thead>
              <tbody>
                {campaigns.slice(0, 5).map((campaign) => (
                  <tr
                    key={campaign.id}
                    className="cursor-pointer border-t transition hover:bg-muted/30"
                    onClick={() => onCampaignClick(campaign.id)}
                  >
                    <td className="px-3 py-3">
                      <div>
                        <p className="font-medium">{campaign.name}</p>
                        <p className="text-xs text-muted-foreground">
                          {campaign.campaign_code} · Last seen {formatRelativeTime(campaign.last_seen_at ?? campaign.first_seen_at)}
                        </p>
                      </div>
                    </td>
                    <td className="px-3 py-3 text-muted-foreground">{campaign.actor_name ?? 'Unassigned'}</td>
                    <td className="px-3 py-3">
                      <CTISeverityBadge severity={campaign.severity_code} size="sm" />
                    </td>
                    <td className="px-3 py-3 font-medium tabular-nums text-orange-500">{formatNumber(campaign.ioc_count)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        <Button variant="ghost" size="sm" className="px-0" onClick={onViewAll}>
          View All
          <ArrowRight className="ml-1 h-4 w-4" />
        </Button>
      </CardContent>
    </Card>
  );
}
