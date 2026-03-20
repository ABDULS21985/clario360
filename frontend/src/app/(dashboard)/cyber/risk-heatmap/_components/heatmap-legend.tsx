'use client';

const BUCKETS = [
  { label: '0', from: 0, to: 0 },
  { label: '1–5', from: 1, to: 5 },
  { label: '6–20', from: 6, to: 20 },
  { label: '21–50', from: 21, to: 50 },
  { label: '50+', from: 51, to: Infinity },
];

const BUCKET_OPACITIES = [0, 0.2, 0.45, 0.7, 0.95];

interface HeatmapLegendProps {
  baseColor?: string;
}

export function HeatmapLegend({ baseColor = 'bg-slate-500' }: HeatmapLegendProps) {
  return (
    <div className="flex items-center gap-3 text-xs text-muted-foreground">
      <span className="font-medium">Intensity:</span>
      <div className="flex items-center gap-1.5">
        {BUCKETS.map((b, i) => (
          <div key={b.label} className="flex flex-col items-center gap-0.5">
            <div
              className={`h-4 w-7 rounded border ${baseColor}`}
              style={{ opacity: BUCKET_OPACITIES[i] + (i === 0 ? 0.05 : 0) }}
            />
            <span>{b.label}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
