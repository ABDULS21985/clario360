"use client";
import { useEffect, useState } from "react";
import { ChartContainer } from "./chart-container";
import { cn } from "@/lib/utils";

interface GaugeChartProps {
  value: number;
  max?: number;
  thresholds?: { good: number; warning: number };
  label?: string;
  loading?: boolean;
  size?: number;
  showValue?: boolean;
  format?: "percentage" | "number";
  className?: string;
}

function getColor(value: number, max: number, thresholds: { good: number; warning: number }): string {
  const pct = (value / max) * 100;
  if (pct >= thresholds.good) return "#22c55e";    // green-500
  if (pct >= thresholds.warning) return "#eab308"; // yellow-500
  return "#ef4444"; // red-500
}

export function GaugeChart({
  value,
  max = 100,
  thresholds = { good: 80, warning: 60 },
  label,
  loading = false,
  size = 200,
  showValue = true,
  format = "percentage",
  className,
}: GaugeChartProps) {
  const [animatedValue, setAnimatedValue] = useState(0);

  useEffect(() => {
    const timer = setTimeout(() => setAnimatedValue(value), 50);
    return () => clearTimeout(timer);
  }, [value]);

  const radius = (size - 20) / 2;
  const circumference = Math.PI * radius; // half circle
  const pct = Math.min(Math.max(animatedValue / max, 0), 1);
  const strokeDashoffset = circumference * (1 - pct);
  const color = getColor(value, max, thresholds);
  const displayValue = format === "percentage"
    ? `${Math.round((value / max) * 100)}%`
    : value.toLocaleString();

  return (
    <ChartContainer loading={loading} empty={false} height={size} className={cn("flex items-center justify-center", className)}>
      <div className="relative flex flex-col items-center">
        <svg width={size} height={size / 2 + 20} style={{ overflow: "visible" }}>
          {/* Background arc */}
          <path
            d={`M 10 ${size / 2} A ${radius} ${radius} 0 0 1 ${size - 10} ${size / 2}`}
            fill="none"
            stroke="hsl(var(--muted))"
            strokeWidth={12}
            strokeLinecap="round"
          />
          {/* Value arc */}
          <path
            d={`M 10 ${size / 2} A ${radius} ${radius} 0 0 1 ${size - 10} ${size / 2}`}
            fill="none"
            stroke={color}
            strokeWidth={12}
            strokeLinecap="round"
            strokeDasharray={circumference}
            strokeDashoffset={strokeDashoffset}
            style={{ transition: "stroke-dashoffset 0.3s ease-out, stroke 0.3s ease-out" }}
          />
        </svg>
        {showValue && (
          <div className="absolute bottom-0 flex flex-col items-center">
            <span className="text-2xl font-bold" style={{ color }}>{displayValue}</span>
            {label && <span className="text-xs text-muted-foreground mt-1">{label}</span>}
          </div>
        )}
      </div>
    </ChartContainer>
  );
}
