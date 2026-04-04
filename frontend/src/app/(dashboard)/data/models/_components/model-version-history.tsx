'use client';

import { Badge } from '@/components/ui/badge';
import { type DataModel } from '@/lib/data-suite';
import { formatMaybeDateTime } from '@/lib/data-suite/utils';

interface ModelVersionHistoryProps {
  versions: DataModel[];
  currentModelId: string;
}

export function ModelVersionHistory({
  versions,
  currentModelId,
}: ModelVersionHistoryProps) {
  if (versions.length === 0) {
    return (
      <div className="rounded-lg border border-dashed p-6 text-sm text-muted-foreground">
        No historical versions are available for this model.
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {versions.map((version) => (
        <div key={version.id} className="rounded-lg border px-4 py-3">
          <div className="flex items-start justify-between gap-3">
            <div>
              <div className="font-medium">
                Version {version.version} • {version.display_name || version.name}
              </div>
              <div className="mt-1 text-xs text-muted-foreground">
                {version.field_count} fields • updated {formatMaybeDateTime(version.updated_at)}
              </div>
            </div>
            {version.id === currentModelId ? <Badge variant="outline">Current</Badge> : null}
          </div>
        </div>
      ))}
    </div>
  );
}
