'use client';

import { useQueries } from '@tanstack/react-query';
import { enterpriseApi } from '@/lib/enterprise';

export function useVisusCTIWidgets() {
  const [overviewQuery, threatMapQuery, sectorsQuery, campaignsQuery, brandAbuseQuery, riskScoreQuery] = useQueries({
    queries: [
      {
        queryKey: ['visus', 'cti', 'overview'],
        queryFn: () => enterpriseApi.visus.getCTIOverview(),
        refetchInterval: 120_000,
      },
      {
        queryKey: ['visus', 'cti', 'threat-map', '24h'],
        queryFn: () => enterpriseApi.visus.getCTIThreatMap('24h'),
        refetchInterval: 120_000,
      },
      {
        queryKey: ['visus', 'cti', 'sectors', '24h'],
        queryFn: () => enterpriseApi.visus.getCTISectors('24h'),
        refetchInterval: 120_000,
      },
      {
        queryKey: ['visus', 'cti', 'campaigns', 5],
        queryFn: () => enterpriseApi.visus.getCTICampaigns(5),
        refetchInterval: 120_000,
      },
      {
        queryKey: ['visus', 'cti', 'brand-abuse', 5],
        queryFn: () => enterpriseApi.visus.getCTIBrandAbuse(5),
        refetchInterval: 120_000,
      },
      {
        queryKey: ['visus', 'cti', 'risk-score'],
        queryFn: () => enterpriseApi.visus.getCTIRiskScore(),
        refetchInterval: 120_000,
      },
    ],
  });

  const error =
    overviewQuery.error
    ?? threatMapQuery.error
    ?? sectorsQuery.error
    ?? campaignsQuery.error
    ?? brandAbuseQuery.error
    ?? riskScoreQuery.error
    ?? null;

  const refetch = async () => {
    await Promise.all([
      overviewQuery.refetch(),
      threatMapQuery.refetch(),
      sectorsQuery.refetch(),
      campaignsQuery.refetch(),
      brandAbuseQuery.refetch(),
      riskScoreQuery.refetch(),
    ]);
  };

  return {
    overview: overviewQuery.data ?? null,
    threatMap: threatMapQuery.data ?? null,
    sectors: sectorsQuery.data ?? null,
    campaigns: campaignsQuery.data ?? null,
    brandAbuse: brandAbuseQuery.data ?? null,
    riskScore: riskScoreQuery.data ?? null,
    isLoading:
      overviewQuery.isLoading
      || threatMapQuery.isLoading
      || sectorsQuery.isLoading
      || campaignsQuery.isLoading
      || brandAbuseQuery.isLoading
      || riskScoreQuery.isLoading,
    error,
    refetch,
    queries: {
      overview: overviewQuery,
      threatMap: threatMapQuery,
      sectors: sectorsQuery,
      campaigns: campaignsQuery,
      brandAbuse: brandAbuseQuery,
      riskScore: riskScoreQuery,
    },
  };
}
