'use client';

import { AlertTriangle, ArrowRight, Minus, TrendingDown, TrendingUp } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import type { CTIExecutiveSnapshot } from '@/types/cti';

interface CTIRiskSummaryWidgetProps {
  snapshot: CTIExecutiveSnapshot | null;
  isLoading: boolean;
  onViewDetails: () => void;
}

function riskTone(score: number): string {
  if (score >= 81) return 'from-rose-500/15 to-rose-900/10 border-rose-500/30';
  if (score >= 61) return 'from-orange-500/15 to-orange-900/10 border-orange-500/30';
  if (score >= 31) return 'from-amber-400/15 to-amber-900/10 border-amber-500/30';
  return 'from-emerald-500/15 to-emerald-900/10 border-emerald-500/30';
}

function trendMeta(direction?: string | null) {
  switch (direction) {
    case 'increasing':
      return { icon: TrendingUp, tone: 'text-rose-500' };
    case 'decreasing':
      return { icon: TrendingDown, tone: 'text-emerald-500' };
    default:
      return { icon: Minus, tone: 'text-muted-foreground' };
  }
}

export function CTIRiskSummaryWidget({
  snapshot,
  isLoading,
  onViewDetails,
}: CTIRiskSummaryWidgetProps) {
  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-4 w-36" />
          <Skeleton className="h-3 w-48" />
        </CardHeader>
        <CardContent className="space-y-4">
          <Skeleton className="h-14 w-28" />
          <div className="grid grid-cols-2 gap-3">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!snapshot) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">CTI Risk Summary</CardTitle>
          <CardDescription>Threat intelligence risk data is currently unavailable.</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  const score = Number(snapshot.risk_score_overall ?? 0);
  const trend = trendMeta(snapshot.trend_direction);
  const TrendIcon = trend.icon;

  return (
    <Card className={cn('border bg-gradient-to-br', riskTone(score))}>
      <CardHeader className="flex flex-row items-start justify-between gap-3 space-y-0">
        <div>
          <CardTitle className="text-base">CTI Risk Summary</CardTitle>
          <CardDescription>Executive threat posture from the live CTI aggregation pipeline.</CardDescription>
        </div>
        <AlertTriangle className="h-5 w-5 text-muted-foreground" />
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="flex items-end justify-between gap-4">
          <div>
            <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Risk Score</p>
            <p className="text-4xl font-semibold tracking-[-0.06em]">{score.toFixed(1)}</p>
          </div>
          <div className={cn('inline-flex items-center gap-1 rounded-full px-3 py-1 text-sm font-medium', trend.tone)}>
            <TrendIcon className="h-4 w-4" />
            <span>{snapshot.trend_percentage.toFixed(1)}%</span>
          </div>
        </div>

        <div className="grid grid-cols-3 gap-3">
          <div className="rounded-lg border bg-background/60 px-3 py-3">
            <p className="text-[11px] uppercase tracking-[0.16em] text-muted-foreground">Events 24h</p>
            <p className="mt-1 text-xl font-semibold tabular-nums">{snapshot.total_events_24h.toLocaleString()}</p>
          </div>
          <div className="rounded-lg border bg-background/60 px-3 py-3">
            <p className="text-[11px] uppercase tracking-[0.16em] text-muted-foreground">MTTD</p>
            <p className="mt-1 text-xl font-semibold tabular-nums">{(snapshot.mean_time_to_detect_hours ?? 0).toFixed(1)}h</p>
          </div>
          <div className="rounded-lg border bg-background/60 px-3 py-3">
            <p className="text-[11px] uppercase tracking-[0.16em] text-muted-foreground">MTTR</p>
            <p className="mt-1 text-xl font-semibold tabular-nums">{(snapshot.mean_time_to_respond_hours ?? 0).toFixed(1)}h</p>
          </div>
        </div>

        <Button variant="ghost" size="sm" className="px-0" onClick={onViewDetails}>
          View Full CTI
          <ArrowRight className="ml-1 h-4 w-4" />
        </Button>
      </CardContent>
    </Card>
  );
}
