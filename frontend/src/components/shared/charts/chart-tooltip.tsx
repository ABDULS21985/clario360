import { cn } from "@/lib/utils";

interface ChartTooltipProps {
  active?: boolean;
  payload?: Array<{
    name: string;
    value: number;
    color: string;
    dataKey: string;
  }>;
  label?: string | number;
  labelFormatter?: (label: string | number) => string;
  valueFormatter?: (value: number, name: string) => string;
  className?: string;
}

export function ChartTooltip({
  active,
  payload,
  label,
  labelFormatter,
  valueFormatter,
  className,
}: ChartTooltipProps) {
  if (!active || !payload || payload.length === 0) return null;
  const displayLabel = label !== undefined && labelFormatter ? labelFormatter(label) : label;

  return (
    <div className={cn("rounded-lg border border-border bg-background shadow-md p-3 text-sm min-w-[120px]", className)}>
      {displayLabel !== undefined && (
        <p className="font-medium mb-2 text-foreground">{String(displayLabel)}</p>
      )}
      <div className="space-y-1">
        {payload.map((entry) => (
          <div key={entry.dataKey} className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-1.5">
              <div className="h-2.5 w-2.5 rounded-full shrink-0" style={{ backgroundColor: entry.color }} aria-hidden />
              <span className="text-muted-foreground">{entry.name}</span>
            </div>
            <span className="font-medium text-foreground">
              {valueFormatter ? valueFormatter(entry.value, entry.name) : entry.value.toLocaleString()}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
