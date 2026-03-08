'use client';

const DAY_LABELS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];

export function ActivityHeatmap({ matrix }: { matrix: number[][] }) {
  const max = Math.max(...matrix.flat(), 1);

  return (
    <div className="space-y-2">
      <div className="grid grid-cols-[48px_repeat(24,minmax(0,1fr))] gap-1 text-[10px] text-muted-foreground">
        <div />
        {Array.from({ length: 24 }).map((_, hour) => (
          <div key={hour} className="text-center">{hour}</div>
        ))}
      </div>
      {DAY_LABELS.map((label, rowIndex) => (
        <div key={label} className="grid grid-cols-[48px_repeat(24,minmax(0,1fr))] gap-1">
          <div className="text-xs text-muted-foreground">{label}</div>
          {Array.from({ length: 24 }).map((_, hour) => {
            const value = matrix[rowIndex]?.[hour] ?? 0;
            const opacity = value === 0 ? 0.08 : Math.max(0.15, value / max);
            return (
              <div
                key={`${label}-${hour}`}
                className="h-4 rounded-[4px] bg-sky-600"
                style={{ opacity }}
                title={`${label} ${hour}:00 — ${value} events`}
              />
            );
          })}
        </div>
      ))}
    </div>
  );
}
