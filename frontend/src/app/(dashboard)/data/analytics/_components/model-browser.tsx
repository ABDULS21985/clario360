'use client';

import { Search } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { type DataModel } from '@/lib/data-suite';
import { getClassificationBadge } from '@/lib/data-suite/utils';

interface ModelBrowserProps {
  models: DataModel[];
  selectedModelId: string | null;
  search: string;
  onSearch: (value: string) => void;
  onSelectModel: (modelId: string) => void;
  onAddColumn: (columnName: string) => void;
}

export function ModelBrowser({
  models,
  selectedModelId,
  search,
  onSearch,
  onSelectModel,
  onAddColumn,
}: ModelBrowserProps) {
  const filteredModels = models.filter((model) => {
    const lowered = search.trim().toLowerCase();
    if (!lowered) {
      return true;
    }
    return (
      model.name.toLowerCase().includes(lowered) ||
      model.display_name.toLowerCase().includes(lowered) ||
      model.schema_definition.some((field) => field.name.toLowerCase().includes(lowered))
    );
  });

  return (
    <div className="space-y-4 rounded-lg border bg-card p-4">
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
        <Input className="pl-9" value={search} onChange={(event) => onSearch(event.target.value)} placeholder="Search models or columns..." />
      </div>

      <ScrollArea className="h-[720px] pr-3">
        <div className="space-y-3">
          {filteredModels.map((model) => {
            const classification = getClassificationBadge(model.data_classification);
            const selected = selectedModelId === model.id;
            return (
              <div key={model.id} className={`rounded-lg border ${selected ? 'border-primary bg-primary/5' : ''}`}>
                <button
                  type="button"
                  className="w-full p-4 text-left"
                  onClick={() => onSelectModel(model.id)}
                >
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <div className="font-medium">{model.display_name || model.name}</div>
                      <div className="mt-1 text-xs text-muted-foreground">
                        {model.field_count} fields {model.contains_pii ? `• ${model.pii_columns.length} PII columns` : ''}
                      </div>
                    </div>
                    <Badge variant="outline" className={classification.className}>
                      {classification.label}
                    </Badge>
                  </div>
                </button>
                {selected ? (
                  <div className="border-t px-4 py-3">
                    <div className="space-y-2">
                      {model.schema_definition.map((field) => (
                        <button
                          key={field.name}
                          type="button"
                          className="flex w-full items-center justify-between rounded-md px-2 py-1.5 text-left text-sm hover:bg-background"
                          onClick={() => onAddColumn(field.name)}
                        >
                          <span>{field.name}</span>
                          <span className="text-xs text-muted-foreground">
                            {field.data_type}{field.pii_type ? ` • ${field.pii_type}` : ''}
                          </span>
                        </button>
                      ))}
                    </div>
                  </div>
                ) : null}
              </div>
            );
          })}
        </div>
      </ScrollArea>
    </div>
  );
}
