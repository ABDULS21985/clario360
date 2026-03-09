'use client';
import type { ComplianceFramework } from '@/types/cyber';

const statusConfig = {
  compliant: {
    label: 'Compliant',
    className: 'bg-green-100 text-green-700',
    barClassName: 'bg-green-500',
  },
  partial: {
    label: 'Partial',
    className: 'bg-amber-100 text-amber-700',
    barClassName: 'bg-amber-500',
  },
  non_compliant: {
    label: 'Non-Compliant',
    className: 'bg-red-100 text-red-700',
    barClassName: 'bg-red-500',
  },
} satisfies Record<
  ComplianceFramework['status'],
  { label: string; className: string; barClassName: string }
>;

export function ComplianceStatusSection({
  frameworks,
}: {
  frameworks: ComplianceFramework[];
}) {
  const list = Array.isArray(frameworks) ? frameworks : [];

  if (list.length === 0) {
    return (
      <p className="text-sm text-muted-foreground py-4 text-center">
        No compliance frameworks configured.
      </p>
    );
  }

  return (
    <div className="space-y-4">
      {list.map((fw) => {
        const config = statusConfig[fw.status];

        return (
          <div key={fw.name} className="rounded-lg border bg-white p-4 space-y-3">
            {/* Name + status badge */}
            <div className="flex items-center justify-between gap-3">
              <span className="text-sm font-semibold text-foreground">{fw.name}</span>
              <span
                className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${config.className}`}
              >
                {config.label}
              </span>
            </div>

            {/* Coverage progress bar */}
            <div className="space-y-1">
              <div className="flex justify-between text-xs text-muted-foreground">
                <span>Coverage</span>
                <span className="font-medium tabular-nums">
                  {fw.coverage_percent.toFixed(0)}%
                </span>
              </div>
              <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                <div
                  className={`h-2 rounded-full transition-all duration-500 ${config.barClassName}`}
                  style={{ width: `${Math.min(fw.coverage_percent, 100)}%` }}
                />
              </div>
            </div>

            {/* Controls passed */}
            <p className="text-xs text-muted-foreground">
              <span className="font-medium text-foreground">{fw.controls_passed}</span>
              {' / '}
              <span className="font-medium text-foreground">{fw.controls_total}</span>
              {' controls passed'}
            </p>
          </div>
        );
      })}
    </div>
  );
}
