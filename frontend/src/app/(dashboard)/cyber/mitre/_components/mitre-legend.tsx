'use client';

export function MitreLegend() {
  return (
    <div className="flex flex-wrap items-center gap-x-6 gap-y-2 text-xs text-muted-foreground">
      <div className="flex items-center gap-1.5">
        <span className="inline-block h-3 w-5 rounded border border-green-400 bg-green-100" />
        <span>Active (rule + alerts)</span>
      </div>
      <div className="flex items-center gap-1.5">
        <span className="inline-block h-3 w-5 rounded border border-yellow-300 bg-yellow-50" />
        <span>Passive (rule, no alerts)</span>
      </div>
      <div className="flex items-center gap-1.5">
        <span className="inline-block h-3 w-5 rounded border border-red-200 bg-red-50" />
        <span>Gap (no rule)</span>
      </div>
      <div className="flex items-center gap-1.5">
        <span className="inline-block h-3 w-5 rounded border border-gray-200 bg-gray-50" />
        <span>Not applicable</span>
      </div>
    </div>
  );
}
