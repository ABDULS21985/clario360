'use client';

import { useRealtimeData } from '@/hooks/use-realtime-data';
import { API_ENDPOINTS } from '@/lib/constants';
import { TrendingDown, TrendingUp, Minus } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { ExposureScore } from '@/types/cyber';

const GRADE_CONFIG: Record<string, { color: string; ring: string; bg: string }> = {
  A: { color: 'text-green-600', ring: 'stroke-green-500', bg: 'bg-green-50 dark:bg-green-950/30' },
  B: { color: 'text-blue-600', ring: 'stroke-blue-500', bg: 'bg-blue-50 dark:bg-blue-950/30' },
  C: { color: 'text-yellow-600', ring: 'stroke-yellow-500', bg: 'bg-yellow-50 dark:bg-yellow-950/30' },
  D: { color: 'text-orange-600', ring: 'stroke-orange-500', bg: 'bg-orange-50 dark:bg-orange-950/30' },
  F: { color: 'text-red-600', ring: 'stroke-red-500', bg: 'bg-red-50 dark:bg-red-950/30' },
};

export function ExposureScoreGauge() {
  const { data: envelope, isLoading } = useRealtimeData<{ data: ExposureScore }>(
    API_ENDPOINTS.CYBER_CTEM_EXPOSURE_SCORE,
    { pollInterval: 120000 },
  );

  if (isLoading) {
    return <div className="h-40 animate-pulse rounded-xl bg-muted" />;
  }

  const score = envelope?.data;
  if (!score) return null;

  const grade = score.grade ?? 'C';
  const config = GRADE_CONFIG[grade] ?? GRADE_CONFIG.C;
  const radius = 54;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (score.score / 100) * circumference;

  // Backend scoring engine returns: 'worsening' (score up = bad), 'improving' (score down = good), 'stable'
  const TrendIcon = score.trend === 'worsening'
    ? TrendingUp
    : score.trend === 'improving'
    ? TrendingDown
    : Minus;

  const trendColor = score.trend === 'worsening'
    ? 'text-red-500'
    : score.trend === 'improving'
    ? 'text-green-500'
    : 'text-muted-foreground';

  return (
    <div className={cn('flex flex-col items-center rounded-xl border p-6', config.bg)}>
      <p className="mb-3 text-sm font-semibold">Exposure Score</p>
      <div className="relative">
        <svg width="140" height="140" className="-rotate-90">
          <circle cx="70" cy="70" r={radius} fill="none" stroke="currentColor" strokeOpacity={0.1} strokeWidth={10} />
          <circle
            cx="70" cy="70" r={radius}
            fill="none"
            className={config.ring}
            strokeWidth={10}
            strokeLinecap="round"
            strokeDasharray={circumference}
            strokeDashoffset={offset}
            style={{ transition: 'stroke-dashoffset 0.8s ease' }}
          />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className={cn('text-4xl font-bold tabular-nums', config.color)}>{score.score}</span>
          <span className={cn('text-2xl font-black', config.color)}>{grade}</span>
        </div>
      </div>
      <div className={cn('mt-2 flex items-center gap-1 text-sm', trendColor)}>
        <TrendIcon className="h-4 w-4" />
        <span>{Math.abs(score.trend_delta).toFixed(1)} pts</span>
      </div>
    </div>
  );
}
