'use client';

import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { Badge } from '@/components/ui/badge';
import { type DarkDataAsset } from '@/lib/data-suite';
import { formatMaybeBytes, formatMaybeDateTime, getClassificationBadge } from '@/lib/data-suite/utils';

interface DarkDataDetailPanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  asset: DarkDataAsset | null;
}

export function DarkDataDetailPanel({
  open,
  onOpenChange,
  asset,
}: DarkDataDetailPanelProps) {
  const classification = getClassificationBadge(asset?.inferred_classification);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-2xl">
        <SheetHeader>
          <SheetTitle>{asset?.name ?? 'Dark data asset'}</SheetTitle>
          <SheetDescription>{asset?.reason ?? 'Select an asset to inspect governance risk.'}</SheetDescription>
        </SheetHeader>
        {asset ? (
          <div className="mt-6 space-y-4">
            <div className="flex flex-wrap gap-2">
              <Badge variant="outline">{asset.asset_type}</Badge>
              <Badge variant="outline">{asset.governance_status}</Badge>
              <Badge variant="outline" className={classification.className}>
                {classification.label}
              </Badge>
            </div>
            <div className="grid gap-3 md:grid-cols-2">
              <Metric label="Risk Score" value={`${asset.risk_score.toFixed(0)}%`} />
              <Metric label="Estimated Size" value={formatMaybeBytes(asset.estimated_size_bytes)} />
              <Metric label="Columns" value={asset.column_count?.toLocaleString() ?? '—'} />
              <Metric label="Last Accessed" value={formatMaybeDateTime(asset.last_accessed_at)} />
            </div>
            <div className="rounded-lg border">
              <pre className="overflow-x-auto p-4 text-xs">{JSON.stringify(asset.risk_factors, null, 2)}</pre>
            </div>
          </div>
        ) : null}
      </SheetContent>
    </Sheet>
  );
}

function Metric({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  return (
    <div className="rounded-lg border bg-muted/20 p-3">
      <div className="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1 text-sm font-medium">{value}</div>
    </div>
  );
}
