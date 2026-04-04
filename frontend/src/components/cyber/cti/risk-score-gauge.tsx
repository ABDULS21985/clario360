'use client';

import { TrendingUp, TrendingDown, Minus } from 'lucide-react';

interface RiskScoreGaugeProps {
  score: number;
  trend: 'increasing' | 'stable' | 'decreasing';
  size?: number;
}

function scoreColor(score: number): string {
  if (score >= 81) return '#FF3B5C';
  if (score >= 61) return '#FF8C42';
  if (score >= 31) return '#FFD93D';
  return '#4ADE80';
}

export function RiskScoreGauge({ score, trend, size = 120 }: RiskScoreGaugeProps) {
  const r = (size - 12) / 2;
  const cx = size / 2;
  const cy = size / 2;
  const circumference = Math.PI * r; // semi-circle
  const pct = Math.min(score, 100) / 100;
  const offset = circumference * (1 - pct);
  const color = scoreColor(score);

  const TrendIcon = trend === 'increasing' ? TrendingUp
    : trend === 'decreasing' ? TrendingDown
    : Minus;

  return (
    <div className="flex flex-col items-center gap-1">
      <svg width={size} height={size / 2 + 20} viewBox={`0 0 ${size} ${size / 2 + 20}`}>
        {/* Background arc */}
        <path
          d={`M 6 ${cy} A ${r} ${r} 0 0 1 ${size - 6} ${cy}`}
          fill="none"
          stroke="currentColor"
          strokeWidth={8}
          className="text-muted-foreground/20"
        />
        {/* Value arc */}
        <path
          d={`M 6 ${cy} A ${r} ${r} 0 0 1 ${size - 6} ${cy}`}
          fill="none"
          stroke={color}
          strokeWidth={8}
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          className="transition-all duration-700"
        />
        <text x={cx} y={cy - 4} textAnchor="middle" className="fill-foreground text-2xl font-bold" style={{ fontSize: 24 }}>
          {Math.round(score)}
        </text>
        <text x={cx} y={cy + 14} textAnchor="middle" className="fill-muted-foreground" style={{ fontSize: 10 }}>
          Risk Score
        </text>
      </svg>
      <span className="flex items-center gap-1 text-xs text-muted-foreground">
        <TrendIcon className="h-3 w-3" />
        {trend}
      </span>
    </div>
  );
}
