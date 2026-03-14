'use client';

import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';
import { ExternalLink } from 'lucide-react';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import type { CampaignCluster } from '@/types/cyber';

interface CampaignResponse {
  items?: CampaignCluster[];
}

const STAGE_COLORS: Record<string, string> = {
  reconnaissance: 'bg-yellow-100 text-yellow-800',
  active_attack: 'bg-red-100 text-red-800',
  expanded_campaign: 'bg-purple-100 text-purple-800',
};

export function CampaignDetection() {
  const { data, isLoading } = useQuery({
    queryKey: ['cyber-analytics-campaigns'],
    queryFn: () => apiGet<{ data: CampaignResponse }>(API_ENDPOINTS.CYBER_ANALYTICS_CAMPAIGNS),
    refetchInterval: 300000,
  });

  const campaigns = data?.data?.items ?? [];

  if (isLoading) {
    return <LoadingSkeleton variant="card" />;
  }

  return (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold">Campaign Detection</h3>
      {campaigns.length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center">
            <p className="text-sm text-muted-foreground">
              No active campaigns detected. The system correlates alerts by IOC overlap, MITRE technique overlap, and temporal proximity.
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          {campaigns.map((campaign) => (
            <Card key={campaign.cluster_id}>
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-sm font-semibold">
                    Campaign #{campaign.cluster_id}
                  </CardTitle>
                  <Badge
                    className={`text-xs ${STAGE_COLORS[campaign.stage] ?? 'bg-gray-100 text-gray-800'}`}
                    variant="secondary"
                  >
                    {campaign.stage.replace(/_/g, ' ')}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="grid grid-cols-2 gap-2 text-xs">
                  <div>
                    <span className="text-muted-foreground">Alerts: </span>
                    <span className="font-medium">{campaign.alert_ids.length}</span>
                  </div>
                  <div>
                    <span className="text-muted-foreground">Confidence: </span>
                    <span className="font-medium tabular-nums">
                      {(campaign.confidence.p50 * 100).toFixed(0)}%
                    </span>
                  </div>
                  <div>
                    <span className="text-muted-foreground">Start: </span>
                    <span className="tabular-nums">
                      {new Date(campaign.start_at).toLocaleDateString()}
                    </span>
                  </div>
                  <div>
                    <span className="text-muted-foreground">End: </span>
                    <span className="tabular-nums">
                      {new Date(campaign.end_at).toLocaleDateString()}
                    </span>
                  </div>
                </div>

                {campaign.mitre_techniques.length > 0 && (
                  <div>
                    <span className="text-xs text-muted-foreground">MITRE Techniques:</span>
                    <div className="flex flex-wrap gap-1 mt-1">
                      {campaign.mitre_techniques.slice(0, 5).map((t) => (
                        <Badge key={t} variant="outline" className="text-xs">{t}</Badge>
                      ))}
                      {campaign.mitre_techniques.length > 5 && (
                        <Badge variant="secondary" className="text-xs">
                          +{campaign.mitre_techniques.length - 5}
                        </Badge>
                      )}
                    </div>
                  </div>
                )}

                {campaign.shared_iocs.length > 0 && (
                  <div>
                    <span className="text-xs text-muted-foreground">Shared IOCs:</span>
                    <div className="flex flex-wrap gap-1 mt-1">
                      {campaign.shared_iocs.slice(0, 3).map((ioc) => (
                        <code key={ioc} className="text-xs bg-muted px-1 py-0.5 rounded">{ioc}</code>
                      ))}
                      {campaign.shared_iocs.length > 3 && (
                        <Badge variant="secondary" className="text-xs">
                          +{campaign.shared_iocs.length - 3}
                        </Badge>
                      )}
                    </div>
                  </div>
                )}

                <Button variant="outline" size="sm" className="w-full" asChild>
                  <Link href={`/cyber/threats?campaign=${campaign.cluster_id}`}>
                    <ExternalLink className="mr-1.5 h-3 w-3" />
                    Investigate
                  </Link>
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
