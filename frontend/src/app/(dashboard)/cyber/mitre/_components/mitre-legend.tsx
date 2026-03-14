'use client';

export function MitreLegend() {
  return (
    <div className="flex flex-wrap items-center gap-x-6 gap-y-2 text-xs text-muted-foreground">
      <div className="flex items-center gap-1.5">
        <span className="inline-block h-3 w-5 rounded border border-emerald-300 bg-emerald-50" />
        <span>Covered by active rules</span>
      </div>
      <div className="flex items-center gap-1.5">
        <span className="inline-block h-3 w-5 rounded border border-amber-300 bg-amber-50" />
        <span>Covered, but noisy</span>
      </div>
      <div className="flex items-center gap-1.5">
        <span className="inline-block h-3 w-5 rounded border border-red-300 bg-red-50" />
        <span>Threat-backed gap</span>
      </div>
      <div className="flex items-center gap-1.5">
        <span className="inline-block h-3 w-5 rounded border border-slate-200 bg-slate-50" />
        <span>Idle / not covered</span>
      </div>
    </div>
  );
}
