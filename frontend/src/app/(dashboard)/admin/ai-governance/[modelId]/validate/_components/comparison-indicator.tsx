'use client';

import { ArrowDownRight, ArrowUpRight, Minus } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { formatPercentage } from '@/lib/format';

interface ComparisonIndicatorProps {
  delta?: number | null;
  inverse?: boolean;
  label?: string;
}

export function ComparisonIndicator({
  delta,
  inverse = false,
  label = 'vs production',
}: ComparisonIndicatorProps) {
  if (delta === null || delta === undefined || Number.isNaN(delta)) {
    return <Badge variant="outline">No baseline</Badge>;
  }

  const isImprovement = inverse ? delta < 0 : delta > 0;
  const isRegression = inverse ? delta > 0 : delta < 0;
  const Icon = isImprovement ? ArrowUpRight : isRegression ? ArrowDownRight : Minus;
  const variant = isImprovement ? 'success' : isRegression ? 'destructive' : 'outline';
  const sign = delta > 0 ? '+' : '';

  return (
    <Badge variant={variant} className="gap-1.5 normal-case tracking-normal">
      <Icon className="h-3.5 w-3.5" />
      {sign}
      {formatPercentage(delta, 1)} {label}
    </Badge>
  );
}
