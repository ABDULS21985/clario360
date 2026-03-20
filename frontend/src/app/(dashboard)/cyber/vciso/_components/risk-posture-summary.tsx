'use client';
import { TrendingDown, TrendingUp, Minus } from 'lucide-react';
import type { RiskPostureSummary } from '@/types/cyber';

function gradeColor(grade: string): string {
  if (grade === 'A' || grade === 'B') return 'text-green-600';
  if (grade === 'C') return 'text-amber-500';
  return 'text-red-600';
}

function gradeBackground(grade: string): string {
  if (grade === 'A' || grade === 'B') return 'bg-green-50 border-green-200';
  if (grade === 'C') return 'bg-amber-50 border-amber-200';
  return 'bg-red-50 border-red-200';
}

function componentBarColor(value: number): string {
  if (value <= 30) return 'bg-green-500';
  if (value <= 60) return 'bg-amber-500';
  return 'bg-red-500';
}

function formatComponentLabel(key: string): string {
  return key
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

export function RiskPostureSummary({ posture }: { posture: RiskPostureSummary }) {
  const isUp = posture.trend === 'increasing';
  const isDown = posture.trend === 'decreasing';

  return (
    <div className="rounded-lg border bg-white p-6 space-y-6">
      {/* Grade and score row */}
      <div className="flex items-center gap-6">
        <div
          className={`flex h-20 w-20 items-center justify-center rounded-xl border-2 text-5xl font-bold ${gradeBackground(posture.grade)} ${gradeColor(posture.grade)}`}
        >
          {posture.grade}
        </div>
        <div className="flex flex-col gap-1">
          <span className="text-sm text-muted-foreground font-medium uppercase tracking-wide">
            Risk Score
          </span>
          <span className="text-4xl font-bold text-foreground">
            {posture.overall_score}
          </span>
          {/* Trend indicator */}
          <div className="flex items-center gap-1 text-sm font-medium">
            {isUp ? (
              <TrendingUp className="h-4 w-4 text-red-500" />
            ) : isDown ? (
              <TrendingDown className="h-4 w-4 text-green-500" />
            ) : (
              <Minus className="h-4 w-4 text-muted-foreground" />
            )}
            <span
              className={
                isUp
                  ? 'text-red-500'
                  : isDown
                  ? 'text-green-600'
                  : 'text-muted-foreground'
              }
            >
              {posture.trend_delta > 0 ? '+' : ''}
              {posture.trend_delta.toFixed(1)}
            </span>
            <span className="text-muted-foreground font-normal">vs last period</span>
          </div>
        </div>
      </div>

      {/* Component bars */}
      {Object.keys(posture.components).length > 0 && (
        <div className="space-y-3">
          <p className="text-sm font-semibold text-foreground">Risk Components</p>
          {Object.entries(posture.components).map(([key, value]) => (
            <div key={key} className="space-y-1">
              <div className="flex justify-between text-xs text-muted-foreground">
                <span>{formatComponentLabel(key)}</span>
                <span className="font-medium tabular-nums">{value}</span>
              </div>
              <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                <div
                  className={`h-2 rounded-full transition-all duration-500 ${componentBarColor(value)}`}
                  style={{ width: `${Math.min(value, 100)}%` }}
                />
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
