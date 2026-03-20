'use client';

import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { type DataModel } from '@/lib/data-suite';
import { getClassificationBadge } from '@/lib/data-suite/utils';

interface ModelSchemaViewerProps {
  model: DataModel;
}

export function ModelSchemaViewer({
  model,
}: ModelSchemaViewerProps) {
  return (
    <div className="rounded-lg border">
      <ScrollArea className="h-[520px]">
        <table className="min-w-full text-sm">
          <thead className="sticky top-0 z-10 bg-background">
            <tr className="border-b">
              <th className="px-3 py-2 text-left font-medium">Field</th>
              <th className="px-3 py-2 text-left font-medium">Type</th>
              <th className="px-3 py-2 text-left font-medium">Nullable</th>
              <th className="px-3 py-2 text-left font-medium">PII</th>
              <th className="px-3 py-2 text-left font-medium">Classification</th>
              <th className="px-3 py-2 text-left font-medium">Description</th>
            </tr>
          </thead>
          <tbody>
            {model.schema_definition.map((field) => {
              const classification = getClassificationBadge(field.classification);
              return (
                <tr key={field.name} className="border-b align-top">
                  <td className="px-3 py-2 font-medium">{field.name}</td>
                  <td className="px-3 py-2 text-muted-foreground">
                    {field.native_type} ({field.data_type})
                  </td>
                  <td className="px-3 py-2 text-muted-foreground">{field.nullable ? 'Yes' : 'No'}</td>
                  <td className="px-3 py-2 text-muted-foreground">{field.pii_type ?? '—'}</td>
                  <td className="px-3 py-2">
                    <Badge variant="outline" className={classification.className}>
                      {classification.label}
                    </Badge>
                  </td>
                  <td className="px-3 py-2 text-muted-foreground">{field.description || '—'}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </ScrollArea>
    </div>
  );
}
