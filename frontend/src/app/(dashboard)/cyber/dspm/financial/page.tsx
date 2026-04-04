'use client';

import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { DollarSign, TrendingUp, AlertTriangle, BarChart3 } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { PortfolioRisk, FinancialImpact } from '@/types/cyber';

const currencyFormatter = new Intl.NumberFormat('en-US', {
  style: 'currency',
  currency: 'USD',
  maximumFractionDigits: 0,
});

const compactCurrencyFormatter = new Intl.NumberFormat('en-US', {
  style: 'currency',
  currency: 'USD',
  notation: 'compact',
  maximumFractionDigits: 1,
});

function formatCurrency(value: number): string {
  return value >= 1_000_000
    ? compactCurrencyFormatter.format(value)
    : currencyFormatter.format(value);
}

function shortenAssetId(id: string): string {
  return id.length > 12 ? `${id.slice(0, 8)}...` : id;
}

function probabilityColor(probability: number): string {
  if (probability > 0.5) return 'text-red-600';
  if (probability > 0.2) return 'text-orange-500';
  return 'text-green-600';
}

function probabilityBadgeVariant(probability: number): 'destructive' | 'default' | 'secondary' {
  if (probability > 0.5) return 'destructive';
  if (probability > 0.2) return 'default';
  return 'secondary';
}

export default function FinancialRiskQuantificationPage() {
  const {
    data: portfolioEnvelope,
    isLoading: portfolioLoading,
    error: portfolioError,
    refetch: refetchPortfolio,
  } = useQuery({
    queryKey: ['dspm-financial-portfolio'],
    queryFn: () => apiGet<{ data: PortfolioRisk }>(API_ENDPOINTS.CYBER_DSPM_FINANCIAL_PORTFOLIO),
    staleTime: 120000,
  });

  const {
    data: topRisksEnvelope,
    isLoading: topRisksLoading,
    error: topRisksError,
    refetch: refetchTopRisks,
  } = useQuery({
    queryKey: ['dspm-financial-top-risks'],
    queryFn: () => apiGet<{ data: FinancialImpact[] }>(API_ENDPOINTS.CYBER_DSPM_FINANCIAL_TOP_RISKS),
    staleTime: 120000,
  });

  const portfolio = portfolioEnvelope?.data;
  const topRisks = topRisksEnvelope?.data ?? [];
  const isLoading = portfolioLoading || topRisksLoading;
  const hasError = portfolioError || topRisksError;

  const kpis = useMemo(() => {
    if (!portfolio) return [];
    return [
      {
        label: 'Total Breach Cost Exposure',
        value: formatCurrency(portfolio.total_breach_cost),
        icon: DollarSign,
        color: 'text-red-600',
      },
      {
        label: 'Annual Expected Loss',
        value: formatCurrency(portfolio.total_expected_loss),
        icon: TrendingUp,
        color: 'text-orange-600',
      },
      {
        label: 'Max Single Breach',
        value: formatCurrency(portfolio.max_single_breach),
        icon: AlertTriangle,
        color: 'text-amber-600',
      },
      {
        label: 'Assets at Risk',
        value: portfolio.asset_count.toLocaleString(),
        icon: BarChart3,
        color: 'text-blue-600',
      },
    ];
  }, [portfolio]);

  function handleRetry() {
    void refetchPortfolio();
    void refetchTopRisks();
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Financial Risk Quantification"
          description="Quantify the financial impact of potential data breaches across your asset portfolio"
        />

        {isLoading ? (
          <LoadingSkeleton variant="card" count={4} />
        ) : hasError ? (
          <ErrorState
            message="Failed to load financial risk data"
            onRetry={handleRetry}
          />
        ) : (
          <>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
              {kpis.map((kpi) => {
                const Icon = kpi.icon;
                return (
                  <Card key={kpi.label}>
                    <CardContent className="flex items-center gap-4 p-5">
                      <div className={`rounded-lg bg-muted p-2.5 ${kpi.color}`}>
                        <Icon className="h-5 w-5" />
                      </div>
                      <div>
                        <p className="text-xs text-muted-foreground">{kpi.label}</p>
                        <p className="text-2xl font-bold tabular-nums">{kpi.value}</p>
                      </div>
                    </CardContent>
                  </Card>
                );
              })}
            </div>

            <div className="rounded-xl border bg-card">
              <div className="border-b px-5 py-4">
                <h3 className="text-sm font-semibold">Top Financial Risks</h3>
                <p className="text-xs text-muted-foreground">
                  Highest-impact assets ranked by estimated breach cost and annual expected loss
                </p>
              </div>
              <div className="overflow-x-auto">
                {topRisks.length === 0 ? (
                  <div className="flex flex-col items-center justify-center py-12 text-center">
                    <BarChart3 className="mb-3 h-8 w-8 text-muted-foreground" />
                    <p className="text-sm font-medium">No financial risk data available</p>
                    <p className="text-xs text-muted-foreground">
                      Run a DSPM financial impact analysis to generate risk quantification data.
                    </p>
                  </div>
                ) : (
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b bg-muted/50">
                        <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">
                          Asset
                        </th>
                        <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground">
                          Breach Cost
                        </th>
                        <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground">
                          Cost per Record
                        </th>
                        <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground">
                          Records
                        </th>
                        <th className="px-5 py-3 text-center text-xs font-medium text-muted-foreground">
                          Breach Probability
                        </th>
                        <th className="px-5 py-3 text-right text-xs font-medium text-muted-foreground">
                          Annual Expected Loss
                        </th>
                        <th className="px-5 py-3 text-left text-xs font-medium text-muted-foreground">
                          Methodology
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y">
                      {topRisks.map((risk) => {
                        const prob = risk.breach_probability_annual;
                        const pctStr = `${(prob * 100).toFixed(1)}%`;
                        return (
                          <tr
                            key={risk.id}
                            className="transition-colors hover:bg-muted/30"
                          >
                            <td className="px-5 py-3 font-medium" title={risk.data_asset_id}>
                              {shortenAssetId(risk.data_asset_id)}
                            </td>
                            <td className="px-5 py-3 text-right tabular-nums">
                              {currencyFormatter.format(risk.estimated_breach_cost)}
                            </td>
                            <td className="px-5 py-3 text-right tabular-nums">
                              {currencyFormatter.format(risk.cost_per_record)}
                            </td>
                            <td className="px-5 py-3 text-right tabular-nums">
                              {risk.record_count.toLocaleString()}
                            </td>
                            <td className="px-5 py-3 text-center">
                              <Badge
                                variant={probabilityBadgeVariant(prob)}
                                className={probabilityColor(prob)}
                              >
                                {pctStr}
                              </Badge>
                            </td>
                            <td className="px-5 py-3 text-right tabular-nums font-medium">
                              {currencyFormatter.format(risk.annual_expected_loss)}
                            </td>
                            <td className="px-5 py-3">
                              <Badge variant="outline" className="text-xs capitalize">
                                {risk.methodology.replace(/_/g, ' ')}
                              </Badge>
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                )}
              </div>
            </div>
          </>
        )}
      </div>
    </PermissionRedirect>
  );
}
